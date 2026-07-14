package vehicle

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/vehicle"
)

// ─── Maintenance (unified) ────────────────────────────────────────────────────

type MaintenanceItemInput struct {
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

type CreateMaintenanceInput struct {
	WorkspaceID         uuid.UUID
	VehicleID           uuid.UUID
	TemplateID          *uuid.UUID
	Type                string
	Title               string
	Description         *string
	OdometerAtService   *int
	ServiceDate         *time.Time // nullable — orçamento ainda não tem data
	Cost                *float64
	SupplierID          *uuid.UUID
	NextServiceOdometer *int
	NextServiceDate     *time.Time
	Notes               *string
	Status              string
	OSNumber            *string
	Technician          *string
	PaymentMethod       *string
	Items               []MaintenanceItemInput
}

type AddMaintenanceItemInput struct {
	WorkspaceID               uuid.UUID
	MaintenanceID             uuid.UUID
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

func (s *Service) CreateMaintenance(ctx context.Context, in CreateMaintenanceInput) (*dom.VehicleMaintenance, error) {
	if _, err := s.repo.GetByID(ctx, in.WorkspaceID, in.VehicleID); err != nil {
		return nil, err
	}

	status := dom.MaintenanceStatus(in.Status)
	if status == "" {
		status = dom.MaintenanceStatusRealizado
	}

	now := time.Now().UTC()
	m := &dom.VehicleMaintenance{
		ID:                  uuid.New(),
		VehicleID:           in.VehicleID,
		WorkspaceID:         in.WorkspaceID,
		TemplateID:          in.TemplateID,
		Type:                in.Type,
		Title:               in.Title,
		Description:         in.Description,
		OdometerAtService:   in.OdometerAtService,
		ServiceDate:         in.ServiceDate,
		Cost:                in.Cost,
		SupplierID:          in.SupplierID,
		NextServiceOdometer: in.NextServiceOdometer,
		NextServiceDate:     in.NextServiceDate,
		Notes:               in.Notes,
		Status:              status,
		OSNumber:            in.OSNumber,
		Technician:          in.Technician,
		PaymentMethod:       in.PaymentMethod,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if m.Type == "" {
		m.Type = "other"
	}
	if m.Title == "" {
		m.Title = m.Type
	}

	if err := s.repo.CreateMaintenance(ctx, m); err != nil {
		return nil, err
	}

	kmAtService := 0
	if in.OdometerAtService != nil {
		kmAtService = *in.OdometerAtService
	}
	for _, inp := range in.Items {
		item := buildMaintenanceItem(m.ID, in.VehicleID, in.WorkspaceID, in.ServiceDate, kmAtService, inp)
		if err := s.repo.CreateMaintenanceItem(ctx, &item); err != nil {
			return nil, err
		}
		if item.ReplacementIntervalKM != nil || item.ReplacementIntervalMonths != nil {
			if err := s.createScheduleFromMaintenanceItem(ctx, m, &item); err != nil {
				s.log.Warn("maintenance: create schedule from item failed", "item_id", item.ID, "error", err)
			}
		}
	}

	if len(in.Items) > 0 {
		if err := s.repo.RecalcMaintenanceTotals(ctx, in.WorkspaceID, m.ID); err != nil {
			return nil, err
		}
	}

	return s.repo.GetMaintenanceByID(ctx, in.WorkspaceID, m.ID)
}

func (s *Service) AddMaintenanceItem(ctx context.Context, in AddMaintenanceItemInput) (*dom.VehicleMaintenanceItem, error) {
	m, err := s.repo.GetMaintenanceByID(ctx, in.WorkspaceID, in.MaintenanceID)
	if err != nil {
		return nil, err
	}
	kmAtService := 0
	if m.OdometerAtService != nil {
		kmAtService = *m.OdometerAtService
	}
	inp := MaintenanceItemInput{
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
	item := buildMaintenanceItem(m.ID, m.VehicleID, m.WorkspaceID, m.ServiceDate, kmAtService, inp)
	if err := s.repo.CreateMaintenanceItem(ctx, &item); err != nil {
		return nil, err
	}
	if err := s.repo.RecalcMaintenanceTotals(ctx, in.WorkspaceID, m.ID); err != nil {
		return nil, err
	}
	if item.ReplacementIntervalKM != nil || item.ReplacementIntervalMonths != nil {
		if err := s.createScheduleFromMaintenanceItem(ctx, m, &item); err != nil {
			s.log.Warn("add_item: create schedule failed", "item_id", item.ID, "error", err)
		}
	}
	return &item, nil
}

func (s *Service) UpdateMaintenanceItem(ctx context.Context, workspaceID, id uuid.UUID, in MaintenanceItemInput) (*dom.VehicleMaintenanceItem, error) {
	item, err := s.repo.GetMaintenanceItemByID(ctx, workspaceID, id)
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
	if err := s.repo.UpdateMaintenanceItem(ctx, item); err != nil {
		return nil, err
	}
	if err := s.repo.RecalcMaintenanceTotals(ctx, workspaceID, item.MaintenanceID); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) DeleteMaintenanceItem(ctx context.Context, workspaceID, id uuid.UUID) error {
	item, err := s.repo.GetMaintenanceItemByID(ctx, workspaceID, id)
	if err != nil {
		return err
	}
	mID := item.MaintenanceID
	if err := s.repo.DeleteMaintenanceItem(ctx, workspaceID, id); err != nil {
		return err
	}
	return s.repo.RecalcMaintenanceTotals(ctx, workspaceID, mID)
}

// ─── Maintenance Catalog ──────────────────────────────────────────────────────

func (s *Service) SearchCatalog(ctx context.Context, query, category string, limit int) ([]dom.MaintenanceCatalogItem, error) {
	return s.repo.SearchCatalog(ctx, query, category, limit)
}

// ─── Schedules ────────────────────────────────────────────────────────────────

type CreateScheduleInput struct {
	WorkspaceID       uuid.UUID
	VehicleID         uuid.UUID
	MaintenanceItemID *uuid.UUID // era ServiceOrderItemID
	Description       string
	Category          string
	ScheduledKM       *int
	ScheduledDate     *time.Time
	Notes             *string
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
		ID:                uuid.New(),
		VehicleID:         in.VehicleID,
		WorkspaceID:       in.WorkspaceID,
		MaintenanceItemID: in.MaintenanceItemID,
		Description:       in.Description,
		Category:          dom.OSItemCategory(in.Category),
		ScheduledKM:       in.ScheduledKM,
		ScheduledDate:     in.ScheduledDate,
		AlertStatus:       dom.ScheduleStatusPending,
		Notes:             in.Notes,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
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

func buildMaintenanceItem(maintenanceID, vehicleID, workspaceID uuid.UUID, serviceDate *time.Time, kmAtService int, inp MaintenanceItemInput) dom.VehicleMaintenanceItem {
	category := dom.OSItemCategory(inp.Category)
	if category == "" {
		category = dom.OSItemCategoryOutros
	}
	q := inp.Quantity
	if q <= 0 {
		q = 1
	}
	total := int64(float64(inp.UnitPriceCents) * q)

	item := dom.VehicleMaintenanceItem{
		ID:                        uuid.New(),
		MaintenanceID:             maintenanceID,
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

	if inp.KMAtInstallation == nil && kmAtService > 0 {
		item.KMAtInstallation = &kmAtService
	}
	// Compute next_due_date in application (date math with months)
	if inp.ReplacementIntervalMonths != nil && serviceDate != nil {
		dueDate := serviceDate.AddDate(0, *inp.ReplacementIntervalMonths, 0)
		item.NextDueDate = &dueDate
	}

	return item
}

func (s *Service) createScheduleFromMaintenanceItem(ctx context.Context, m *dom.VehicleMaintenance, item *dom.VehicleMaintenanceItem) error {
	sched := &dom.MaintenanceSchedule{
		ID:                uuid.New(),
		VehicleID:         m.VehicleID,
		WorkspaceID:       m.WorkspaceID,
		MaintenanceItemID: &item.ID,
		Description:       item.Description,
		Category:          item.Category,
		ScheduledKM:       item.NextDueKM,
		ScheduledDate:     item.NextDueDate,
		AlertStatus:       dom.ScheduleStatusPending,
		Notes:             item.Notes,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	return s.repo.CreateSchedule(ctx, sched)
}
