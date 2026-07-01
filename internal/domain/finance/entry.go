package finance

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Kind diferencia receita (credit) de despesa (debit).
type Kind string

const (
	KindCredit Kind = "credit"
	KindDebit  Kind = "debit"
)

// Status representa o ciclo de vida de um lançamento.
type Status string

const (
	StatusPrevista  Status = "prevista"
	StatusRealizada Status = "realizada"
	StatusCancelada Status = "cancelada"
)

// Recurrence define a periodicidade de geração das ocorrências.
type Recurrence string

const (
	RecurrenceNone    Recurrence = "none"
	RecurrenceWeekly  Recurrence = "weekly"
	RecurrenceMonthly Recurrence = "monthly"
	RecurrenceYearly  Recurrence = "yearly"
)

// FinancialEntry é um lançamento único de crédito ou débito.
type FinancialEntry struct {
	ID                uuid.UUID
	WorkspaceID       uuid.UUID
	Kind              Kind
	Status            Status
	AmountCents       int64
	DueDate           time.Time
	FamilyMemberID    *uuid.UUID
	SourceID          *uuid.UUID
	Type              *string
	Description       string
	Recurrence        Recurrence
	RecurrenceGroupID *uuid.UUID
	CardID            *uuid.UUID
	ParentID          *uuid.UUID
	InstallmentNumber *int
	InstallmentTotal  *int
	Notes             *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Validate valida invariantes do lançamento.
func (e *FinancialEntry) Validate() error {
	if e.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	switch e.Kind {
	case KindCredit, KindDebit:
	default:
		return &ValidationError{Msg: "kind do lançamento inválido"}
	}
	switch e.Status {
	case StatusPrevista, StatusRealizada, StatusCancelada:
	case "":
		e.Status = StatusPrevista
	default:
		return &ValidationError{Msg: "status do lançamento inválido"}
	}
	switch e.Recurrence {
	case RecurrenceNone, RecurrenceWeekly, RecurrenceMonthly, RecurrenceYearly:
	case "":
		e.Recurrence = RecurrenceNone
	default:
		return &ValidationError{Msg: "recurrence do lançamento inválido"}
	}
	if e.AmountCents == 0 {
		return &ValidationError{Msg: "amount_cents não pode ser zero"}
	}
	if e.DueDate.IsZero() {
		return &ValidationError{Msg: "due_date é obrigatória"}
	}
	e.Description = strings.TrimSpace(e.Description)
	return nil
}

// FinancialEntryFilter filtra a listagem de lançamentos.
type FinancialEntryFilter struct {
	Kind           *string
	Status         *string
	FamilyMemberID *uuid.UUID
	Type           *string
	Year           *int
	Month          *int
	CardID         *uuid.UUID
	ParentID       *uuid.UUID
	TopLevelOnly   bool
}

// FinancialEntryRepository persiste lançamentos com escopo de workspace.
type FinancialEntryRepository interface {
	Create(ctx context.Context, e *FinancialEntry) error
	CreateBatch(ctx context.Context, es []*FinancialEntry) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*FinancialEntry, error)
	Update(ctx context.Context, e *FinancialEntry) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, filter FinancialEntryFilter, limit, offset int) ([]FinancialEntry, int64, error)
}
