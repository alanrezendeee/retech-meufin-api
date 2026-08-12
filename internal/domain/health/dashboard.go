package health

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EvolutionPoint é um ponto do histórico de um marcador (um resultado no tempo).
type EvolutionPoint struct {
	ExamDate       time.Time
	Value          *float64
	Unit           *string
	RefMin         *float64
	RefMax         *float64
	RefText        *string
	LabID          *uuid.UUID
	Interpretation *string
	// Normalized é calculado pelo serviço: posição na própria referência (−1..+1).
	Normalized *float64
}

// DashboardCounts resume o módulo para o workspace.
type DashboardCounts struct {
	FamilyMembers          int64
	ExamResults            int64
	TenantMarkers          int64
	DocumentsPendingReview int64
}

type DashboardRepository interface {
	MarkerEvolution(ctx context.Context, workspaceID, markerID uuid.UUID, familyMemberID *uuid.UUID, from, to *time.Time) ([]EvolutionPoint, error)
	// EvolutionAll retorna, em UMA query, o histórico de todos os marcadores
	// com resultado numérico (opcionalmente de um membro), keyed por marker_id
	// e ordenado por data — base da visão de painéis.
	EvolutionAll(ctx context.Context, workspaceID uuid.UUID, familyMemberID *uuid.UUID) (map[uuid.UUID][]EvolutionPoint, error)
	Counts(ctx context.Context, workspaceID uuid.UUID) (DashboardCounts, error)
}

// PanelMarker é um marcador com histórico dentro de um painel.
type PanelMarker struct {
	Marker      Marker
	DefaultMode string // "absolute" | "normalized"
	Points      []EvolutionPoint
}

// Panel agrupa os marcadores de uma categoria do catálogo: os com resultado
// (Markers, com histórico) e os que nunca tiveram resultado (Missing).
type Panel struct {
	Category string
	Markers  []PanelMarker
	Missing  []Marker
}

// NormalizeToReference mapeia o valor para −1..+1 dentro da referência.
// 0 = meio da faixa; ±1 = limites; fora da faixa passa de ±1. nil se faltar dado.
func NormalizeToReference(value, min, max *float64) *float64 {
	if value == nil || min == nil || max == nil {
		return nil
	}
	if *max <= *min {
		return nil
	}
	n := 2*(*value-*min)/(*max-*min) - 1
	return &n
}
