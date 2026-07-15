package persistence

import (
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/warranty"
)

// ─── GORM models ─────────────────────────────────────────────────────────────

// WarrantyModel mapeia a tabela warranties.
type WarrantyModel struct {
	ID                        uuid.UUID  `gorm:"type:uuid;primaryKey;column:id"`
	WorkspaceID               uuid.UUID  `gorm:"type:uuid;column:workspace_id;index:idx_warranties_workspace"`
	ItemName                  string     `gorm:"column:item_name;size:200"`
	Category                  string     `gorm:"column:category;size:30"`
	Brand                     *string    `gorm:"column:brand;size:120"`
	Model                     *string    `gorm:"column:model;size:120"`
	SerialNumber              *string    `gorm:"column:serial_number;size:120"`
	Store                     *string    `gorm:"column:store;size:150"`
	SupplierName              *string    `gorm:"column:supplier_name;size:150"`
	PurchaseDate              time.Time  `gorm:"column:purchase_date;type:date"`
	PriceCents                *int64     `gorm:"column:price_cents"`
	InvoiceNumber             *string    `gorm:"column:invoice_number;size:80"`
	EntryID                   *uuid.UUID `gorm:"type:uuid;column:entry_id"`
	FiscalItemID              *uuid.UUID `gorm:"type:uuid;column:fiscal_item_id"`
	LegalWarrantyDays         int        `gorm:"column:legal_warranty_days"`
	ContractualWarrantyMonths int        `gorm:"column:contractual_warranty_months"`
	ExtendedWarrantyMonths    int        `gorm:"column:extended_warranty_months"`
	ExtendedProvider          *string    `gorm:"column:extended_provider;size:150"`
	ExtendedCostCents         int64      `gorm:"column:extended_cost_cents"`
	CoverageNotes             *string    `gorm:"column:coverage_notes"`
	Notes                     *string    `gorm:"column:notes"`
	Active                    bool       `gorm:"column:active"`
	CreatedAt                 time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt                 time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (WarrantyModel) TableName() string { return "warranties" }

// WarrantyDocumentModel mapeia a tabela warranty_documents.
type WarrantyDocumentModel struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;column:id"`
	WarrantyID       uuid.UUID `gorm:"type:uuid;column:warranty_id;index:idx_warranty_documents_warranty"`
	WorkspaceID      uuid.UUID `gorm:"type:uuid;column:workspace_id;index:idx_warranty_documents_workspace"`
	DocType          string    `gorm:"column:doc_type;size:30"`
	FileName         string    `gorm:"column:file_name;size:255"`
	OriginalFileName string    `gorm:"column:original_file_name;size:255"`
	ObjectKey        string    `gorm:"column:object_key;size:500"`
	ContentType      string    `gorm:"column:content_type;size:100"`
	SizeBytes        int64     `gorm:"column:size_bytes"`
	Notes            *string   `gorm:"column:notes"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (WarrantyDocumentModel) TableName() string { return "warranty_documents" }

// ─── conversões ──────────────────────────────────────────────────────────────

func warrantyToModel(w *dom.Warranty) WarrantyModel {
	return WarrantyModel{
		ID:                        w.ID,
		WorkspaceID:               w.WorkspaceID,
		ItemName:                  w.ItemName,
		Category:                  string(w.Category),
		Brand:                     w.Brand,
		Model:                     w.Model,
		SerialNumber:              w.SerialNumber,
		Store:                     w.Store,
		SupplierName:              w.SupplierName,
		PurchaseDate:              w.PurchaseDate,
		PriceCents:                w.PriceCents,
		InvoiceNumber:             w.InvoiceNumber,
		EntryID:                   w.EntryID,
		FiscalItemID:              w.FiscalItemID,
		LegalWarrantyDays:         w.LegalWarrantyDays,
		ContractualWarrantyMonths: w.ContractualWarrantyMonths,
		ExtendedWarrantyMonths:    w.ExtendedWarrantyMonths,
		ExtendedProvider:          w.ExtendedProvider,
		ExtendedCostCents:         w.ExtendedCostCents,
		CoverageNotes:             w.CoverageNotes,
		Notes:                     w.Notes,
		Active:                    w.Active,
		CreatedAt:                 w.CreatedAt,
		UpdatedAt:                 w.UpdatedAt,
	}
}

func modelToWarranty(m *WarrantyModel) dom.Warranty {
	return dom.Warranty{
		ID:                        m.ID,
		WorkspaceID:               m.WorkspaceID,
		ItemName:                  m.ItemName,
		Category:                  dom.Category(m.Category),
		Brand:                     m.Brand,
		Model:                     m.Model,
		SerialNumber:              m.SerialNumber,
		Store:                     m.Store,
		SupplierName:              m.SupplierName,
		PurchaseDate:              m.PurchaseDate,
		PriceCents:                m.PriceCents,
		InvoiceNumber:             m.InvoiceNumber,
		EntryID:                   m.EntryID,
		FiscalItemID:              m.FiscalItemID,
		LegalWarrantyDays:         m.LegalWarrantyDays,
		ContractualWarrantyMonths: m.ContractualWarrantyMonths,
		ExtendedWarrantyMonths:    m.ExtendedWarrantyMonths,
		ExtendedProvider:          m.ExtendedProvider,
		ExtendedCostCents:         m.ExtendedCostCents,
		CoverageNotes:             m.CoverageNotes,
		Notes:                     m.Notes,
		Active:                    m.Active,
		CreatedAt:                 m.CreatedAt,
		UpdatedAt:                 m.UpdatedAt,
	}
}

func warrantyDocumentToModel(d *dom.Document) WarrantyDocumentModel {
	return WarrantyDocumentModel{
		ID:               d.ID,
		WarrantyID:       d.WarrantyID,
		WorkspaceID:      d.WorkspaceID,
		DocType:          string(d.DocType),
		FileName:         d.FileName,
		OriginalFileName: d.OriginalFileName,
		ObjectKey:        d.ObjectKey,
		ContentType:      d.ContentType,
		SizeBytes:        d.SizeBytes,
		Notes:            d.Notes,
		CreatedAt:        d.CreatedAt,
	}
}

func modelToWarrantyDocument(m *WarrantyDocumentModel) dom.Document {
	return dom.Document{
		ID:               m.ID,
		WarrantyID:       m.WarrantyID,
		WorkspaceID:      m.WorkspaceID,
		DocType:          dom.DocType(m.DocType),
		FileName:         m.FileName,
		OriginalFileName: m.OriginalFileName,
		ObjectKey:        m.ObjectKey,
		ContentType:      m.ContentType,
		SizeBytes:        m.SizeBytes,
		Notes:            m.Notes,
		CreatedAt:        m.CreatedAt,
	}
}
