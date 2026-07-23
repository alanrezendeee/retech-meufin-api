package finance

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type stubReconRepo struct {
	purchases []InvoicePurchase
	cupons    []ReconcileCandidate
}

func (s stubReconRepo) InvoicePurchases(context.Context, uuid.UUID, uuid.UUID) ([]InvoicePurchase, error) {
	return s.purchases, nil
}
func (s stubReconRepo) ReconcilableCupons(context.Context, uuid.UUID) ([]ReconcileCandidate, error) {
	return s.cupons, nil
}

func d(day int) time.Time { return time.Date(2026, 7, day, 0, 0, 0, 0, time.UTC) }

func TestSuggestForInvoice_MatchesByValueAndDateWindow(t *testing.T) {
	p1, p2 := uuid.New(), uuid.New()
	c1, c2, c3 := uuid.New(), uuid.New(), uuid.New()
	repo := stubReconRepo{
		purchases: []InvoicePurchase{
			{EntryID: p1, AmountCents: 92135, Date: d(18)},
			{EntryID: p2, AmountCents: 5000, Date: d(10)},
		},
		cupons: []ReconcileCandidate{
			{CupomEntryID: c1, AmountCents: 92135, Date: d(16)}, // casa p1 (2 dias)
			{CupomEntryID: c2, AmountCents: 5000, Date: d(2)},   // NÃO casa p2 (8 dias fora)
			{CupomEntryID: c3, AmountCents: 999, Date: d(10)},   // valor diferente
		},
	}
	svc := NewReconciliationService(repo, nil, nil, nil)
	matches, err := svc.SuggestForInvoice(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("SuggestForInvoice: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("esperava 1 match, veio %d", len(matches))
	}
	if matches[0].Purchase.EntryID != p1 || matches[0].Cupom.CupomEntryID != c1 {
		t.Fatalf("match errado: %+v", matches[0])
	}
	if matches[0].DaysDiff != 2 {
		t.Fatalf("days_diff esperado 2, veio %d", matches[0].DaysDiff)
	}
}

func TestSuggestForInvoice_CupomUsedOnce(t *testing.T) {
	p1, p2 := uuid.New(), uuid.New()
	c1 := uuid.New()
	repo := stubReconRepo{
		purchases: []InvoicePurchase{
			{EntryID: p1, AmountCents: 1000, Date: d(10)},
			{EntryID: p2, AmountCents: 1000, Date: d(10)}, // mesmo valor/data
		},
		cupons: []ReconcileCandidate{
			{CupomEntryID: c1, AmountCents: 1000, Date: d(10)}, // só 1 cupom
		},
	}
	svc := NewReconciliationService(repo, nil, nil, nil)
	matches, _ := svc.SuggestForInvoice(context.Background(), uuid.New(), uuid.New())
	if len(matches) != 1 {
		t.Fatalf("um cupom deve casar com só uma compra; veio %d matches", len(matches))
	}
}
