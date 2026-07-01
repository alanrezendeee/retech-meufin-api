package persistence

import (
	"context"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"gorm.io/gorm"
)

type IncomeSourceRepository struct {
	db *gorm.DB
}

func NewIncomeSourceRepository(db *gorm.DB) *IncomeSourceRepository {
	return &IncomeSourceRepository{db: db}
}

func (r *IncomeSourceRepository) Create(ctx context.Context, s *dom.IncomeSource) error {
	model := incomeSourceToModel(s)
	return mapFinanceErr(r.db.WithContext(ctx).Create(&model).Error)
}

func (r *IncomeSourceRepository) Update(ctx context.Context, s *dom.IncomeSource) error {
	model := incomeSourceToModel(s)
	res := r.db.WithContext(ctx).Model(&IncomeSourceModel{}).
		Where("id = ? AND workspace_id = ?", s.ID, s.WorkspaceID).
		Updates(map[string]any{
			"name":       model.Name,
			"kind":       model.Kind,
			"active":     model.Active,
			"notes":      model.Notes,
			"updated_at": model.UpdatedAt,
		})
	if res.Error != nil {
		return mapFinanceErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *IncomeSourceRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&IncomeSourceModel{})
	if res.Error != nil {
		return mapFinanceErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *IncomeSourceRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.IncomeSource, error) {
	var m IncomeSourceModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	return modelToIncomeSource(&m), nil
}

func (r *IncomeSourceRepository) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]dom.IncomeSource, int64, error) {
	base := r.db.WithContext(ctx).Model(&IncomeSourceModel{}).Where("workspace_id = ?", workspaceID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, mapFinanceErr(err)
	}
	var rows []IncomeSourceModel
	if err := base.Order("name ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, mapFinanceErr(err)
	}
	out := make([]dom.IncomeSource, len(rows))
	for i := range rows {
		out[i] = *modelToIncomeSource(&rows[i])
	}
	return out, total, nil
}

// --- conversões ---

func incomeSourceToModel(s *dom.IncomeSource) IncomeSourceModel {
	return IncomeSourceModel{
		ID:          s.ID,
		WorkspaceID: s.WorkspaceID,
		Name:        s.Name,
		Kind:        string(s.Kind),
		Active:      s.Active,
		Notes:       s.Notes,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

func modelToIncomeSource(m *IncomeSourceModel) *dom.IncomeSource {
	return &dom.IncomeSource{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		Name:        m.Name,
		Kind:        dom.IncomeSourceKind(m.Kind),
		Active:      m.Active,
		Notes:       m.Notes,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
