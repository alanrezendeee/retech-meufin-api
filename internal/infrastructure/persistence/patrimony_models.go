package persistence

import (
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/patrimony"
)

// ─── GORM models ─────────────────────────────────────────────────────────────

type PropertyModel struct {
	ID                 string     `gorm:"primaryKey;column:id"`
	WorkspaceID        string     `gorm:"column:workspace_id"`
	Name               string     `gorm:"column:name;size:150"`
	PropertyType       string     `gorm:"column:property_type;size:20"`
	Address            *string    `gorm:"column:address;size:255"`
	City               *string    `gorm:"column:city;size:120"`
	State              *string    `gorm:"column:state;size:40"`
	ZipCode            *string    `gorm:"column:zip_code;size:20"`
	RegistrationNumber *string    `gorm:"column:registration_number;size:80"`
	AreaM2             *float64   `gorm:"column:area_m2"`
	PurchaseDate       *time.Time `gorm:"column:purchase_date"`
	PurchaseValueCents *int64     `gorm:"column:purchase_value_cents"`
	CurrentValueCents  *int64     `gorm:"column:current_value_cents"`
	Notes              *string    `gorm:"column:notes"`
	Active             bool       `gorm:"column:active"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (PropertyModel) TableName() string { return "properties" }

type PropertyDocumentModel struct {
	ID          string    `gorm:"primaryKey;column:id"`
	PropertyID  string    `gorm:"column:property_id"`
	WorkspaceID string    `gorm:"column:workspace_id"`
	DocType     string    `gorm:"column:doc_type;size:30"`
	FileName    string    `gorm:"column:file_name;size:255"`
	ObjectKey   string    `gorm:"column:object_key;size:500"`
	ContentType string    `gorm:"column:content_type;size:100"`
	SizeBytes   int64     `gorm:"column:size_bytes"`
	Notes       *string   `gorm:"column:notes"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (PropertyDocumentModel) TableName() string { return "property_documents" }

type AssetTaxModel struct {
	ID                string     `gorm:"primaryKey;column:id"`
	WorkspaceID       string     `gorm:"column:workspace_id"`
	AssetType         string     `gorm:"column:asset_type;size:20"`
	PropertyID        *string    `gorm:"column:property_id"`
	VehicleID         *string    `gorm:"column:vehicle_id"`
	TaxType           string     `gorm:"column:tax_type;size:30"`
	ReferenceYear     int        `gorm:"column:reference_year"`
	Description       *string    `gorm:"column:description;size:255"`
	DueDate           *time.Time `gorm:"column:due_date"`
	AmountCents       int64      `gorm:"column:amount_cents"`
	PaidCents         int64      `gorm:"column:paid_cents"`
	PaidDate          *time.Time `gorm:"column:paid_date"`
	Status            string     `gorm:"column:status;size:20"`
	InstallmentsTotal int        `gorm:"column:installments_total"`
	InstallmentNumber int        `gorm:"column:installment_number"`
	Notes             *string    `gorm:"column:notes"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (AssetTaxModel) TableName() string { return "asset_taxes" }

// ─── Converters ────────────────────────────────────────────────────────────────

func propertyToModel(p *dom.Property) PropertyModel {
	return PropertyModel{
		ID:                 p.ID.String(),
		WorkspaceID:        p.WorkspaceID.String(),
		Name:               p.Name,
		PropertyType:       string(p.PropertyType),
		Address:            p.Address,
		City:               p.City,
		State:              p.State,
		ZipCode:            p.ZipCode,
		RegistrationNumber: p.RegistrationNumber,
		AreaM2:             p.AreaM2,
		PurchaseDate:       p.PurchaseDate,
		PurchaseValueCents: p.PurchaseValueCents,
		CurrentValueCents:  p.CurrentValueCents,
		Notes:              p.Notes,
		Active:             p.Active,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}

func modelToProperty(m PropertyModel) dom.Property {
	id, _ := uuid.Parse(m.ID)
	wsID, _ := uuid.Parse(m.WorkspaceID)
	return dom.Property{
		ID:                 id,
		WorkspaceID:        wsID,
		Name:               m.Name,
		PropertyType:       dom.PropertyType(m.PropertyType),
		Address:            m.Address,
		City:               m.City,
		State:              m.State,
		ZipCode:            m.ZipCode,
		RegistrationNumber: m.RegistrationNumber,
		AreaM2:             m.AreaM2,
		PurchaseDate:       m.PurchaseDate,
		PurchaseValueCents: m.PurchaseValueCents,
		CurrentValueCents:  m.CurrentValueCents,
		Notes:              m.Notes,
		Active:             m.Active,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

func propertyDocToModel(d *dom.PropertyDocument) PropertyDocumentModel {
	return PropertyDocumentModel{
		ID:          d.ID.String(),
		PropertyID:  d.PropertyID.String(),
		WorkspaceID: d.WorkspaceID.String(),
		DocType:     string(d.DocType),
		FileName:    d.FileName,
		ObjectKey:   d.ObjectKey,
		ContentType: d.ContentType,
		SizeBytes:   d.SizeBytes,
		Notes:       d.Notes,
		CreatedAt:   d.CreatedAt,
	}
}

func modelToPropertyDoc(m PropertyDocumentModel) dom.PropertyDocument {
	id, _ := uuid.Parse(m.ID)
	pid, _ := uuid.Parse(m.PropertyID)
	wsID, _ := uuid.Parse(m.WorkspaceID)
	return dom.PropertyDocument{
		ID:          id,
		PropertyID:  pid,
		WorkspaceID: wsID,
		DocType:     dom.PropertyDocType(m.DocType),
		FileName:    m.FileName,
		ObjectKey:   m.ObjectKey,
		ContentType: m.ContentType,
		SizeBytes:   m.SizeBytes,
		Notes:       m.Notes,
		CreatedAt:   m.CreatedAt,
	}
}

func taxToModel(t *dom.AssetTax) AssetTaxModel {
	m := AssetTaxModel{
		ID:                t.ID.String(),
		WorkspaceID:       t.WorkspaceID.String(),
		AssetType:         string(t.AssetType),
		TaxType:           string(t.TaxType),
		ReferenceYear:     t.ReferenceYear,
		Description:       t.Description,
		DueDate:           t.DueDate,
		AmountCents:       t.AmountCents,
		PaidCents:         t.PaidCents,
		PaidDate:          t.PaidDate,
		Status:            string(t.Status),
		InstallmentsTotal: t.InstallmentsTotal,
		InstallmentNumber: t.InstallmentNumber,
		Notes:             t.Notes,
		CreatedAt:         t.CreatedAt,
		UpdatedAt:         t.UpdatedAt,
	}
	if t.PropertyID != nil {
		s := t.PropertyID.String()
		m.PropertyID = &s
	}
	if t.VehicleID != nil {
		s := t.VehicleID.String()
		m.VehicleID = &s
	}
	return m
}

func modelToTax(m AssetTaxModel) dom.AssetTax {
	id, _ := uuid.Parse(m.ID)
	wsID, _ := uuid.Parse(m.WorkspaceID)
	t := dom.AssetTax{
		ID:                id,
		WorkspaceID:       wsID,
		AssetType:         dom.AssetType(m.AssetType),
		TaxType:           dom.TaxType(m.TaxType),
		ReferenceYear:     m.ReferenceYear,
		Description:       m.Description,
		DueDate:           m.DueDate,
		AmountCents:       m.AmountCents,
		PaidCents:         m.PaidCents,
		PaidDate:          m.PaidDate,
		Status:            dom.TaxStatus(m.Status),
		InstallmentsTotal: m.InstallmentsTotal,
		InstallmentNumber: m.InstallmentNumber,
		Notes:             m.Notes,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
	if m.PropertyID != nil {
		pid, _ := uuid.Parse(*m.PropertyID)
		t.PropertyID = &pid
	}
	if m.VehicleID != nil {
		vid, _ := uuid.Parse(*m.VehicleID)
		t.VehicleID = &vid
	}
	return t
}
