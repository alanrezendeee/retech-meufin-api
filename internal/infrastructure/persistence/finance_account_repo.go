package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"gorm.io/gorm"
)

// FinanceAccountModel mapeia a tabela finance_accounts.
type FinanceAccountModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index:idx_finance_accounts_workspace"`
	Name        string    `gorm:"size:255;not null"`
	Kind        string    `gorm:"size:20;not null"`
	BankName    *string   `gorm:"size:255"`
	Active      bool      `gorm:"not null;default:true"`
	Notes       *string   `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
	DeletedAt   gorm.DeletedAt
}

func (FinanceAccountModel) TableName() string { return "finance_accounts" }

type FinanceAccountRepository struct {
	db *gorm.DB
}

func NewFinanceAccountRepository(db *gorm.DB) *FinanceAccountRepository {
	return &FinanceAccountRepository{db: db}
}

func (r *FinanceAccountRepository) Create(ctx context.Context, a *dom.Account) error {
	model := financeAccountToModel(a)
	return mapFinanceErr(r.db.WithContext(ctx).Create(&model).Error)
}

func (r *FinanceAccountRepository) Update(ctx context.Context, a *dom.Account) error {
	model := financeAccountToModel(a)
	res := r.db.WithContext(ctx).Model(&FinanceAccountModel{}).
		Where("id = ? AND workspace_id = ?", a.ID, a.WorkspaceID).
		Updates(map[string]any{
			"name":       model.Name,
			"kind":       model.Kind,
			"bank_name":  model.BankName,
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

func (r *FinanceAccountRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&FinanceAccountModel{})
	if res.Error != nil {
		return mapFinanceErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *FinanceAccountRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Account, error) {
	var m FinanceAccountModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	return modelToFinanceAccount(&m), nil
}

func (r *FinanceAccountRepository) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]dom.Account, int64, error) {
	base := r.db.WithContext(ctx).Model(&FinanceAccountModel{}).Where("workspace_id = ?", workspaceID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, mapFinanceErr(err)
	}
	var rows []FinanceAccountModel
	if err := base.Order("name ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, mapFinanceErr(err)
	}
	out := make([]dom.Account, len(rows))
	for i := range rows {
		out[i] = *modelToFinanceAccount(&rows[i])
	}
	return out, total, nil
}

// --- conversões ---

func financeAccountToModel(a *dom.Account) FinanceAccountModel {
	return FinanceAccountModel{
		ID:          a.ID,
		WorkspaceID: a.WorkspaceID,
		Name:        a.Name,
		Kind:        string(a.Kind),
		BankName:    a.BankName,
		Active:      a.Active,
		Notes:       a.Notes,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

func modelToFinanceAccount(m *FinanceAccountModel) *dom.Account {
	return &dom.Account{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		Name:        m.Name,
		Kind:        dom.AccountKind(m.Kind),
		BankName:    m.BankName,
		Active:      m.Active,
		Notes:       m.Notes,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
