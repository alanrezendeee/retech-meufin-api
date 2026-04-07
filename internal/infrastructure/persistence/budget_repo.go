package persistence

import (
	"context"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/budget"
	"gorm.io/gorm"
)

type BudgetRepository struct {
	db *gorm.DB
}

func NewBudgetRepository(db *gorm.DB) *BudgetRepository {
	return &BudgetRepository{db: db}
}

func (r *BudgetRepository) Create(ctx context.Context, b *dom.Budget) error {
	m := budgetToModel(b)
	return mapBudgetErr(r.db.WithContext(ctx).Create(&m).Error)
}

func (r *BudgetRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Budget, error) {
	var m BudgetModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapBudgetErr(err)
	}
	return modelToBudget(&m), nil
}

func (r *BudgetRepository) Update(ctx context.Context, b *dom.Budget) error {
	m := budgetToModel(b)
	return mapBudgetErr(r.db.WithContext(ctx).Save(&m).Error)
}

func (r *BudgetRepository) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&BudgetModel{})
	if res.Error != nil {
		return mapBudgetErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *BudgetRepository) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]dom.Budget, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&BudgetModel{}).Where("workspace_id = ?", workspaceID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []BudgetModel
	if err := q.Order("year DESC, month DESC, created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]dom.Budget, len(rows))
	for i := range rows {
		out[i] = *modelToBudget(&rows[i])
	}
	return out, total, nil
}

func (r *BudgetRepository) GetByCategoryPeriod(ctx context.Context, workspaceID, categoryID uuid.UUID, year, month int) (*dom.Budget, error) {
	var m BudgetModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND category_id = ? AND year = ? AND month = ?", workspaceID, categoryID, year, month).
		First(&m).Error
	if err != nil {
		return nil, mapBudgetErr(err)
	}
	return modelToBudget(&m), nil
}

func (r *BudgetRepository) ListByWorkspaceMonth(ctx context.Context, workspaceID uuid.UUID, year, month int) ([]dom.Budget, error) {
	var rows []BudgetModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND year = ? AND month = ?", workspaceID, year, month).
		Order("category_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]dom.Budget, len(rows))
	for i := range rows {
		out[i] = *modelToBudget(&rows[i])
	}
	return out, nil
}

func budgetToModel(b *dom.Budget) BudgetModel {
	return BudgetModel{
		ID: b.ID, WorkspaceID: b.WorkspaceID, CategoryID: b.CategoryID,
		Year: b.Year, Month: b.Month, LimitCents: b.LimitCents,
		CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
	}
}

func modelToBudget(m *BudgetModel) *dom.Budget {
	return &dom.Budget{
		ID: m.ID, WorkspaceID: m.WorkspaceID, CategoryID: m.CategoryID,
		Year: m.Year, Month: m.Month, LimitCents: m.LimitCents,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
