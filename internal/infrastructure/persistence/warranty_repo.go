package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/warranty"
	"gorm.io/gorm"
)

// WarrantyRepository implementa dom.Repository com GORM/Postgres.
type WarrantyRepository struct {
	db *gorm.DB
}

func NewWarrantyRepository(db *gorm.DB) *WarrantyRepository {
	return &WarrantyRepository{db: db}
}

// ─── Warranties ───────────────────────────────────────────────────────────────

func (r *WarrantyRepository) Create(ctx context.Context, w *dom.Warranty) error {
	m := warrantyToModel(w)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("warranty: create: %w", err)
	}
	return nil
}

func (r *WarrantyRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Warranty, error) {
	var m WarrantyModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	w := modelToWarranty(&m)
	return &w, nil
}

func (r *WarrantyRepository) List(ctx context.Context, workspaceID uuid.UUID, p dom.ListParams) ([]dom.Warranty, int64, error) {
	q := r.db.WithContext(ctx).Model(&WarrantyModel{}).
		Where("workspace_id = ?", workspaceID)
	if p.Category != "" {
		q = q.Where("category = ?", p.Category)
	}
	if s := strings.TrimSpace(p.Query); s != "" {
		like := "%" + strings.ToLower(s) + "%"
		q = q.Where("LOWER(item_name) LIKE ? OR LOWER(COALESCE(brand, '')) LIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	var models []WarrantyModel
	if err := q.Limit(limit).Offset(p.Offset).
		Order("purchase_date DESC").
		Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]dom.Warranty, len(models))
	for i := range models {
		out[i] = modelToWarranty(&models[i])
	}
	return out, total, nil
}

func (r *WarrantyRepository) ListActive(ctx context.Context, workspaceID uuid.UUID) ([]dom.Warranty, error) {
	var models []WarrantyModel
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND active = true", workspaceID).
		Order("purchase_date DESC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.Warranty, len(models))
	for i := range models {
		out[i] = modelToWarranty(&models[i])
	}
	return out, nil
}

func (r *WarrantyRepository) Update(ctx context.Context, w *dom.Warranty) error {
	m := warrantyToModel(w)
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", w.ID, w.WorkspaceID).
		Omit("id", "workspace_id", "created_at").
		Updates(&m)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *WarrantyRepository) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&WarrantyModel{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

// ─── Documents ────────────────────────────────────────────────────────────────

func (r *WarrantyRepository) CreateDocument(ctx context.Context, d *dom.Document) error {
	m := warrantyDocumentToModel(d)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("warranty: create document: %w", err)
	}
	return nil
}

func (r *WarrantyRepository) GetDocumentByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Document, error) {
	var m WarrantyDocumentModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	d := modelToWarrantyDocument(&m)
	return &d, nil
}

func (r *WarrantyRepository) ListDocuments(ctx context.Context, workspaceID, warrantyID uuid.UUID) ([]dom.Document, error) {
	var models []WarrantyDocumentModel
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND warranty_id = ?", workspaceID, warrantyID).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.Document, len(models))
	for i := range models {
		out[i] = modelToWarrantyDocument(&models[i])
	}
	return out, nil
}

func (r *WarrantyRepository) DeleteDocument(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&WarrantyDocumentModel{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}
