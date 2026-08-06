package finance

import (
	"context"
	"testing"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// fakeDashboardRepo implementa só o necessário para testar a montagem da DFC.
type fakeDashboardRepo struct {
	opening int64
	months  []dom.CashFlowMonth
}

func (f *fakeDashboardRepo) Summary(_ context.Context, _ uuid.UUID, _, _ int, _ *uuid.UUID) (*dom.DashboardSummary, error) {
	return nil, nil
}
func (f *fakeDashboardRepo) MonthlySeries(_ context.Context, _ uuid.UUID, _ int, _ *uuid.UUID) ([]dom.MonthlyPoint, error) {
	return nil, nil
}
func (f *fakeDashboardRepo) CategoryEntries(_ context.Context, _ uuid.UUID, _ string, _, _ int, _ *uuid.UUID) ([]dom.FinancialEntry, error) {
	return nil, nil
}
func (f *fakeDashboardRepo) CashFlowRaw(_ context.Context, _ uuid.UUID, _ int, _ *uuid.UUID) (int64, []dom.CashFlowMonth, error) {
	return f.opening, f.months, nil
}

// A identidade contábil da DFC: saldo inicial + líquidos = saldo final, e o
// acumulado carrega de mês em mês — inclusive pelos meses sem movimento.
func TestCashFlowAcumulaSaldo(t *testing.T) {
	repo := &fakeDashboardRepo{
		opening: 100_000, // R$ 1.000 acumulados antes do ano
		months: []dom.CashFlowMonth{
			{Month: 1, InflowCents: 500_000, OutflowCents: 300_000}, // +2000
			{Month: 3, InflowCents: 100_000, OutflowCents: 250_000}, // -1500
		},
	}
	svc := NewFinanceDashboardService(repo)

	cf, err := svc.CashFlow(context.Background(), uuid.New(), 2026, nil)
	if err != nil {
		t.Fatalf("CashFlow: %v", err)
	}
	if len(cf.Months) != 12 {
		t.Fatalf("meses = %d, quer 12 (zeros preenchidos)", len(cf.Months))
	}

	jan := cf.Months[0]
	if jan.NetCents != 200_000 || jan.BalanceCents != 300_000 {
		t.Errorf("jan: net=%d saldo=%d, quer 200000/300000", jan.NetCents, jan.BalanceCents)
	}
	// Fevereiro sem movimento: líquido zero, saldo carregado de janeiro.
	fev := cf.Months[1]
	if fev.NetCents != 0 || fev.BalanceCents != 300_000 {
		t.Errorf("fev: net=%d saldo=%d, quer 0/300000", fev.NetCents, fev.BalanceCents)
	}
	mar := cf.Months[2]
	if mar.NetCents != -150_000 || mar.BalanceCents != 150_000 {
		t.Errorf("mar: net=%d saldo=%d, quer -150000/150000", mar.NetCents, mar.BalanceCents)
	}
	// Dezembro repete o saldo de março (nada depois) e fecha o ano.
	if cf.Months[11].BalanceCents != 150_000 || cf.ClosingBalanceCents != 150_000 {
		t.Errorf("fechamento = %d/%d, quer 150000", cf.Months[11].BalanceCents, cf.ClosingBalanceCents)
	}
	// Identidade: abertura + soma dos líquidos = fechamento.
	var net int64
	for _, m := range cf.Months {
		net += m.NetCents
	}
	if cf.OpeningBalanceCents+net != cf.ClosingBalanceCents {
		t.Errorf("identidade violada: %d + %d != %d", cf.OpeningBalanceCents, net, cf.ClosingBalanceCents)
	}
}
