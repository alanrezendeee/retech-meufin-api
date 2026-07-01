package persistence

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// FinanceExtractionJobModel mapeia a tabela finance_extraction_jobs.
// A tabela NÃO possui deleted_at — sem soft delete.
type FinanceExtractionJobModel struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey"`
	WorkspaceID   uuid.UUID      `gorm:"type:uuid;not null"`
	DocumentID    uuid.UUID      `gorm:"type:uuid;not null;index:idx_finance_extraction_jobs_document"`
	Provider      string         `gorm:"size:30;not null"`
	Model         *string        `gorm:"size:100"`
	Status        string         `gorm:"size:20;not null;default:pending"`
	InputType     string         `gorm:"size:10;not null"`
	PromptVersion *string        `gorm:"column:prompt_version;size:30"`
	RawResponse   datatypes.JSON `gorm:"column:raw_response;type:jsonb"`
	ErrorMessage  *string        `gorm:"column:error_message;type:text"`
	StartedAt     *time.Time     `gorm:"column:started_at"`
	FinishedAt    *time.Time     `gorm:"column:finished_at"`
	CreatedAt     time.Time      `gorm:"not null"`
	UpdatedAt     time.Time      `gorm:"not null"`
}

func (FinanceExtractionJobModel) TableName() string { return "finance_extraction_jobs" }
