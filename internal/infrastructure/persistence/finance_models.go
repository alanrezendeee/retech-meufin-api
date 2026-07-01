package persistence

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IncomeSourceModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index:idx_income_sources_workspace"`
	Name        string    `gorm:"size:255;not null"`
	Kind        string    `gorm:"size:20;not null"`
	Active      bool      `gorm:"not null;default:true"`
	Notes       *string   `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
	DeletedAt   gorm.DeletedAt
}

func (IncomeSourceModel) TableName() string { return "income_sources" }

type FinancialEntryModel struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID       uuid.UUID  `gorm:"type:uuid;not null;index:idx_financial_entries_workspace"`
	Kind              string     `gorm:"size:10;not null"`
	Status            string     `gorm:"size:15;not null;default:prevista"`
	AmountCents       int64      `gorm:"not null"`
	DueDate           time.Time  `gorm:"column:due_date;type:date;not null"`
	FamilyMemberID    *uuid.UUID `gorm:"type:uuid"`
	SourceID          *uuid.UUID `gorm:"type:uuid"`
	Type              *string    `gorm:"size:30"`
	Description       string     `gorm:"type:text;not null;default:''"`
	Recurrence        string     `gorm:"size:10;not null;default:none"`
	RecurrenceGroupID *uuid.UUID `gorm:"type:uuid"`
	Notes             *string    `gorm:"type:text"`
	CreatedAt         time.Time  `gorm:"not null"`
	UpdatedAt         time.Time  `gorm:"not null"`
	DeletedAt         gorm.DeletedAt
}

func (FinancialEntryModel) TableName() string { return "financial_entries" }
