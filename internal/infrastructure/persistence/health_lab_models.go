package persistence

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type HealthLabModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID    uuid.UUID `gorm:"type:uuid;not null;index:idx_health_labs_workspace"`
	Name           string    `gorm:"size:255;not null"`
	WebsiteURL     *string   `gorm:"column:website_url;size:500"`
	ExamResultsURL *string   `gorm:"column:exam_results_url;size:500"`
	ContactPhone   *string   `gorm:"column:contact_phone;size:50"`
	Address        *string   `gorm:"size:500"`
	Notes          *string   `gorm:"type:text"`
	Active         bool      `gorm:"not null;default:true"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
	DeletedAt      gorm.DeletedAt
}

func (HealthLabModel) TableName() string { return "health_labs" }
