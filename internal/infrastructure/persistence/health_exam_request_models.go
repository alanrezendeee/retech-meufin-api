package persistence

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type HealthExamRequestModel struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID    uuid.UUID  `gorm:"type:uuid;not null;index:idx_health_exam_requests_workspace"`
	FamilyMemberID uuid.UUID  `gorm:"type:uuid;not null;index:idx_health_exam_requests_member"`
	LabID          *uuid.UUID `gorm:"type:uuid"`
	RequestedBy    *string    `gorm:"size:255"`
	RequestDate    time.Time  `gorm:"type:date;not null"`
	Status         string     `gorm:"size:30;not null;default:draft"`
	Notes          *string    `gorm:"type:text"`
	CreatedAt      time.Time  `gorm:"not null"`
	UpdatedAt      time.Time  `gorm:"not null"`
	DeletedAt      gorm.DeletedAt
	Items          []HealthExamRequestItemModel `gorm:"foreignKey:ExamRequestID"`
}

func (HealthExamRequestModel) TableName() string { return "health_exam_requests" }

type HealthExamRequestItemModel struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID   uuid.UUID  `gorm:"type:uuid;not null"`
	ExamRequestID uuid.UUID  `gorm:"type:uuid;not null;index:idx_health_exam_request_items_request"`
	MarkerID      *uuid.UUID `gorm:"type:uuid"`
	ExamName      string     `gorm:"size:255;not null"`
	ExamCode      *string    `gorm:"size:50"`
	BodyArea      *string    `gorm:"size:100"`
	Notes         *string    `gorm:"type:text"`
	Status        string     `gorm:"size:20;not null;default:pending"`
	CreatedAt     time.Time  `gorm:"not null"`
	UpdatedAt     time.Time  `gorm:"not null"`
	DeletedAt     gorm.DeletedAt
}

func (HealthExamRequestItemModel) TableName() string { return "health_exam_request_items" }
