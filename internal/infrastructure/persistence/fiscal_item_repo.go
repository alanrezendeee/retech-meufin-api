package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"gorm.io/gorm"
)

// FinanceFiscalItemModel é o registro GORM de finance_fiscal_items.
type FinanceFiscalItemModel struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID   uuid.UUID `gorm:"type:uuid;not null;index:idx_finance_fiscal_items_entry"`
	EntryID       uuid.UUID `gorm:"type:uuid;not null;index:idx_finance_fiscal_items_entry"`
	DocumentID    uuid.UUID `gorm:"type:uuid;not null"`
	Description   string    `gorm:"type:text;not null"`
	QuantityMilli int64     `gorm:"column:quantity_milli;not null;default:0"`
	UnitCents     int64     `gorm:"column:unit_cents;not null;default:0"`
	AmountCents   int64     `gorm:"column:amount_cents;not null"`
	Category      *string   `gorm:"size:30"`
	UnitOfMeasure *string   `gorm:"column:unit_of_measure;size:10"`
	CreatedAt     time.Time `gorm:"not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}

func (FinanceFiscalItemModel) TableName() string { return "finance_fiscal_items" }

type FiscalItemRepository struct{ db *gorm.DB }

func NewFiscalItemRepository(db *gorm.DB) *FiscalItemRepository {
	return &FiscalItemRepository{db: db}
}

func (r *FiscalItemRepository) CreateBatch(ctx context.Context, items []*dom.FiscalItem) error {
	if len(items) == 0 {
		return nil
	}
	models := make([]FinanceFiscalItemModel, len(items))
	for i := range items {
		models[i] = fiscalItemToModel(items[i])
	}
	return mapFinanceErr(r.db.WithContext(ctx).Create(&models).Error)
}

func (r *FiscalItemRepository) ListByEntry(ctx context.Context, workspaceID, entryID uuid.UUID) ([]dom.FiscalItem, error) {
	var rows []FinanceFiscalItemModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND entry_id = ?", workspaceID, entryID).
		Order("created_at ASC, id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	out := make([]dom.FiscalItem, len(rows))
	for i := range rows {
		out[i] = *modelToFiscalItem(&rows[i])
	}
	return out, nil
}

func (r *FiscalItemRepository) DeleteByEntry(ctx context.Context, workspaceID, entryID uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("workspace_id = ? AND entry_id = ?", workspaceID, entryID).
		Delete(&FinanceFiscalItemModel{})
	return mapFinanceErr(res.Error)
}

func (r *FiscalItemRepository) ReassignEntry(ctx context.Context, workspaceID, fromEntryID, toEntryID uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&FinanceFiscalItemModel{}).
		Where("workspace_id = ? AND entry_id = ?", workspaceID, fromEntryID).
		Update("entry_id", toEntryID)
	return mapFinanceErr(res.Error)
}

func fiscalItemToModel(i *dom.FiscalItem) FinanceFiscalItemModel {
	return FinanceFiscalItemModel{
		ID:            i.ID,
		WorkspaceID:   i.WorkspaceID,
		EntryID:       i.EntryID,
		DocumentID:    i.DocumentID,
		Description:   i.Description,
		QuantityMilli: i.QuantityMilli,
		UnitCents:     i.UnitCents,
		AmountCents:   i.AmountCents,
		Category:      i.Category,
		UnitOfMeasure: i.UnitOfMeasure,
		CreatedAt:     i.CreatedAt,
		UpdatedAt:     i.UpdatedAt,
	}
}

func modelToFiscalItem(m *FinanceFiscalItemModel) *dom.FiscalItem {
	return &dom.FiscalItem{
		ID:            m.ID,
		WorkspaceID:   m.WorkspaceID,
		EntryID:       m.EntryID,
		DocumentID:    m.DocumentID,
		Description:   m.Description,
		QuantityMilli: m.QuantityMilli,
		UnitCents:     m.UnitCents,
		AmountCents:   m.AmountCents,
		Category:      m.Category,
		UnitOfMeasure: m.UnitOfMeasure,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}
