package persistence

import (
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/vehicle"
)

// ─── GORM models ─────────────────────────────────────────────────────────────

type MaintenanceItemModel struct {
	ID                        string     `gorm:"primaryKey;column:id"`
	MaintenanceID             string     `gorm:"column:maintenance_id"`
	VehicleID                 string     `gorm:"column:vehicle_id"`
	WorkspaceID               string     `gorm:"column:workspace_id"`
	CatalogItemID             *string    `gorm:"column:catalog_item_id"`
	ItemType                  string     `gorm:"column:item_type;size:10"`
	Category                  string     `gorm:"column:category;size:30"`
	Description               string     `gorm:"column:description"`
	Quantity                  float64    `gorm:"column:quantity"`
	UnitPriceCents            int64      `gorm:"column:unit_price_cents"`
	TotalPriceCents           int64      `gorm:"column:total_price_cents"`
	KMAtInstallation          *int       `gorm:"column:km_at_installation"`
	ReplacementIntervalKM     *int       `gorm:"column:replacement_interval_km"`
	ReplacementIntervalMonths *int       `gorm:"column:replacement_interval_months"`
	NextDueKM                 *int       `gorm:"column:next_due_km;<-:false"`
	NextDueDate               *time.Time `gorm:"column:next_due_date;type:date"`
	WarrantyExpiresDate       *time.Time `gorm:"column:warranty_expires_date;type:date"`
	WarrantyExpiresKM         *int       `gorm:"column:warranty_expires_km"`
	Notes                     *string    `gorm:"column:notes"`
	CreatedAt                 time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (MaintenanceItemModel) TableName() string { return "vehicle_maintenance_items" }

type MaintenanceCatalogItemModel struct {
	ID                    string  `gorm:"primaryKey;column:id"`
	Category              string  `gorm:"column:category;size:30"`
	ItemType              string  `gorm:"column:item_type;size:10"`
	Name                  string  `gorm:"column:name"`
	Description           *string `gorm:"column:description"`
	DefaultIntervalKM     *int    `gorm:"column:default_interval_km"`
	DefaultIntervalMonths *int    `gorm:"column:default_interval_months"`
	DefaultWarrantyMonths *int    `gorm:"column:default_warranty_months"`
	Active                bool    `gorm:"column:active"`
	SortOrder             int     `gorm:"column:sort_order"`
}

func (MaintenanceCatalogItemModel) TableName() string { return "maintenance_catalog_items" }

type MaintenanceScheduleModel struct {
	ID                string     `gorm:"primaryKey;column:id"`
	VehicleID         string     `gorm:"column:vehicle_id"`
	WorkspaceID       string     `gorm:"column:workspace_id"`
	MaintenanceItemID *string    `gorm:"column:maintenance_item_id"` // era service_order_item_id
	Description       string     `gorm:"column:description"`
	Category          string     `gorm:"column:category;size:30"`
	ScheduledKM       *int       `gorm:"column:scheduled_km"`
	ScheduledDate     *time.Time `gorm:"column:scheduled_date;type:date"`
	AlertStatus       string     `gorm:"column:alert_status;size:20"`
	CompletedAt       *time.Time `gorm:"column:completed_at;type:date"`
	Notes             *string    `gorm:"column:notes"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (MaintenanceScheduleModel) TableName() string { return "vehicle_maintenance_schedules" }

// ─── Converters ───────────────────────────────────────────────────────────────

func maintenanceItemToModel(item *dom.VehicleMaintenanceItem) MaintenanceItemModel {
	m := MaintenanceItemModel{
		ID:                        item.ID.String(),
		MaintenanceID:             item.MaintenanceID.String(),
		VehicleID:                 item.VehicleID.String(),
		WorkspaceID:               item.WorkspaceID.String(),
		ItemType:                  string(item.ItemType),
		Category:                  string(item.Category),
		Description:               item.Description,
		Quantity:                  item.Quantity,
		UnitPriceCents:            item.UnitPriceCents,
		TotalPriceCents:           item.TotalPriceCents,
		KMAtInstallation:          item.KMAtInstallation,
		ReplacementIntervalKM:     item.ReplacementIntervalKM,
		ReplacementIntervalMonths: item.ReplacementIntervalMonths,
		NextDueDate:               item.NextDueDate,
		WarrantyExpiresDate:       item.WarrantyExpiresDate,
		WarrantyExpiresKM:         item.WarrantyExpiresKM,
		Notes:                     item.Notes,
		CreatedAt:                 item.CreatedAt,
	}
	if item.CatalogItemID != nil {
		s := item.CatalogItemID.String()
		m.CatalogItemID = &s
	}
	return m
}

func modelToMaintenanceItem(m MaintenanceItemModel) dom.VehicleMaintenanceItem {
	id, _ := uuid.Parse(m.ID)
	mID, _ := uuid.Parse(m.MaintenanceID)
	vid, _ := uuid.Parse(m.VehicleID)
	wsID, _ := uuid.Parse(m.WorkspaceID)
	item := dom.VehicleMaintenanceItem{
		ID:                        id,
		MaintenanceID:             mID,
		VehicleID:                 vid,
		WorkspaceID:               wsID,
		ItemType:                  dom.OSItemType(m.ItemType),
		Category:                  dom.OSItemCategory(m.Category),
		Description:               m.Description,
		Quantity:                  m.Quantity,
		UnitPriceCents:            m.UnitPriceCents,
		TotalPriceCents:           m.TotalPriceCents,
		KMAtInstallation:          m.KMAtInstallation,
		ReplacementIntervalKM:     m.ReplacementIntervalKM,
		ReplacementIntervalMonths: m.ReplacementIntervalMonths,
		NextDueKM:                 m.NextDueKM,
		NextDueDate:               m.NextDueDate,
		WarrantyExpiresDate:       m.WarrantyExpiresDate,
		WarrantyExpiresKM:         m.WarrantyExpiresKM,
		Notes:                     m.Notes,
		CreatedAt:                 m.CreatedAt,
	}
	if m.CatalogItemID != nil {
		cid, _ := uuid.Parse(*m.CatalogItemID)
		item.CatalogItemID = &cid
	}
	return item
}

func modelToCatalogItem(m MaintenanceCatalogItemModel) dom.MaintenanceCatalogItem {
	id, _ := uuid.Parse(m.ID)
	return dom.MaintenanceCatalogItem{
		ID:                    id,
		Category:              dom.OSItemCategory(m.Category),
		ItemType:              dom.OSItemType(m.ItemType),
		Name:                  m.Name,
		Description:           m.Description,
		DefaultIntervalKM:     m.DefaultIntervalKM,
		DefaultIntervalMonths: m.DefaultIntervalMonths,
		DefaultWarrantyMonths: m.DefaultWarrantyMonths,
		Active:                m.Active,
	}
}

func scheduleToModel(s *dom.MaintenanceSchedule) MaintenanceScheduleModel {
	m := MaintenanceScheduleModel{
		ID:            s.ID.String(),
		VehicleID:     s.VehicleID.String(),
		WorkspaceID:   s.WorkspaceID.String(),
		Description:   s.Description,
		Category:      string(s.Category),
		ScheduledKM:   s.ScheduledKM,
		ScheduledDate: s.ScheduledDate,
		AlertStatus:   string(s.AlertStatus),
		CompletedAt:   s.CompletedAt,
		Notes:         s.Notes,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
	if s.MaintenanceItemID != nil {
		str := s.MaintenanceItemID.String()
		m.MaintenanceItemID = &str
	}
	return m
}

func modelToSchedule(m MaintenanceScheduleModel) dom.MaintenanceSchedule {
	id, _ := uuid.Parse(m.ID)
	vid, _ := uuid.Parse(m.VehicleID)
	wsID, _ := uuid.Parse(m.WorkspaceID)
	s := dom.MaintenanceSchedule{
		ID:            id,
		VehicleID:     vid,
		WorkspaceID:   wsID,
		Description:   m.Description,
		Category:      dom.OSItemCategory(m.Category),
		ScheduledKM:   m.ScheduledKM,
		ScheduledDate: m.ScheduledDate,
		AlertStatus:   dom.ScheduleStatus(m.AlertStatus),
		CompletedAt:   m.CompletedAt,
		Notes:         m.Notes,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
	if m.MaintenanceItemID != nil {
		itemID, _ := uuid.Parse(*m.MaintenanceItemID)
		s.MaintenanceItemID = &itemID
	}
	return s
}
