package persistence

import (
	"context"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type HealthDocumentRepository struct {
	db *gorm.DB
}

func NewHealthDocumentRepository(db *gorm.DB) *HealthDocumentRepository {
	return &HealthDocumentRepository{db: db}
}

func (r *HealthDocumentRepository) Create(ctx context.Context, d *dom.Document) error {
	model := documentToModel(d)
	return mapHealthErr(r.db.WithContext(ctx).Create(&model).Error)
}

func (r *HealthDocumentRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Document, error) {
	var m HealthDocumentModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	return modelToDocument(&m), nil
}

func (r *HealthDocumentRepository) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]dom.Document, int64, error) {
	base := r.db.WithContext(ctx).Model(&HealthDocumentModel{}).
		Where("workspace_id = ?", workspaceID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	var rows []HealthDocumentModel
	if err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	out := make([]dom.Document, len(rows))
	for i := range rows {
		out[i] = *modelToDocument(&rows[i])
	}
	return out, total, nil
}

func (r *HealthDocumentRepository) UpdateExtraction(ctx context.Context, d *dom.Document) error {
	res := r.db.WithContext(ctx).Model(&HealthDocumentModel{}).
		Where("id = ? AND workspace_id = ?", d.ID, d.WorkspaceID).
		Updates(map[string]any{
			"extraction_status": string(d.ExtractionStatus),
			"extracted_text":    d.ExtractedText,
			"extracted_json":    rawToJSON(d.ExtractedJSON),
			"updated_at":        d.UpdatedAt,
		})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *HealthDocumentRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&HealthDocumentModel{})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

// --- conversões ---

func rawToJSON(b []byte) datatypes.JSON {
	if len(b) == 0 {
		return nil
	}
	return datatypes.JSON(b)
}

func documentToModel(d *dom.Document) HealthDocumentModel {
	return HealthDocumentModel{
		ID:               d.ID,
		WorkspaceID:      d.WorkspaceID,
		FamilyMemberID:   d.FamilyMemberID,
		LabID:            d.LabID,
		ExamRequestID:    d.ExamRequestID,
		ExamResultID:     d.ExamResultID,
		DocumentType:     string(d.DocumentType),
		FileName:         d.FileName,
		OriginalFileName: d.OriginalFileName,
		MimeType:         d.MimeType,
		SizeBytes:        d.SizeBytes,
		StorageProvider:  d.StorageProvider,
		Bucket:           d.Bucket,
		ObjectKey:        d.ObjectKey,
		Checksum:         d.Checksum,
		UploadedByUserID: d.UploadedByUserID,
		ExtractionStatus: string(d.ExtractionStatus),
		ExtractedText:    d.ExtractedText,
		ExtractedJSON:    rawToJSON(d.ExtractedJSON),
		Metadata:         rawToJSON(d.Metadata),
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}
}

func modelToDocument(m *HealthDocumentModel) *dom.Document {
	var extracted, metadata []byte
	if len(m.ExtractedJSON) > 0 {
		extracted = []byte(m.ExtractedJSON)
	}
	if len(m.Metadata) > 0 {
		metadata = []byte(m.Metadata)
	}
	return &dom.Document{
		ID:               m.ID,
		WorkspaceID:      m.WorkspaceID,
		FamilyMemberID:   m.FamilyMemberID,
		LabID:            m.LabID,
		ExamRequestID:    m.ExamRequestID,
		ExamResultID:     m.ExamResultID,
		DocumentType:     dom.DocumentType(m.DocumentType),
		FileName:         m.FileName,
		OriginalFileName: m.OriginalFileName,
		MimeType:         m.MimeType,
		SizeBytes:        m.SizeBytes,
		StorageProvider:  m.StorageProvider,
		Bucket:           m.Bucket,
		ObjectKey:        m.ObjectKey,
		Checksum:         m.Checksum,
		UploadedByUserID: m.UploadedByUserID,
		ExtractionStatus: dom.ExtractionStatus(m.ExtractionStatus),
		ExtractedText:    m.ExtractedText,
		ExtractedJSON:    extracted,
		Metadata:         metadata,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}
