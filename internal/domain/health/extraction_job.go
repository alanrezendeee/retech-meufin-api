package health

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ExtractionStatus modela o ciclo de vida de um job de extração.
type ExtractionStatus string

const (
	ExtractionPending    ExtractionStatus = "pending"
	ExtractionProcessing ExtractionStatus = "processing"
	ExtractionCompleted  ExtractionStatus = "completed"
	ExtractionFailed     ExtractionStatus = "failed"
)

// ExtractionInputType indica o tipo do documento processado.
type ExtractionInputType string

const (
	ExtractionInputPDF   ExtractionInputType = "pdf"
	ExtractionInputImage ExtractionInputType = "image"
)

// ExtractionJob é o registro de um processamento de extração OCR/LLM de um
// documento de saúde. Mapeia a tabela health_extraction_jobs.
type ExtractionJob struct {
	ID            uuid.UUID
	WorkspaceID   uuid.UUID
	DocumentID    uuid.UUID
	Provider      string
	Model         *string
	Status        ExtractionStatus
	InputType     ExtractionInputType
	PromptVersion *string
	RawResponse   json.RawMessage
	ErrorMessage  *string
	StartedAt     *time.Time
	FinishedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ExtractionJobRepository abstrai a persistência de jobs de extração,
// sempre no escopo do tenant (workspace).
type ExtractionJobRepository interface {
	Create(ctx context.Context, j *ExtractionJob) error
	Update(ctx context.Context, j *ExtractionJob) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*ExtractionJob, error)
	// GetByDocument retorna o job mais recente do documento.
	GetByDocument(ctx context.Context, workspaceID, documentID uuid.UUID) (*ExtractionJob, error)
}
