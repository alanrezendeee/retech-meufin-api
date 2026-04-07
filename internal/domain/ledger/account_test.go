package ledger

import (
	"testing"

	"github.com/google/uuid"
)

func TestAccount_Validate(t *testing.T) {
	ws := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	t.Run("ok", func(t *testing.T) {
		a := &Account{WorkspaceID: ws, Name: "  Conta corrente ", Currency: "brl"}
		if err := a.Validate(); err != nil {
			t.Fatal(err)
		}
		if a.Name != "Conta corrente" || a.Currency != "BRL" {
			t.Fatalf("normalização: %+v", a)
		}
	})
	t.Run("nome vazio", func(t *testing.T) {
		a := &Account{WorkspaceID: ws, Name: "   ", Currency: "BRL"}
		if err := a.Validate(); err == nil {
			t.Fatal("esperado erro")
		}
	})
	t.Run("moeda inválida", func(t *testing.T) {
		a := &Account{WorkspaceID: ws, Name: "X", Currency: "BR"}
		if err := a.Validate(); err == nil {
			t.Fatal("esperado erro")
		}
	})
}
