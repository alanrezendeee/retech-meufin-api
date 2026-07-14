package vehicle

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/vehicle"
	"github.com/retechfin/retechfin-api/internal/infrastructure/fipe"
)

// Service orquestra as regras de negócio do módulo de frota familiar.
type Service struct {
	repo dom.VehicleRepository
	fipe fipe.Searcher
	log  *slog.Logger
}

func NewService(repo dom.VehicleRepository, f fipe.Searcher, log *slog.Logger) *Service {
	return &Service{repo: repo, fipe: f, log: log}
}

// ─── Vehicle CRUD ─────────────────────────────────────────────────────────────

type CreateVehicleInput struct {
	WorkspaceID      uuid.UUID
	Nickname         *string
	Make             string
	Model            string
	YearManufacture  int
	YearModel        int
	Color            *string
	Plate            *string
	FuelType         string
	FipeVehicleType  string
	FipeCode         *string
	FipeBrandCode    *string
	FipeModelCode    *string
	FipeYearCode     *string
	AcquisitionDate  *time.Time
	AcquisitionPrice *float64
	CurrentOdometer  int
	Notes            *string
	MemberIDs        []MemberInput
}

type MemberInput struct {
	MemberID uuid.UUID
	Role     string
}

type UpdateVehicleInput struct {
	WorkspaceID      uuid.UUID
	ID               uuid.UUID
	Nickname         *string
	Make             string
	Model            string
	YearManufacture  int
	YearModel        int
	Color            *string
	Plate            *string
	FuelType         string
	FipeVehicleType  string
	FipeCode         *string
	FipeBrandCode    *string
	FipeModelCode    *string
	FipeYearCode     *string
	AcquisitionDate  *time.Time
	AcquisitionPrice *float64
	CurrentOdometer  int
	Status           string
	SoldAt           *time.Time
	SoldPrice        *float64
	Notes            *string
	MemberIDs        []MemberInput
}

func (s *Service) Create(ctx context.Context, in CreateVehicleInput) (*dom.Vehicle, error) {
	now := time.Now().UTC()
	fvt := in.FipeVehicleType
	if fvt == "" {
		fvt = fipe.VehicleCarros
	}

	members := toMembers(uuid.Nil, in.MemberIDs)
	v := &dom.Vehicle{
		ID:               uuid.New(),
		WorkspaceID:      in.WorkspaceID,
		Nickname:         in.Nickname,
		Make:             in.Make,
		Model:            in.Model,
		YearManufacture:  in.YearManufacture,
		YearModel:        in.YearModel,
		Color:            in.Color,
		Plate:            in.Plate,
		FuelType:         dom.FuelType(in.FuelType),
		FipeVehicleType:  fvt,
		FipeCode:         in.FipeCode,
		FipeBrandCode:    in.FipeBrandCode,
		FipeModelCode:    in.FipeModelCode,
		FipeYearCode:     in.FipeYearCode,
		AcquisitionDate:  in.AcquisitionDate,
		AcquisitionPrice: in.AcquisitionPrice,
		CurrentOdometer:  in.CurrentOdometer,
		Status:           dom.StatusActive,
		Notes:            in.Notes,
		Members:          members,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	// Fix member vehicle IDs after v.ID is set.
	for i := range v.Members {
		v.Members[i].VehicleID = v.ID
	}
	if err := v.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Service) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Vehicle, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

type ListVehiclesResult struct {
	Items []dom.Vehicle
	Total int64
}

func (s *Service) List(ctx context.Context, workspaceID uuid.UUID, status string, limit, offset int) (*ListVehiclesResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, dom.ListVehiclesParams{
		Status: status,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	return &ListVehiclesResult{Items: items, Total: total}, nil
}

func (s *Service) Update(ctx context.Context, in UpdateVehicleInput) (*dom.Vehicle, error) {
	v, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}

	fvt := in.FipeVehicleType
	if fvt == "" {
		fvt = fipe.VehicleCarros
	}
	status := dom.VehicleStatus(in.Status)
	if status == "" {
		status = v.Status
	}

	v.Nickname = in.Nickname
	v.Make = in.Make
	v.Model = in.Model
	v.YearManufacture = in.YearManufacture
	v.YearModel = in.YearModel
	v.Color = in.Color
	v.Plate = in.Plate
	v.FuelType = dom.FuelType(in.FuelType)
	v.FipeVehicleType = fvt
	v.FipeCode = in.FipeCode
	v.FipeBrandCode = in.FipeBrandCode
	v.FipeModelCode = in.FipeModelCode
	v.FipeYearCode = in.FipeYearCode
	v.AcquisitionDate = in.AcquisitionDate
	v.AcquisitionPrice = in.AcquisitionPrice
	v.CurrentOdometer = in.CurrentOdometer
	v.Status = status
	v.SoldAt = in.SoldAt
	v.SoldPrice = in.SoldPrice
	v.Notes = in.Notes
	v.Members = toMembers(v.ID, in.MemberIDs)
	v.UpdatedAt = time.Now().UTC()

	if err := v.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Service) UpdateOdometer(ctx context.Context, workspaceID, id uuid.UUID, odometer int) (*dom.Vehicle, error) {
	v, err := s.repo.GetByID(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}
	if odometer < v.CurrentOdometer {
		return nil, &dom.ValidationError{Msg: "odômetro não pode ser menor que o valor atual"}
	}
	v.CurrentOdometer = odometer
	v.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Service) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.Delete(ctx, workspaceID, id)
}

// ─── Maintenance ──────────────────────────────────────────────────────────────

type UpdateMaintenanceInput struct {
	WorkspaceID         uuid.UUID
	ID                  uuid.UUID
	Title               string
	Description         *string
	OdometerAtService   *int
	ServiceDate         *time.Time // nullable
	Cost                *float64
	SupplierID          *uuid.UUID
	NextServiceOdometer *int
	NextServiceDate     *time.Time
	Notes               *string
	Status              string
}

func (s *Service) GetMaintenance(ctx context.Context, workspaceID, id uuid.UUID) (*dom.VehicleMaintenance, error) {
	return s.repo.GetMaintenanceByID(ctx, workspaceID, id)
}

func (s *Service) ListMaintenance(ctx context.Context, workspaceID, vehicleID uuid.UUID) ([]dom.VehicleMaintenance, error) {
	if _, err := s.repo.GetByID(ctx, workspaceID, vehicleID); err != nil {
		return nil, err
	}
	return s.repo.ListMaintenance(ctx, workspaceID, vehicleID)
}

func (s *Service) UpdateMaintenance(ctx context.Context, in UpdateMaintenanceInput) (*dom.VehicleMaintenance, error) {
	m, err := s.repo.GetMaintenanceByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	m.Title = in.Title
	m.Description = in.Description
	m.OdometerAtService = in.OdometerAtService
	m.ServiceDate = in.ServiceDate
	m.Cost = in.Cost
	m.SupplierID = in.SupplierID
	m.NextServiceOdometer = in.NextServiceOdometer
	m.NextServiceDate = in.NextServiceDate
	m.Notes = in.Notes
	if in.Status != "" {
		m.Status = dom.MaintenanceStatus(in.Status)
	}
	m.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateMaintenance(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Service) DeleteMaintenance(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.DeleteMaintenance(ctx, workspaceID, id)
}

// ─── Maintenance plans ────────────────────────────────────────────────────────

type UpdatePlanInput struct {
	WorkspaceID  uuid.UUID
	VehicleID    uuid.UUID
	TemplateID   uuid.UUID
	IntervalKM   *int
	IntervalDays *int
	Enabled      bool
}

func (s *Service) GetPlans(ctx context.Context, workspaceID, vehicleID uuid.UUID) ([]dom.VehicleMaintenancePlan, error) {
	if _, err := s.repo.GetByID(ctx, workspaceID, vehicleID); err != nil {
		return nil, err
	}
	return s.repo.GetVehiclePlans(ctx, workspaceID, vehicleID)
}

func (s *Service) UpdatePlan(ctx context.Context, in UpdatePlanInput) (*dom.VehicleMaintenancePlan, error) {
	if _, err := s.repo.GetByID(ctx, in.WorkspaceID, in.VehicleID); err != nil {
		return nil, err
	}
	p := &dom.VehicleMaintenancePlan{
		VehicleID:    in.VehicleID,
		WorkspaceID:  in.WorkspaceID,
		TemplateID:   in.TemplateID,
		IntervalKM:   in.IntervalKM,
		IntervalDays: in.IntervalDays,
		Enabled:      in.Enabled,
		UpdatedAt:    time.Now().UTC(),
	}
	if err := s.repo.UpsertVehiclePlan(ctx, p); err != nil {
		return nil, err
	}
	// Reload with template
	plans, err := s.repo.GetVehiclePlans(ctx, in.WorkspaceID, in.VehicleID)
	if err != nil {
		return nil, err
	}
	for i := range plans {
		if plans[i].TemplateID == in.TemplateID {
			return &plans[i], nil
		}
	}
	return p, nil
}

// ─── Alerts ───────────────────────────────────────────────────────────────────

func (s *Service) CalculateAlerts(ctx context.Context, workspaceID, vehicleID uuid.UUID) ([]dom.MaintenanceAlert, error) {
	v, err := s.repo.GetByID(ctx, workspaceID, vehicleID)
	if err != nil {
		return nil, err
	}
	plans, err := s.repo.GetVehiclePlans(ctx, workspaceID, vehicleID)
	if err != nil {
		return nil, err
	}
	lastByType, err := s.repo.LastMaintenanceByType(ctx, vehicleID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	alerts := make([]dom.MaintenanceAlert, 0, len(plans))

	for _, plan := range plans {
		if !plan.Enabled || plan.Template == nil {
			continue
		}
		alert := dom.MaintenanceAlert{
			TemplateID: plan.TemplateID,
			Type:       plan.Template.Type,
			Title:      plan.Template.Name,
			Status:     dom.AlertUnknown,
		}

		effectiveKM := plan.IntervalKM
		if effectiveKM == nil {
			effectiveKM = plan.Template.DefaultIntervalKM
		}
		effectiveDays := plan.IntervalDays
		if effectiveDays == nil {
			effectiveDays = plan.Template.DefaultIntervalDays
		}

		last, hasLast := lastByType[plan.Template.Type]
		if !hasLast {
			if effectiveKM != nil {
				dueKM := *effectiveKM
				alert.DueAtKM = &dueKM
			}
			if effectiveDays != nil {
				suggested := now.AddDate(0, 0, *effectiveDays)
				alert.DueAtDate = &suggested
			}
			alerts = append(alerts, alert)
			continue
		}

		alert.LastDate = last.ServiceDate
		alert.LastOdometer = last.OdometerAtService

		overdue := false
		dueSoon := false

		if effectiveKM != nil && last.OdometerAtService != nil {
			dueKM := *last.OdometerAtService + *effectiveKM
			remaining := dueKM - v.CurrentOdometer
			alert.DueAtKM = &dueKM
			alert.KMRemaining = &remaining
			if remaining <= 0 {
				overdue = true
			} else if remaining <= 500 {
				dueSoon = true
			}
		}

		if effectiveDays != nil && last.ServiceDate != nil {
			dueDate := last.ServiceDate.AddDate(0, 0, *effectiveDays)
			remaining := int(dueDate.Sub(now).Hours() / 24)
			alert.DueAtDate = &dueDate
			alert.DaysRemaining = &remaining
			if remaining <= 0 {
				overdue = true
			} else if remaining <= 30 {
				dueSoon = true
			}
		}

		switch {
		case overdue:
			alert.Status = dom.AlertOverdue
		case dueSoon:
			alert.Status = dom.AlertDueSoon
		default:
			alert.Status = dom.AlertOK
		}

		alerts = append(alerts, alert)
	}
	return alerts, nil
}

// ─── FIPE search (proxy com cache) ───────────────────────────────────────────

func (s *Service) FipeListBrands(ctx context.Context, vehicleType string) ([]fipe.Brand, error) {
	return s.fipe.ListBrands(ctx, vehicleType)
}

func (s *Service) FipeListModels(ctx context.Context, vehicleType, brandCode string) ([]fipe.Model, error) {
	return s.fipe.ListModels(ctx, vehicleType, brandCode)
}

func (s *Service) FipeListYears(ctx context.Context, vehicleType, brandCode, modelCode string) ([]fipe.Year, error) {
	return s.fipe.ListYears(ctx, vehicleType, brandCode, modelCode)
}

func (s *Service) FipeGetPrice(ctx context.Context, vehicleType, brandCode, modelCode, yearCode string) (*fipe.Price, error) {
	return s.fipe.GetPrice(ctx, vehicleType, brandCode, modelCode, yearCode)
}

func (s *Service) FipeGetAllYearPrices(ctx context.Context, vehicleType, brandCode, modelCode string) ([]fipe.Price, error) {
	return s.fipe.GetAllYearPrices(ctx, vehicleType, brandCode, modelCode)
}

// ─── Depreciation ─────────────────────────────────────────────────────────────

func (s *Service) GetDepreciation(ctx context.Context, workspaceID, vehicleID uuid.UUID) (*dom.DepreciationReport, error) {
	v, err := s.repo.GetByID(ctx, workspaceID, vehicleID)
	if err != nil {
		return nil, err
	}
	history, err := s.repo.ListFipeHistory(ctx, vehicleID)
	if err != nil {
		return nil, err
	}

	report := &dom.DepreciationReport{
		AcquisitionPrice: v.AcquisitionPrice,
		History:          history,
	}

	if len(history) == 0 {
		return report, nil
	}

	latestFipe := history[len(history)-1].FipeValue
	report.CurrentFipeValue = &latestFipe

	if v.AcquisitionPrice != nil && *v.AcquisitionPrice > 0 {
		totalR := *v.AcquisitionPrice - latestFipe
		totalPct := totalR / *v.AcquisitionPrice * 100
		report.TotalDepreciationR = &totalR
		report.TotalDepreciationPct = &totalPct
	}

	if v.AcquisitionDate != nil {
		months := monthsBetween(*v.AcquisitionDate, time.Now())
		if months < 1 {
			months = 1
		}
		report.MonthsOwned = months
		if report.TotalDepreciationR != nil {
			monthly := *report.TotalDepreciationR / float64(months)
			annual := monthly * 12
			report.MonthlyAvgDeprecR = &monthly
			report.AnnualAvgDeprecR = &annual
		}
	}

	// Tendência: média das variações dos últimos 6 meses do histórico.
	if len(history) >= 2 {
		end := len(history)
		start := end - 6
		if start < 1 {
			start = 1
		}
		var sumDelta float64
		count := 0
		for i := start; i < end; i++ {
			sumDelta += history[i].FipeValue - history[i-1].FipeValue
			count++
		}
		if count > 0 {
			trend := sumDelta / float64(count)
			report.Trend6MonthsR = &trend
		}
	}

	return report, nil
}

func (s *Service) ListFipeHistory(ctx context.Context, workspaceID, vehicleID uuid.UUID) ([]dom.VehicleFipeHistory, error) {
	if _, err := s.repo.GetByID(ctx, workspaceID, vehicleID); err != nil {
		return nil, err
	}
	return s.repo.ListFipeHistory(ctx, vehicleID)
}

// ─── Cron: atualização mensal do valor FIPE ───────────────────────────────────

// RefreshFipeHistory busca o valor FIPE atual de todos os veículos ativos
// que têm fipe_codes configurados e persiste no histórico mensal.
// referenceMonth deve estar no formato "01/2006" (MM/AAAA).
func (s *Service) RefreshFipeHistory(ctx context.Context, referenceMonth string) (int, error) {
	vehicles, err := s.repo.ListActiveVehiclesForFipeUpdate(ctx)
	if err != nil {
		return 0, err
	}

	updated := 0
	for _, v := range vehicles {
		price, err := s.fipe.GetPrice(ctx, v.FipeVehicleType, *v.FipeBrandCode, *v.FipeModelCode, *v.FipeYearCode)
		if err != nil {
			s.log.Warn("fipe: refresh: skip vehicle",
				slog.String("vehicle_id", v.ID.String()),
				slog.String("error", err.Error()),
			)
			continue
		}
		val := fipe.ParseFipeValue(price.Value)
		fuel := price.Fuel
		if err := s.repo.AddFipeHistory(ctx, &dom.VehicleFipeHistory{
			VehicleID:      v.ID,
			WorkspaceID:    v.WorkspaceID,
			ReferenceMonth: referenceMonth,
			FipeValue:      val,
			FipeFuel:       &fuel,
		}); err != nil {
			s.log.Warn("fipe: refresh: persist failed",
				slog.String("vehicle_id", v.ID.String()),
				slog.String("error", err.Error()),
			)
			continue
		}
		updated++
	}
	return updated, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func toMembers(vehicleID uuid.UUID, inputs []MemberInput) []dom.VehicleMember {
	members := make([]dom.VehicleMember, 0, len(inputs))
	for _, m := range inputs {
		role := dom.MemberRole(m.Role)
		if role == "" {
			role = dom.RoleDriver
		}
		members = append(members, dom.VehicleMember{
			VehicleID: vehicleID,
			MemberID:  m.MemberID,
			Role:      role,
		})
	}
	return members
}

func monthsBetween(from, to time.Time) int {
	years := to.Year() - from.Year()
	months := int(to.Month()) - int(from.Month())
	return years*12 + months
}
