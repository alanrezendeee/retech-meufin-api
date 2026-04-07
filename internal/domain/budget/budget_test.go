package budget

import (
	"testing"

	"github.com/google/uuid"
)

func TestBudget_Validate(t *testing.T) {
	ws := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	cat := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	t.Run("ok", func(t *testing.T) {
		b := &Budget{WorkspaceID: ws, CategoryID: cat, Year: 2026, Month: 4, LimitCents: 50_000_00}
		if err := b.Validate(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("mês inválido", func(t *testing.T) {
		b := &Budget{WorkspaceID: ws, CategoryID: cat, Year: 2026, Month: 13, LimitCents: 1}
		if err := b.Validate(); err == nil {
			t.Fatal("esperado erro")
		}
	})
}
