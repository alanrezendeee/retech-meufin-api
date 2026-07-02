package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"gorm.io/gorm"
)

// FamilyMemberDocumentModel mapeia a tabela family_member_documents.
type FamilyMemberDocumentModel struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID      uuid.UUID  `gorm:"type:uuid;not null;index:idx_family_member_documents_workspace"`
	FamilyMemberID   uuid.UUID  `gorm:"type:uuid;not null;index:idx_family_member_documents_member"`
	DocType          string     `gorm:"column:doc_type;size:30;not null"`
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

func (FamilyMemberDocumentModel) TableName() string { return "family_member_documents" }

type HealthMemberDocumentRepository struct {
	db *gorm.DB
}

func NewHealthMemberDocumentRepository(db *gorm.DB) *HealthMemberDocumentRepository {
	return &HealthMemberDocumentRepository{db: db}
}

func (r *HealthMemberDocumentRepository) Create(ctx context.Context, d *dom.MemberDocument) error {
	model := memberDocumentToModel(d)
	return mapHealthErr(r.db.WithContext(ctx).Create(&model).Error)
}

func (r *HealthMemberDocumentRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.MemberDocument, error) {
	var m FamilyMemberDocumentModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	return modelToMemberDocument(&m), nil
}

func (r *HealthMemberDocumentRepository) ListByMember(ctx context.Context, workspaceID, familyMemberID uuid.UUID, limit, offset int) ([]dom.MemberDocument, int64, error) {
	base := r.db.WithContext(ctx).Model(&FamilyMemberDocumentModel{}).
		Where("workspace_id = ? AND family_member_id = ?", workspaceID, familyMemberID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	var rows []FamilyMemberDocumentModel
	if err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	out := make([]dom.MemberDocument, len(rows))
	for i := range rows {
		out[i] = *modelToMemberDocument(&rows[i])
	}
	return out, total, nil
}

func (r *HealthMemberDocumentRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&FamilyMemberDocumentModel{})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

// --- conversões ---

func memberDocumentToModel(d *dom.MemberDocument) FamilyMemberDocumentModel {
	return FamilyMemberDocumentModel{
		ID:               d.ID,
		WorkspaceID:      d.WorkspaceID,
		FamilyMemberID:   d.FamilyMemberID,
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

func modelToMemberDocument(m *FamilyMemberDocumentModel) *dom.MemberDocument {
	return &dom.MemberDocument{
		ID:               m.ID,
		WorkspaceID:      m.WorkspaceID,
		FamilyMemberID:   m.FamilyMemberID,
		DocType:          dom.MemberDocType(m.DocType),
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
