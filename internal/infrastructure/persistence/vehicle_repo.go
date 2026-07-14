package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/vehicle"
	"gorm.io/gorm"
)

// VehicleRepository implementa dom.VehicleRepository com GORM/Postgres.
type VehicleRepository struct {
	db *gorm.DB
}

func NewVehicleRepository(db *gorm.DB) *VehicleRepository {
	return &VehicleRepository{db: db}
}

// ─── Vehicles ─────────────────────────────────────────────────────────────────

func (r *VehicleRepository) Create(ctx context.Context, v *dom.Vehicle) error {
	m := vehicleToModel(v)
	if err := r.db.WithContext(ctx).Omit("Members").Create(&m).Error; err != nil {
		return fmt.Errorf("vehicle: create: %w", err)
	}
	if len(v.Members) > 0 {
		return r.SetMembers(ctx, v.ID, v.Members)
	}
	return nil
}

func (r *VehicleRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Vehicle, error) {
	var m VehicleModel
	err := r.db.WithContext(ctx).
		Preload("Members").
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	v := modelToVehicle(m)
	return &v, nil
}

func (r *VehicleRepository) List(ctx context.Context, workspaceID uuid.UUID, p dom.ListVehiclesParams) ([]dom.Vehicle, int64, error) {
	q := r.db.WithContext(ctx).Model(&VehicleModel{}).
		Where("workspace_id = ?", workspaceID.String())
	if p.Status != "" {
		q = q.Where("status = ?", p.Status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	var models []VehicleModel
	if err := q.Preload("Members").
		Limit(limit).Offset(p.Offset).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, 0, err
	}

	out := make([]dom.Vehicle, len(models))
	for i, m := range models {
		out[i] = modelToVehicle(m)
	}
	return out, total, nil
}

func (r *VehicleRepository) Update(ctx context.Context, v *dom.Vehicle) error {
	m := vehicleToModel(v)
	if err := r.db.WithContext(ctx).Omit("Members", "CreatedAt").Save(&m).Error; err != nil {
		return fmt.Errorf("vehicle: update: %w", err)
	}
	return r.SetMembers(ctx, v.ID, v.Members)
}

func (r *VehicleRepository) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		Delete(&VehicleModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *VehicleRepository) SetMembers(ctx context.Context, vehicleID uuid.UUID, members []dom.VehicleMember) error {
	if err := r.db.WithContext(ctx).
		Where("vehicle_id = ?", vehicleID.String()).
		Delete(&VehicleMemberModel{}).Error; err != nil {
		return err
	}
	if len(members) == 0 {
		return nil
	}
	models := make([]VehicleMemberModel, len(members))
	for i, m := range members {
		models[i] = VehicleMemberModel{
			VehicleID: vehicleID.String(),
			MemberID:  m.MemberID.String(),
			Role:      string(m.Role),
		}
	}
	return r.db.WithContext(ctx).Create(&models).Error
}

// ─── Maintenance ──────────────────────────────────────────────────────────────

func (r *VehicleRepository) CreateMaintenance(ctx context.Context, m *dom.VehicleMaintenance) error {
	model := maintenanceToModel(m)
	if err := r.db.WithContext(ctx).Omit("Items").Create(&model).Error; err != nil {
		return fmt.Errorf("maintenance: create: %w", err)
	}
	return nil
}

func (r *VehicleRepository) GetMaintenanceByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.VehicleMaintenance, error) {
	var m VehicleMaintenanceModel
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
	out := modelToMaintenance(m)
	return &out, nil
}

func (r *VehicleRepository) ListMaintenance(ctx context.Context, workspaceID, vehicleID uuid.UUID) ([]dom.VehicleMaintenance, error) {
	var models []VehicleMaintenanceModel
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Where("workspace_id = ? AND vehicle_id = ?", workspaceID.String(), vehicleID.String()).
		Order("COALESCE(service_date, created_at) DESC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.VehicleMaintenance, len(models))
	for i, m := range models {
		out[i] = modelToMaintenance(m)
	}
	return out, nil
}

func (r *VehicleRepository) UpdateMaintenance(ctx context.Context, m *dom.VehicleMaintenance) error {
	model := maintenanceToModel(m)
	return r.db.WithContext(ctx).Omit("Items", "CreatedAt").Save(&model).Error
}

func (r *VehicleRepository) DeleteMaintenance(ctx context.Context, workspaceID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		Delete(&VehicleMaintenanceModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *VehicleRepository) LastMaintenanceByType(ctx context.Context, vehicleID uuid.UUID) (map[string]*dom.VehicleMaintenance, error) {
	var types []string
	if err := r.db.WithContext(ctx).
		Model(&VehicleMaintenanceModel{}).
		Where("vehicle_id = ?", vehicleID.String()).
		Distinct().
		Pluck("type", &types).Error; err != nil {
		return nil, err
	}

	out := make(map[string]*dom.VehicleMaintenance, len(types))
	for _, t := range types {
		var m VehicleMaintenanceModel
		if err := r.db.WithContext(ctx).
			Where("vehicle_id = ? AND type = ? AND service_date IS NOT NULL", vehicleID.String(), t).
			Order("service_date DESC NULLS LAST").
			First(&m).Error; err == nil {
			v := modelToMaintenance(m)
			out[t] = &v
		}
	}
	return out, nil
}

// ─── Maintenance plans ────────────────────────────────────────────────────────

func (r *VehicleRepository) ListTemplates(ctx context.Context) ([]dom.MaintenancePlanTemplate, error) {
	var models []MaintenancePlanTemplateModel
	if err := r.db.WithContext(ctx).Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.MaintenancePlanTemplate, len(models))
	for i, m := range models {
		out[i] = modelToTemplate(m)
	}
	return out, nil
}

func (r *VehicleRepository) GetVehiclePlans(ctx context.Context, workspaceID, vehicleID uuid.UUID) ([]dom.VehicleMaintenancePlan, error) {
	templates, err := r.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}

	var customizations []VehicleMaintenancePlanModel
	if err := r.db.WithContext(ctx).
		Preload("Template").
		Where("workspace_id = ? AND vehicle_id = ?", workspaceID.String(), vehicleID.String()).
		Find(&customizations).Error; err != nil {
		return nil, err
	}

	customByTemplate := make(map[string]VehicleMaintenancePlanModel, len(customizations))
	for _, c := range customizations {
		customByTemplate[c.TemplateID] = c
	}

	out := make([]dom.VehicleMaintenancePlan, 0, len(templates))
	for _, t := range templates {
		if c, ok := customByTemplate[t.ID.String()]; ok {
			out = append(out, modelToMaintenancePlan(c))
		} else {
			tCopy := t
			out = append(out, dom.VehicleMaintenancePlan{
				VehicleID:    vehicleID,
				WorkspaceID:  workspaceID,
				TemplateID:   t.ID,
				IntervalKM:   t.DefaultIntervalKM,
				IntervalDays: t.DefaultIntervalDays,
				Enabled:      true,
				Template:     &tCopy,
			})
		}
	}
	return out, nil
}

func (r *VehicleRepository) UpsertVehiclePlan(ctx context.Context, p *dom.VehicleMaintenancePlan) error {
	var existing VehicleMaintenancePlanModel
	err := r.db.WithContext(ctx).
		Where("vehicle_id = ? AND template_id = ?", p.VehicleID.String(), p.TemplateID.String()).
		First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		newID := uuid.New()
		m := VehicleMaintenancePlanModel{
			ID:           newID.String(),
			VehicleID:    p.VehicleID.String(),
			WorkspaceID:  p.WorkspaceID.String(),
			TemplateID:   p.TemplateID.String(),
			IntervalKM:   p.IntervalKM,
			IntervalDays: p.IntervalDays,
			Enabled:      p.Enabled,
		}
		result := r.db.WithContext(ctx).Create(&m)
		if result.Error != nil {
			return result.Error
		}
		p.ID = newID
		return nil
	}
	if err != nil {
		return err
	}

	p.ID, _ = uuid.Parse(existing.ID)
	return r.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
		"interval_km":   p.IntervalKM,
		"interval_days": p.IntervalDays,
		"enabled":       p.Enabled,
	}).Error
}

// ─── FIPE history ─────────────────────────────────────────────────────────────

func (r *VehicleRepository) AddFipeHistory(ctx context.Context, h *dom.VehicleFipeHistory) error {
	var existing VehicleFipeHistoryModel
	err := r.db.WithContext(ctx).
		Where("vehicle_id = ? AND reference_month = ?", h.VehicleID.String(), h.ReferenceMonth).
		First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		m := VehicleFipeHistoryModel{
			ID:             uuid.New().String(),
			VehicleID:      h.VehicleID.String(),
			WorkspaceID:    h.WorkspaceID.String(),
			ReferenceMonth: h.ReferenceMonth,
			FipeValue:      h.FipeValue,
			FipeFuel:       h.FipeFuel,
		}
		return r.db.WithContext(ctx).Create(&m).Error
	}
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
		"fipe_value": h.FipeValue,
		"fipe_fuel":  h.FipeFuel,
	}).Error
}

func (r *VehicleRepository) ListFipeHistory(ctx context.Context, vehicleID uuid.UUID) ([]dom.VehicleFipeHistory, error) {
	var models []VehicleFipeHistoryModel
	if err := r.db.WithContext(ctx).
		Where("vehicle_id = ?", vehicleID.String()).
		Order("reference_month ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.VehicleFipeHistory, len(models))
	for i, m := range models {
		out[i] = modelToFipeHistory(m)
	}
	return out, nil
}

func (r *VehicleRepository) ListActiveVehiclesForFipeUpdate(ctx context.Context) ([]dom.Vehicle, error) {
	var models []VehicleModel
	if err := r.db.WithContext(ctx).
		Where("status = 'active' AND fipe_brand_code IS NOT NULL AND fipe_model_code IS NOT NULL AND fipe_year_code IS NOT NULL").
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.Vehicle, len(models))
	for i, m := range models {
		out[i] = modelToVehicle(m)
	}
	return out, nil
}
