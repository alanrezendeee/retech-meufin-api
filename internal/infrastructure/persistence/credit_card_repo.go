package persistence

import (
	"context"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"gorm.io/gorm"
)

type CreditCardRepository struct {
	db *gorm.DB
}

func NewCreditCardRepository(db *gorm.DB) *CreditCardRepository {
	return &CreditCardRepository{db: db}
}

func (r *CreditCardRepository) Create(ctx context.Context, c *dom.CreditCard) error {
	model := creditCardToModel(c)
	return mapFinanceErr(r.db.WithContext(ctx).Create(&model).Error)
}

func (r *CreditCardRepository) Update(ctx context.Context, c *dom.CreditCard) error {
	model := creditCardToModel(c)
	res := r.db.WithContext(ctx).Model(&CreditCardModel{}).
		Where("id = ? AND workspace_id = ?", c.ID, c.WorkspaceID).
		Updates(map[string]any{
			"name":        model.Name,
			"brand":       model.Brand,
			"closing_day": model.ClosingDay,
			"due_day":     model.DueDay,
			"active":      model.Active,
			"notes":       model.Notes,
			"updated_at":  model.UpdatedAt,
		})
	if res.Error != nil {
		return mapFinanceErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *CreditCardRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&CreditCardModel{})
	if res.Error != nil {
		return mapFinanceErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *CreditCardRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.CreditCard, error) {
	var m CreditCardModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	return modelToCreditCard(&m), nil
}

func (r *CreditCardRepository) List(ctx context.Context, workspaceID uuid.UUID, filter dom.CreditCardFilter, limit, offset int) ([]dom.CreditCard, int64, error) {
	base := r.db.WithContext(ctx).Model(&CreditCardModel{}).Where("workspace_id = ?", workspaceID)
	if filter.Query != "" {
		base = base.Where("name ILIKE ?", "%"+filter.Query+"%")
	}
	if filter.Active != nil {
		base = base.Where("active = ?", *filter.Active)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, mapFinanceErr(err)
	}
	var rows []CreditCardModel
	if err := base.Order("name ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, mapFinanceErr(err)
	}
	out := make([]dom.CreditCard, len(rows))
	for i := range rows {
		out[i] = *modelToCreditCard(&rows[i])
	}
	return out, total, nil
}

// --- conversões ---

func creditCardToModel(c *dom.CreditCard) CreditCardModel {
	return CreditCardModel{
		ID:          c.ID,
		WorkspaceID: c.WorkspaceID,
		Name:        c.Name,
		Brand:       c.Brand,
		ClosingDay:  c.ClosingDay,
		DueDay:      c.DueDay,
		Active:      c.Active,
		Notes:       c.Notes,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func modelToCreditCard(m *CreditCardModel) *dom.CreditCard {
	return &dom.CreditCard{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		Name:        m.Name,
		Brand:       m.Brand,
		ClosingDay:  m.ClosingDay,
		DueDay:      m.DueDay,
		Active:      m.Active,
		Notes:       m.Notes,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
