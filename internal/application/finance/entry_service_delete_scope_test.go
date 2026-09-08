package finance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

func TestDeleteScopeOneKeepsSiblings(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})
	ws := uuid.New()
	occs := createInstallmentSeries(t, svc, ws, 6)

	res, err := svc.Delete(context.Background(), ws, occs[2].ID, DeleteScopeOne)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if res.Deleted != 1 || len(repo.entries) != 5 {
		t.Fatalf("esperava 1 excluída e 5 restantes, veio %d / %d", res.Deleted, len(repo.entries))
	}
}

func TestDeleteScopeFutureInstallmentsKeepsPaid(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})
	ws := uuid.New()
	occs := createInstallmentSeries(t, svc, ws, 6)

	// Parcelas 1 e 2 pagas; a 5ª também (fora de ordem) — paga futura não cai.
	for _, i := range []int{0, 1, 4} {
		repo.entries[occs[i].ID].Status = dom.StatusRealizada
	}

	res, err := svc.Delete(context.Background(), ws, occs[2].ID, DeleteScopeFuture)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// Âncora (3ª) + previstas futuras (4ª e 6ª) = 3. A 5ª paga sobrevive.
	if res.Deleted != 3 || res.DeletedPaid != 0 || res.RecurrenceEnded {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	for _, i := range []int{0, 1, 4} {
		if _, ok := repo.entries[occs[i].ID]; !ok {
			t.Fatalf("parcela %d paga foi excluída", i+1)
		}
	}
	for _, i := range []int{2, 3, 5} {
		if _, ok := repo.entries[occs[i].ID]; ok {
			t.Fatalf("parcela %d deveria ter sido excluída", i+1)
		}
	}
}

func TestDeleteScopeAllRemovesPaidAndResiduals(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})
	ws := uuid.New()
	occs := createInstallmentSeries(t, svc, ws, 4)
	repo.entries[occs[0].ID].Status = dom.StatusRealizada

	// Residual de pagamento parcial da 2ª parcela.
	residual := &dom.FinancialEntry{
		ID: uuid.New(), WorkspaceID: ws, Kind: dom.KindDebit, Status: dom.StatusPrevista,
		AmountCents: 10000, DueDate: occs[1].DueDate, Description: dom.ResidualPrefix + "Financiamento",
		ResidualOfID: &occs[1].ID,
	}
	repo.entries[residual.ID] = residual

	res, err := svc.Delete(context.Background(), ws, occs[3].ID, DeleteScopeAll)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if res.Deleted != 4 || res.DeletedPaid != 1 || res.DeletedResiduals != 1 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if len(repo.entries) != 0 {
		t.Fatalf("esperava repositório vazio, sobraram %d", len(repo.entries))
	}
}

func TestDeleteScopeFutureRecurringEndsSeries(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})
	ws := uuid.New()
	occs := createMonthlySeries(t, svc, ws, time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC), 100000)

	anchor := occs[3]
	res, err := svc.Delete(context.Background(), ws, anchor.ID, DeleteScopeFuture)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !res.RecurrenceEnded {
		t.Fatalf("esperava recorrência encerrada: %+v", res)
	}
	if res.Deleted != dom.RollingMonths-4 {
		t.Fatalf("esperava %d excluídas, veio %d", dom.RollingMonths-4, res.Deleted)
	}
	got := repo.entries[anchor.ID]
	if got == nil || got.Status != dom.StatusCancelada || got.CancelReason == nil || *got.CancelReason != dom.CancelReasonEncerramento {
		t.Fatalf("âncora deveria virar cancelada/encerramento: %+v", got)
	}
	// O extensor não pode ressuscitar os meses apagados.
	created, err := svc.ExtendRecurrences(context.Background())
	if err != nil {
		t.Fatalf("ExtendRecurrences: %v", err)
	}
	if created != 0 {
		t.Fatalf("extensor recriou %d ocorrências", created)
	}
}

func TestDeleteScopeRequiresSeries(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})
	ws := uuid.New()
	occs, err := svc.Create(context.Background(), CreateEntryInput{
		WorkspaceID: ws, Kind: "debit", AmountCents: 5000,
		DueDate: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC), Description: "Único",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_, err = svc.Delete(context.Background(), ws, occs[0].ID, DeleteScopeAll)
	var ve *dom.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("esperava ValidationError, veio %v", err)
	}
}

func TestCreateWithCustomInstallments(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})
	ws := uuid.New()
	paid1 := time.Date(2025, 11, 12, 0, 0, 0, 0, time.UTC)
	paidAmt := int64(19500)
	paid2 := time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC)
	specs := []dom.InstallmentSpec{
		// Fora de ordem de propósito: o service numera por vencimento.
		{DueDate: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC), AmountCents: 20000},
		{DueDate: time.Date(2025, 11, 10, 0, 0, 0, 0, time.UTC), AmountCents: 20000, PaidAt: &paid1, PaidAmountCents: &paidAmt},
		{DueDate: time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC), AmountCents: 21000, PaidAt: &paid2},
	}
	total := 3
	occs, err := svc.Create(context.Background(), CreateEntryInput{
		WorkspaceID: ws, Kind: "debit", AmountCents: 20000,
		DueDate: specs[1].DueDate, Description: "Geladeira",
		InstallmentsTotal: &total, Installments: specs,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(occs) != 3 {
		t.Fatalf("esperava 3 parcelas, veio %d", len(occs))
	}
	first, second, third := occs[0], occs[1], occs[2]
	if *first.InstallmentNumber != 1 || !first.DueDate.Equal(specs[1].DueDate) || first.Status != dom.StatusRealizada || *first.PaidAmountCents != paidAmt || !first.PaidAt.Equal(paid1) {
		t.Fatalf("1ª parcela inesperada: %+v", first)
	}
	if *second.InstallmentNumber != 2 || second.AmountCents != 21000 || second.Status != dom.StatusRealizada || *second.PaidAmountCents != 21000 {
		t.Fatalf("2ª parcela inesperada: %+v", second)
	}
	if *third.InstallmentNumber != 3 || third.Status != dom.StatusPrevista || third.PaidAt != nil {
		t.Fatalf("3ª parcela inesperada: %+v", third)
	}
	if first.RecurrenceGroupID == nil || *first.RecurrenceGroupID != *third.RecurrenceGroupID || *third.InstallmentTotal != 3 {
		t.Fatalf("grupo/total inconsistentes")
	}
}

func TestCreateWithCustomInstallmentsValidation(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})
	ws := uuid.New()
	due := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		total *int
		specs []dom.InstallmentSpec
	}{
		{"uma só", nil, []dom.InstallmentSpec{{DueDate: due, AmountCents: 100}}},
		{"total divergente", intPtr(3), []dom.InstallmentSpec{{DueDate: due, AmountCents: 100}, {DueDate: due.AddDate(0, 1, 0), AmountCents: 100}}},
		{"valor zero", nil, []dom.InstallmentSpec{{DueDate: due, AmountCents: 0}, {DueDate: due.AddDate(0, 1, 0), AmountCents: 100}}},
		{"pago sem data", nil, []dom.InstallmentSpec{{DueDate: due, AmountCents: 100, PaidAmountCents: intPtr64(100)}, {DueDate: due.AddDate(0, 1, 0), AmountCents: 100}}},
	}
	for _, tc := range cases {
		_, err := svc.Create(context.Background(), CreateEntryInput{
			WorkspaceID: ws, Kind: "debit", AmountCents: 100, DueDate: due, Description: "X",
			InstallmentsTotal: tc.total, Installments: tc.specs,
		})
		var ve *dom.ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("%s: esperava ValidationError, veio %v", tc.name, err)
		}
	}
}

func intPtr(n int) *int       { return &n }
func intPtr64(n int64) *int64 { return &n }
