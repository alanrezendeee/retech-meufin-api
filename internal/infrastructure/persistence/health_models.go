package persistence

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type HealthMarkerModel struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Scope          string     `gorm:"size:10;not null"`
	WorkspaceID    *uuid.UUID `gorm:"type:uuid;index:idx_health_markers_workspace"`
	CanonicalName  string     `gorm:"size:255;not null"`
	NormalizedKey  string     `gorm:"size:255;not null"`
	LoincCode      *string    `gorm:"size:20"`
	Category       string     `gorm:"size:50;not null"`
	Comparability  string     `gorm:"column:comparability_class;size:20;not null;default:standardized"`
	CanonicalUnit  *string    `gorm:"size:30"`
	DefaultRefMin  *float64   `gorm:"column:default_ref_min"`
	DefaultRefMax  *float64   `gorm:"column:default_ref_max"`
	DefaultRefText *string    `gorm:"column:default_ref_text;size:255"`
	Active         bool       `gorm:"not null;default:true"`
	CreatedAt      time.Time  `gorm:"not null"`
	UpdatedAt      time.Time  `gorm:"not null"`
	DeletedAt      gorm.DeletedAt
	Aliases        []HealthMarkerAliasModel `gorm:"foreignKey:MarkerID"`
}

func (HealthMarkerModel) TableName() string { return "health_markers" }

type HealthMarkerAliasModel struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey"`
	MarkerID        uuid.UUID  `gorm:"type:uuid;not null;index:idx_health_marker_aliases_marker"`
	Scope           string     `gorm:"size:10;not null"`
	WorkspaceID     *uuid.UUID `gorm:"type:uuid"`
	Alias           string     `gorm:"size:255;not null"`
	NormalizedAlias string     `gorm:"size:255;not null"`
	Source          *string    `gorm:"size:50"`
	CreatedAt       time.Time  `gorm:"not null"`
	UpdatedAt       time.Time  `gorm:"not null"`
	DeletedAt       gorm.DeletedAt
}

func (HealthMarkerAliasModel) TableName() string { return "health_marker_aliases" }
