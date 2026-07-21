package persistence

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// FinanceExtractionJobRepository persiste jobs de extração de faturas, escopo do tenant.
type FinanceExtractionJobRepository struct {
	db *gorm.DB
}

func NewFinanceExtractionJobRepository(db *gorm.DB) *FinanceExtractionJobRepository {
	return &FinanceExtractionJobRepository{db: db}
}

func (r *FinanceExtractionJobRepository) Create(ctx context.Context, j *dom.FinanceExtractionJob) error {
	model := financeExtractionJobToModel(j)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return mapFinanceErr(err)
	}
	*j = *modelToFinanceExtractionJob(&model)
	return nil
}

func (r *FinanceExtractionJobRepository) Update(ctx context.Context, j *dom.FinanceExtractionJob) error {
	model := financeExtractionJobToModel(j)
	res := r.db.WithContext(ctx).Model(&FinanceExtractionJobModel{}).
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
		return mapFinanceErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *FinanceExtractionJobRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.FinanceExtractionJob, error) {
	var m FinanceExtractionJobModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	return modelToFinanceExtractionJob(&m), nil
}

func (r *FinanceExtractionJobRepository) GetByDocument(ctx context.Context, workspaceID, documentID uuid.UUID) (*dom.FinanceExtractionJob, error) {
	var m FinanceExtractionJobModel
	err := r.db.WithContext(ctx).
		Where("document_id = ? AND workspace_id = ?", documentID, workspaceID).
		Order("created_at DESC").
		First(&m).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	return modelToFinanceExtractionJob(&m), nil
}

func (r *FinanceExtractionJobRepository) ListStale(
	ctx context.Context,
	statuses []dom.ExtractionJobStatus,
	updatedBefore time.Time,
	limit int,
) ([]dom.FinanceExtractionJob, error) {
	if limit <= 0 {
		limit = 100
	}
	ss := make([]string, len(statuses))
	for i, s := range statuses {
		ss[i] = string(s)
	}
	var models []FinanceExtractionJobModel
	err := r.db.WithContext(ctx).
		Where("status IN ? AND updated_at < ?", ss, updatedBefore).
		Order("updated_at ASC").
		Limit(limit).
		Find(&models).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	out := make([]dom.FinanceExtractionJob, len(models))
	for i := range models {
		out[i] = *modelToFinanceExtractionJob(&models[i])
	}
	return out, nil
}

// --- conversões ---

func financeExtractionJobToModel(j *dom.FinanceExtractionJob) FinanceExtractionJobModel {
	m := FinanceExtractionJobModel{
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

func modelToFinanceExtractionJob(m *FinanceExtractionJobModel) *dom.FinanceExtractionJob {
	j := &dom.FinanceExtractionJob{
		ID:            m.ID,
		WorkspaceID:   m.WorkspaceID,
		DocumentID:    m.DocumentID,
		Provider:      m.Provider,
		Model:         m.Model,
		Status:        dom.ExtractionJobStatus(m.Status),
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
