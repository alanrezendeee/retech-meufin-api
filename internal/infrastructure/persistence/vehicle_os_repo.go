package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/vehicle"
	"gorm.io/gorm"
)

// ─── Maintenance Items ────────────────────────────────────────────────────────

func (r *VehicleRepository) CreateMaintenanceItem(ctx context.Context, item *dom.VehicleMaintenanceItem) error {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	m := maintenanceItemToModel(item)
	return r.db.WithContext(ctx).Create(&m).Error
}

func (r *VehicleRepository) GetMaintenanceItemByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.VehicleMaintenanceItem, error) {
	var m MaintenanceItemModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	out := modelToMaintenanceItem(m)
	return &out, nil
}

func (r *VehicleRepository) ListMaintenanceItems(ctx context.Context, workspaceID, maintenanceID uuid.UUID) ([]dom.VehicleMaintenanceItem, error) {
	var models []MaintenanceItemModel
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND maintenance_id = ?", workspaceID.String(), maintenanceID.String()).
		Order("created_at ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.VehicleMaintenanceItem, len(models))
	for i, m := range models {
		out[i] = modelToMaintenanceItem(m)
	}
	return out, nil
}

func (r *VehicleRepository) UpdateMaintenanceItem(ctx context.Context, item *dom.VehicleMaintenanceItem) error {
	m := maintenanceItemToModel(item)
	return r.db.WithContext(ctx).Omit("CreatedAt").Save(&m).Error
}

func (r *VehicleRepository) DeleteMaintenanceItem(ctx context.Context, workspaceID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		Delete(&MaintenanceItemModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *VehicleRepository) RecalcMaintenanceTotals(ctx context.Context, workspaceID, maintenanceID uuid.UUID) error {
	type totals struct {
		ProductsCents int64
		ServicesCents int64
	}
	var t totals
	err := r.db.WithContext(ctx).
		Model(&MaintenanceItemModel{}).
		Select(
			"COALESCE(SUM(CASE WHEN item_type = 'product' THEN total_price_cents ELSE 0 END), 0) AS products_cents, " +
				"COALESCE(SUM(CASE WHEN item_type = 'service' THEN total_price_cents ELSE 0 END), 0) AS services_cents",
		).
		Where("maintenance_id = ? AND workspace_id = ?", maintenanceID.String(), workspaceID.String()).
		Scan(&t).Error
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).
		Model(&VehicleMaintenanceModel{}).
		Where("id = ? AND workspace_id = ?", maintenanceID.String(), workspaceID.String()).
		Updates(map[string]any{
			"total_products_cents": t.ProductsCents,
			"total_services_cents": t.ServicesCents,
			"total_cents":          t.ProductsCents + t.ServicesCents,
		}).Error
}

// ─── Maintenance Catalog ──────────────────────────────────────────────────────

func (r *VehicleRepository) SearchCatalog(ctx context.Context, query, category string, limit int) ([]dom.MaintenanceCatalogItem, error) {
	q := r.db.WithContext(ctx).
		Model(&MaintenanceCatalogItemModel{}).
		Where("active = true")
	if query != "" {
		q = q.Where("name ILIKE ?", "%"+query+"%")
	}
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if limit <= 0 {
		limit = 20
	}
	var models []MaintenanceCatalogItemModel
	if err := q.Order("sort_order ASC, name ASC").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.MaintenanceCatalogItem, len(models))
	for i, m := range models {
		out[i] = modelToCatalogItem(m)
	}
	return out, nil
}

// ─── Maintenance Schedules ────────────────────────────────────────────────────

func (r *VehicleRepository) CreateSchedule(ctx context.Context, s *dom.MaintenanceSchedule) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	m := scheduleToModel(s)
	return r.db.WithContext(ctx).Create(&m).Error
}

func (r *VehicleRepository) GetScheduleByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.MaintenanceSchedule, error) {
	var m MaintenanceScheduleModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	out := modelToSchedule(m)
	return &out, nil
}

func (r *VehicleRepository) ListSchedules(ctx context.Context, workspaceID, vehicleID uuid.UUID) ([]dom.MaintenanceSchedule, error) {
	var models []MaintenanceScheduleModel
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND vehicle_id = ?", workspaceID.String(), vehicleID.String()).
		Order("CASE alert_status WHEN 'overdue' THEN 1 WHEN 'due_soon' THEN 2 WHEN 'pending' THEN 3 ELSE 4 END, scheduled_date ASC NULLS LAST").
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.MaintenanceSchedule, len(models))
	for i, m := range models {
		out[i] = modelToSchedule(m)
	}
	return out, nil
}

func (r *VehicleRepository) UpdateSchedule(ctx context.Context, s *dom.MaintenanceSchedule) error {
	m := scheduleToModel(s)
	return r.db.WithContext(ctx).Omit("CreatedAt").Save(&m).Error
}

func (r *VehicleRepository) DeleteSchedule(ctx context.Context, workspaceID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		Delete(&MaintenanceScheduleModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

// RefreshScheduleStatuses recalcula alert_status baseado no km atual e na data de hoje.
// overdue: km/date já passou | due_soon: dentro de 1000km ou 30 dias | pending: fora disso.
func (r *VehicleRepository) RefreshScheduleStatuses(ctx context.Context, vehicleID uuid.UUID, currentKM int) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	dueSoonDate := today.AddDate(0, 0, 30)

	// Overdue por km
	if err := r.db.WithContext(ctx).
		Model(&MaintenanceScheduleModel{}).
		Where("vehicle_id = ? AND alert_status NOT IN ('done','cancelled') AND scheduled_km IS NOT NULL AND scheduled_km <= ?", vehicleID.String(), currentKM).
		Update("alert_status", "overdue").Error; err != nil {
		return err
	}

	// Overdue por data
	if err := r.db.WithContext(ctx).
		Model(&MaintenanceScheduleModel{}).
		Where("vehicle_id = ? AND alert_status NOT IN ('done','cancelled') AND scheduled_date IS NOT NULL AND scheduled_date < ?", vehicleID.String(), today).
		Update("alert_status", "overdue").Error; err != nil {
		return err
	}

	// Due soon por km
	if err := r.db.WithContext(ctx).
		Model(&MaintenanceScheduleModel{}).
		Where("vehicle_id = ? AND alert_status NOT IN ('done','cancelled','overdue') AND scheduled_km IS NOT NULL AND scheduled_km <= ?", vehicleID.String(), currentKM+1000).
		Update("alert_status", "due_soon").Error; err != nil {
		return err
	}

	// Due soon por data
	if err := r.db.WithContext(ctx).
		Model(&MaintenanceScheduleModel{}).
		Where("vehicle_id = ? AND alert_status NOT IN ('done','cancelled','overdue') AND scheduled_date IS NOT NULL AND scheduled_date <= ?", vehicleID.String(), dueSoonDate).
		Update("alert_status", "due_soon").Error; err != nil {
		return err
	}

	// Restante = pending
	return r.db.WithContext(ctx).
		Model(&MaintenanceScheduleModel{}).
		Where("vehicle_id = ? AND alert_status NOT IN ('done','cancelled','overdue','due_soon')", vehicleID.String()).
		Update("alert_status", "pending").Error
}

// ─── Analytics ────────────────────────────────────────────────────────────────

func (r *VehicleRepository) GetAnalytics(ctx context.Context, workspaceID, vehicleID uuid.UUID, months int) (*dom.VehicleAnalytics, error) {
	if months <= 0 {
		months = 12
	}
	since := time.Now().UTC().AddDate(0, -months, 0)

	wsID := workspaceID.String()
	vID := vehicleID.String()

	// Totais gerais a partir de vehicle_maintenance
	type summary struct {
		TotalCents    int64
		ProductsCents int64
		ServicesCents int64
		TotalCount    int
	}
	var s summary
	if err := r.db.WithContext(ctx).
		Model(&VehicleMaintenanceModel{}).
		Select(
			"COALESCE(SUM(total_cents),0) AS total_cents," +
				"COALESCE(SUM(total_products_cents),0) AS products_cents," +
				"COALESCE(SUM(total_services_cents),0) AS services_cents," +
				"COUNT(*) FILTER (WHERE status = 'realizado') AS total_count",
		).
		Where("workspace_id = ? AND vehicle_id = ? AND status != 'cancelado' AND service_date >= ?", wsID, vID, since).
		Scan(&s).Error; err != nil {
		return nil, err
	}

	avg := int64(0)
	if s.TotalCount > 0 {
		avg = s.TotalCents / int64(s.TotalCount)
	}

	// Odômetro atual para custo por km
	var vehicle VehicleModel
	var costPerKM *float64
	if err := r.db.WithContext(ctx).Select("current_odometer").Where("id = ?", vID).First(&vehicle).Error; err == nil && vehicle.CurrentOdometer > 0 && s.TotalCents > 0 {
		val := float64(s.TotalCents) / 100.0 / float64(vehicle.CurrentOdometer)
		costPerKM = &val
	}

	// Gasto por categoria via items
	type catRow struct {
		Category   string
		TotalCents int64
	}
	var catRows []catRow
	r.db.WithContext(ctx).
		Model(&MaintenanceItemModel{}).
		Select("category, COALESCE(SUM(total_price_cents),0) AS total_cents").
		Where("workspace_id = ? AND vehicle_id = ? AND EXISTS (SELECT 1 FROM vehicle_maintenance m WHERE m.id = maintenance_id AND m.status != 'cancelado' AND m.service_date >= ?)", wsID, vID, since).
		Group("category").
		Order("total_cents DESC").
		Scan(&catRows)
	byCategory := make([]dom.CategorySpending, len(catRows))
	for i, row := range catRows {
		byCategory[i] = dom.CategorySpending{Category: row.Category, TotalCents: row.TotalCents}
	}

	// Gasto por fornecedor
	type supRow struct {
		SupplierID   *string
		SupplierName string
		TotalCents   int64
	}
	var supRows []supRow
	r.db.WithContext(ctx).
		Model(&VehicleMaintenanceModel{}).
		Joins("LEFT JOIN suppliers s ON s.id::text = vehicle_maintenance.supplier_id::text").
		Select("vehicle_maintenance.supplier_id, COALESCE(s.name,'Sem fornecedor') AS supplier_name, COALESCE(SUM(vehicle_maintenance.total_cents),0) AS total_cents").
		Where("vehicle_maintenance.workspace_id = ? AND vehicle_maintenance.vehicle_id = ? AND vehicle_maintenance.status != 'cancelado' AND vehicle_maintenance.service_date >= ?", wsID, vID, since).
		Group("vehicle_maintenance.supplier_id, s.name").
		Order("total_cents DESC").
		Limit(10).
		Scan(&supRows)
	bySupplier := make([]dom.SupplierSpending, len(supRows))
	for i, row := range supRows {
		sid := ""
		if row.SupplierID != nil {
			sid = *row.SupplierID
		}
		bySupplier[i] = dom.SupplierSpending{SupplierID: sid, SupplierName: row.SupplierName, TotalCents: row.TotalCents}
	}

	// Gasto mensal
	type monthRow struct {
		Month      string
		TotalCents int64
	}
	var monthRows []monthRow
	r.db.WithContext(ctx).
		Model(&VehicleMaintenanceModel{}).
		Select("TO_CHAR(service_date, 'YYYY-MM') AS month, COALESCE(SUM(total_cents),0) AS total_cents").
		Where("workspace_id = ? AND vehicle_id = ? AND status != 'cancelado' AND service_date >= ?", wsID, vID, since).
		Group("month").
		Order("month ASC").
		Scan(&monthRows)
	monthly := make([]dom.MonthlySpending, len(monthRows))
	for i, row := range monthRows {
		monthly[i] = dom.MonthlySpending{Month: row.Month, TotalCents: row.TotalCents}
	}

	return &dom.VehicleAnalytics{
		TotalSpentCents:    s.TotalCents,
		TotalProductsCents: s.ProductsCents,
		TotalServicesCents: s.ServicesCents,
		CostPerKM:          costPerKM,
		TotalCount:         s.TotalCount,
		AvgCostPerOSCents:  avg,
		SpendingByCategory: byCategory,
		SpendingBySupplier: bySupplier,
		MonthlySpending:    monthly,
	}, nil
}
