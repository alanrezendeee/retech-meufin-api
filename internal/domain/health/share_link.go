package health

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ShareLink é um link público temporário para compartilhar dados de saúde de
// um membro (ex.: com o médico), sem login. Ver docs/health-share-links.md.
type ShareLink struct {
	ID             uuid.UUID
	WorkspaceID    uuid.UUID
	Token          string // hex de 32 bytes (256 bits) — nunca logar
	Scope          string // MVP: member_panels
	FamilyMemberID uuid.UUID
	Title          *string
	ExpiresAt      time.Time
	ViewCount      int
	LastViewedAt   *time.Time
	CreatedBy      *uuid.UUID
	RevokedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

const ScopeMemberPanels = "member_panels"

// MaxActiveShareLinks limita links ativos por workspace (decisão da spec).
const MaxActiveShareLinks = 50

// AllowedShareTTLHours são as validades aceitas (spec).
var AllowedShareTTLHours = map[int]bool{1: true, 6: true, 12: true, 24: true, 48: true, 72: true, 168: true}

// Active informa se o link ainda é utilizável.
func (l *ShareLink) Active(now time.Time) bool {
	return l.RevokedAt == nil && now.Before(l.ExpiresAt)
}

// ErrShareLinkGone: token expirado, revogado ou inexistente. Um único erro de
// propósito — não revelar qual dos casos ocorreu (spec: anti-enumeração).
type ShareLinkGoneError struct{}

func (e *ShareLinkGoneError) Error() string {
	return "este link expirou ou foi revogado — solicite um novo link ao paciente"
}

type ShareLinkRepository interface {
	Create(ctx context.Context, l *ShareLink) error
	// List retorna os links do workspace (ativos e não), mais recentes primeiro.
	List(ctx context.Context, workspaceID uuid.UUID) ([]ShareLink, error)
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*ShareLink, error)
	// GetByToken busca sem escopo de workspace (acesso público).
	GetByToken(ctx context.Context, token string) (*ShareLink, error)
	Update(ctx context.Context, l *ShareLink) error
	CountActive(ctx context.Context, workspaceID uuid.UUID, now time.Time) (int64, error)
	// RegisterView incrementa view_count e marca last_viewed_at atomicamente.
	RegisterView(ctx context.Context, id uuid.UUID, when time.Time) error
}
