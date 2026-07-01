package finance

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ExtractionJobStatus modela o ciclo de vida de um job de extração LLM.
type ExtractionJobStatus string

const (
	JobPending    ExtractionJobStatus = "pending"
	JobProcessing ExtractionJobStatus = "processing"
	JobCompleted  ExtractionJobStatus = "completed"
	JobFailed     ExtractionJobStatus = "failed"
)

// ExtractionInputType indica o tipo do documento processado.
type ExtractionInputType string

const (
	ExtractionInputPDF   ExtractionInputType = "pdf"
	ExtractionInputImage ExtractionInputType = "image"
)

// FinanceExtractionJob é o registro de um processamento de extração LLM de um
// documento financeiro (fatura). Mapeia a tabela finance_extraction_jobs.
type FinanceExtractionJob struct {
	ID            uuid.UUID
	WorkspaceID   uuid.UUID
	DocumentID    uuid.UUID
	Provider      string
	Model         *string
	Status        ExtractionJobStatus
	InputType     ExtractionInputType
	PromptVersion *string
	RawResponse   json.RawMessage
	ErrorMessage  *string
	StartedAt     *time.Time
	FinishedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// FinanceExtractionJobRepository abstrai a persistência de jobs de extração,
// sempre no escopo do tenant (workspace).
type FinanceExtractionJobRepository interface {
	Create(ctx context.Context, j *FinanceExtractionJob) error
	Update(ctx context.Context, j *FinanceExtractionJob) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*FinanceExtractionJob, error)
	// GetByDocument retorna o job mais recente do documento.
	GetByDocument(ctx context.Context, workspaceID, documentID uuid.UUID) (*FinanceExtractionJob, error)
}
