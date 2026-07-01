package persistence

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// HealthExtractionJobRepository persiste jobs de extração, escopo do tenant.
type HealthExtractionJobRepository struct {
	db *gorm.DB
}

func NewHealthExtractionJobRepository(db *gorm.DB) *HealthExtractionJobRepository {
	return &HealthExtractionJobRepository{db: db}
}

func (r *HealthExtractionJobRepository) Create(ctx context.Context, j *dom.ExtractionJob) error {
	model := extractionJobToModel(j)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return mapHealthErr(err)
	}
	*j = *modelToExtractionJob(&model)
	return nil
}

func (r *HealthExtractionJobRepository) Update(ctx context.Context, j *dom.ExtractionJob) error {
	model := extractionJobToModel(j)
	res := r.db.WithContext(ctx).Model(&HealthExtractionJobModel{}).
		Where("id = ? AND workspace_id = ?", j.ID, j.WorkspaceID).
		Updates(map[string]any{
			"provider":       model.Provider,
			"model":          model.Model,
			"status":         model.Status,
			"input_type":     model.InputType,
			"prompt_version": model.PromptVersion,
			"raw_response":   model.RawResponse,
			"error_message":  model.ErrorMessage,
			"started_at":     model.StartedAt,
			"finished_at":    model.FinishedAt,
			"updated_at":     model.UpdatedAt,
		})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *HealthExtractionJobRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.ExtractionJob, error) {
	var m HealthExtractionJobModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	return modelToExtractionJob(&m), nil
}

func (r *HealthExtractionJobRepository) GetByDocument(ctx context.Context, workspaceID, documentID uuid.UUID) (*dom.ExtractionJob, error) {
	var m HealthExtractionJobModel
	err := r.db.WithContext(ctx).
		Where("document_id = ? AND workspace_id = ?", documentID, workspaceID).
		Order("created_at DESC").
		First(&m).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	return modelToExtractionJob(&m), nil
}

// --- conversões ---

func extractionJobToModel(j *dom.ExtractionJob) HealthExtractionJobModel {
	m := HealthExtractionJobModel{
		ID:            j.ID,
		WorkspaceID:   j.WorkspaceID,
		DocumentID:    j.DocumentID,
		Provider:      j.Provider,
		Model:         j.Model,
		Status:        string(j.Status),
		InputType:     string(j.InputType),
		PromptVersion: j.PromptVersion,
		ErrorMessage:  j.ErrorMessage,
		StartedAt:     j.StartedAt,
		FinishedAt:    j.FinishedAt,
		CreatedAt:     j.CreatedAt,
		UpdatedAt:     j.UpdatedAt,
	}
	if len(j.RawResponse) > 0 {
		m.RawResponse = datatypes.JSON(j.RawResponse)
	}
	return m
}

func modelToExtractionJob(m *HealthExtractionJobModel) *dom.ExtractionJob {
	j := &dom.ExtractionJob{
		ID:            m.ID,
		WorkspaceID:   m.WorkspaceID,
		DocumentID:    m.DocumentID,
		Provider:      m.Provider,
		Model:         m.Model,
		Status:        dom.ExtractionStatus(m.Status),
		InputType:     dom.ExtractionInputType(m.InputType),
		PromptVersion: m.PromptVersion,
		ErrorMessage:  m.ErrorMessage,
		StartedAt:     m.StartedAt,
		FinishedAt:    m.FinishedAt,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
	if len(m.RawResponse) > 0 {
		j.RawResponse = json.RawMessage(m.RawResponse)
	}
	return j
}
