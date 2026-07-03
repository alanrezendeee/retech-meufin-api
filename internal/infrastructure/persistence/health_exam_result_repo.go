package persistence

import (
	"context"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"gorm.io/gorm"
)

type HealthExamResultRepository struct {
	db *gorm.DB
}

func NewHealthExamResultRepository(db *gorm.DB) *HealthExamResultRepository {
	return &HealthExamResultRepository{db: db}
}

func (r *HealthExamResultRepository) Create(ctx context.Context, res *dom.ExamResult) error {
	m := examResultToModel(res)
	m.Items = examResultItemsToModels(res.Items)
	if err := mapHealthErr(r.db.WithContext(ctx).Create(&m).Error); err != nil {
		return err
	}
	// devolve IDs/timestamps gerados para os itens
	res.Items = modelsToExamResultItems(m.Items)
	return nil
}

func (r *HealthExamResultRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.ExamResult, error) {
	var m HealthExamResultModel
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	return modelToExamResult(&m), nil
}

func (r *HealthExamResultRepository) Update(ctx context.Context, res *dom.ExamResult) error {
	m := examResultToModel(res)
	upd := r.db.WithContext(ctx).Model(&HealthExamResultModel{}).
		Where("id = ? AND workspace_id = ?", res.ID, res.WorkspaceID).
		Updates(map[string]any{
			"family_member_id": m.FamilyMemberID,
			"lab_id":           m.LabID,
			"exam_request_id":  m.ExamRequestID,
			"exam_date":        m.ExamDate,
			"collection_date":  m.CollectionDate,
			"release_date":     m.ReleaseDate,
			"source_type":      m.SourceType,
			"status":           m.Status,
			"summary":          m.Summary,
			"notes":            m.Notes,
			"updated_at":       m.UpdatedAt,
		})
	if upd.Error != nil {
		return mapHealthErr(upd.Error)
	}
	if upd.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *HealthExamResultRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ? AND workspace_id = ?", id, workspaceID).
			Delete(&HealthExamResultModel{})
		if res.Error != nil {
			return mapHealthErr(res.Error)
		}
		if res.RowsAffected == 0 {
			return dom.ErrNotFound
		}
		if err := tx.Where("exam_result_id = ?", id).Delete(&HealthExamResultItemModel{}).Error; err != nil {
			return mapHealthErr(err)
		}
		return nil
	})
}

func (r *HealthExamResultRepository) List(ctx context.Context, workspaceID uuid.UUID, filter dom.ExamResultFilter, limit, offset int) ([]dom.ExamResult, int64, error) {
	q := r.db.WithContext(ctx).Model(&HealthExamResultModel{}).Where("workspace_id = ?", workspaceID)
	if filter.Query != "" {
		q = q.Where("summary ILIKE ?", "%"+filter.Query+"%")
	}
	if filter.FamilyMemberID != nil {
		q = q.Where("family_member_id = ?", *filter.FamilyMemberID)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	var rows []HealthExamResultModel
	if err := q.Preload("Items").Order("exam_date DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	out := make([]dom.ExamResult, len(rows))
	for i := range rows {
		out[i] = *modelToExamResult(&rows[i])
	}
	return out, total, nil
}

func (r *HealthExamResultRepository) AddItem(ctx context.Context, workspaceID uuid.UUID, item *dom.ExamResultItem) error {
	// garante que o resultado pertence ao workspace
	var parent HealthExamResultModel
	if err := r.db.WithContext(ctx).
		Select("id").
		Where("id = ? AND workspace_id = ?", item.ExamResultID, workspaceID).
		First(&parent).Error; err != nil {
		return mapHealthErr(err)
	}
	m := examResultItemToModel(item)
	if err := mapHealthErr(r.db.WithContext(ctx).Create(&m).Error); err != nil {
		return err
	}
	*item = *modelToExamResultItem(&m)
	return nil
}

func (r *HealthExamResultRepository) UpdateItem(ctx context.Context, item *dom.ExamResultItem) error {
	m := examResultItemToModel(item)
	upd := r.db.WithContext(ctx).Model(&HealthExamResultItemModel{}).
		Where("id = ? AND workspace_id = ? AND exam_result_id = ?", item.ID, item.WorkspaceID, item.ExamResultID).
		Updates(map[string]any{
			"marker_id":               m.MarkerID,
			"raw_marker_name":         m.RawMarkerName,
			"result_value":            m.ResultValue,
			"result_numeric":          m.ResultNumeric,
			"unit":                    m.Unit,
			"reference_min":           m.ReferenceMin,
			"reference_max":           m.ReferenceMax,
			"reference_text":          m.ReferenceText,
			"interpretation":          m.Interpretation,
			"interpretation_computed": m.InterpretationComputed,
			"method":                  m.Method,
			"material":                m.Material,
			"raw_text":                m.RawText,
			"updated_at":              m.UpdatedAt,
		})
	if upd.Error != nil {
		return mapHealthErr(upd.Error)
	}
	if upd.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *HealthExamResultRepository) SoftDeleteItem(ctx context.Context, workspaceID, resultID, itemID uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ? AND exam_result_id = ?", itemID, workspaceID, resultID).
		Delete(&HealthExamResultItemModel{})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

// --- conversões ---

func examResultToModel(r *dom.ExamResult) HealthExamResultModel {
	return HealthExamResultModel{
		ID:             r.ID,
		WorkspaceID:    r.WorkspaceID,
		FamilyMemberID: r.FamilyMemberID,
		LabID:          r.LabID,
		ExamRequestID:  r.ExamRequestID,
		ExamDate:       r.ExamDate,
		CollectionDate: r.CollectionDate,
		ReleaseDate:    r.ReleaseDate,
		SourceType:     string(r.SourceType),
		Status:         string(r.Status),
		Summary:        r.Summary,
		Notes:          r.Notes,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func examResultItemToModel(it *dom.ExamResultItem) HealthExamResultItemModel {
	return HealthExamResultItemModel{
		ID:                     it.ID,
		WorkspaceID:            it.WorkspaceID,
		ExamResultID:           it.ExamResultID,
		MarkerID:               it.MarkerID,
		RawMarkerName:          it.RawMarkerName,
		ResultValue:            it.ResultValue,
		ResultNumeric:          it.ResultNumeric,
		Unit:                   it.Unit,
		ReferenceMin:           it.ReferenceMin,
		ReferenceMax:           it.ReferenceMax,
		ReferenceText:          it.ReferenceText,
		Interpretation:         it.Interpretation,
		InterpretationComputed: it.InterpretationComputed,
		Method:                 it.Method,
		Material:               it.Material,
		RawText:                it.RawText,
		CreatedAt:              it.CreatedAt,
		UpdatedAt:              it.UpdatedAt,
	}
}

func examResultItemsToModels(items []dom.ExamResultItem) []HealthExamResultItemModel {
	out := make([]HealthExamResultItemModel, len(items))
	for i := range items {
		out[i] = examResultItemToModel(&items[i])
	}
	return out
}

func modelToExamResult(m *HealthExamResultModel) *dom.ExamResult {
	return &dom.ExamResult{
		ID:             m.ID,
		WorkspaceID:    m.WorkspaceID,
		FamilyMemberID: m.FamilyMemberID,
		LabID:          m.LabID,
		ExamRequestID:  m.ExamRequestID,
		ExamDate:       m.ExamDate,
		CollectionDate: m.CollectionDate,
		ReleaseDate:    m.ReleaseDate,
		SourceType:     dom.SourceType(m.SourceType),
		Status:         dom.ExamResultStatus(m.Status),
		Summary:        m.Summary,
		Notes:          m.Notes,
		Items:          modelsToExamResultItems(m.Items),
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func modelToExamResultItem(m *HealthExamResultItemModel) *dom.ExamResultItem {
	return &dom.ExamResultItem{
		ID:                     m.ID,
		WorkspaceID:            m.WorkspaceID,
		ExamResultID:           m.ExamResultID,
		MarkerID:               m.MarkerID,
		RawMarkerName:          m.RawMarkerName,
		ResultValue:            m.ResultValue,
		ResultNumeric:          m.ResultNumeric,
		Unit:                   m.Unit,
		ReferenceMin:           m.ReferenceMin,
		ReferenceMax:           m.ReferenceMax,
		ReferenceText:          m.ReferenceText,
		Interpretation:         m.Interpretation,
		InterpretationComputed: m.InterpretationComputed,
		Method:                 m.Method,
		Material:               m.Material,
		RawText:                m.RawText,
		CreatedAt:              m.CreatedAt,
		UpdatedAt:              m.UpdatedAt,
	}
}

func modelsToExamResultItems(models []HealthExamResultItemModel) []dom.ExamResultItem {
	if len(models) == 0 {
		return nil
	}
	out := make([]dom.ExamResultItem, len(models))
	for i := range models {
		out[i] = *modelToExamResultItem(&models[i])
	}
	return out
}
