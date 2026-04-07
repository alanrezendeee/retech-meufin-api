package ledger

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxAccountNameLen = 255

// Account representa uma conta financeira (caixa, banco, cartão sintético, etc.).
type Account struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	Currency    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

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
	cur := strings.TrimSpace(strings.ToUpper(a.Currency))
	if len(cur) != 3 {
		return &ValidationError{Msg: "moeda deve ser código ISO de 3 letras"}
	}
	a.Name = name
	a.Currency = cur
	return nil
}
