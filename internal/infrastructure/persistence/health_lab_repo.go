package persistence

import (
	"context"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"gorm.io/gorm"
)

type HealthLabRepository struct {
	db *gorm.DB
}

func NewHealthLabRepository(db *gorm.DB) *HealthLabRepository {
	return &HealthLabRepository{db: db}
}

func (r *HealthLabRepository) Create(ctx context.Context, l *dom.Lab) error {
	m := labToModel(l)
	return mapHealthErr(r.db.WithContext(ctx).Create(&m).Error)
}

func (r *HealthLabRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Lab, error) {
	var m HealthLabModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	return modelToLab(&m), nil
}

func (r *HealthLabRepository) Update(ctx context.Context, l *dom.Lab) error {
	m := labToModel(l)
	res := r.db.WithContext(ctx).Model(&HealthLabModel{}).
		Where("id = ? AND workspace_id = ?", l.ID, l.WorkspaceID).
		Updates(map[string]any{
			"name":             m.Name,
			"kind":             m.Kind,
			"website_url":      m.WebsiteURL,
			"exam_results_url": m.ExamResultsURL,
			"contact_phone":    m.ContactPhone,
			"address":          m.Address,
			"notes":            m.Notes,
			"active":           m.Active,
			"updated_at":       m.UpdatedAt,
		})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *HealthLabRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&HealthLabModel{})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *HealthLabRepository) List(ctx context.Context, workspaceID uuid.UUID, filter dom.LabFilter, limit, offset int) ([]dom.Lab, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&HealthLabModel{}).Where("workspace_id = ?", workspaceID)
	if filter.Query != "" {
		q = q.Where("name ILIKE ?", "%"+filter.Query+"%")
	}
	if filter.Active != nil {
		q = q.Where("active = ?", *filter.Active)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	var rows []HealthLabModel
	if err := q.Order("name ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	out := make([]dom.Lab, len(rows))
	for i := range rows {
		out[i] = *modelToLab(&rows[i])
	}
	return out, total, nil
}

// --- conversões ---

func labToModel(l *dom.Lab) HealthLabModel {
	return HealthLabModel{
		ID:             l.ID,
		WorkspaceID:    l.WorkspaceID,
		Name:           l.Name,
		Kind:           string(l.Kind),
		WebsiteURL:     l.WebsiteURL,
		ExamResultsURL: l.ExamResultsURL,
		ContactPhone:   l.ContactPhone,
		Address:        l.Address,
		Notes:          l.Notes,
		Active:         l.Active,
		CreatedAt:      l.CreatedAt,
		UpdatedAt:      l.UpdatedAt,
	}
}

func modelToLab(m *HealthLabModel) *dom.Lab {
	return &dom.Lab{
		ID:             m.ID,
		WorkspaceID:    m.WorkspaceID,
		Name:           m.Name,
		Kind:           dom.LabKind(m.Kind),
		WebsiteURL:     m.WebsiteURL,
		ExamResultsURL: m.ExamResultsURL,
		ContactPhone:   m.ContactPhone,
		Address:        m.Address,
		Notes:          m.Notes,
		Active:         m.Active,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}
