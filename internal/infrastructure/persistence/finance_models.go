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

type CreditCardModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index:idx_credit_cards_workspace"`
	Name        string    `gorm:"size:255;not null"`
	Brand       *string   `gorm:"size:50"`
	ClosingDay  *int      `gorm:"column:closing_day"`
	DueDay      *int      `gorm:"column:due_day"`
	Active      bool      `gorm:"not null;default:true"`
	Notes       *string   `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
	DeletedAt   gorm.DeletedAt
}

func (CreditCardModel) TableName() string { return "credit_cards" }

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
	CardID            *uuid.UUID `gorm:"column:card_id;type:uuid"`
	ParentID          *uuid.UUID `gorm:"column:parent_id;type:uuid"`
	InstallmentNumber *int       `gorm:"column:installment_number"`
	InstallmentTotal  *int       `gorm:"column:installment_total"`
	Notes             *string    `gorm:"type:text"`
	PaidAt            *time.Time `gorm:"column:paid_at"`
	PaidAmountCents   *int64     `gorm:"column:paid_amount_cents"`
	PaymentMethod     *string    `gorm:"column:payment_method;size:20"`
	PaymentAccountID  *uuid.UUID `gorm:"column:payment_account_id;type:uuid"`
	PaymentCardID     *uuid.UUID `gorm:"column:payment_card_id;type:uuid"`
	CreatedAt         time.Time  `gorm:"not null"`
	UpdatedAt         time.Time  `gorm:"not null"`
	DeletedAt         gorm.DeletedAt
}

func (FinancialEntryModel) TableName() string { return "financial_entries" }
