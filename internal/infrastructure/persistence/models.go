package persistence

import (
	"time"

	"github.com/google/uuid"
)

type FinancialAccountModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index:idx_financial_accounts_workspace"`
	Name        string    `gorm:"size:255;not null"`
	Currency    string    `gorm:"size:3;not null;default:BRL"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (FinancialAccountModel) TableName() string { return "financial_accounts" }

type CategoryModel struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID  `gorm:"type:uuid;not null;index:idx_categories_workspace"`
	Name        string     `gorm:"size:255;not null"`
	Kind        string     `gorm:"size:20;not null"`
	ParentID    *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time  `gorm:"not null"`
	UpdatedAt   time.Time  `gorm:"not null"`
}

func (CategoryModel) TableName() string { return "categories" }

type TransactionModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index:idx_transactions_workspace_occurred"`
	AccountID   uuid.UUID `gorm:"type:uuid;not null"`
	CategoryID  uuid.UUID `gorm:"type:uuid;not null;index:idx_transactions_category"`
	AmountCents int64     `gorm:"not null"`
	Flow        string    `gorm:"size:10;not null"`
	Description string    `gorm:"type:text;not null"`
	OccurredAt  time.Time `gorm:"not null"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (TransactionModel) TableName() string { return "transactions" }

type BudgetModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index:idx_budgets_workspace"`
	CategoryID  uuid.UUID `gorm:"type:uuid;not null"`
	Year        int       `gorm:"not null"`
	Month       int       `gorm:"not null"`
	LimitCents  int64     `gorm:"not null"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (BudgetModel) TableName() string { return "budgets" }
