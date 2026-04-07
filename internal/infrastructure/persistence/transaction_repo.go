package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/ledger"
	"gorm.io/gorm"
)

type TransactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, t *dom.Transaction) error {
	m := transactionToModel(t)
	return mapLedgerErr(r.db.WithContext(ctx).Create(&m).Error)
}

func (r *TransactionRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Transaction, error) {
	var m TransactionModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapLedgerErr(err)
	}
	return modelToTransaction(&m), nil
}

func (r *TransactionRepository) Update(ctx context.Context, t *dom.Transaction) error {
	m := transactionToModel(t)
	return mapLedgerErr(r.db.WithContext(ctx).Save(&m).Error)
}

func (r *TransactionRepository) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&TransactionModel{})
	if res.Error != nil {
		return mapLedgerErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *TransactionRepository) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]dom.Transaction, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&TransactionModel{}).Where("workspace_id = ?", workspaceID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []TransactionModel
	if err := q.Order("occurred_at DESC, created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]dom.Transaction, len(rows))
	for i := range rows {
		out[i] = *modelToTransaction(&rows[i])
	}
	return out, total, nil
}

func (r *TransactionRepository) SumOutflowsByCategoryInMonth(ctx context.Context, workspaceID, categoryID uuid.UUID, year, month int) (int64, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	var sum int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(amount_cents), 0)
		FROM transactions
		WHERE workspace_id = ? AND category_id = ? AND flow = ?
		  AND occurred_at >= ? AND occurred_at < ?`,
		workspaceID, categoryID, string(dom.FlowOut), start, end,
	).Scan(&sum).Error
	return sum, err
}

func transactionToModel(t *dom.Transaction) TransactionModel {
	return TransactionModel{
		ID: t.ID, WorkspaceID: t.WorkspaceID, AccountID: t.AccountID, CategoryID: t.CategoryID,
		AmountCents: t.AmountCents, Flow: string(t.Flow), Description: t.Description,
		OccurredAt: t.OccurredAt, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func modelToTransaction(m *TransactionModel) *dom.Transaction {
	return &dom.Transaction{
		ID: m.ID, WorkspaceID: m.WorkspaceID, AccountID: m.AccountID, CategoryID: m.CategoryID,
		AmountCents: m.AmountCents, Flow: dom.Flow(m.Flow), Description: m.Description,
		OccurredAt: m.OccurredAt, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
