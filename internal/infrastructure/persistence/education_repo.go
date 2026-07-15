package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/education"
	"gorm.io/gorm"
)

// EducationRepository implementa dom.Repository com GORM/Postgres.
type EducationRepository struct {
	db *gorm.DB
}

func NewEducationRepository(db *gorm.DB) *EducationRepository {
	return &EducationRepository{db: db}
}

// ─── Enrollments ─────────────────────────────────────────────────────────────

func (r *EducationRepository) CreateEnrollment(ctx context.Context, e *dom.SchoolEnrollment) error {
	m := enrollmentToModel(e)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("education: create enrollment: %w", err)
	}
	return nil
}

func (r *EducationRepository) GetEnrollment(ctx context.Context, workspaceID, id uuid.UUID) (*dom.SchoolEnrollment, error) {
	var m SchoolEnrollmentModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	e := modelToEnrollment(m)
	return &e, nil
}

func (r *EducationRepository) ListEnrollments(ctx context.Context, workspaceID uuid.UUID, p dom.ListEnrollmentsParams) ([]dom.SchoolEnrollment, error) {
	q := r.db.WithContext(ctx).Model(&SchoolEnrollmentModel{}).
		Where("workspace_id = ?", workspaceID.String())
	if p.MemberID != nil {
		q = q.Where("member_id = ?", p.MemberID.String())
	}
	if p.SchoolYear != nil {
		q = q.Where("school_year = ?", *p.SchoolYear)
	}
	var models []SchoolEnrollmentModel
	if err := q.Order("school_year DESC, created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.SchoolEnrollment, len(models))
	for i, m := range models {
		out[i] = modelToEnrollment(m)
	}
	return out, nil
}

func (r *EducationRepository) UpdateEnrollment(ctx context.Context, e *dom.SchoolEnrollment) error {
	m := enrollmentToModel(e)
	return r.db.WithContext(ctx).Omit("CreatedAt").Save(&m).Error
}

func (r *EducationRepository) DeleteEnrollment(ctx context.Context, workspaceID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		Delete(&SchoolEnrollmentModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

// ─── Supply lists ────────────────────────────────────────────────────────────

func (r *EducationRepository) CreateList(ctx context.Context, l *dom.SchoolSupplyList) error {
	m := listToModel(l)
	if err := r.db.WithContext(ctx).Omit("Items").Create(&m).Error; err != nil {
		return fmt.Errorf("education: create list: %w", err)
	}
	return nil
}

func (r *EducationRepository) GetList(ctx context.Context, workspaceID, id uuid.UUID) (*dom.SchoolSupplyList, error) {
	var m SchoolSupplyListModel
	err := r.db.WithContext(ctx).
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	l := modelToList(m)
	return &l, nil
}

func (r *EducationRepository) ListSupplyLists(ctx context.Context, workspaceID uuid.UUID, p dom.ListSupplyListsParams) ([]dom.SchoolSupplyList, error) {
	q := r.db.WithContext(ctx).Model(&SchoolSupplyListModel{}).
		Where("school_supply_lists.workspace_id = ?", workspaceID.String())
	if p.EnrollmentID != nil {
		q = q.Where("school_supply_lists.enrollment_id = ?", p.EnrollmentID.String())
	}
	if p.Status != "" {
		q = q.Where("school_supply_lists.status = ?", p.Status)
	}
	if p.SchoolYear != nil {
		q = q.Joins("JOIN school_enrollments ON school_enrollments.id = school_supply_lists.enrollment_id").
			Where("school_enrollments.school_year = ?", *p.SchoolYear)
	}
	var models []SchoolSupplyListModel
	if err := q.Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Order("school_supply_lists.created_at DESC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.SchoolSupplyList, len(models))
	for i, m := range models {
		out[i] = modelToList(m)
	}
	return out, nil
}

func (r *EducationRepository) UpdateList(ctx context.Context, l *dom.SchoolSupplyList) error {
	m := listToModel(l)
	return r.db.WithContext(ctx).Omit("Items", "CreatedAt").Save(&m).Error
}

func (r *EducationRepository) DeleteList(ctx context.Context, workspaceID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		Delete(&SchoolSupplyListModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

// ─── Supply items ────────────────────────────────────────────────────────────

func (r *EducationRepository) CreateItem(ctx context.Context, i *dom.SchoolSupplyItem) error {
	m := itemToModel(i)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("education: create item: %w", err)
	}
	return nil
}

func (r *EducationRepository) GetItem(ctx context.Context, workspaceID, id uuid.UUID) (*dom.SchoolSupplyItem, error) {
	var m SchoolSupplyItemModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	it := modelToItem(m)
	return &it, nil
}

func (r *EducationRepository) UpdateItem(ctx context.Context, i *dom.SchoolSupplyItem) error {
	m := itemToModel(i)
	return r.db.WithContext(ctx).Omit("CreatedAt").Save(&m).Error
}

func (r *EducationRepository) DeleteItem(ctx context.Context, workspaceID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		Delete(&SchoolSupplyItemModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

// ─── Dashboard raw ───────────────────────────────────────────────────────────

func (r *EducationRepository) AllEnrollments(ctx context.Context, workspaceID uuid.UUID) ([]dom.SchoolEnrollment, error) {
	var models []SchoolEnrollmentModel
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ?", workspaceID.String()).
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.SchoolEnrollment, len(models))
	for i, m := range models {
		out[i] = modelToEnrollment(m)
	}
	return out, nil
}

func (r *EducationRepository) AllLists(ctx context.Context, workspaceID uuid.UUID) ([]dom.SchoolSupplyList, error) {
	var models []SchoolSupplyListModel
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ?", workspaceID.String()).
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.SchoolSupplyList, len(models))
	for i, m := range models {
		out[i] = modelToList(m)
	}
	return out, nil
}

func (r *EducationRepository) AllItems(ctx context.Context, workspaceID uuid.UUID) ([]dom.SchoolSupplyItem, error) {
	var models []SchoolSupplyItemModel
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ?", workspaceID.String()).
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.SchoolSupplyItem, len(models))
	for i, m := range models {
		out[i] = modelToItem(m)
	}
	return out, nil
}

// MemberNames retorna id → full_name dos membros da família do workspace.
func (r *EducationRepository) MemberNames(ctx context.Context, workspaceID uuid.UUID) (map[string]string, error) {
	type row struct {
		ID       string
		FullName string
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Table("health_family_members").
		Select("id, full_name").
		Where("workspace_id = ?", workspaceID.String()).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, rw := range rows {
		out[rw.ID] = rw.FullName
	}
	return out, nil
}
