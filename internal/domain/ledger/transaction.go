package ledger

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Flow indica entrada ou saída de caixa.
type Flow string

const (
	FlowIn  Flow = "in"
	FlowOut Flow = "out"
)

const maxDescriptionLen = 2000

type Transaction struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	AccountID   uuid.UUID
	CategoryID  uuid.UUID
	AmountCents int64
	Flow        Flow
	Description string
	OccurredAt  time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (f Flow) IsValid() bool {
	switch f {
	case FlowIn, FlowOut:
		return true
	default:
		return false
	}
}

// Validate valida invariantes do lançamento. category deve ser informada quando
// a regra income/expense vs flow for aplicada (CreateTransaction use case).
func (t *Transaction) Validate() error {
	if t.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	if t.AccountID == uuid.Nil {
		return &ValidationError{Msg: "account_id é obrigatório"}
	}
	if t.CategoryID == uuid.Nil {
		return &ValidationError{Msg: "category_id é obrigatório"}
	}
	if t.AmountCents <= 0 {
		return &ValidationError{Msg: "valor deve ser positivo em centavos"}
	}
	if !t.Flow.IsValid() {
		return &ValidationError{Msg: "flow inválido: use in ou out"}
	}
	if t.OccurredAt.IsZero() {
		return &ValidationError{Msg: "data de ocorrência é obrigatória"}
	}
	desc := strings.TrimSpace(t.Description)
	if len(desc) > maxDescriptionLen {
		return &ValidationError{Msg: "descrição excede o tamanho máximo"}
	}
	t.Description = desc
	return nil
}

// MatchesCategoryKind garante que entrada use categoria income e saída use expense.
func (t *Transaction) MatchesCategoryKind(kind CategoryKind) bool {
	switch t.Flow {
	case FlowIn:
		return kind == CategoryKindIncome
	case FlowOut:
		return kind == CategoryKindExpense
	default:
		return false
	}
}
