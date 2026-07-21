package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/entitlement"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EntitlementRepository implementa dom.Repository com GORM/Postgres.
type EntitlementRepository struct {
	db *gorm.DB
}

func NewEntitlementRepository(db *gorm.DB) *EntitlementRepository {
	return &EntitlementRepository{db: db}
}

func (r *EntitlementRepository) Get(ctx context.Context, workspaceID uuid.UUID) (*dom.Entitlement, error) {
	var m EntitlementModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ?", workspaceID).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dom.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("entitlement: get: %w", err)
	}
	e := modelToEntitlement(&m)
	return &e, nil
}

func (r *EntitlementRepository) Upsert(ctx context.Context, e *dom.Entitlement) error {
	m := entitlementToModel(e)
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "workspace_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"tier", "fiscal_sefaz_quota", "updated_at"}),
		}).
		Create(&m).Error
	if err != nil {
		return fmt.Errorf("entitlement: upsert: %w", err)
	}
	return nil
}
