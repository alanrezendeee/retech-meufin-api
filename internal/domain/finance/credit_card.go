package finance

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxCreditCardNameLen = 255

// CreditCard é um cartão de crédito cadastrado pelo tenant.
type CreditCard struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	Brand       *string
	ClosingDay  *int
	DueDay      *int
	Active      bool
	Notes       *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate normaliza campos e valida invariantes.
func (c *CreditCard) Validate() error {
	if c.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	name := strings.TrimSpace(c.Name)
	if name == "" {
		return &ValidationError{Msg: "nome do cartão é obrigatório"}
	}
	if len(name) > maxCreditCardNameLen {
		return &ValidationError{Msg: "nome do cartão excede o tamanho máximo"}
	}
	c.Name = name

	if c.ClosingDay != nil && (*c.ClosingDay < 1 || *c.ClosingDay > 31) {
		return &ValidationError{Msg: "closing_day deve estar entre 1 e 31"}
	}
	if c.DueDay != nil && (*c.DueDay < 1 || *c.DueDay > 31) {
		return &ValidationError{Msg: "due_day deve estar entre 1 e 31"}
	}
	return nil
}

// CreditCardFilter recorta a listagem da tela de gestão.
type CreditCardFilter struct {
	Query  string // busca por nome (case-insensitive)
	Active *bool
}

// CreditCardRepository persiste cartões de crédito com escopo de workspace.
type CreditCardRepository interface {
	Create(ctx context.Context, c *CreditCard) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*CreditCard, error)
	Update(ctx context.Context, c *CreditCard) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, filter CreditCardFilter, limit, offset int) ([]CreditCard, int64, error)
}
