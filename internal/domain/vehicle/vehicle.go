package vehicle

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// FuelType é o combustível do veículo.
type FuelType string

const (
	FuelGasolina FuelType = "gasolina"
	FuelEtanol   FuelType = "etanol"
	FuelFlex     FuelType = "flex"
	FuelDiesel   FuelType = "diesel"
	FuelEletrico FuelType = "eletrico"
	FuelHibrido  FuelType = "hibrido"
)

// VehicleStatus representa o estado do veículo na frota familiar.
type VehicleStatus string

const (
	StatusActive   VehicleStatus = "active"
	StatusSold     VehicleStatus = "sold"
	StatusInactive VehicleStatus = "inactive"
)

// MemberRole é o papel do membro familiar em relação ao veículo.
type MemberRole string

const (
	RoleOwner  MemberRole = "owner"
	RoleDriver MemberRole = "driver"
)

// AlertStatus indica a urgência de um alerta de manutenção.
type AlertStatus string

const (
	AlertOverdue AlertStatus = "overdue"
	AlertDueSoon AlertStatus = "due_soon"
	AlertOK      AlertStatus = "ok"
	AlertUnknown AlertStatus = "unknown"
)

// Vehicle é o agregado central do módulo de frota familiar.
type Vehicle struct {
	ID               uuid.UUID
	WorkspaceID      uuid.UUID
	Nickname         *string
	Make             string
	Model            string
	YearManufacture  int
	YearModel        int
	Color            *string
	Plate            *string
	FuelType         FuelType
	FipeVehicleType  string    // "carros" | "motos" | "caminhoes"
	FipeCode         *string
	FipeBrandCode    *string
	FipeModelCode    *string
	FipeYearCode     *string
	AcquisitionDate  *time.Time
	AcquisitionPrice *float64
	CurrentOdometer  int
	Status           VehicleStatus
	SoldAt           *time.Time
	SoldPrice        *float64
	Notes            *string
	Members          []VehicleMember
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// VehicleMember vincula um membro familiar a um veículo (relação N:N).
type VehicleMember struct {
	VehicleID uuid.UUID
	MemberID  uuid.UUID
	Role      MemberRole
}

// MaintenancePlanTemplate é um plano global de manutenção (ex: Troca de óleo / 5.000 km).
type MaintenancePlanTemplate struct {
	ID                  uuid.UUID
	Type                string
	Name                string
	DefaultIntervalKM   *int
	DefaultIntervalDays *int
	Scope               string
}

// VehicleMaintenancePlan é a customização por veículo de um plano global.
type VehicleMaintenancePlan struct {
	ID           uuid.UUID
	VehicleID    uuid.UUID
	WorkspaceID  uuid.UUID
	TemplateID   uuid.UUID
	IntervalKM   *int
	IntervalDays *int
	Enabled      bool
	UpdatedAt    time.Time
	Template     *MaintenancePlanTemplate
}

// VehicleMaintenance é um registro de manutenção executada.
type VehicleMaintenance struct {
	ID                  uuid.UUID
	VehicleID           uuid.UUID
	WorkspaceID         uuid.UUID
	TemplateID          *uuid.UUID
	Type                string
	Title               string
	Description         *string
	OdometerAtService   *int
	ServiceDate         time.Time
	Cost                *float64
	SupplierID          *uuid.UUID
	NextServiceOdometer *int
	NextServiceDate     *time.Time
	Notes               *string
	CreatedAt           time.Time
}

// VehicleFipeHistory é um snapshot mensal do valor FIPE do veículo.
type VehicleFipeHistory struct {
	ID             uuid.UUID
	VehicleID      uuid.UUID
	WorkspaceID    uuid.UUID
	ReferenceMonth string // "07/2026"
	FipeValue      float64
	FipeFuel       *string
	RecordedAt     time.Time
}

// DepreciationReport agrega os indicadores de depreciação de um veículo.
type DepreciationReport struct {
	AcquisitionPrice     *float64
	CurrentFipeValue     *float64
	TotalDepreciationPct *float64
	TotalDepreciationR   *float64
	MonthsOwned          int
	MonthlyAvgDeprecR    *float64
	AnnualAvgDeprecR     *float64
	Trend6MonthsR        *float64 // média da variação dos últimos 6 meses (negativo = depreciação)
	History              []VehicleFipeHistory
}

// MaintenanceAlert é o resultado calculado de um alerta para um plano de manutenção.
type MaintenanceAlert struct {
	TemplateID    uuid.UUID
	Type          string
	Title         string
	Status        AlertStatus
	DueAtKM       *int
	DueAtDate     *time.Time
	KMRemaining   *int
	DaysRemaining *int
	LastOdometer  *int
	LastDate      *time.Time
}

// ─── Service Orders ───────────────────────────────────────────────────────────

type OSStatus string

const (
	OSStatusDraft     OSStatus = "draft"
	OSStatusCompleted OSStatus = "completed"
	OSStatusCancelled OSStatus = "cancelled"
)

type OSItemType string

const (
	OSItemProduct OSItemType = "product"
	OSItemService OSItemType = "service"
)

type OSItemCategory string

const (
	OSItemCategoryMotor          OSItemCategory = "motor"
	OSItemCategoryFreios         OSItemCategory = "freios"
	OSItemCategorySuspensao      OSItemCategory = "suspensao"
	OSItemCategoryTransmissao    OSItemCategory = "transmissao"
	OSItemCategoryArrefecimento  OSItemCategory = "arrefecimento"
	OSItemCategoryEletrico       OSItemCategory = "eletrico"
	OSItemCategoryPneus          OSItemCategory = "pneus"
	OSItemCategoryArCondicionado OSItemCategory = "ar_condicionado"
	OSItemCategoryCarroceria     OSItemCategory = "carroceria"
	OSItemCategoryServico        OSItemCategory = "servico"
	OSItemCategoryOutros         OSItemCategory = "outros"
)

type ScheduleStatus string

const (
	ScheduleStatusPending   ScheduleStatus = "pending"
	ScheduleStatusDueSoon   ScheduleStatus = "due_soon"
	ScheduleStatusOverdue   ScheduleStatus = "overdue"
	ScheduleStatusDone      ScheduleStatus = "done"
	ScheduleStatusCancelled ScheduleStatus = "cancelled"
)

type ServiceOrder struct {
	ID                 uuid.UUID
	VehicleID          uuid.UUID
	WorkspaceID        uuid.UUID
	SupplierID         *uuid.UUID
	OSNumber           *string
	ServiceDate        time.Time
	KMAtService        int
	TotalProductsCents int64
	TotalServicesCents int64
	TotalCents         int64
	PaymentMethod      *string
	Technician         *string
	Notes              *string
	Status             OSStatus
	Items              []ServiceOrderItem
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ServiceOrderItem struct {
	ID                        uuid.UUID
	ServiceOrderID            uuid.UUID
	VehicleID                 uuid.UUID
	WorkspaceID               uuid.UUID
	CatalogItemID             *uuid.UUID
	ItemType                  OSItemType
	Category                  OSItemCategory
	Description               string
	Quantity                  float64
	UnitPriceCents            int64
	TotalPriceCents           int64
	KMAtInstallation          *int
	ReplacementIntervalKM     *int
	ReplacementIntervalMonths *int
	NextDueKM                 *int
	NextDueDate               *time.Time
	WarrantyExpiresDate       *time.Time
	WarrantyExpiresKM         *int
	Notes                     *string
	CreatedAt                 time.Time
}

type MaintenanceCatalogItem struct {
	ID                    uuid.UUID
	Category              OSItemCategory
	ItemType              OSItemType
	Name                  string
	Description           *string
	DefaultIntervalKM     *int
	DefaultIntervalMonths *int
	DefaultWarrantyMonths *int
	Active                bool
}

type MaintenanceSchedule struct {
	ID                 uuid.UUID
	VehicleID          uuid.UUID
	WorkspaceID        uuid.UUID
	ServiceOrderItemID *uuid.UUID
	Description        string
	Category           OSItemCategory
	ScheduledKM        *int
	ScheduledDate      *time.Time
	AlertStatus        ScheduleStatus
	CompletedAt        *time.Time
	Notes              *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CategorySpending struct {
	Category   string
	TotalCents int64
}

type SupplierSpending struct {
	SupplierID   string
	SupplierName string
	TotalCents   int64
}

type MonthlySpending struct {
	Month      string
	TotalCents int64
}

type VehicleAnalytics struct {
	TotalSpentCents    int64
	TotalProductsCents int64
	TotalServicesCents int64
	CostPerKM          *float64
	TotalOSCount       int
	AvgCostPerOSCents  int64
	SpendingByCategory []CategorySpending
	SpendingBySupplier []SupplierSpending
	MonthlySpending    []MonthlySpending
}

// ValidationError é retornado quando a entidade viola regras de domínio.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

// ErrNotFound é retornado quando um veículo ou manutenção não existe (ou não pertence ao workspace).
var ErrNotFound = &ValidationError{Msg: "não encontrado"}

// Validate verifica as regras de domínio do Vehicle.
func (v *Vehicle) Validate() error {
	if v.Make == "" {
		return &ValidationError{Msg: "make é obrigatório"}
	}
	if v.Model == "" {
		return &ValidationError{Msg: "model é obrigatório"}
	}
	if v.YearManufacture < 1900 || v.YearManufacture > 2100 {
		return &ValidationError{Msg: fmt.Sprintf("year_manufacture inválido: %d", v.YearManufacture)}
	}
	if v.YearModel < 1900 || v.YearModel > 2100 {
		return &ValidationError{Msg: fmt.Sprintf("year_model inválido: %d", v.YearModel)}
	}
	switch v.FuelType {
	case FuelGasolina, FuelEtanol, FuelFlex, FuelDiesel, FuelEletrico, FuelHibrido:
	default:
		return &ValidationError{Msg: fmt.Sprintf("fuel_type inválido: %s", v.FuelType)}
	}
	switch v.Status {
	case StatusActive, StatusSold, StatusInactive:
	default:
		return &ValidationError{Msg: fmt.Sprintf("status inválido: %s", v.Status)}
	}
	return nil
}

// ListVehiclesParams filtra a listagem de veículos.
type ListVehiclesParams struct {
	Status string
	Limit  int
	Offset int
}

// VehicleRepository define as operações de persistência do módulo de frota familiar.
type VehicleRepository interface {
	Create(ctx context.Context, v *Vehicle) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*Vehicle, error)
	List(ctx context.Context, workspaceID uuid.UUID, p ListVehiclesParams) ([]Vehicle, int64, error)
	Update(ctx context.Context, v *Vehicle) error
	Delete(ctx context.Context, workspaceID, id uuid.UUID) error
	SetMembers(ctx context.Context, vehicleID uuid.UUID, members []VehicleMember) error

	CreateMaintenance(ctx context.Context, m *VehicleMaintenance) error
	GetMaintenanceByID(ctx context.Context, workspaceID, id uuid.UUID) (*VehicleMaintenance, error)
	ListMaintenance(ctx context.Context, workspaceID, vehicleID uuid.UUID) ([]VehicleMaintenance, error)
	UpdateMaintenance(ctx context.Context, m *VehicleMaintenance) error
	DeleteMaintenance(ctx context.Context, workspaceID, id uuid.UUID) error
	LastMaintenanceByType(ctx context.Context, vehicleID uuid.UUID) (map[string]*VehicleMaintenance, error)

	ListTemplates(ctx context.Context) ([]MaintenancePlanTemplate, error)
	GetVehiclePlans(ctx context.Context, workspaceID, vehicleID uuid.UUID) ([]VehicleMaintenancePlan, error)
	UpsertVehiclePlan(ctx context.Context, p *VehicleMaintenancePlan) error

	AddFipeHistory(ctx context.Context, h *VehicleFipeHistory) error
	ListFipeHistory(ctx context.Context, vehicleID uuid.UUID) ([]VehicleFipeHistory, error)
	ListActiveVehiclesForFipeUpdate(ctx context.Context) ([]Vehicle, error)

	// Service Orders
	CreateServiceOrder(ctx context.Context, o *ServiceOrder) error
	GetServiceOrderByID(ctx context.Context, workspaceID, id uuid.UUID) (*ServiceOrder, error)
	ListServiceOrders(ctx context.Context, workspaceID, vehicleID uuid.UUID) ([]ServiceOrder, error)
	UpdateServiceOrder(ctx context.Context, o *ServiceOrder) error
	DeleteServiceOrder(ctx context.Context, workspaceID, id uuid.UUID) error

	// Service Order Items
	CreateServiceOrderItem(ctx context.Context, item *ServiceOrderItem) error
	GetServiceOrderItemByID(ctx context.Context, workspaceID, id uuid.UUID) (*ServiceOrderItem, error)
	ListServiceOrderItems(ctx context.Context, workspaceID, serviceOrderID uuid.UUID) ([]ServiceOrderItem, error)
	UpdateServiceOrderItem(ctx context.Context, item *ServiceOrderItem) error
	DeleteServiceOrderItem(ctx context.Context, workspaceID, id uuid.UUID) error
	RecalcServiceOrderTotals(ctx context.Context, workspaceID, serviceOrderID uuid.UUID) error

	// Catalog
	SearchCatalog(ctx context.Context, query, category string, limit int) ([]MaintenanceCatalogItem, error)

	// Schedules
	CreateSchedule(ctx context.Context, s *MaintenanceSchedule) error
	GetScheduleByID(ctx context.Context, workspaceID, id uuid.UUID) (*MaintenanceSchedule, error)
	ListSchedules(ctx context.Context, workspaceID, vehicleID uuid.UUID) ([]MaintenanceSchedule, error)
	UpdateSchedule(ctx context.Context, s *MaintenanceSchedule) error
	DeleteSchedule(ctx context.Context, workspaceID, id uuid.UUID) error
	RefreshScheduleStatuses(ctx context.Context, vehicleID uuid.UUID, currentKM int) error

	// Analytics
	GetAnalytics(ctx context.Context, workspaceID, vehicleID uuid.UUID, months int) (*VehicleAnalytics, error)
}
