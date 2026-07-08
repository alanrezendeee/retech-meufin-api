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

	// Terceira ocorrência realizada — é futura à editada: recebe o novo dia,
	// mas não o valor (fato histórico/pago).
	occs[2].Status = dom.StatusRealizada
	repo.entries[occs[2].ID].Status = dom.StatusRealizada

	// Edita a primeira: dia 10 → 15, valor 1000 → 1200, aplicar em diante.
	// Editando a 1ª ocorrência, "em diante" alcança a série toda.
	first := occs[0]
	_, stats, err := svc.Update(context.Background(), UpdateEntryInput{
		WorkspaceID: ws, ID: first.ID, Kind: "debit",
		AmountCents:   120000,
		DueDate:       time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		Description:   first.Description,
		ApplyToFuture: true,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	// Dia: todas as 11 irmãs. Valor: 10 previstas futuras (realizada fica fora).
	if stats.DueDates != 11 || stats.Fields != 10 || stats.Total != 11 {
		t.Fatalf("stats esperado {11 10 11}, veio %+v", stats)
	}

	for i, occ := range occs {
		got := repo.entries[occ.ID]
		switch {
		case i == 0:
			if got.DueDate.Day() != 15 || got.AmountCents != 120000 {
				t.Fatalf("ocorrência editada não atualizada: %v %d", got.DueDate, got.AmountCents)
			}
		case i == 2: // realizada: dia ajustado, valor histórico preservado
			if got.DueDate.Day() != 15 || got.AmountCents != 100000 {
				t.Fatalf("realizada: quer dia 15 e valor 100000, veio %v %d", got.DueDate, got.AmountCents)
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
	_, _, err := svc.Update(context.Background(), UpdateEntryInput{
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
	_, _, err := svc.Update(context.Background(), UpdateEntryInput{
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

	_, _, err := svc.Update(context.Background(), UpdateEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, Kind: "debit",
		AmountCents: e.AmountCents, DueDate: e.DueDate, Description: e.Description,
		ApplyToFuture: true,
	})
	var vErr *dom.ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("esperava ValidationError para lançamento fora de série, veio %v", err)
	}
}

func TestUpdateApplyToFuturePropagatesInstallments(t *testing.T) {
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
	if len(occs) != total {
		t.Fatalf("esperava %d parcelas, veio %d", total, len(occs))
	}

	// Segunda parcela realizada — intocada.
	repo.entries[occs[1].ID].Status = dom.StatusRealizada

	// Edita a terceira parcela: dia 5 → 20, aplicar às próximas.
	third := occs[2]
	num, tot := 3, total
	_, stats, err := svc.Update(context.Background(), UpdateEntryInput{
		WorkspaceID: ws, ID: third.ID, Kind: "debit",
		AmountCents:       third.AmountCents,
		DueDate:           time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		Description:       third.Description,
		InstallmentNumber: &num, InstallmentTotal: &tot,
		ApplyToFuture: true,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	// Dia: só as 3 parcelas seguintes (4, 5 e 6); anteriores preservadas.
	if stats.DueDates != 3 || stats.Fields != 0 || stats.Total != 3 {
		t.Fatalf("stats esperado {3 0 3}, veio %+v", stats)
	}

	for i, occ := range occs {
		got := repo.entries[occ.ID]
		if i < 2 {
			// Anteriores à editada (inclusive a realizada): vencimento histórico preservado.
			if got.DueDate.Day() != 5 {
				t.Fatalf("parcela %d anterior foi alterada: %v", i+1, got.DueDate)
			}
			continue
		}
		// Editada e seguintes: dia 20, mês preservado, numeração intacta.
		if got.DueDate.Day() != 20 {
			t.Fatalf("parcela %d: dia esperado 20, veio %d", i+1, got.DueDate.Day())
		}
		if got.DueDate.Month() != occ.DueDate.Month() {
			t.Fatalf("parcela %d: mês mudou de %v para %v", i+1, occ.DueDate.Month(), got.DueDate.Month())
		}
		if got.InstallmentNumber == nil || *got.InstallmentNumber != i+1 {
			t.Fatalf("parcela %d: installment_number perdido/alterado: %v", i+1, got.InstallmentNumber)
		}
	}
}
