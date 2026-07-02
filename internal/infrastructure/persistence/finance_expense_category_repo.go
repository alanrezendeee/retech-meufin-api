package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"gorm.io/gorm"
)

// FinanceExpenseCategoryModel mapeia a tabela finance_expense_categories.
type FinanceExpenseCategoryModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index:idx_finance_expense_categories_workspace"`
	Slug        string    `gorm:"size:40;not null"`
	Name        string    `gorm:"size:80;not null"`
	GroupSlug   string    `gorm:"column:group_slug;size:30;not null"`
	Active      bool      `gorm:"not null;default:true"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
	DeletedAt   gorm.DeletedAt
}

func (FinanceExpenseCategoryModel) TableName() string { return "finance_expense_categories" }

type FinanceExpenseCategoryRepository struct {
	db *gorm.DB
}

func NewFinanceExpenseCategoryRepository(db *gorm.DB) *FinanceExpenseCategoryRepository {
	return &FinanceExpenseCategoryRepository{db: db}
}

func (r *FinanceExpenseCategoryRepository) Create(ctx context.Context, c *dom.ExpenseCategory) error {
	model := expenseCategoryToModel(c)
	return mapFinanceErr(r.db.WithContext(ctx).Create(&model).Error)
}

func (r *FinanceExpenseCategoryRepository) CreateBatch(ctx context.Context, cs []*dom.ExpenseCategory) error {
	if len(cs) == 0 {
		return nil
	}
	models := make([]FinanceExpenseCategoryModel, len(cs))
	for i := range cs {
		models[i] = expenseCategoryToModel(cs[i])
	}
	return mapFinanceErr(r.db.WithContext(ctx).Create(&models).Error)
}

func (r *FinanceExpenseCategoryRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.ExpenseCategory, error) {
	var m FinanceExpenseCategoryModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	return modelToExpenseCategory(&m), nil
}

func (r *FinanceExpenseCategoryRepository) ExistsBySlug(ctx context.Context, workspaceID uuid.UUID, slug string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&FinanceExpenseCategoryModel{}).
		Where("workspace_id = ? AND slug = ? AND active", workspaceID, slug).
		Count(&count).Error
	if err != nil {
		return false, mapFinanceErr(err)
	}
	return count > 0, nil
}

func (r *FinanceExpenseCategoryRepository) Update(ctx context.Context, c *dom.ExpenseCategory) error {
	model := expenseCategoryToModel(c)
	res := r.db.WithContext(ctx).Model(&FinanceExpenseCategoryModel{}).
		Where("id = ? AND workspace_id = ?", c.ID, c.WorkspaceID).
		Updates(map[string]any{
			"name":       model.Name,
			"group_slug": model.GroupSlug,
			"active":     model.Active,
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

func (r *FinanceExpenseCategoryRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&FinanceExpenseCategoryModel{})
	if res.Error != nil {
		return mapFinanceErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *FinanceExpenseCategoryRepository) List(ctx context.Context, workspaceID uuid.UUID) ([]dom.ExpenseCategory, error) {
	var rows []FinanceExpenseCategoryModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ?", workspaceID).
		Order("name ASC").
		Find(&rows).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	out := make([]dom.ExpenseCategory, len(rows))
	for i := range rows {
		out[i] = *modelToExpenseCategory(&rows[i])
	}
	return out, nil
}

// --- conversões ---

func expenseCategoryToModel(c *dom.ExpenseCategory) FinanceExpenseCategoryModel {
	return FinanceExpenseCategoryModel{
		ID:          c.ID,
		WorkspaceID: c.WorkspaceID,
		Slug:        c.Slug,
		Name:        c.Name,
		GroupSlug:   c.GroupSlug,
		Active:      c.Active,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func modelToExpenseCategory(m *FinanceExpenseCategoryModel) *dom.ExpenseCategory {
	return &dom.ExpenseCategory{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		Slug:        m.Slug,
		Name:        m.Name,
		GroupSlug:   m.GroupSlug,
		Active:      m.Active,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
