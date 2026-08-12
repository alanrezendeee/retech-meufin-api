package health

import (
	"context"
	"sort"
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
	fallbackMin, fallbackMax := marker.DefaultRefMin, marker.DefaultRefMax
	for i := range points {
		// Ponto sem faixa do laudo herda a faixa de curadoria do catálogo
		// (ex.: VLDL) — sem isso o modo normalizado fica vazio para o marcador.
		if points[i].RefMin == nil && points[i].RefMax == nil {
			points[i].RefMin = fallbackMin
			points[i].RefMax = fallbackMax
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

// panelCategoryOrder define a ordem clínica dos painéis; categorias fora da
// lista vêm depois, em ordem alfabética.
var panelCategoryOrder = []string{
	"hematologia", "bioquimica", "lipidico", "hepatico", "renal",
	"eletrolitos", "hormonios", "vitaminas", "inflamacao", "coagulacao",
	"sorologia", "urina",
}

// Panels monta a visão de painéis: todos os marcadores do catálogo agrupados
// por categoria, separando os que têm histórico (com pontos, curadoria e
// normalização aplicadas) dos que nunca tiveram resultado.
func (s *DashboardService) Panels(ctx context.Context, workspaceID uuid.UUID, familyMemberID *uuid.UUID) ([]dom.Panel, error) {
	history, err := s.dashboard.EvolutionAll(ctx, workspaceID, familyMemberID)
	if err != nil {
		return nil, err
	}
	catalog, err := s.markers.Candidates(ctx, workspaceID, 5000)
	if err != nil {
		return nil, err
	}

	byCategory := map[string]*dom.Panel{}
	for i := range catalog {
		m := catalog[i]
		if !m.Active {
			continue
		}
		cat := m.Category
		if cat == "" {
			cat = "outros"
		}
		p, ok := byCategory[cat]
		if !ok {
			p = &dom.Panel{Category: cat}
			byCategory[cat] = p
		}
		points, has := history[m.ID]
		if !has {
			p.Missing = append(p.Missing, m)
			continue
		}
		// Mesmos fallbacks da evolução unitária: ponto sem faixa herda a
		// curadoria do marcador; normalizado −1..+1 na própria referência.
		for j := range points {
			if points[j].RefMin == nil && points[j].RefMax == nil {
				points[j].RefMin = m.DefaultRefMin
				points[j].RefMax = m.DefaultRefMax
			}
			points[j].Normalized = dom.NormalizeToReference(points[j].Value, points[j].RefMin, points[j].RefMax)
		}
		mode := "absolute"
		if m.Comparability == dom.ComparabilityMethodDependent {
			mode = "normalized"
		}
		p.Markers = append(p.Markers, dom.PanelMarker{Marker: m, DefaultMode: mode, Points: points})
	}

	// Ordena marcadores dentro do painel por nome; painéis pela ordem clínica.
	rank := map[string]int{}
	for i, c := range panelCategoryOrder {
		rank[c] = i
	}
	out := make([]dom.Panel, 0, len(byCategory))
	for _, p := range byCategory {
		sort.Slice(p.Markers, func(i, j int) bool {
			return p.Markers[i].Marker.CanonicalName < p.Markers[j].Marker.CanonicalName
		})
		sort.Slice(p.Missing, func(i, j int) bool {
			return p.Missing[i].CanonicalName < p.Missing[j].CanonicalName
		})
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool {
		ri, iok := rank[out[i].Category]
		rj, jok := rank[out[j].Category]
		switch {
		case iok && jok:
			return ri < rj
		case iok:
			return true
		case jok:
			return false
		default:
			return out[i].Category < out[j].Category
		}
	})
	return out, nil
}
