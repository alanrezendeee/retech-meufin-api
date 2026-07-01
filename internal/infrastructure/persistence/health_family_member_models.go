package persistence

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type HealthFamilyMemberModel struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID  uuid.UUID  `gorm:"type:uuid;not null;index:idx_health_family_members_workspace"`
	FullName     string     `gorm:"size:255;not null"`
	Relationship string     `gorm:"size:20;not null"`
	BirthDate    *time.Time `gorm:"column:birth_date"`
	Gender       *string    `gorm:"size:20"`
	Document     *string    `gorm:"size:50"`
	Notes        *string    `gorm:"type:text"`
	Active       bool       `gorm:"not null;default:true"`
	CreatedAt    time.Time  `gorm:"not null"`
	UpdatedAt    time.Time  `gorm:"not null"`
	DeletedAt    gorm.DeletedAt
}

func (HealthFamilyMemberModel) TableName() string { return "health_family_members" }
