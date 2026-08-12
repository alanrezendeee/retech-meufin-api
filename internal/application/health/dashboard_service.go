package health

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
)

type DashboardService struct {
	dashboard dom.DashboardRepository
	markers   dom.MarkerRepository
}

func NewDashboardService(dashboard dom.DashboardRepository, markers dom.MarkerRepository) *DashboardService {
	return &DashboardService{dashboard: dashboard, markers: markers}
}

type MarkerEvolutionResult struct {
	Marker      *dom.Marker
	DefaultMode string // "absolute" | "normalized"
	Points      []dom.EvolutionPoint
}

// MarkerEvolution retorna os pontos do histórico do marcador, com o valor
// normalizado calculado e o modo default sugerido conforme a comparabilidade.
func (s *DashboardService) MarkerEvolution(ctx context.Context, workspaceID, markerID uuid.UUID, familyMemberID *uuid.UUID, from, to *time.Time) (*MarkerEvolutionResult, error) {
	marker, err := s.markers.GetByID(ctx, workspaceID, markerID)
	if err != nil {
		return nil, err
	}
	points, err := s.dashboard.MarkerEvolution(ctx, workspaceID, markerID, familyMemberID, from, to)
	if err != nil {
		return nil, err
	}
	for i := range points {
		// Ponto sem faixa do laudo herda a faixa de curadoria do catálogo
		// (ex.: VLDL) — sem isso o modo normalizado fica vazio para o marcador.
		if points[i].RefMin == nil && points[i].RefMax == nil {
			points[i].RefMin = marker.DefaultRefMin
			points[i].RefMax = marker.DefaultRefMax
		}
		points[i].Normalized = dom.NormalizeToReference(points[i].Value, points[i].RefMin, points[i].RefMax)
	}
	mode := "absolute"
	if marker.Comparability == dom.ComparabilityMethodDependent {
		mode = "normalized"
	}
	return &MarkerEvolutionResult{Marker: marker, DefaultMode: mode, Points: points}, nil
}

func (s *DashboardService) Counts(ctx context.Context, workspaceID uuid.UUID) (dom.DashboardCounts, error) {
	return s.dashboard.Counts(ctx, workspaceID)
}
