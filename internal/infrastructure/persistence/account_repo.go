package persistence

import (
	"context"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/ledger"
	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Create(ctx context.Context, a *dom.Account) error {
	m := accountToModel(a)
	return mapLedgerErr(r.db.WithContext(ctx).Create(&m).Error)
}

func (r *AccountRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Account, error) {
	var m FinancialAccountModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapLedgerErr(err)
	}
	return modelToAccount(&m), nil
}

func (r *AccountRepository) Update(ctx context.Context, a *dom.Account) error {
	m := accountToModel(a)
	return mapLedgerErr(r.db.WithContext(ctx).Save(&m).Error)
}

func (r *AccountRepository) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&FinancialAccountModel{})
	if res.Error != nil {
		return mapLedgerErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *AccountRepository) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]dom.Account, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&FinancialAccountModel{}).Where("workspace_id = ?", workspaceID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []FinancialAccountModel
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]dom.Account, len(rows))
	for i := range rows {
		out[i] = *modelToAccount(&rows[i])
	}
	return out, total, nil
}

func accountToModel(a *dom.Account) FinancialAccountModel {
	return FinancialAccountModel{
		ID: a.ID, WorkspaceID: a.WorkspaceID, Name: a.Name, Currency: a.Currency,
		CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}

func modelToAccount(m *FinancialAccountModel) *dom.Account {
	return &dom.Account{
		ID: m.ID, WorkspaceID: m.WorkspaceID, Name: m.Name, Currency: m.Currency,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
