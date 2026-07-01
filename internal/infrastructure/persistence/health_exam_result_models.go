package persistence

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type HealthExamResultModel struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID    uuid.UUID  `gorm:"type:uuid;not null;index:idx_health_exam_results_workspace"`
	FamilyMemberID uuid.UUID  `gorm:"type:uuid;not null"`
	LabID          *uuid.UUID `gorm:"type:uuid"`
	ExamRequestID  *uuid.UUID `gorm:"type:uuid"`
	ExamDate       time.Time  `gorm:"type:date;not null"`
	CollectionDate *time.Time `gorm:"type:date"`
	ReleaseDate    *time.Time `gorm:"type:date"`
	SourceType     string     `gorm:"size:20;not null;default:manual"`
	Status         string     `gorm:"size:20;not null;default:draft"`
	Summary        *string    `gorm:"type:text"`
	Notes          *string    `gorm:"type:text"`
	CreatedAt      time.Time  `gorm:"not null"`
	UpdatedAt      time.Time  `gorm:"not null"`
	DeletedAt      gorm.DeletedAt
	Items          []HealthExamResultItemModel `gorm:"foreignKey:ExamResultID"`
}

func (HealthExamResultModel) TableName() string { return "health_exam_results" }

type HealthExamResultItemModel struct {
	ID                     uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID            uuid.UUID  `gorm:"type:uuid;not null"`
	ExamResultID           uuid.UUID  `gorm:"type:uuid;not null;index:idx_health_exam_result_items_result"`
	MarkerID               *uuid.UUID `gorm:"type:uuid"`
	RawMarkerName          *string    `gorm:"column:raw_marker_name;size:255"`
	ResultValue            string     `gorm:"column:result_value;size:255;not null"`
	ResultNumeric          *float64   `gorm:"column:result_numeric"`
	Unit                   *string    `gorm:"size:30"`
	ReferenceMin           *float64   `gorm:"column:reference_min"`
	ReferenceMax           *float64   `gorm:"column:reference_max"`
	ReferenceText          *string    `gorm:"column:reference_text;size:255"`
	Interpretation         *string    `gorm:"size:20"`
	InterpretationComputed *string    `gorm:"column:interpretation_computed;size:20"`
	Method                 *string    `gorm:"size:100"`
	Material               *string    `gorm:"size:100"`
	RawText                *string    `gorm:"column:raw_text;type:text"`
	CreatedAt              time.Time  `gorm:"not null"`
	UpdatedAt              time.Time  `gorm:"not null"`
	DeletedAt              gorm.DeletedAt
}

func (HealthExamResultItemModel) TableName() string { return "health_exam_result_items" }
