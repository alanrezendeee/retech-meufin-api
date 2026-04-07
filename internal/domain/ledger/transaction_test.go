package ledger

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTransaction_Validate(t *testing.T) {
	ws := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	acc := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	cat := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	t.Run("ok", func(t *testing.T) {
		tx := &Transaction{
			WorkspaceID: ws, AccountID: acc, CategoryID: cat,
			AmountCents: 100, Flow: FlowOut, OccurredAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC),
		}
		if err := tx.Validate(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("valor zero", func(t *testing.T) {
		tx := &Transaction{
			WorkspaceID: ws, AccountID: acc, CategoryID: cat,
			AmountCents: 0, Flow: FlowOut, OccurredAt: time.Now().UTC(),
		}
		if err := tx.Validate(); err == nil {
			t.Fatal("esperado erro")
		}
	})
}

func TestTransaction_MatchesCategoryKind(t *testing.T) {
	tx := &Transaction{Flow: FlowIn}
	if !tx.MatchesCategoryKind(CategoryKindIncome) {
		t.Fatal("entrada deve casar com income")
	}
	if tx.MatchesCategoryKind(CategoryKindExpense) {
		t.Fatal("entrada não deve casar com expense")
	}
}
