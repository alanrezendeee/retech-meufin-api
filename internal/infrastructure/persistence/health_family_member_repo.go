package persistence

import (
	"context"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"gorm.io/gorm"
)

type HealthFamilyMemberRepository struct {
	db *gorm.DB
}

func NewHealthFamilyMemberRepository(db *gorm.DB) *HealthFamilyMemberRepository {
	return &HealthFamilyMemberRepository{db: db}
}

func (r *HealthFamilyMemberRepository) Create(ctx context.Context, f *dom.FamilyMember) error {
	m := familyMemberToModel(f)
	return mapHealthErr(r.db.WithContext(ctx).Create(&m).Error)
}

func (r *HealthFamilyMemberRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.FamilyMember, error) {
	var m HealthFamilyMemberModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	return modelToFamilyMember(&m), nil
}

func (r *HealthFamilyMemberRepository) Update(ctx context.Context, f *dom.FamilyMember) error {
	res := r.db.WithContext(ctx).Model(&HealthFamilyMemberModel{}).
		Where("id = ? AND workspace_id = ?", f.ID, f.WorkspaceID).
		Updates(map[string]any{
			"full_name":    f.FullName,
			"relationship": f.Relationship,
			"birth_date":   f.BirthDate,
			"gender":       f.Gender,
			"document":     f.Document,
			"notes":        f.Notes,
			"height_cm":    f.HeightCm,
			"weight_kg":    f.WeightKg,
			"active":       f.Active,
			"updated_at":   f.UpdatedAt,
		})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *HealthFamilyMemberRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&HealthFamilyMemberModel{})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *HealthFamilyMemberRepository) List(ctx context.Context, workspaceID uuid.UUID, filter dom.FamilyMemberFilter, limit, offset int) ([]dom.FamilyMember, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&HealthFamilyMemberModel{}).Where("workspace_id = ?", workspaceID)
	if filter.Query != "" {
		q = q.Where("full_name ILIKE ?", "%"+filter.Query+"%")
	}
	if filter.Relationship != "" {
		q = q.Where("relationship = ?", filter.Relationship)
	}
	if filter.Active != nil {
		q = q.Where("active = ?", *filter.Active)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	var rows []HealthFamilyMemberModel
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	out := make([]dom.FamilyMember, len(rows))
	for i := range rows {
		out[i] = *modelToFamilyMember(&rows[i])
	}
	return out, total, nil
}

// ListWithBirthDate retorna membros ativos com data de nascimento preenchida.
func (r *HealthFamilyMemberRepository) ListWithBirthDate(ctx context.Context, workspaceID uuid.UUID) ([]dom.FamilyMember, error) {
	var rows []HealthFamilyMemberModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND active = ? AND birth_date IS NOT NULL", workspaceID, true).
		Order("full_name ASC").
		Find(&rows).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	out := make([]dom.FamilyMember, len(rows))
	for i := range rows {
		out[i] = *modelToFamilyMember(&rows[i])
	}
	return out, nil
}

// --- conversões ---

func familyMemberToModel(f *dom.FamilyMember) HealthFamilyMemberModel {
	return HealthFamilyMemberModel{
		ID:           f.ID,
		WorkspaceID:  f.WorkspaceID,
		FullName:     f.FullName,
		Relationship: f.Relationship,
		BirthDate:    f.BirthDate,
		Gender:       f.Gender,
		Document:     f.Document,
		Notes:        f.Notes,
		HeightCm:     f.HeightCm,
		WeightKg:     f.WeightKg,
		Active:       f.Active,
		CreatedAt:    f.CreatedAt,
		UpdatedAt:    f.UpdatedAt,
	}
}

func modelToFamilyMember(m *HealthFamilyMemberModel) *dom.FamilyMember {
	return &dom.FamilyMember{
		ID:           m.ID,
		WorkspaceID:  m.WorkspaceID,
		FullName:     m.FullName,
		Relationship: m.Relationship,
		BirthDate:    m.BirthDate,
		Gender:       m.Gender,
		Document:     m.Document,
		Notes:        m.Notes,
		HeightCm:     m.HeightCm,
		WeightKg:     m.WeightKg,
		Active:       m.Active,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
