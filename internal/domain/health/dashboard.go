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
	Counts(ctx context.Context, workspaceID uuid.UUID) (DashboardCounts, error)
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
