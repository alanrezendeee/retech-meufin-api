package health

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DocumentType classifica o documento de saúde armazenado.
type DocumentType string

const (
	DocTypeExamRequest   DocumentType = "exam_request"
	DocTypeExamResult    DocumentType = "exam_result"
	DocTypeImageReport   DocumentType = "image_report"
	DocTypeMedicalReport DocumentType = "medical_report"
	DocTypePrescription  DocumentType = "prescription"
	DocTypeOther         DocumentType = "other"
)

// Estados de extração específicos de documentos (complementam os de extraction_job.go).
// ExtractionStatus, ExtractionPending, ExtractionProcessing e ExtractionFailed são
// declarados em extraction_job.go e reutilizados aqui.
const (
	ExtractionExtracted   ExtractionStatus = "extracted"
	ExtractionNotRequired ExtractionStatus = "not_required"
)

// Document é um arquivo (PDF/imagem) vinculado ao domínio de saúde, armazenado no object storage.
type Document struct {
	ID               uuid.UUID
	WorkspaceID      uuid.UUID
	FamilyMemberID   *uuid.UUID
	LabID            *uuid.UUID
	ExamRequestID    *uuid.UUID
	ExamResultID     *uuid.UUID
	DocumentType     DocumentType
	FileName         string
	OriginalFileName string
	MimeType         string
	SizeBytes        int64
	StorageProvider  string
	Bucket           string
	ObjectKey        string
	Checksum         *string
	UploadedByUserID uuid.UUID
	ExtractionStatus ExtractionStatus
	ExtractedText    *string
	ExtractedJSON    []byte // JSON cru (JSONB); nil quando ausente
	Metadata         []byte // JSON cru (JSONB); nil quando ausente
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ValidDocumentType indica se o tipo de documento é conhecido.
func ValidDocumentType(t DocumentType) bool {
	switch t {
	case DocTypeExamRequest, DocTypeExamResult, DocTypeImageReport,
		DocTypeMedicalReport, DocTypePrescription, DocTypeOther:
		return true
	default:
		return false
	}
}

// Validate valida invariantes do documento.
func (d *Document) Validate() error {
	if d.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	if d.UploadedByUserID == uuid.Nil {
		return &ValidationError{Msg: "uploaded_by_user_id é obrigatório"}
	}
	if !ValidDocumentType(d.DocumentType) {
		return &ValidationError{Msg: "document_type inválido"}
	}
	if strings.TrimSpace(d.FileName) == "" {
		return &ValidationError{Msg: "file_name é obrigatório"}
	}
	if strings.TrimSpace(d.ObjectKey) == "" {
		return &ValidationError{Msg: "object_key é obrigatório"}
	}
	if strings.TrimSpace(d.Bucket) == "" {
		return &ValidationError{Msg: "bucket é obrigatório"}
	}
	return nil
}

// DocumentRepository persiste documentos de saúde (workspace-scoped, soft-delete).
type DocumentRepository interface {
	Create(ctx context.Context, d *Document) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*Document, error)
	List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]Document, int64, error)
	// UpdateExtraction atualiza extraction_status/extracted_text/extracted_json.
	UpdateExtraction(ctx context.Context, d *Document) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
}
