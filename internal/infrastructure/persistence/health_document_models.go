package persistence

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type HealthDocumentModel struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey"`
	WorkspaceID      uuid.UUID      `gorm:"type:uuid;not null;index:idx_health_documents_workspace"`
	FamilyMemberID   *uuid.UUID     `gorm:"type:uuid"`
	LabID            *uuid.UUID     `gorm:"type:uuid"`
	ExamRequestID    *uuid.UUID     `gorm:"type:uuid"`
	ExamResultID     *uuid.UUID     `gorm:"type:uuid;index:idx_health_documents_result"`
	DocumentType     string         `gorm:"size:30;not null"`
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
	CreatedAt        time.Time      `gorm:"not null"`
	UpdatedAt        time.Time      `gorm:"not null"`
	DeletedAt        gorm.DeletedAt
}

func (HealthDocumentModel) TableName() string { return "health_documents" }
