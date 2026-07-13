package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/vehicle"
	"gorm.io/gorm"
)

// ─── Service Orders ───────────────────────────────────────────────────────────

func (r *VehicleRepository) CreateServiceOrder(ctx context.Context, o *dom.ServiceOrder) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	m := serviceOrderToModel(o)
	return r.db.WithContext(ctx).Omit("Items").Create(&m).Error
}

func (r *VehicleRepository) GetServiceOrderByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.ServiceOrder, error) {
	var m ServiceOrderModel
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	out := modelToServiceOrder(m)
	return &out, nil
}

func (r *VehicleRepository) ListServiceOrders(ctx context.Context, workspaceID, vehicleID uuid.UUID) ([]dom.ServiceOrder, error) {
	var models []ServiceOrderModel
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Where("workspace_id = ? AND vehicle_id = ?", workspaceID.String(), vehicleID.String()).
		Order("service_date DESC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.ServiceOrder, len(models))
	for i, m := range models {
		out[i] = modelToServiceOrder(m)
	}
	return out, nil
}

func (r *VehicleRepository) UpdateServiceOrder(ctx context.Context, o *dom.ServiceOrder) error {
	m := serviceOrderToModel(o)
	return r.db.WithContext(ctx).Omit("Items", "CreatedAt").Save(&m).Error
}

func (r *VehicleRepository) DeleteServiceOrder(ctx context.Context, workspaceID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		Delete(&ServiceOrderModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

// ─── Service Order Items ──────────────────────────────────────────────────────

func (r *VehicleRepository) CreateServiceOrderItem(ctx context.Context, item *dom.ServiceOrderItem) error {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	m := serviceOrderItemToModel(item)
	return r.db.WithContext(ctx).Create(&m).Error
}

func (r *VehicleRepository) GetServiceOrderItemByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.ServiceOrderItem, error) {
	var m ServiceOrderItemModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	out := modelToServiceOrderItem(m)
	return &out, nil
}

func (r *VehicleRepository) ListServiceOrderItems(ctx context.Context, workspaceID, serviceOrderID uuid.UUID) ([]dom.ServiceOrderItem, error) {
	var models []ServiceOrderItemModel
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND service_order_id = ?", workspaceID.String(), serviceOrderID.String()).
		Order("created_at ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.ServiceOrderItem, len(models))
	for i, m := range models {
		out[i] = modelToServiceOrderItem(m)
	}
	return out, nil
}

func (r *VehicleRepository) UpdateServiceOrderItem(ctx context.Context, item *dom.ServiceOrderItem) error {
	m := serviceOrderItemToModel(item)
	return r.db.WithContext(ctx).Omit("CreatedAt").Save(&m).Error
}

func (r *VehicleRepository) DeleteServiceOrderItem(ctx context.Context, workspaceID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		Delete(&ServiceOrderItemModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *VehicleRepository) RecalcServiceOrderTotals(ctx context.Context, workspaceID, serviceOrderID uuid.UUID) error {
	type totals struct {
		ProductsCents int64
		ServicesCents int64
	}
	var t totals
	err := r.db.WithContext(ctx).
		Model(&ServiceOrderItemModel{}).
		Select(
			"COALESCE(SUM(CASE WHEN item_type = 'product' THEN total_price_cents ELSE 0 END), 0) AS products_cents, "+
				"COALESCE(SUM(CASE WHEN item_type = 'service' THEN total_price_cents ELSE 0 END), 0) AS services_cents",
		).
		Where("service_order_id = ? AND workspace_id = ?", serviceOrderID.String(), workspaceID.String()).
		Scan(&t).Error
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).
		Model(&ServiceOrderModel{}).
		Where("id = ? AND workspace_id = ?", serviceOrderID.String(), workspaceID.String()).
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

	// Due soon por km (overdue já foi tratado acima, não precisa excluir 'overdue' aqui pois condição due_soon só se aplica a não-overdue)
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

	baseQ := r.db.WithContext(ctx).
		Model(&ServiceOrderModel{}).
		Where("workspace_id = ? AND vehicle_id = ? AND status != 'cancelled' AND service_date >= ?", wsID, vID, since)

	// Totais gerais
	type summary struct {
		TotalCents    int64
		ProductsCents int64
		ServicesCents int64
		OSCount       int
	}
	var s summary
	if err := baseQ.Select(
		"COALESCE(SUM(total_cents),0) AS total_cents," +
			"COALESCE(SUM(total_products_cents),0) AS products_cents," +
			"COALESCE(SUM(total_services_cents),0) AS services_cents," +
			"COUNT(*) AS os_count",
	).Scan(&s).Error; err != nil {
		return nil, err
	}

	avg := int64(0)
	if s.OSCount > 0 {
		avg = s.TotalCents / int64(s.OSCount)
	}

	// Odômetro atual para custo por km
	var vehicle VehicleModel
	var costPerKM *float64
	if err := r.db.WithContext(ctx).Select("current_odometer").Where("id = ?", vID).First(&vehicle).Error; err == nil && vehicle.CurrentOdometer > 0 && s.TotalCents > 0 {
		val := float64(s.TotalCents) / 100.0 / float64(vehicle.CurrentOdometer)
		costPerKM = &val
	}

	// Gasto por categoria
	type catRow struct {
		Category   string
		TotalCents int64
	}
	var catRows []catRow
	r.db.WithContext(ctx).
		Model(&ServiceOrderItemModel{}).
		Select("category, COALESCE(SUM(total_price_cents),0) AS total_cents").
		Where("workspace_id = ? AND vehicle_id = ? AND EXISTS (SELECT 1 FROM vehicle_service_orders o WHERE o.id = service_order_id AND o.status != 'cancelled' AND o.service_date >= ?)", wsID, vID, since).
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
		Model(&ServiceOrderModel{}).
		Joins("LEFT JOIN suppliers s ON s.id::text = vehicle_service_orders.supplier_id::text").
		Select("vehicle_service_orders.supplier_id, COALESCE(s.name,'Sem fornecedor') AS supplier_name, COALESCE(SUM(vehicle_service_orders.total_cents),0) AS total_cents").
		Where("vehicle_service_orders.workspace_id = ? AND vehicle_service_orders.vehicle_id = ? AND vehicle_service_orders.status != 'cancelled' AND vehicle_service_orders.service_date >= ?", wsID, vID, since).
		Group("vehicle_service_orders.supplier_id, s.name").
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
		Model(&ServiceOrderModel{}).
		Select("TO_CHAR(service_date, 'YYYY-MM') AS month, COALESCE(SUM(total_cents),0) AS total_cents").
		Where("workspace_id = ? AND vehicle_id = ? AND status != 'cancelled' AND service_date >= ?", wsID, vID, since).
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
		TotalOSCount:       s.OSCount,
		AvgCostPerOSCents:  avg,
		SpendingByCategory: byCategory,
		SpendingBySupplier: bySupplier,
		MonthlySpending:    monthly,
	}, nil
}
