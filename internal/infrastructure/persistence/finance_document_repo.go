package persistence

import (
	"context"
	"strings"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"gorm.io/gorm"
)

// FinanceDocumentRepository persiste documentos financeiros (workspace-scoped, soft-delete).
type FinanceDocumentRepository struct {
	db *gorm.DB
}

func NewFinanceDocumentRepository(db *gorm.DB) *FinanceDocumentRepository {
	return &FinanceDocumentRepository{db: db}
}

func (r *FinanceDocumentRepository) Create(ctx context.Context, d *dom.FinanceDocument) error {
	model := financeDocumentToModel(d)
	return mapFinanceErr(r.db.WithContext(ctx).Create(&model).Error)
}

func (r *FinanceDocumentRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.FinanceDocument, error) {
	var m FinanceDocumentModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	return modelToFinanceDocument(&m), nil
}

func (r *FinanceDocumentRepository) List(ctx context.Context, workspaceID uuid.UUID, filter dom.FinanceDocumentFilter, limit, offset int) ([]dom.FinanceDocument, int64, error) {
	base := r.db.WithContext(ctx).Model(&FinanceDocumentModel{}).
		Where("workspace_id = ?", workspaceID)
	if filter.Kind != nil {
		base = base.Where("kind = ?", string(*filter.Kind))
	}
	if filter.EntryID != nil {
		base = base.Where("entry_id = ?", *filter.EntryID)
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		base = base.Where("original_file_name ILIKE ?", "%"+q+"%")
	}
	if filter.Status != nil {
		base = base.Where("extraction_status = ?", string(*filter.Status))
	}
	if filter.Linked != nil {
		if *filter.Linked {
			base = base.Where("entry_id IS NOT NULL")
		} else {
			base = base.Where("entry_id IS NULL")
		}
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, mapFinanceErr(err)
	}
	var rows []FinanceDocumentModel
	if err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, mapFinanceErr(err)
	}
	out := make([]dom.FinanceDocument, len(rows))
	for i := range rows {
		out[i] = *modelToFinanceDocument(&rows[i])
	}
	return out, total, nil
}

func (r *FinanceDocumentRepository) UpdateExtraction(ctx context.Context, d *dom.FinanceDocument) error {
	res := r.db.WithContext(ctx).Model(&FinanceDocumentModel{}).
		Where("id = ? AND workspace_id = ?", d.ID, d.WorkspaceID).
		Updates(map[string]any{
			"extraction_status": string(d.ExtractionStatus),
			"extracted_text":    d.ExtractedText,
			"extracted_json":    rawToJSON(d.ExtractedJSON),
			"entry_id":          d.EntryID,
			"fiscal_source":     d.FiscalSource,
			"payment_method":    d.PaymentMethod,
			"updated_at":        d.UpdatedAt,
		})
	if res.Error != nil {
		return mapFinanceErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *FinanceDocumentRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&FinanceDocumentModel{})
	if res.Error != nil {
		return mapFinanceErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

// --- conversões ---

func financeDocumentToModel(d *dom.FinanceDocument) FinanceDocumentModel {
	return FinanceDocumentModel{
		ID:               d.ID,
		WorkspaceID:      d.WorkspaceID,
		CardID:           d.CardID,
		EntryID:          d.EntryID,
		Kind:             string(d.Kind),
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
		FiscalSource:     d.FiscalSource,
		PaymentMethod:    d.PaymentMethod,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}
}

func modelToFinanceDocument(m *FinanceDocumentModel) *dom.FinanceDocument {
	var extracted, metadata []byte
	if len(m.ExtractedJSON) > 0 {
		extracted = []byte(m.ExtractedJSON)
	}
	if len(m.Metadata) > 0 {
		metadata = []byte(m.Metadata)
	}
	return &dom.FinanceDocument{
		ID:               m.ID,
		WorkspaceID:      m.WorkspaceID,
		CardID:           m.CardID,
		EntryID:          m.EntryID,
		Kind:             dom.DocumentKind(m.Kind),
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
		FiscalSource:     m.FiscalSource,
		PaymentMethod:    m.PaymentMethod,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}
