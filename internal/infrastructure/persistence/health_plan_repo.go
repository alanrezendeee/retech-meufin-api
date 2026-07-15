package persistence

import (
	"context"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"gorm.io/gorm"
)

type HealthPlanRepository struct {
	db *gorm.DB
}

func NewHealthPlanRepository(db *gorm.DB) *HealthPlanRepository {
	return &HealthPlanRepository{db: db}
}

func (r *HealthPlanRepository) Create(ctx context.Context, p *dom.Plan) error {
	model := planToModel(p)
	model.Members = planMembersToModels(p.Members)
	return mapHealthErr(r.db.WithContext(ctx).Create(&model).Error)
}

func (r *HealthPlanRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Plan, error) {
	var m HealthPlanModel
	err := r.db.WithContext(ctx).
		Preload("Members").
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	return modelToPlan(&m), nil
}

func (r *HealthPlanRepository) Update(ctx context.Context, p *dom.Plan) error {
	model := planToModel(p)
	res := r.db.WithContext(ctx).Model(&HealthPlanModel{}).
		Where("id = ? AND workspace_id = ?", p.ID, p.WorkspaceID).
		Updates(map[string]any{
			"name":              model.Name,
			"operator":          model.Operator,
			"plan_type":         model.PlanType,
			"ans_code":          model.AnsCode,
			"monthly_fee_cents": model.MonthlyFeeCents,
			"coverage_notes":    model.CoverageNotes,
			"active":            model.Active,
			"updated_at":        model.UpdatedAt,
		})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *HealthPlanRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&HealthPlanModel{})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *HealthPlanRepository) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]dom.Plan, int64, error) {
	base := r.db.WithContext(ctx).Model(&HealthPlanModel{}).Where("workspace_id = ?", workspaceID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	var rows []HealthPlanModel
	if err := base.Preload("Members").Order("name ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	out := make([]dom.Plan, len(rows))
	for i := range rows {
		out[i] = *modelToPlan(&rows[i])
	}
	return out, total, nil
}

func (r *HealthPlanRepository) ReplaceMembers(ctx context.Context, workspaceID, planID uuid.UUID, members []dom.PlanMember) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// garante que o plano existe no workspace
		var count int64
		if err := tx.Model(&HealthPlanModel{}).
			Where("id = ? AND workspace_id = ?", planID, workspaceID).
			Count(&count).Error; err != nil {
			return mapHealthErr(err)
		}
		if count == 0 {
			return dom.ErrNotFound
		}
		if err := tx.Where("plan_id = ? AND workspace_id = ?", planID, workspaceID).
			Delete(&HealthPlanMemberModel{}).Error; err != nil {
			return mapHealthErr(err)
		}
		if len(members) == 0 {
			return nil
		}
		models := planMembersToModels(members)
		return mapHealthErr(tx.Create(&models).Error)
	})
}

// --- conversões ---

func planToModel(p *dom.Plan) HealthPlanModel {
	return HealthPlanModel{
		ID:              p.ID,
		WorkspaceID:     p.WorkspaceID,
		Name:            p.Name,
		Operator:        p.Operator,
		PlanType:        string(p.PlanType),
		AnsCode:         p.AnsCode,
		MonthlyFeeCents: p.MonthlyFeeCents,
		CoverageNotes:   p.CoverageNotes,
		Active:          p.Active,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

func planMemberToModel(m *dom.PlanMember) HealthPlanMemberModel {
	return HealthPlanMemberModel{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		PlanID:      m.PlanID,
		MemberID:    m.MemberID,
		CardNumber:  m.CardNumber,
		Holder:      m.Holder,
		CreatedAt:   m.CreatedAt,
	}
}

func planMembersToModels(members []dom.PlanMember) []HealthPlanMemberModel {
	out := make([]HealthPlanMemberModel, len(members))
	for i := range members {
		out[i] = planMemberToModel(&members[i])
	}
	return out
}

func modelToPlan(m *HealthPlanModel) *dom.Plan {
	out := &dom.Plan{
		ID:              m.ID,
		WorkspaceID:     m.WorkspaceID,
		Name:            m.Name,
		Operator:        m.Operator,
		PlanType:        dom.PlanType(m.PlanType),
		AnsCode:         m.AnsCode,
		MonthlyFeeCents: m.MonthlyFeeCents,
		CoverageNotes:   m.CoverageNotes,
		Active:          m.Active,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
	for i := range m.Members {
		out.Members = append(out.Members, *modelToPlanMember(&m.Members[i]))
	}
	return out
}

func modelToPlanMember(m *HealthPlanMemberModel) *dom.PlanMember {
	return &dom.PlanMember{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		PlanID:      m.PlanID,
		MemberID:    m.MemberID,
		CardNumber:  m.CardNumber,
		Holder:      m.Holder,
		CreatedAt:   m.CreatedAt,
	}
}
