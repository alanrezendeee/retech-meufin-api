package finance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// createMonthlySeries cria uma despesa recorrente mensal e retorna as
// ocorrências ordenadas por due_date.
func createMonthlySeries(t *testing.T, svc *FinancialEntryService, ws uuid.UUID, due time.Time, amount int64) []dom.FinancialEntry {
	t.Helper()
	occs, err := svc.Create(context.Background(), CreateEntryInput{
		WorkspaceID: ws,
		Kind:        "debit",
		AmountCents: amount,
		DueDate:     due,
		Description: "Aluguel",
		Recurrence:  "monthly",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(occs) != dom.RollingMonths {
		t.Fatalf("esperava %d ocorrências, veio %d", dom.RollingMonths, len(occs))
	}
	return occs
}

func TestUpdateApplyToFuturePropagatesDayAmount(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})
	ws := uuid.New()
	occs := createMonthlySeries(t, svc, ws, time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC), 100000)

	// Terceira ocorrência realizada — não pode ser tocada.
	occs[2].Status = dom.StatusRealizada
	repo.entries[occs[2].ID].Status = dom.StatusRealizada

	// Edita a primeira: dia 10 → 15, valor 1000 → 1200, aplicar às próximas.
	first := occs[0]
	_, err := svc.Update(context.Background(), UpdateEntryInput{
		WorkspaceID: ws, ID: first.ID, Kind: "debit",
		AmountCents:   120000,
		DueDate:       time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		Description:   first.Description,
		ApplyToFuture: true,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	for i, occ := range occs {
		got := repo.entries[occ.ID]
		switch {
		case i == 0:
			if got.DueDate.Day() != 15 || got.AmountCents != 120000 {
				t.Fatalf("ocorrência editada não atualizada: %v %d", got.DueDate, got.AmountCents)
			}
		case i == 2: // realizada: intocada
			if got.DueDate.Day() != 10 || got.AmountCents != 100000 {
				t.Fatalf("ocorrência realizada foi alterada: %v %d", got.DueDate, got.AmountCents)
			}
		default: // previstas futuras: dia e valor propagados, mês preservado
			if got.DueDate.Day() != 15 {
				t.Fatalf("ocorrência %d: dia esperado 15, veio %d", i, got.DueDate.Day())
			}
			if got.DueDate.Month() != occ.DueDate.Month() {
				t.Fatalf("ocorrência %d: mês mudou de %v para %v", i, occ.DueDate.Month(), got.DueDate.Month())
			}
			if got.AmountCents != 120000 {
				t.Fatalf("ocorrência %d: valor esperado 120000, veio %d", i, got.AmountCents)
			}
		}
	}
}

func TestUpdateApplyToFutureClampsEndOfMonth(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})
	ws := uuid.New()
	occs := createMonthlySeries(t, svc, ws, time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC), 50000)

	first := occs[0]
	_, err := svc.Update(context.Background(), UpdateEntryInput{
		WorkspaceID: ws, ID: first.ID, Kind: "debit",
		AmountCents:   first.AmountCents,
		DueDate:       time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
		Description:   first.Description,
		ApplyToFuture: true,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Fevereiro/2026 não tem dia 31 — clamp para 28.
	feb := repo.entries[occs[1].ID]
	if feb.DueDate.Month() != time.February || feb.DueDate.Day() != 28 {
		t.Fatalf("clamp de fevereiro falhou: %v", feb.DueDate)
	}
	// Março tem 31.
	mar := repo.entries[occs[2].ID]
	if mar.DueDate.Day() != 31 {
		t.Fatalf("março deveria ser dia 31: %v", mar.DueDate)
	}
}

func TestUpdateApplyToFutureOnlyChangedFields(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})
	ws := uuid.New()
	occs := createMonthlySeries(t, svc, ws, time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC), 80000)

	// Nota individual numa ocorrência futura — não pode ser sobrescrita.
	note := "negociado desconto neste mês"
	repo.entries[occs[4].ID].Notes = &note

	first := occs[0]
	_, err := svc.Update(context.Background(), UpdateEntryInput{
		WorkspaceID: ws, ID: first.ID, Kind: "debit",
		AmountCents:   first.AmountCents, // sem mudança
		DueDate:       first.DueDate,     // sem mudança
		Description:   "Aluguel reajustado",
		ApplyToFuture: true,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	got := repo.entries[occs[4].ID]
	if got.Description != "Aluguel reajustado" {
		t.Fatalf("descrição não propagada: %q", got.Description)
	}
	if got.AmountCents != 80000 || got.DueDate.Day() != 10 {
		t.Fatalf("campos sem mudança foram alterados: %d %v", got.AmountCents, got.DueDate)
	}
	if got.Notes == nil || *got.Notes != note {
		t.Fatalf("nota individual da ocorrência foi perdida")
	}
}

func TestUpdateApplyToFutureRejectsNonSeries(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})
	e := seedEntry(repo, dom.StatusPrevista)

	_, err := svc.Update(context.Background(), UpdateEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, Kind: "debit",
		AmountCents: e.AmountCents, DueDate: e.DueDate, Description: e.Description,
		ApplyToFuture: true,
	})
	var vErr *dom.ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("esperava ValidationError para lançamento fora de série, veio %v", err)
	}
}

func TestUpdateApplyToFutureRejectsInstallments(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})
	ws := uuid.New()
	total := 6
	occs, err := svc.Create(context.Background(), CreateEntryInput{
		WorkspaceID:       ws,
		Kind:              "debit",
		AmountCents:       30000,
		DueDate:           time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC),
		Description:       "Notebook 6x",
		InstallmentsTotal: &total,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = svc.Update(context.Background(), UpdateEntryInput{
		WorkspaceID: ws, ID: occs[0].ID, Kind: "debit",
		AmountCents: occs[0].AmountCents, DueDate: occs[0].DueDate, Description: occs[0].Description,
		ApplyToFuture: true,
	})
	var vErr *dom.ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("esperava ValidationError para parcelamento, veio %v", err)
	}
}
