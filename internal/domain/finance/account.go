package finance

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxAccountNameLen = 255

// AccountKind classifica a conta usada para pagar/receber lançamentos.
type AccountKind string

const (
	AccountCorrente AccountKind = "corrente"
	AccountPoupanca AccountKind = "poupanca"
	AccountCarteira AccountKind = "carteira"
	AccountDigital  AccountKind = "digital"
)

// Account é uma conta do tenant (corrente, poupança, carteira/dinheiro, digital).
type Account struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	Kind        AccountKind
	BankName    *string
	Active      bool
	Notes       *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate normaliza campos e valida invariantes.
func (a *Account) Validate() error {
	if a.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	name := strings.TrimSpace(a.Name)
	if name == "" {
		return &ValidationError{Msg: "nome da conta é obrigatório"}
	}
	if len(name) > maxAccountNameLen {
		return &ValidationError{Msg: "nome da conta excede o tamanho máximo"}
	}
	a.Name = name

	switch a.Kind {
	case AccountCorrente, AccountPoupanca, AccountCarteira, AccountDigital:
	default:
		return &ValidationError{Msg: "kind da conta inválido"}
	}
	return nil
}

// AccountFilter recorta a listagem da tela de gestão.
type AccountFilter struct {
	Query  string // busca por nome ou banco (case-insensitive)
	Kind   string
	Active *bool
}

// AccountRepository persiste contas com escopo de workspace.
type AccountRepository interface {
	Create(ctx context.Context, a *Account) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*Account, error)
	Update(ctx context.Context, a *Account) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, filter AccountFilter, limit, offset int) ([]Account, int64, error)
}
