package health

import (
	"context"

	"github.com/google/uuid"
)

// MarkerRepository abstrai a persistência do catálogo de marcadores.
// Todas as leituras consideram o escopo system + o tenant informado.
type MarkerRepository interface {
	Create(ctx context.Context, m *Marker) error
	Update(ctx context.Context, m *Marker) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error

	// GetByID retorna um marcador do sistema OU do próprio tenant.
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*Marker, error)

	// Search filtra por texto (nome/alias normalizado) e categoria, escopo system + tenant.
	Search(ctx context.Context, workspaceID uuid.UUID, query, category string, limit, offset int) ([]Marker, int64, error)

	// MatchExact acha por normalized_key ou alias exato, escopo system + tenant.
	// Retorna ErrNotFound quando não há correspondência.
	MatchExact(ctx context.Context, workspaceID uuid.UUID, normalized string) (*Marker, error)

	// Candidates lista marcadores do escopo (system + tenant) para ranqueamento fuzzy em memória.
	Candidates(ctx context.Context, workspaceID uuid.UUID, limit int) ([]Marker, error)

	// UpsertSystem insere marcador system de forma idempotente (não recria se normalized_key já existe).
	// Retorna true quando inseriu.
	UpsertSystem(ctx context.Context, m *Marker) (bool, error)
}
