package finance

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ExtractionStatus modela o estado de extração de um documento financeiro.
// Reutilizado tanto em finance_documents (pending|processing|extracted|failed|not_required)
// quanto (parcialmente) em finance_extraction_jobs.
type ExtractionStatus string

const (
	ExtractionPending     ExtractionStatus = "pending"
	ExtractionProcessing  ExtractionStatus = "processing"
	ExtractionExtracted   ExtractionStatus = "extracted"
	ExtractionFailed      ExtractionStatus = "failed"
	ExtractionNotRequired ExtractionStatus = "not_required"
)

// FinanceDocument é um arquivo (PDF/imagem) vinculado ao módulo Financeiro
// (tipicamente uma fatura de cartão), armazenado no object storage.
// Mapeia a tabela finance_documents.
type FinanceDocument struct {
	ID               uuid.UUID
	WorkspaceID      uuid.UUID
	CardID           *uuid.UUID
	EntryID          *uuid.UUID // a fatura criada a partir do documento
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

// Validate valida invariantes do documento financeiro.
func (d *FinanceDocument) Validate() error {
	if d.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	if d.UploadedByUserID == uuid.Nil {
		return &ValidationError{Msg: "uploaded_by_user_id é obrigatório"}
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

// FinanceDocumentRepository persiste documentos financeiros (workspace-scoped, soft-delete).
type FinanceDocumentRepository interface {
	Create(ctx context.Context, d *FinanceDocument) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*FinanceDocument, error)
	List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]FinanceDocument, int64, error)
	// UpdateExtraction atualiza extraction_status/extracted_text/extracted_json/entry_id.
	UpdateExtraction(ctx context.Context, d *FinanceDocument) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
}
