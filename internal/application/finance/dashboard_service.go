package finance

import (
	"context"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// FinanceDashboardService responde as 4 perguntas da dashboard:
// como estou este mês / o que ainda vem / pra onde foi / quanto do futuro
// já está comprometido.
type FinanceDashboardService struct {
	repo dom.FinanceDashboardRepository
}

func NewFinanceDashboardService(repo dom.FinanceDashboardRepository) *FinanceDashboardService {
	return &FinanceDashboardService{repo: repo}
}

func (s *FinanceDashboardService) Summary(ctx context.Context, workspaceID uuid.UUID, year, month int, familyMemberID *uuid.UUID) (*dom.DashboardSummary, error) {
	if year < 2000 || year > 2100 {
		return nil, &dom.ValidationError{Msg: "year inválido"}
	}
	if month < 1 || month > 12 {
		return nil, &dom.ValidationError{Msg: "month inválido"}
	}
	return s.repo.Summary(ctx, workspaceID, year, month, familyMemberID)
}

// MonthlySeries retorna sempre 12 pontos (meses sem lançamentos vêm zerados).
func (s *FinanceDashboardService) MonthlySeries(ctx context.Context, workspaceID uuid.UUID, year int, familyMemberID *uuid.UUID) ([]dom.MonthlyPoint, error) {
	if year < 2000 || year > 2100 {
		return nil, &dom.ValidationError{Msg: "year inválido"}
	}
	points, err := s.repo.MonthlySeries(ctx, workspaceID, year, familyMemberID)
	if err != nil {
		return nil, err
	}
	full := make([]dom.MonthlyPoint, 12)
	for i := range full {
		full[i] = dom.MonthlyPoint{Month: i + 1}
	}
	for _, p := range points {
		if p.Month >= 1 && p.Month <= 12 {
			full[p.Month-1] = p
		}
	}
	return full, nil
}

// CategoryEntriesResult é o drill-down de uma barra de categoria.
type CategoryEntriesResult struct {
	Items []dom.FinancialEntry
	// TotalCents soma pelo MESMO critério da barra: realizada pelo valor
	// pago, prevista pelo valor do lançamento.
	TotalCents int64
}

func (s *FinanceDashboardService) CategoryEntries(ctx context.Context, workspaceID uuid.UUID, category string, year, month int, familyMemberID *uuid.UUID) (*CategoryEntriesResult, error) {
	items, err := s.repo.CategoryEntries(ctx, workspaceID, category, year, month, familyMemberID)
	if err != nil {
		return nil, err
	}
	out := &CategoryEntriesResult{Items: items}
	for i := range items {
		e := &items[i]
		if e.Status == dom.StatusRealizada && e.PaidAmountCents != nil {
			out.TotalCents += *e.PaidAmountCents
		} else {
			out.TotalCents += e.AmountCents
		}
	}
	return out, nil
}

// CashFlow monta a DFC do ano: saldo inicial + entradas pagas − saídas pagas
// = saldo final, mês a mês. Os 12 meses saem preenchidos (zeros onde não
// houve movimento) para a série do gráfico não ter buracos.
func (s *FinanceDashboardService) CashFlow(ctx context.Context, workspaceID uuid.UUID, year int, familyMemberID *uuid.UUID) (*dom.CashFlow, error) {
	opening, raw, err := s.repo.CashFlowRaw(ctx, workspaceID, year, familyMemberID)
	if err != nil {
		return nil, err
	}
	byMonth := map[int]dom.CashFlowMonth{}
	for _, m := range raw {
		byMonth[m.Month] = m
	}
	out := &dom.CashFlow{Year: year, OpeningBalanceCents: opening}
	balance := opening
	for m := 1; m <= 12; m++ {
		cm := byMonth[m]
		cm.Month = m
		cm.NetCents = cm.InflowCents - cm.OutflowCents
		balance += cm.NetCents
		cm.BalanceCents = balance
		out.Months = append(out.Months, cm)
	}
	out.ClosingBalanceCents = balance
	return out, nil
}
