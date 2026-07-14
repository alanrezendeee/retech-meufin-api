package persistence

import (
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/vehicle"
)

// ─── GORM models ─────────────────────────────────────────────────────────────

type ServiceOrderModel struct {
	ID                 string    `gorm:"primaryKey;column:id"`
	VehicleID          string    `gorm:"column:vehicle_id"`
	WorkspaceID        string    `gorm:"column:workspace_id"`
	SupplierID         *string   `gorm:"column:supplier_id"`
	OSNumber           *string   `gorm:"column:os_number"`
	ServiceDate        time.Time `gorm:"column:service_date;type:date"`
	KMAtService        int       `gorm:"column:km_at_service"`
	TotalProductsCents int64     `gorm:"column:total_products_cents"`
	TotalServicesCents int64     `gorm:"column:total_services_cents"`
	TotalCents         int64     `gorm:"column:total_cents"`
	PaymentMethod      *string   `gorm:"column:payment_method"`
	Technician         *string   `gorm:"column:technician"`
	Notes              *string   `gorm:"column:notes"`
	Status             string    `gorm:"column:status;size:20"`
	Items              []ServiceOrderItemModel `gorm:"foreignKey:ServiceOrderID;references:ID"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (ServiceOrderModel) TableName() string { return "vehicle_service_orders" }

type ServiceOrderItemModel struct {
	ID                        string     `gorm:"primaryKey;column:id"`
	ServiceOrderID            string     `gorm:"column:service_order_id"`
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

func (ServiceOrderItemModel) TableName() string { return "vehicle_service_order_items" }

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
	ID                 string     `gorm:"primaryKey;column:id"`
	VehicleID          string     `gorm:"column:vehicle_id"`
	WorkspaceID        string     `gorm:"column:workspace_id"`
	ServiceOrderItemID *string    `gorm:"column:service_order_item_id"`
	Description        string     `gorm:"column:description"`
	Category           string     `gorm:"column:category;size:30"`
	ScheduledKM        *int       `gorm:"column:scheduled_km"`
	ScheduledDate      *time.Time `gorm:"column:scheduled_date;type:date"`
	AlertStatus        string     `gorm:"column:alert_status;size:20"`
	CompletedAt        *time.Time `gorm:"column:completed_at;type:date"`
	Notes              *string    `gorm:"column:notes"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (MaintenanceScheduleModel) TableName() string { return "vehicle_maintenance_schedules" }

// ─── Converters ───────────────────────────────────────────────────────────────

func serviceOrderToModel(o *dom.ServiceOrder) ServiceOrderModel {
	m := ServiceOrderModel{
		ID:                 o.ID.String(),
		VehicleID:          o.VehicleID.String(),
		WorkspaceID:        o.WorkspaceID.String(),
		OSNumber:           o.OSNumber,
		ServiceDate:        o.ServiceDate,
		KMAtService:        o.KMAtService,
		TotalProductsCents: o.TotalProductsCents,
		TotalServicesCents: o.TotalServicesCents,
		TotalCents:         o.TotalCents,
		PaymentMethod:      o.PaymentMethod,
		Technician:         o.Technician,
		Notes:              o.Notes,
		Status:             string(o.Status),
		CreatedAt:          o.CreatedAt,
		UpdatedAt:          o.UpdatedAt,
	}
	if o.SupplierID != nil {
		s := o.SupplierID.String()
		m.SupplierID = &s
	}
	return m
}

func modelToServiceOrder(m ServiceOrderModel) dom.ServiceOrder {
	id, _ := uuid.Parse(m.ID)
	vid, _ := uuid.Parse(m.VehicleID)
	wsID, _ := uuid.Parse(m.WorkspaceID)
	o := dom.ServiceOrder{
		ID:                 id,
		VehicleID:          vid,
		WorkspaceID:        wsID,
		OSNumber:           m.OSNumber,
		ServiceDate:        m.ServiceDate,
		KMAtService:        m.KMAtService,
		TotalProductsCents: m.TotalProductsCents,
		TotalServicesCents: m.TotalServicesCents,
		TotalCents:         m.TotalCents,
		PaymentMethod:      m.PaymentMethod,
		Technician:         m.Technician,
		Notes:              m.Notes,
		Status:             dom.OSStatus(m.Status),
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
	if m.SupplierID != nil {
		sid, _ := uuid.Parse(*m.SupplierID)
		o.SupplierID = &sid
	}
	o.Items = make([]dom.ServiceOrderItem, len(m.Items))
	for i, item := range m.Items {
		o.Items[i] = modelToServiceOrderItem(item)
	}
	return o
}

func serviceOrderItemToModel(item *dom.ServiceOrderItem) ServiceOrderItemModel {
	m := ServiceOrderItemModel{
		ID:                        item.ID.String(),
		ServiceOrderID:            item.ServiceOrderID.String(),
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

func modelToServiceOrderItem(m ServiceOrderItemModel) dom.ServiceOrderItem {
	id, _ := uuid.Parse(m.ID)
	osID, _ := uuid.Parse(m.ServiceOrderID)
	vid, _ := uuid.Parse(m.VehicleID)
	wsID, _ := uuid.Parse(m.WorkspaceID)
	item := dom.ServiceOrderItem{
		ID:                        id,
		ServiceOrderID:            osID,
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
	if s.ServiceOrderItemID != nil {
		str := s.ServiceOrderItemID.String()
		m.ServiceOrderItemID = &str
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
	if m.ServiceOrderItemID != nil {
		itemID, _ := uuid.Parse(*m.ServiceOrderItemID)
		s.ServiceOrderItemID = &itemID
	}
	return s
}
