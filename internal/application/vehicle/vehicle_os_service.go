package vehicle

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/vehicle"
)

// ─── Service Orders ───────────────────────────────────────────────────────────

type CreateServiceOrderInput struct {
	WorkspaceID   uuid.UUID
	VehicleID     uuid.UUID
	SupplierID    *uuid.UUID
	OSNumber      *string
	ServiceDate   time.Time
	KMAtService   int
	PaymentMethod *string
	Technician    *string
	Notes         *string
	Status        string
	Items         []ServiceOrderItemInput
}

type ServiceOrderItemInput struct {
	CatalogItemID             *uuid.UUID
	ItemType                  string
	Category                  string
	Description               string
	Quantity                  float64
	UnitPriceCents            int64
	KMAtInstallation          *int
	ReplacementIntervalKM     *int
	ReplacementIntervalMonths *int
	WarrantyExpiresDate       *time.Time
	WarrantyExpiresKM         *int
	Notes                     *string
}

type UpdateServiceOrderInput struct {
	WorkspaceID   uuid.UUID
	ID            uuid.UUID
	SupplierID    *uuid.UUID
	OSNumber      *string
	ServiceDate   time.Time
	KMAtService   int
	PaymentMethod *string
	Technician    *string
	Notes         *string
	Status        string
}

func (s *Service) CreateServiceOrder(ctx context.Context, in CreateServiceOrderInput) (*dom.ServiceOrder, error) {
	if _, err := s.repo.GetByID(ctx, in.WorkspaceID, in.VehicleID); err != nil {
		return nil, err
	}

	status := dom.OSStatus(in.Status)
	if status == "" {
		status = dom.OSStatusCompleted
	}

	o := &dom.ServiceOrder{
		ID:          uuid.New(),
		VehicleID:   in.VehicleID,
		WorkspaceID: in.WorkspaceID,
		SupplierID:  in.SupplierID,
		OSNumber:    in.OSNumber,
		ServiceDate: in.ServiceDate,
		KMAtService: in.KMAtService,
		PaymentMethod: in.PaymentMethod,
		Technician:  in.Technician,
		Notes:       in.Notes,
		Status:      status,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.repo.CreateServiceOrder(ctx, o); err != nil {
		return nil, err
	}

	for _, inp := range in.Items {
		item := buildItem(o.ID, in.VehicleID, in.WorkspaceID, in.ServiceDate, in.KMAtService, inp)
		if err := s.repo.CreateServiceOrderItem(ctx, &item); err != nil {
			return nil, err
		}
		if item.ReplacementIntervalKM != nil || item.ReplacementIntervalMonths != nil {
			if err := s.createScheduleFromItem(ctx, o, &item); err != nil {
				s.log.Warn("service_order: create schedule from item failed", "item_id", item.ID, "error", err)
			}
		}
	}

	if err := s.repo.RecalcServiceOrderTotals(ctx, in.WorkspaceID, o.ID); err != nil {
		return nil, err
	}

	return s.repo.GetServiceOrderByID(ctx, in.WorkspaceID, o.ID)
}

func (s *Service) GetServiceOrder(ctx context.Context, workspaceID, id uuid.UUID) (*dom.ServiceOrder, error) {
	return s.repo.GetServiceOrderByID(ctx, workspaceID, id)
}

func (s *Service) ListServiceOrders(ctx context.Context, workspaceID, vehicleID uuid.UUID) ([]dom.ServiceOrder, error) {
	if _, err := s.repo.GetByID(ctx, workspaceID, vehicleID); err != nil {
		return nil, err
	}
	return s.repo.ListServiceOrders(ctx, workspaceID, vehicleID)
}

func (s *Service) UpdateServiceOrder(ctx context.Context, in UpdateServiceOrderInput) (*dom.ServiceOrder, error) {
	o, err := s.repo.GetServiceOrderByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}

	o.SupplierID = in.SupplierID
	o.OSNumber = in.OSNumber
	o.ServiceDate = in.ServiceDate
	o.KMAtService = in.KMAtService
	o.PaymentMethod = in.PaymentMethod
	o.Technician = in.Technician
	o.Notes = in.Notes
	if in.Status != "" {
		o.Status = dom.OSStatus(in.Status)
	}
	o.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateServiceOrder(ctx, o); err != nil {
		return nil, err
	}
	return s.repo.GetServiceOrderByID(ctx, in.WorkspaceID, in.ID)
}

func (s *Service) DeleteServiceOrder(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.DeleteServiceOrder(ctx, workspaceID, id)
}

// ─── Service Order Items ──────────────────────────────────────────────────────

type AddServiceOrderItemInput struct {
	WorkspaceID               uuid.UUID
	ServiceOrderID            uuid.UUID
	CatalogItemID             *uuid.UUID
	ItemType                  string
	Category                  string
	Description               string
	Quantity                  float64
	UnitPriceCents            int64
	KMAtInstallation          *int
	ReplacementIntervalKM     *int
	ReplacementIntervalMonths *int
	WarrantyExpiresDate       *time.Time
	WarrantyExpiresKM         *int
	Notes                     *string
}

func (s *Service) AddServiceOrderItem(ctx context.Context, in AddServiceOrderItemInput) (*dom.ServiceOrderItem, error) {
	o, err := s.repo.GetServiceOrderByID(ctx, in.WorkspaceID, in.ServiceOrderID)
	if err != nil {
		return nil, err
	}
	inp := ServiceOrderItemInput{
		CatalogItemID:             in.CatalogItemID,
		ItemType:                  in.ItemType,
		Category:                  in.Category,
		Description:               in.Description,
		Quantity:                  in.Quantity,
		UnitPriceCents:            in.UnitPriceCents,
		KMAtInstallation:          in.KMAtInstallation,
		ReplacementIntervalKM:     in.ReplacementIntervalKM,
		ReplacementIntervalMonths: in.ReplacementIntervalMonths,
		WarrantyExpiresDate:       in.WarrantyExpiresDate,
		WarrantyExpiresKM:         in.WarrantyExpiresKM,
		Notes:                     in.Notes,
	}
	item := buildItem(o.ID, o.VehicleID, o.WorkspaceID, o.ServiceDate, o.KMAtService, inp)
	if err := s.repo.CreateServiceOrderItem(ctx, &item); err != nil {
		return nil, err
	}
	if err := s.repo.RecalcServiceOrderTotals(ctx, in.WorkspaceID, o.ID); err != nil {
		return nil, err
	}
	if item.ReplacementIntervalKM != nil || item.ReplacementIntervalMonths != nil {
		if err := s.createScheduleFromItem(ctx, o, &item); err != nil {
			s.log.Warn("add_item: create schedule failed", "item_id", item.ID, "error", err)
		}
	}
	return &item, nil
}

func (s *Service) UpdateServiceOrderItem(ctx context.Context, workspaceID, id uuid.UUID, in ServiceOrderItemInput) (*dom.ServiceOrderItem, error) {
	item, err := s.repo.GetServiceOrderItemByID(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}
	category := dom.OSItemCategory(in.Category)
	if category == "" {
		category = dom.OSItemCategoryOutros
	}
	q := in.Quantity
	if q <= 0 {
		q = 1
	}
	item.CatalogItemID = in.CatalogItemID
	item.ItemType = dom.OSItemType(in.ItemType)
	item.Category = category
	item.Description = in.Description
	item.Quantity = q
	item.UnitPriceCents = in.UnitPriceCents
	item.TotalPriceCents = int64(float64(in.UnitPriceCents) * q)
	item.KMAtInstallation = in.KMAtInstallation
	item.ReplacementIntervalKM = in.ReplacementIntervalKM
	item.ReplacementIntervalMonths = in.ReplacementIntervalMonths
	item.WarrantyExpiresDate = in.WarrantyExpiresDate
	item.WarrantyExpiresKM = in.WarrantyExpiresKM
	item.Notes = in.Notes
	if err := s.repo.UpdateServiceOrderItem(ctx, item); err != nil {
		return nil, err
	}
	if err := s.repo.RecalcServiceOrderTotals(ctx, workspaceID, item.ServiceOrderID); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) DeleteServiceOrderItem(ctx context.Context, workspaceID, id uuid.UUID) error {
	item, err := s.repo.GetServiceOrderItemByID(ctx, workspaceID, id)
	if err != nil {
		return err
	}
	osID := item.ServiceOrderID
	if err := s.repo.DeleteServiceOrderItem(ctx, workspaceID, id); err != nil {
		return err
	}
	return s.repo.RecalcServiceOrderTotals(ctx, workspaceID, osID)
}

// ─── Maintenance Catalog ──────────────────────────────────────────────────────

func (s *Service) SearchCatalog(ctx context.Context, query, category string, limit int) ([]dom.MaintenanceCatalogItem, error) {
	return s.repo.SearchCatalog(ctx, query, category, limit)
}

// ─── Schedules ────────────────────────────────────────────────────────────────

type CreateScheduleInput struct {
	WorkspaceID        uuid.UUID
	VehicleID          uuid.UUID
	ServiceOrderItemID *uuid.UUID
	Description        string
	Category           string
	ScheduledKM        *int
	ScheduledDate      *time.Time
	Notes              *string
}

type UpdateScheduleInput struct {
	WorkspaceID   uuid.UUID
	ID            uuid.UUID
	Description   string
	Category      string
	ScheduledKM   *int
	ScheduledDate *time.Time
	AlertStatus   string
	CompletedAt   *time.Time
	Notes         *string
}

func (s *Service) CreateSchedule(ctx context.Context, in CreateScheduleInput) (*dom.MaintenanceSchedule, error) {
	if _, err := s.repo.GetByID(ctx, in.WorkspaceID, in.VehicleID); err != nil {
		return nil, err
	}
	sched := &dom.MaintenanceSchedule{
		ID:                 uuid.New(),
		VehicleID:          in.VehicleID,
		WorkspaceID:        in.WorkspaceID,
		ServiceOrderItemID: in.ServiceOrderItemID,
		Description:        in.Description,
		Category:           dom.OSItemCategory(in.Category),
		ScheduledKM:        in.ScheduledKM,
		ScheduledDate:      in.ScheduledDate,
		AlertStatus:        dom.ScheduleStatusPending,
		Notes:              in.Notes,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if sched.Category == "" {
		sched.Category = dom.OSItemCategoryOutros
	}
	if err := s.repo.CreateSchedule(ctx, sched); err != nil {
		return nil, err
	}
	return sched, nil
}

func (s *Service) ListSchedules(ctx context.Context, workspaceID, vehicleID uuid.UUID) ([]dom.MaintenanceSchedule, error) {
	v, err := s.repo.GetByID(ctx, workspaceID, vehicleID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.RefreshScheduleStatuses(ctx, vehicleID, v.CurrentOdometer); err != nil {
		s.log.Warn("schedules: refresh statuses failed", "vehicle_id", vehicleID, "error", err)
	}
	return s.repo.ListSchedules(ctx, workspaceID, vehicleID)
}

func (s *Service) UpdateSchedule(ctx context.Context, in UpdateScheduleInput) (*dom.MaintenanceSchedule, error) {
	sched, err := s.repo.GetScheduleByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	sched.Description = in.Description
	if in.Category != "" {
		sched.Category = dom.OSItemCategory(in.Category)
	}
	sched.ScheduledKM = in.ScheduledKM
	sched.ScheduledDate = in.ScheduledDate
	if in.AlertStatus != "" {
		sched.AlertStatus = dom.ScheduleStatus(in.AlertStatus)
	}
	sched.CompletedAt = in.CompletedAt
	sched.Notes = in.Notes
	sched.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateSchedule(ctx, sched); err != nil {
		return nil, err
	}
	return sched, nil
}

func (s *Service) DeleteSchedule(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.DeleteSchedule(ctx, workspaceID, id)
}

// ─── Analytics ────────────────────────────────────────────────────────────────

func (s *Service) GetAnalytics(ctx context.Context, workspaceID, vehicleID uuid.UUID, months int) (*dom.VehicleAnalytics, error) {
	if _, err := s.repo.GetByID(ctx, workspaceID, vehicleID); err != nil {
		return nil, err
	}
	return s.repo.GetAnalytics(ctx, workspaceID, vehicleID, months)
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func buildItem(osID, vehicleID, workspaceID uuid.UUID, serviceDate time.Time, kmAtService int, inp ServiceOrderItemInput) dom.ServiceOrderItem {
	category := dom.OSItemCategory(inp.Category)
	if category == "" {
		category = dom.OSItemCategoryOutros
	}
	q := inp.Quantity
	if q <= 0 {
		q = 1
	}
	total := int64(float64(inp.UnitPriceCents) * q)

	item := dom.ServiceOrderItem{
		ID:                        uuid.New(),
		ServiceOrderID:            osID,
		VehicleID:                 vehicleID,
		WorkspaceID:               workspaceID,
		CatalogItemID:             inp.CatalogItemID,
		ItemType:                  dom.OSItemType(inp.ItemType),
		Category:                  category,
		Description:               inp.Description,
		Quantity:                  q,
		UnitPriceCents:            inp.UnitPriceCents,
		TotalPriceCents:           total,
		KMAtInstallation:          inp.KMAtInstallation,
		ReplacementIntervalKM:     inp.ReplacementIntervalKM,
		ReplacementIntervalMonths: inp.ReplacementIntervalMonths,
		WarrantyExpiresDate:       inp.WarrantyExpiresDate,
		WarrantyExpiresKM:         inp.WarrantyExpiresKM,
		Notes:                     inp.Notes,
		CreatedAt:                 time.Now().UTC(),
	}

	// Compute next_due_date in application (date math with months)
	if inp.KMAtInstallation == nil && kmAtService > 0 {
		item.KMAtInstallation = &kmAtService
	}
	if inp.ReplacementIntervalMonths != nil {
		dueDate := serviceDate.AddDate(0, *inp.ReplacementIntervalMonths, 0)
		item.NextDueDate = &dueDate
	}

	return item
}

func (s *Service) createScheduleFromItem(ctx context.Context, o *dom.ServiceOrder, item *dom.ServiceOrderItem) error {
	sched := &dom.MaintenanceSchedule{
		ID:                 uuid.New(),
		VehicleID:          o.VehicleID,
		WorkspaceID:        o.WorkspaceID,
		ServiceOrderItemID: &item.ID,
		Description:        item.Description,
		Category:           item.Category,
		ScheduledKM:        item.NextDueKM,
		ScheduledDate:      item.NextDueDate,
		AlertStatus:        dom.ScheduleStatusPending,
		Notes:              item.Notes,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	return s.repo.CreateSchedule(ctx, sched)
}
