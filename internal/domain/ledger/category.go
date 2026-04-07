package ledger

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// CategoryKind classifica a natureza da categoria para validação de lançamentos.
type CategoryKind string

const (
	CategoryKindIncome  CategoryKind = "income"
	CategoryKindExpense CategoryKind = "expense"
)

const maxCategoryNameLen = 255

type Category struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	Kind        CategoryKind
	ParentID    *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (k CategoryKind) IsValid() bool {
	switch k {
	case CategoryKindIncome, CategoryKindExpense:
		return true
	default:
		return false
	}
}

func (c *Category) Validate() error {
	if c.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	name := strings.TrimSpace(c.Name)
	if name == "" {
		return &ValidationError{Msg: "nome da categoria é obrigatório"}
	}
	if len(name) > maxCategoryNameLen {
		return &ValidationError{Msg: "nome da categoria excede o tamanho máximo"}
	}
	if !c.Kind.IsValid() {
		return &ValidationError{Msg: "tipo de categoria inválido: use income ou expense"}
	}
	c.Name = name
	return nil
}
