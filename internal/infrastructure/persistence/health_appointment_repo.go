package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"gorm.io/gorm"
)

type HealthAppointmentRepository struct {
	db *gorm.DB
}

func NewHealthAppointmentRepository(db *gorm.DB) *HealthAppointmentRepository {
	return &HealthAppointmentRepository{db: db}
}

func (r *HealthAppointmentRepository) Create(ctx context.Context, a *dom.Appointment) error {
	m := appointmentToModel(a)
	return mapHealthErr(r.db.WithContext(ctx).Create(&m).Error)
}

func (r *HealthAppointmentRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Appointment, error) {
	var m HealthAppointmentModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	return modelToAppointment(&m), nil
}

func (r *HealthAppointmentRepository) Update(ctx context.Context, a *dom.Appointment) error {
	m := appointmentToModel(a)
	res := r.db.WithContext(ctx).Model(&HealthAppointmentModel{}).
		Where("id = ? AND workspace_id = ?", a.ID, a.WorkspaceID).
		Updates(map[string]any{
			"family_member_id":  m.FamilyMemberID,
			"kind":              m.Kind,
			"specialty":         m.Specialty,
			"professional_name": m.ProfessionalName,
			"lab_id":            m.LabID,
			"exam_request_id":   m.ExamRequestID,
			"plan_id":           m.PlanID,
			"scheduled_at":      m.ScheduledAt,
			"status":            m.Status,
			"reason":            m.Reason,
			"outcome":           m.Outcome,
			"price_cents":       m.PriceCents,
			"covered_by_plan":   m.CoveredByPlan,
			"notes":             m.Notes,
			"updated_at":        m.UpdatedAt,
		})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *HealthAppointmentRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&HealthAppointmentModel{})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

// enrichedRow espelha as colunas retornadas pela query com joins.
type enrichedRow struct {
	HealthAppointmentModel
	MemberName string
	LabName    *string
	PlanName   *string
}

func (r *HealthAppointmentRepository) enrichedQuery(ctx context.Context, workspaceID uuid.UUID) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("health_appointments a").
		Select("a.*, fm.full_name AS member_name, l.name AS lab_name, p.name AS plan_name").
		Joins("JOIN health_family_members fm ON fm.id = a.family_member_id").
		Joins("LEFT JOIN health_labs l ON l.id = a.lab_id").
		Joins("LEFT JOIN health_plans p ON p.id = a.plan_id").
		Where("a.workspace_id = ? AND a.deleted_at IS NULL", workspaceID)
}

func applyAppointmentFilter(q *gorm.DB, f dom.AppointmentFilter) *gorm.DB {
	if f.FamilyMemberID != nil {
		q = q.Where("a.family_member_id = ?", *f.FamilyMemberID)
	}
	if f.Status != "" {
		q = q.Where("a.status = ?", string(f.Status))
	}
	if f.Kind != "" {
		q = q.Where("a.kind = ?", string(f.Kind))
	}
	if f.LabID != nil {
		q = q.Where("a.lab_id = ?", *f.LabID)
	}
	if f.PlanID != nil {
		q = q.Where("a.plan_id = ?", *f.PlanID)
	}
	if f.From != nil {
		q = q.Where("a.scheduled_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("a.scheduled_at <= ?", *f.To)
	}
	return q
}

func (r *HealthAppointmentRepository) List(ctx context.Context, workspaceID uuid.UUID, filter dom.AppointmentFilter, limit, offset int) ([]dom.AppointmentEnriched, int64, error) {
	// Contagem sem joins/select — todos os filtros incidem sobre "a".
	countQ := applyAppointmentFilter(
		r.db.WithContext(ctx).Table("health_appointments a").
			Where("a.workspace_id = ? AND a.deleted_at IS NULL", workspaceID),
		filter,
	)
	var total int64
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}

	q := applyAppointmentFilter(r.enrichedQuery(ctx, workspaceID), filter)
	var rows []enrichedRow
	if err := q.Order("a.scheduled_at DESC").Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	return enrichedRowsToDomain(rows), total, nil
}

func (r *HealthAppointmentRepository) Upcoming(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]dom.AppointmentEnriched, error) {
	q := r.enrichedQuery(ctx, workspaceID).
		Where("a.scheduled_at >= ? AND a.scheduled_at <= ?", from, to).
		Where("a.status IN ?", []string{string(dom.AppointmentStatusAgendada), string(dom.AppointmentStatusConfirmada)}).
		Order("a.scheduled_at ASC")
	var rows []enrichedRow
	if err := q.Scan(&rows).Error; err != nil {
		return nil, mapHealthErr(err)
	}
	return enrichedRowsToDomain(rows), nil
}

func (r *HealthAppointmentRepository) StatusCounts(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]dom.AgendaStatusCount, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Table("health_appointments a").
		Select("a.status, COUNT(*) AS count").
		Where("a.workspace_id = ? AND a.deleted_at IS NULL AND a.scheduled_at >= ? AND a.scheduled_at <= ?", workspaceID, from, to).
		Group("a.status").
		Scan(&rows).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	out := make([]dom.AgendaStatusCount, len(rows))
	for i := range rows {
		out[i] = dom.AgendaStatusCount{Status: rows[i].Status, Count: rows[i].Count}
	}
	return out, nil
}

func (r *HealthAppointmentRepository) RealizedSpendCents(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) (int64, error) {
	var total *int64
	err := r.db.WithContext(ctx).
		Table("health_appointments a").
		Select("COALESCE(SUM(a.price_cents), 0)").
		Where("a.workspace_id = ? AND a.deleted_at IS NULL AND a.status = ? AND a.scheduled_at >= ? AND a.scheduled_at <= ?",
			workspaceID, string(dom.AppointmentStatusRealizada), from, to).
		Scan(&total).Error
	if err != nil {
		return 0, mapHealthErr(err)
	}
	if total == nil {
		return 0, nil
	}
	return *total, nil
}

func (r *HealthAppointmentRepository) SpecialtyCounts(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]dom.AgendaSpecialtyCount, error) {
	type row struct {
		Specialty *string
		Count     int64
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Table("health_appointments a").
		Select("a.specialty, COUNT(*) AS count").
		Where("a.workspace_id = ? AND a.deleted_at IS NULL AND a.specialty IS NOT NULL AND a.scheduled_at >= ? AND a.scheduled_at <= ?", workspaceID, from, to).
		Group("a.specialty").
		Order("count DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	out := make([]dom.AgendaSpecialtyCount, 0, len(rows))
	for i := range rows {
		sp := ""
		if rows[i].Specialty != nil {
			sp = *rows[i].Specialty
		}
		out = append(out, dom.AgendaSpecialtyCount{Specialty: sp, Count: rows[i].Count})
	}
	return out, nil
}

func (r *HealthAppointmentRepository) MemberCounts(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]dom.AgendaMemberCount, error) {
	type row struct {
		MemberID   uuid.UUID
		MemberName string
		Count      int64
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Table("health_appointments a").
		Select("a.family_member_id AS member_id, fm.full_name AS member_name, COUNT(*) AS count").
		Joins("JOIN health_family_members fm ON fm.id = a.family_member_id").
		Where("a.workspace_id = ? AND a.deleted_at IS NULL AND a.scheduled_at >= ? AND a.scheduled_at <= ?", workspaceID, from, to).
		Group("a.family_member_id, fm.full_name").
		Order("count DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	out := make([]dom.AgendaMemberCount, len(rows))
	for i := range rows {
		out[i] = dom.AgendaMemberCount{MemberID: rows[i].MemberID, MemberName: rows[i].MemberName, Count: rows[i].Count}
	}
	return out, nil
}

func (r *HealthAppointmentRepository) ActivePlansMonthlyFeeCents(ctx context.Context, workspaceID uuid.UUID) (int64, error) {
	var total *int64
	err := r.db.WithContext(ctx).
		Table("health_plans p").
		Select("COALESCE(SUM(p.monthly_fee_cents), 0)").
		Where("p.workspace_id = ? AND p.deleted_at IS NULL AND p.active = true", workspaceID).
		Scan(&total).Error
	if err != nil {
		return 0, mapHealthErr(err)
	}
	if total == nil {
		return 0, nil
	}
	return *total, nil
}

// --- conversões ---

func appointmentToModel(a *dom.Appointment) HealthAppointmentModel {
	var specialty *string
	if a.Specialty != nil {
		s := string(*a.Specialty)
		specialty = &s
	}
	return HealthAppointmentModel{
		ID:               a.ID,
		WorkspaceID:      a.WorkspaceID,
		FamilyMemberID:   a.FamilyMemberID,
		Kind:             string(a.Kind),
		Specialty:        specialty,
		ProfessionalName: a.ProfessionalName,
		LabID:            a.LabID,
		ExamRequestID:    a.ExamRequestID,
		PlanID:           a.PlanID,
		ScheduledAt:      a.ScheduledAt,
		Status:           string(a.Status),
		Reason:           a.Reason,
		Outcome:          a.Outcome,
		PriceCents:       a.PriceCents,
		CoveredByPlan:    a.CoveredByPlan,
		Notes:            a.Notes,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
	}
}

func modelToAppointment(m *HealthAppointmentModel) *dom.Appointment {
	var specialty *dom.Specialty
	if m.Specialty != nil {
		s := dom.Specialty(*m.Specialty)
		specialty = &s
	}
	return &dom.Appointment{
		ID:               m.ID,
		WorkspaceID:      m.WorkspaceID,
		FamilyMemberID:   m.FamilyMemberID,
		Kind:             dom.AppointmentKind(m.Kind),
		Specialty:        specialty,
		ProfessionalName: m.ProfessionalName,
		LabID:            m.LabID,
		ExamRequestID:    m.ExamRequestID,
		PlanID:           m.PlanID,
		ScheduledAt:      m.ScheduledAt,
		Status:           dom.AppointmentStatus(m.Status),
		Reason:           m.Reason,
		Outcome:          m.Outcome,
		PriceCents:       m.PriceCents,
		CoveredByPlan:    m.CoveredByPlan,
		Notes:            m.Notes,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

func enrichedRowsToDomain(rows []enrichedRow) []dom.AppointmentEnriched {
	out := make([]dom.AppointmentEnriched, len(rows))
	for i := range rows {
		out[i] = dom.AppointmentEnriched{
			Appointment: *modelToAppointment(&rows[i].HealthAppointmentModel),
			MemberName:  rows[i].MemberName,
			LabName:     rows[i].LabName,
			PlanName:    rows[i].PlanName,
		}
	}
	return out
}
