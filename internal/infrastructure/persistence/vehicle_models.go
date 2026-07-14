package persistence

import (
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/vehicle"
)

// ─── GORM models ─────────────────────────────────────────────────────────────

type MaintenancePlanTemplateModel struct {
	ID                  string `gorm:"primaryKey;column:id"`
	Type                string `gorm:"column:type"`
	Name                string `gorm:"column:name"`
	DefaultIntervalKM   *int   `gorm:"column:default_interval_km"`
	DefaultIntervalDays *int   `gorm:"column:default_interval_days"`
	Scope               string `gorm:"column:scope"`
}

func (MaintenancePlanTemplateModel) TableName() string { return "maintenance_plan_templates" }

type VehicleModel struct {
	ID               string     `gorm:"primaryKey;column:id"`
	WorkspaceID      string     `gorm:"column:workspace_id"`
	Nickname         *string    `gorm:"column:nickname;size:100"`
	Make             string     `gorm:"column:make;size:80"`
	Model            string     `gorm:"column:model;size:120"`
	YearManufacture  int        `gorm:"column:year_manufacture"`
	YearModel        int        `gorm:"column:year_model"`
	Color            *string    `gorm:"column:color;size:40"`
	Plate            *string    `gorm:"column:plate;size:10"`
	FuelType         string     `gorm:"column:fuel_type;size:20"`
	FipeVehicleType  string     `gorm:"column:fipe_vehicle_type;size:20"`
	FipeCode         *string    `gorm:"column:fipe_code;size:20"`
	FipeBrandCode    *string    `gorm:"column:fipe_brand_code;size:20"`
	FipeModelCode    *string    `gorm:"column:fipe_model_code;size:20"`
	FipeYearCode     *string    `gorm:"column:fipe_year_code;size:20"`
	AcquisitionDate  *time.Time `gorm:"column:acquisition_date"`
	AcquisitionPrice *float64   `gorm:"column:acquisition_price"`
	CurrentOdometer  int        `gorm:"column:current_odometer"`
	Status           string     `gorm:"column:status;size:20"`
	SoldAt           *time.Time `gorm:"column:sold_at"`
	SoldPrice        *float64   `gorm:"column:sold_price"`
	Notes            *string    `gorm:"column:notes"`
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	Members          []VehicleMemberModel `gorm:"foreignKey:VehicleID"`
}

func (VehicleModel) TableName() string { return "vehicles" }

type VehicleMemberModel struct {
	VehicleID string `gorm:"primaryKey;column:vehicle_id"`
	MemberID  string `gorm:"primaryKey;column:member_id"`
	Role      string `gorm:"column:role;size:20"`
}

func (VehicleMemberModel) TableName() string { return "vehicle_members" }

type VehicleMaintenancePlanModel struct {
	ID           string    `gorm:"primaryKey;column:id"`
	VehicleID    string    `gorm:"column:vehicle_id"`
	WorkspaceID  string    `gorm:"column:workspace_id"`
	TemplateID   string    `gorm:"column:template_id"`
	IntervalKM   *int      `gorm:"column:interval_km"`
	IntervalDays *int      `gorm:"column:interval_days"`
	Enabled      bool      `gorm:"column:enabled"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
	Template     MaintenancePlanTemplateModel `gorm:"foreignKey:TemplateID;references:ID"`
}

func (VehicleMaintenancePlanModel) TableName() string { return "vehicle_maintenance_plans" }

type VehicleMaintenanceModel struct {
	ID                  string     `gorm:"primaryKey;column:id"`
	VehicleID           string     `gorm:"column:vehicle_id"`
	WorkspaceID         string     `gorm:"column:workspace_id"`
	TemplateID          *string    `gorm:"column:template_id"`
	Type                string     `gorm:"column:type;size:50"`
	Title               string     `gorm:"column:title;size:150"`
	Description         *string    `gorm:"column:description"`
	OdometerAtService   *int       `gorm:"column:odometer_at_service"`
	ServiceDate         *time.Time `gorm:"column:service_date"` // nullable — orçamento ainda não tem data
	Cost                *float64   `gorm:"column:cost"`
	SupplierID          *string    `gorm:"column:supplier_id"`
	NextServiceOdometer *int       `gorm:"column:next_service_odometer"`
	NextServiceDate     *time.Time `gorm:"column:next_service_date"`
	Notes               *string    `gorm:"column:notes"`
	Status              string     `gorm:"column:status;size:20"`
	OSNumber            *string    `gorm:"column:os_number"`
	Technician          *string    `gorm:"column:technician"`
	PaymentMethod       *string    `gorm:"column:payment_method"`
	TotalProductsCents  int64      `gorm:"column:total_products_cents"`
	TotalServicesCents  int64      `gorm:"column:total_services_cents"`
	TotalCents          int64      `gorm:"column:total_cents"`
	CreatedAt           time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt           time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	Items               []MaintenanceItemModel `gorm:"foreignKey:MaintenanceID;references:ID"`
}

func (VehicleMaintenanceModel) TableName() string { return "vehicle_maintenance" }

type VehicleFipeHistoryModel struct {
	ID             string    `gorm:"primaryKey;column:id"`
	VehicleID      string    `gorm:"column:vehicle_id"`
	WorkspaceID    string    `gorm:"column:workspace_id"`
	ReferenceMonth string    `gorm:"column:reference_month;size:10"`
	FipeValue      float64   `gorm:"column:fipe_value"`
	FipeFuel       *string   `gorm:"column:fipe_fuel;size:40"`
	RecordedAt     time.Time `gorm:"column:recorded_at;autoCreateTime"`
}

func (VehicleFipeHistoryModel) TableName() string { return "vehicle_fipe_history" }

// ─── Converters domain → model ────────────────────────────────────────────────

func vehicleToModel(v *dom.Vehicle) VehicleModel {
	members := make([]VehicleMemberModel, len(v.Members))
	for i, m := range v.Members {
		members[i] = VehicleMemberModel{
			VehicleID: v.ID.String(),
			MemberID:  m.MemberID.String(),
			Role:      string(m.Role),
		}
	}
	fvt := v.FipeVehicleType
	if fvt == "" {
		fvt = "carros"
	}
	return VehicleModel{
		ID:               v.ID.String(),
		WorkspaceID:      v.WorkspaceID.String(),
		Nickname:         v.Nickname,
		Make:             v.Make,
		Model:            v.Model,
		YearManufacture:  v.YearManufacture,
		YearModel:        v.YearModel,
		Color:            v.Color,
		Plate:            v.Plate,
		FuelType:         string(v.FuelType),
		FipeVehicleType:  fvt,
		FipeCode:         v.FipeCode,
		FipeBrandCode:    v.FipeBrandCode,
		FipeModelCode:    v.FipeModelCode,
		FipeYearCode:     v.FipeYearCode,
		AcquisitionDate:  v.AcquisitionDate,
		AcquisitionPrice: v.AcquisitionPrice,
		CurrentOdometer:  v.CurrentOdometer,
		Status:           string(v.Status),
		SoldAt:           v.SoldAt,
		SoldPrice:        v.SoldPrice,
		Notes:            v.Notes,
		CreatedAt:        v.CreatedAt,
		UpdatedAt:        v.UpdatedAt,
		Members:          members,
	}
}

func modelToVehicle(m VehicleModel) dom.Vehicle {
	id, _ := uuid.Parse(m.ID)
	wsID, _ := uuid.Parse(m.WorkspaceID)
	members := make([]dom.VehicleMember, len(m.Members))
	for i, mm := range m.Members {
		vid, _ := uuid.Parse(mm.VehicleID)
		mid, _ := uuid.Parse(mm.MemberID)
		members[i] = dom.VehicleMember{
			VehicleID: vid,
			MemberID:  mid,
			Role:      dom.MemberRole(mm.Role),
		}
	}
	return dom.Vehicle{
		ID:               id,
		WorkspaceID:      wsID,
		Nickname:         m.Nickname,
		Make:             m.Make,
		Model:            m.Model,
		YearManufacture:  m.YearManufacture,
		YearModel:        m.YearModel,
		Color:            m.Color,
		Plate:            m.Plate,
		FuelType:         dom.FuelType(m.FuelType),
		FipeVehicleType:  m.FipeVehicleType,
		FipeCode:         m.FipeCode,
		FipeBrandCode:    m.FipeBrandCode,
		FipeModelCode:    m.FipeModelCode,
		FipeYearCode:     m.FipeYearCode,
		AcquisitionDate:  m.AcquisitionDate,
		AcquisitionPrice: m.AcquisitionPrice,
		CurrentOdometer:  m.CurrentOdometer,
		Status:           dom.VehicleStatus(m.Status),
		SoldAt:           m.SoldAt,
		SoldPrice:        m.SoldPrice,
		Notes:            m.Notes,
		Members:          members,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

func modelToTemplate(m MaintenancePlanTemplateModel) dom.MaintenancePlanTemplate {
	id, _ := uuid.Parse(m.ID)
	return dom.MaintenancePlanTemplate{
		ID:                  id,
		Type:                m.Type,
		Name:                m.Name,
		DefaultIntervalKM:   m.DefaultIntervalKM,
		DefaultIntervalDays: m.DefaultIntervalDays,
		Scope:               m.Scope,
	}
}

func modelToMaintenancePlan(m VehicleMaintenancePlanModel) dom.VehicleMaintenancePlan {
	id, _ := uuid.Parse(m.ID)
	vid, _ := uuid.Parse(m.VehicleID)
	wsID, _ := uuid.Parse(m.WorkspaceID)
	tid, _ := uuid.Parse(m.TemplateID)
	t := modelToTemplate(m.Template)
	return dom.VehicleMaintenancePlan{
		ID:           id,
		VehicleID:    vid,
		WorkspaceID:  wsID,
		TemplateID:   tid,
		IntervalKM:   m.IntervalKM,
		IntervalDays: m.IntervalDays,
		Enabled:      m.Enabled,
		UpdatedAt:    m.UpdatedAt,
		Template:     &t,
	}
}

func modelToMaintenance(m VehicleMaintenanceModel) dom.VehicleMaintenance {
	id, _ := uuid.Parse(m.ID)
	vid, _ := uuid.Parse(m.VehicleID)
	wsID, _ := uuid.Parse(m.WorkspaceID)
	status := dom.MaintenanceStatus(m.Status)
	if status == "" {
		status = dom.MaintenanceStatusRealizado
	}
	out := dom.VehicleMaintenance{
		ID:                  id,
		VehicleID:           vid,
		WorkspaceID:         wsID,
		Type:                m.Type,
		Title:               m.Title,
		Description:         m.Description,
		OdometerAtService:   m.OdometerAtService,
		ServiceDate:         m.ServiceDate,
		Cost:                m.Cost,
		NextServiceOdometer: m.NextServiceOdometer,
		NextServiceDate:     m.NextServiceDate,
		Notes:               m.Notes,
		Status:              status,
		OSNumber:            m.OSNumber,
		Technician:          m.Technician,
		PaymentMethod:       m.PaymentMethod,
		TotalProductsCents:  m.TotalProductsCents,
		TotalServicesCents:  m.TotalServicesCents,
		TotalCents:          m.TotalCents,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}
	if m.TemplateID != nil {
		tid, _ := uuid.Parse(*m.TemplateID)
		out.TemplateID = &tid
	}
	if m.SupplierID != nil {
		sid, _ := uuid.Parse(*m.SupplierID)
		out.SupplierID = &sid
	}
	out.Items = make([]dom.VehicleMaintenanceItem, len(m.Items))
	for i, it := range m.Items {
		out.Items[i] = modelToMaintenanceItem(it)
	}
	return out
}

func maintenanceToModel(m *dom.VehicleMaintenance) VehicleMaintenanceModel {
	status := string(m.Status)
	if status == "" {
		status = string(dom.MaintenanceStatusRealizado)
	}
	out := VehicleMaintenanceModel{
		ID:                  m.ID.String(),
		VehicleID:           m.VehicleID.String(),
		WorkspaceID:         m.WorkspaceID.String(),
		Type:                m.Type,
		Title:               m.Title,
		Description:         m.Description,
		OdometerAtService:   m.OdometerAtService,
		ServiceDate:         m.ServiceDate,
		Cost:                m.Cost,
		NextServiceOdometer: m.NextServiceOdometer,
		NextServiceDate:     m.NextServiceDate,
		Notes:               m.Notes,
		Status:              status,
		OSNumber:            m.OSNumber,
		Technician:          m.Technician,
		PaymentMethod:       m.PaymentMethod,
		TotalProductsCents:  m.TotalProductsCents,
		TotalServicesCents:  m.TotalServicesCents,
		TotalCents:          m.TotalCents,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}
	if m.TemplateID != nil {
		s := m.TemplateID.String()
		out.TemplateID = &s
	}
	if m.SupplierID != nil {
		s := m.SupplierID.String()
		out.SupplierID = &s
	}
	return out
}

func modelToFipeHistory(m VehicleFipeHistoryModel) dom.VehicleFipeHistory {
	id, _ := uuid.Parse(m.ID)
	vid, _ := uuid.Parse(m.VehicleID)
	wsID, _ := uuid.Parse(m.WorkspaceID)
	return dom.VehicleFipeHistory{
		ID:             id,
		VehicleID:      vid,
		WorkspaceID:    wsID,
		ReferenceMonth: m.ReferenceMonth,
		FipeValue:      m.FipeValue,
		FipeFuel:       m.FipeFuel,
		RecordedAt:     m.RecordedAt,
	}
}
