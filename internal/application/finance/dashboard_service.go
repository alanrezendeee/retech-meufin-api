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
