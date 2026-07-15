package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/patrimony"
	"gorm.io/gorm"
)

// PatrimonyRepository implementa dom.Repository (imóveis + impostos) com GORM/Postgres.
type PatrimonyRepository struct {
	db *gorm.DB
}

func NewPatrimonyRepository(db *gorm.DB) *PatrimonyRepository {
	return &PatrimonyRepository{db: db}
}

// ─── Properties ────────────────────────────────────────────────────────────────

func (r *PatrimonyRepository) CreateProperty(ctx context.Context, p *dom.Property) error {
	m := propertyToModel(p)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("property: create: %w", err)
	}
	return nil
}

func (r *PatrimonyRepository) GetProperty(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Property, error) {
	var m PropertyModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p := modelToProperty(m)
	return &p, nil
}

func (r *PatrimonyRepository) ListProperties(ctx context.Context, workspaceID uuid.UUID, params dom.ListPropertiesParams) ([]dom.Property, int64, error) {
	q := r.db.WithContext(ctx).Model(&PropertyModel{}).
		Where("workspace_id = ?", workspaceID.String())
	if params.OnlyActive {
		q = q.Where("active = ?", true)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	var models []PropertyModel
	if err := q.Limit(limit).Offset(params.Offset).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]dom.Property, len(models))
	for i, m := range models {
		out[i] = modelToProperty(m)
	}
	return out, total, nil
}

func (r *PatrimonyRepository) UpdateProperty(ctx context.Context, p *dom.Property) error {
	m := propertyToModel(p)
	if err := r.db.WithContext(ctx).Omit("CreatedAt").Save(&m).Error; err != nil {
		return fmt.Errorf("property: update: %w", err)
	}
	return nil
}

func (r *PatrimonyRepository) DeleteProperty(ctx context.Context, workspaceID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		Delete(&PropertyModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

// ─── Taxes ─────────────────────────────────────────────────────────────────────

func (r *PatrimonyRepository) CreateTax(ctx context.Context, t *dom.AssetTax) error {
	m := taxToModel(t)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("tax: create: %w", err)
	}
	return nil
}

func (r *PatrimonyRepository) GetTax(ctx context.Context, workspaceID, id uuid.UUID) (*dom.AssetTax, error) {
	var m AssetTaxModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	t := modelToTax(m)
	return &t, nil
}

func (r *PatrimonyRepository) ListTaxes(ctx context.Context, workspaceID uuid.UUID, params dom.ListTaxesParams) ([]dom.AssetTax, int64, error) {
	q := r.db.WithContext(ctx).Model(&AssetTaxModel{}).
		Where("workspace_id = ?", workspaceID.String())
	if params.AssetType != "" {
		q = q.Where("asset_type = ?", params.AssetType)
	}
	if params.TaxType != "" {
		q = q.Where("tax_type = ?", params.TaxType)
	}
	if params.ReferenceYear > 0 {
		q = q.Where("reference_year = ?", params.ReferenceYear)
	}
	if params.Status != "" {
		q = q.Where("status = ?", params.Status)
	}
	if params.PropertyID != nil {
		q = q.Where("property_id = ?", params.PropertyID.String())
	}
	if params.VehicleID != nil {
		q = q.Where("vehicle_id = ?", params.VehicleID.String())
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}
	var models []AssetTaxModel
	if err := q.Limit(limit).Offset(params.Offset).
		Order("reference_year DESC, due_date ASC NULLS LAST").
		Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]dom.AssetTax, len(models))
	for i, m := range models {
		out[i] = modelToTax(m)
	}
	return out, total, nil
}

func (r *PatrimonyRepository) UpdateTax(ctx context.Context, t *dom.AssetTax) error {
	m := taxToModel(t)
	if err := r.db.WithContext(ctx).Omit("CreatedAt").Save(&m).Error; err != nil {
		return fmt.Errorf("tax: update: %w", err)
	}
	return nil
}

func (r *PatrimonyRepository) DeleteTax(ctx context.Context, workspaceID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		Delete(&AssetTaxModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *PatrimonyRepository) ListAllTaxes(ctx context.Context, workspaceID uuid.UUID) ([]dom.AssetTax, error) {
	var models []AssetTaxModel
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ?", workspaceID.String()).
		Order("reference_year ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.AssetTax, len(models))
	for i, m := range models {
		out[i] = modelToTax(m)
	}
	return out, nil
}
