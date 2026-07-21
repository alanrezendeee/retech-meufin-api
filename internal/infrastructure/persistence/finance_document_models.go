package persistence

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// FinanceDocumentModel mapeia a tabela finance_documents.
type FinanceDocumentModel struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey"`
	WorkspaceID      uuid.UUID      `gorm:"type:uuid;not null;index:idx_finance_documents_workspace"`
	CardID           *uuid.UUID     `gorm:"type:uuid;index:idx_finance_documents_card"`
	EntryID          *uuid.UUID     `gorm:"type:uuid;index:idx_finance_documents_entry"`
	Kind             string         `gorm:"size:20;not null;default:import"`
	FileName         string         `gorm:"size:255;not null"`
	OriginalFileName string         `gorm:"size:255;not null"`
	MimeType         string         `gorm:"size:100;not null"`
	SizeBytes        int64          `gorm:"not null;default:0"`
	StorageProvider  string         `gorm:"size:20;not null;default:minio"`
	Bucket           string         `gorm:"size:255;not null"`
	ObjectKey        string         `gorm:"size:500;not null"`
	Checksum         *string        `gorm:"size:128"`
	UploadedByUserID uuid.UUID      `gorm:"type:uuid;not null"`
	ExtractionStatus string         `gorm:"size:20;not null;default:pending"`
	ExtractedText    *string        `gorm:"type:text"`
	ExtractedJSON    datatypes.JSON `gorm:"type:jsonb"`
	Metadata         datatypes.JSON `gorm:"type:jsonb"`
	FiscalSource     *string        `gorm:"column:fiscal_source;size:20"`
	PaymentMethod    *string        `gorm:"column:payment_method;size:20"`
	CreatedAt        time.Time      `gorm:"not null"`
	UpdatedAt        time.Time      `gorm:"not null"`
	DeletedAt        gorm.DeletedAt
}

func (FinanceDocumentModel) TableName() string { return "finance_documents" }
