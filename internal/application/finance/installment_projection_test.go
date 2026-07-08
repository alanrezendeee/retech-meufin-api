package finance

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

func seedInstallmentChild(repo *fakeEntryRepo, ws, parent, card uuid.UUID, desc string, number, total int, cents int64, due time.Time) {
	e := &dom.FinancialEntry{
		ID: uuid.New(), WorkspaceID: ws, Kind: dom.KindDebit,
		Status: dom.StatusRealizada, AmountCents: cents, DueDate: due,
		Description: desc, ParentID: &parent, CardID: &card,
		InstallmentNumber: &number, InstallmentTotal: &total,
	}
	repo.entries[e.ID] = e
}

func TestInstallmentsProjection(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})
	ws := uuid.New()
	card := uuid.New()
	junho := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	julho := time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC)

	// Mesma compra em duas faturas: 3/10 (junho) e 4/10 (julho) →
	// um grupo só, estado 4/10, restam 6.
	seedInstallmentChild(repo, ws, uuid.New(), card, "LOJA MOVEIS PARC 03/10", 3, 10, 42890, junho)
	seedInstallmentChild(repo, ws, uuid.New(), card, "LOJA MOVEIS PARC 04/10", 4, 10, 42890, julho)

	// Parcelamento quitado (10/10) → fora da projeção.
	seedInstallmentChild(repo, ws, uuid.New(), card, "NOTEBOOK PARC 10/10", 10, 10, 15000, julho)

	// Outro parcelamento ativo: 1/3 em julho → restam 2.
	seedInstallmentChild(repo, ws, uuid.New(), card, "CELULAR 1/3", 1, 3, 50000, julho)

	proj, err := svc.InstallmentsProjection(context.Background(), ws)
	if err != nil {
		t.Fatalf("InstallmentsProjection: %v", err)
	}

	if len(proj.Groups) != 2 {
		t.Fatalf("quer 2 grupos ativos, veio %d: %+v", len(proj.Groups), proj.Groups)
	}

	// Total restante: 6×428,90 + 2×500,00 = 2.573,40 + 1.000,00
	want := int64(6*42890 + 2*50000)
	if proj.RemainingTotalCents != want {
		t.Fatalf("restante total: quer %d, veio %d", want, proj.RemainingTotalCents)
	}

	// Grupo maior primeiro (móveis: 2.573,40 > celular: 1.000,00)
	if proj.Groups[0].Description != "LOJA MOVEIS PARC" || proj.Groups[0].RemainingCount != 6 || proj.Groups[0].LastKnownNumber != 4 {
		t.Fatalf("grupo móveis errado: %+v", proj.Groups[0])
	}

	// Agosto/2026: parcela dos móveis (5/10) + celular (2/3) = 42890+50000
	var ago *MonthlyCommitment
	for i := range proj.Monthly {
		if proj.Monthly[i].Month == "2026-08" {
			ago = &proj.Monthly[i]
		}
	}
	if ago == nil || ago.TotalCents != 42890+50000 || ago.Count != 2 {
		t.Fatalf("agosto: quer %d em 2 parcelas, veio %+v", 42890+50000, ago)
	}

	// Último mês projetado: móveis termina em 2027-01 (6 meses após julho)
	last := proj.Monthly[len(proj.Monthly)-1]
	if last.Month != "2027-01" || last.TotalCents != 42890 {
		t.Fatalf("último mês: quer 2027-01 com 42890, veio %+v", last)
	}
}

func TestInstallmentsProjectionIncludesStandaloneExpenses(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})
	ws := uuid.New()

	// Financiamento 6x de R$ 1.898,77 criado pelo fluxo "Parcelado":
	// parcelas reais, 2 primeiras realizadas.
	total := 6
	occs, err := svc.Create(context.Background(), CreateEntryInput{
		WorkspaceID:       ws,
		Kind:              "debit",
		AmountCents:       189877,
		DueDate:           time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
		Description:       "Refinanciamento Equinox",
		InstallmentsTotal: &total,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	repo.entries[occs[0].ID].Status = dom.StatusRealizada
	repo.entries[occs[1].ID].Status = dom.StatusRealizada

	proj, err := svc.InstallmentsProjection(context.Background(), ws)
	if err != nil {
		t.Fatalf("InstallmentsProjection: %v", err)
	}
	if len(proj.Groups) != 1 {
		t.Fatalf("esperava 1 grupo, veio %d", len(proj.Groups))
	}
	g := proj.Groups[0]
	if g.Source != SourceExpense {
		t.Fatalf("source esperado expense, veio %s", g.Source)
	}
	if g.RemainingCount != 4 || g.LastKnownNumber != 2 || g.InstallmentTotal != 6 {
		t.Fatalf("estado do grupo errado: %+v", g)
	}
	if g.RemainingCents != 4*189877 {
		t.Fatalf("restante: quer %d, veio %d", 4*189877, g.RemainingCents)
	}
	// Termina na última parcela real: 15/10/2026 (maio + 5 meses).
	if g.EndsAt.Format("2006-01") != "2026-10" {
		t.Fatalf("ends_at: quer 2026-10, veio %v", g.EndsAt)
	}
	// Mensal usa os vencimentos reais das previstas (julho..outubro).
	if len(proj.Monthly) != 4 || proj.Monthly[0].Month != "2026-07" || proj.Monthly[3].Month != "2026-10" {
		t.Fatalf("mensal errado: %+v", proj.Monthly)
	}
	if proj.Monthly[0].TotalCents != 189877 || proj.Monthly[0].Count != 1 {
		t.Fatalf("julho errado: %+v", proj.Monthly[0])
	}
}
