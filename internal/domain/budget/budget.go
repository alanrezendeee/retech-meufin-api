package budget

import (
	"time"

	"github.com/google/uuid"
)

// Budget define limite mensal de gastos (saídas) para uma categoria de despesa.
type Budget struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	CategoryID  uuid.UUID
	Year        int
	Month       int
	LimitCents  int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (b *Budget) Validate() error {
	if b.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	if b.CategoryID == uuid.Nil {
		return &ValidationError{Msg: "category_id é obrigatório"}
	}
	if b.Year < 2000 || b.Year > 2100 {
		return &ValidationError{Msg: "ano fora do intervalo permitido"}
	}
	if b.Month < 1 || b.Month > 12 {
		return &ValidationError{Msg: "mês deve estar entre 1 e 12"}
	}
	if b.LimitCents <= 0 {
		return &ValidationError{Msg: "limite deve ser positivo em centavos"}
	}
	return nil
}
