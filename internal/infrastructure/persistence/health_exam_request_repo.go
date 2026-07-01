package persistence

import (
	"context"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"gorm.io/gorm"
)

type HealthExamRequestRepository struct {
	db *gorm.DB
}

func NewHealthExamRequestRepository(db *gorm.DB) *HealthExamRequestRepository {
	return &HealthExamRequestRepository{db: db}
}

func (r *HealthExamRequestRepository) Create(ctx context.Context, req *dom.ExamRequest) error {
	model := examRequestToModel(req)
	model.Items = examItemsToModels(req.Items)
	// Create com associação Items persiste request + itens numa transação implícita do GORM.
	return mapHealthErr(r.db.WithContext(ctx).Create(&model).Error)
}

func (r *HealthExamRequestRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.ExamRequest, error) {
	var m HealthExamRequestModel
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	return modelToExamRequest(&m), nil
}

func (r *HealthExamRequestRepository) Update(ctx context.Context, req *dom.ExamRequest) error {
	model := examRequestToModel(req)
	res := r.db.WithContext(ctx).Model(&HealthExamRequestModel{}).
		Where("id = ? AND workspace_id = ?", req.ID, req.WorkspaceID).
		Updates(map[string]any{
			"family_member_id": model.FamilyMemberID,
			"lab_id":           model.LabID,
			"requested_by":     model.RequestedBy,
			"request_date":     model.RequestDate,
			"status":           model.Status,
			"notes":            model.Notes,
			"updated_at":       model.UpdatedAt,
		})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *HealthExamRequestRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ? AND workspace_id = ?", id, workspaceID).
			Delete(&HealthExamRequestModel{})
		if res.Error != nil {
			return mapHealthErr(res.Error)
		}
		if res.RowsAffected == 0 {
			return dom.ErrNotFound
		}
		// soft-delete dos itens da solicitação
		if err := tx.Where("exam_request_id = ? AND workspace_id = ?", id, workspaceID).
			Delete(&HealthExamRequestItemModel{}).Error; err != nil {
			return mapHealthErr(err)
		}
		return nil
	})
}

func (r *HealthExamRequestRepository) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]dom.ExamRequest, int64, error) {
	base := r.db.WithContext(ctx).Model(&HealthExamRequestModel{}).Where("workspace_id = ?", workspaceID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	var rows []HealthExamRequestModel
	if err := base.Preload("Items").Order("request_date DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	out := make([]dom.ExamRequest, len(rows))
	for i := range rows {
		out[i] = *modelToExamRequest(&rows[i])
	}
	return out, total, nil
}

func (r *HealthExamRequestRepository) AddItem(ctx context.Context, it *dom.ExamRequestItem) error {
	// garante que a solicitação existe no workspace
	var count int64
	if err := r.db.WithContext(ctx).Model(&HealthExamRequestModel{}).
		Where("id = ? AND workspace_id = ?", it.ExamRequestID, it.WorkspaceID).
		Count(&count).Error; err != nil {
		return mapHealthErr(err)
	}
	if count == 0 {
		return dom.ErrNotFound
	}
	model := examItemToModel(it)
	return mapHealthErr(r.db.WithContext(ctx).Create(&model).Error)
}

func (r *HealthExamRequestRepository) UpdateItem(ctx context.Context, it *dom.ExamRequestItem) error {
	model := examItemToModel(it)
	res := r.db.WithContext(ctx).Model(&HealthExamRequestItemModel{}).
		Where("id = ? AND exam_request_id = ? AND workspace_id = ?", it.ID, it.ExamRequestID, it.WorkspaceID).
		Updates(map[string]any{
			"marker_id":  model.MarkerID,
			"exam_name":  model.ExamName,
			"exam_code":  model.ExamCode,
			"body_area":  model.BodyArea,
			"notes":      model.Notes,
			"status":     model.Status,
			"updated_at": model.UpdatedAt,
		})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *HealthExamRequestRepository) SoftDeleteItem(ctx context.Context, workspaceID, requestID, itemID uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND exam_request_id = ? AND workspace_id = ?", itemID, requestID, workspaceID).
		Delete(&HealthExamRequestItemModel{})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

// --- conversões ---

func examRequestToModel(r *dom.ExamRequest) HealthExamRequestModel {
	return HealthExamRequestModel{
		ID:             r.ID,
		WorkspaceID:    r.WorkspaceID,
		FamilyMemberID: r.FamilyMemberID,
		LabID:          r.LabID,
		RequestedBy:    r.RequestedBy,
		RequestDate:    r.RequestDate,
		Status:         string(r.Status),
		Notes:          r.Notes,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func examItemToModel(it *dom.ExamRequestItem) HealthExamRequestItemModel {
	return HealthExamRequestItemModel{
		ID:            it.ID,
		WorkspaceID:   it.WorkspaceID,
		ExamRequestID: it.ExamRequestID,
		MarkerID:      it.MarkerID,
		ExamName:      it.ExamName,
		ExamCode:      it.ExamCode,
		BodyArea:      it.BodyArea,
		Notes:         it.Notes,
		Status:        string(it.Status),
		CreatedAt:     it.CreatedAt,
		UpdatedAt:     it.UpdatedAt,
	}
}

func examItemsToModels(items []dom.ExamRequestItem) []HealthExamRequestItemModel {
	out := make([]HealthExamRequestItemModel, len(items))
	for i := range items {
		out[i] = examItemToModel(&items[i])
	}
	return out
}

func modelToExamRequest(m *HealthExamRequestModel) *dom.ExamRequest {
	out := &dom.ExamRequest{
		ID:             m.ID,
		WorkspaceID:    m.WorkspaceID,
		FamilyMemberID: m.FamilyMemberID,
		LabID:          m.LabID,
		RequestedBy:    m.RequestedBy,
		RequestDate:    m.RequestDate,
		Status:         dom.ExamRequestStatus(m.Status),
		Notes:          m.Notes,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
	for i := range m.Items {
		out.Items = append(out.Items, *modelToExamItem(&m.Items[i]))
	}
	return out
}

func modelToExamItem(m *HealthExamRequestItemModel) *dom.ExamRequestItem {
	return &dom.ExamRequestItem{
		ID:            m.ID,
		WorkspaceID:   m.WorkspaceID,
		ExamRequestID: m.ExamRequestID,
		MarkerID:      m.MarkerID,
		ExamName:      m.ExamName,
		ExamCode:      m.ExamCode,
		BodyArea:      m.BodyArea,
		Notes:         m.Notes,
		Status:        dom.ExamRequestItemStatus(m.Status),
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}
