package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"gorm.io/gorm"
)

// HealthPlanDocumentModel mapeia a tabela health_plan_documents.
type HealthPlanDocumentModel struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID      uuid.UUID  `gorm:"type:uuid;not null;index:idx_health_plan_documents_workspace"`
	PlanID           uuid.UUID  `gorm:"type:uuid;not null;index:idx_health_plan_documents_plan"`
	DocType          string     `gorm:"column:doc_type;size:40;not null"`
	Label            *string    `gorm:"size:255"`
	DocNumber        *string    `gorm:"column:doc_number;size:100"`
	ValidUntil       *time.Time `gorm:"column:valid_until;type:date"`
	Notes            *string    `gorm:"type:text"`
	FileName         string     `gorm:"size:255;not null"`
	OriginalFileName string     `gorm:"size:255;not null"`
	MimeType         string     `gorm:"size:100;not null"`
	SizeBytes        int64      `gorm:"not null;default:0"`
	StorageProvider  string     `gorm:"size:20;not null;default:minio"`
	Bucket           string     `gorm:"size:255;not null"`
	ObjectKey        string     `gorm:"size:500;not null"`
	UploadedByUserID uuid.UUID  `gorm:"type:uuid;not null"`
	CreatedAt        time.Time  `gorm:"not null"`
	UpdatedAt        time.Time  `gorm:"not null"`
	DeletedAt        gorm.DeletedAt
}

func (HealthPlanDocumentModel) TableName() string { return "health_plan_documents" }

type HealthPlanDocumentRepository struct {
	db *gorm.DB
}

func NewHealthPlanDocumentRepository(db *gorm.DB) *HealthPlanDocumentRepository {
	return &HealthPlanDocumentRepository{db: db}
}

func (r *HealthPlanDocumentRepository) Create(ctx context.Context, d *dom.PlanDocument) error {
	model := planDocumentToModel(d)
	return mapHealthErr(r.db.WithContext(ctx).Create(&model).Error)
}

func (r *HealthPlanDocumentRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.PlanDocument, error) {
	var m HealthPlanDocumentModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	return modelToPlanDocument(&m), nil
}

func (r *HealthPlanDocumentRepository) ListByPlan(ctx context.Context, workspaceID, planID uuid.UUID, limit, offset int) ([]dom.PlanDocument, int64, error) {
	base := r.db.WithContext(ctx).Model(&HealthPlanDocumentModel{}).
		Where("workspace_id = ? AND plan_id = ?", workspaceID, planID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	var rows []HealthPlanDocumentModel
	if err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	out := make([]dom.PlanDocument, len(rows))
	for i := range rows {
		out[i] = *modelToPlanDocument(&rows[i])
	}
	return out, total, nil
}

func (r *HealthPlanDocumentRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&HealthPlanDocumentModel{})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

// --- conversões ---

func planDocumentToModel(d *dom.PlanDocument) HealthPlanDocumentModel {
	return HealthPlanDocumentModel{
		ID:               d.ID,
		WorkspaceID:      d.WorkspaceID,
		PlanID:           d.PlanID,
		DocType:          string(d.DocType),
		Label:            d.Label,
		DocNumber:        d.DocNumber,
		ValidUntil:       d.ValidUntil,
		Notes:            d.Notes,
		FileName:         d.FileName,
		OriginalFileName: d.OriginalFileName,
		MimeType:         d.MimeType,
		SizeBytes:        d.SizeBytes,
		StorageProvider:  d.StorageProvider,
		Bucket:           d.Bucket,
		ObjectKey:        d.ObjectKey,
		UploadedByUserID: d.UploadedByUserID,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}
}

func modelToPlanDocument(m *HealthPlanDocumentModel) *dom.PlanDocument {
	return &dom.PlanDocument{
		ID:               m.ID,
		WorkspaceID:      m.WorkspaceID,
		PlanID:           m.PlanID,
		DocType:          dom.PlanDocType(m.DocType),
		Label:            m.Label,
		DocNumber:        m.DocNumber,
		ValidUntil:       m.ValidUntil,
		Notes:            m.Notes,
		FileName:         m.FileName,
		OriginalFileName: m.OriginalFileName,
		MimeType:         m.MimeType,
		SizeBytes:        m.SizeBytes,
		StorageProvider:  m.StorageProvider,
		Bucket:           m.Bucket,
		ObjectKey:        m.ObjectKey,
		UploadedByUserID: m.UploadedByUserID,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}
