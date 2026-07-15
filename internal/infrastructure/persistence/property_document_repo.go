package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/patrimony"
	"gorm.io/gorm"
)

// PropertyDocumentRepository implementa dom.DocumentRepository com GORM/Postgres.
type PropertyDocumentRepository struct {
	db *gorm.DB
}

func NewPropertyDocumentRepository(db *gorm.DB) *PropertyDocumentRepository {
	return &PropertyDocumentRepository{db: db}
}

func (r *PropertyDocumentRepository) Create(ctx context.Context, d *dom.PropertyDocument) error {
	m := propertyDocToModel(d)
	return r.db.WithContext(ctx).Create(&m).Error
}

func (r *PropertyDocumentRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.PropertyDocument, error) {
	var m PropertyDocumentModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	doc := modelToPropertyDoc(m)
	return &doc, nil
}

func (r *PropertyDocumentRepository) ListByProperty(ctx context.Context, workspaceID, propertyID uuid.UUID) ([]dom.PropertyDocument, error) {
	var models []PropertyDocumentModel
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND property_id = ?", workspaceID.String(), propertyID.String()).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.PropertyDocument, len(models))
	for i, m := range models {
		out[i] = modelToPropertyDoc(m)
	}
	return out, nil
}

func (r *PropertyDocumentRepository) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		Delete(&PropertyDocumentModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}
