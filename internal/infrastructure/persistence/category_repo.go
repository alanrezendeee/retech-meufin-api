package persistence

import (
	"context"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/ledger"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(ctx context.Context, c *dom.Category) error {
	m := categoryToModel(c)
	return mapLedgerErr(r.db.WithContext(ctx).Create(&m).Error)
}

func (r *CategoryRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Category, error) {
	var m CategoryModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapLedgerErr(err)
	}
	return modelToCategory(&m), nil
}

func (r *CategoryRepository) Update(ctx context.Context, c *dom.Category) error {
	m := categoryToModel(c)
	return mapLedgerErr(r.db.WithContext(ctx).Save(&m).Error)
}

func (r *CategoryRepository) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&CategoryModel{})
	if res.Error != nil {
		return mapLedgerErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *CategoryRepository) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]dom.Category, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&CategoryModel{}).Where("workspace_id = ?", workspaceID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []CategoryModel
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]dom.Category, len(rows))
	for i := range rows {
		out[i] = *modelToCategory(&rows[i])
	}
	return out, total, nil
}

func categoryToModel(c *dom.Category) CategoryModel {
	return CategoryModel{
		ID: c.ID, WorkspaceID: c.WorkspaceID, Name: c.Name, Kind: string(c.Kind),
		ParentID: c.ParentID, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func modelToCategory(m *CategoryModel) *dom.Category {
	return &dom.Category{
		ID: m.ID, WorkspaceID: m.WorkspaceID, Name: m.Name, Kind: dom.CategoryKind(m.Kind),
		ParentID: m.ParentID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
