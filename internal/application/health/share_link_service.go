package health

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
)

// ShareLinkService gerencia links públicos temporários de saúde
// (docs/health-share-links.md). MVP: scope member_panels — painéis evolutivos
// de um membro, somente leitura.
type ShareLinkService struct {
	repo    dom.ShareLinkRepository
	members dom.FamilyMemberRepository
	// publicBaseURL monta a URL da página pública (admin) — env ADMIN_BASE_URL.
	publicBaseURL string
}

func NewShareLinkService(repo dom.ShareLinkRepository, members dom.FamilyMemberRepository) *ShareLinkService {
	return &ShareLinkService{
		repo:          repo,
		members:       members,
		publicBaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("ADMIN_BASE_URL")), "/"),
	}
}

type CreateShareLinkInput struct {
	WorkspaceID    uuid.UUID
	FamilyMemberID uuid.UUID
	Title          *string
	ExpiresInHours int
	CreatedBy      *uuid.UUID
}

type ShareLinkWithURL struct {
	Link dom.ShareLink
	URL  string
}

// URL monta o endereço público de um link.
func (s *ShareLinkService) URL(l *dom.ShareLink) string {
	return fmt.Sprintf("%s/compartilhado/%s", s.publicBaseURL, l.Token)
}

func (s *ShareLinkService) Create(ctx context.Context, in CreateShareLinkInput) (*ShareLinkWithURL, error) {
	if s.publicBaseURL == "" {
		return nil, fmt.Errorf("ADMIN_BASE_URL não configurada — impossível montar a URL pública")
	}
	if !dom.AllowedShareTTLHours[in.ExpiresInHours] {
		return nil, &dom.ValidationError{Msg: "expires_in_hours inválido (1, 6, 12, 24, 48, 72 ou 168)"}
	}
	// Membro precisa existir e pertencer ao workspace.
	member, err := s.members.GetByID(ctx, in.WorkspaceID, in.FamilyMemberID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	active, err := s.repo.CountActive(ctx, in.WorkspaceID, now)
	if err != nil {
		return nil, err
	}
	if active >= dom.MaxActiveShareLinks {
		return nil, &dom.ValidationError{Msg: fmt.Sprintf("limite de %d links ativos atingido — revogue links antigos antes de criar novos", dom.MaxActiveShareLinks)}
	}

	token, err := generateShareToken()
	if err != nil {
		return nil, err
	}
	var title *string
	if in.Title != nil {
		if t := strings.TrimSpace(*in.Title); t != "" {
			title = &t
		}
	}
	link := &dom.ShareLink{
		ID:             uuid.New(),
		WorkspaceID:    in.WorkspaceID,
		Token:          token,
		Scope:          dom.ScopeMemberPanels,
		FamilyMemberID: member.ID,
		Title:          title,
		ExpiresAt:      now.Add(time.Duration(in.ExpiresInHours) * time.Hour),
		CreatedBy:      in.CreatedBy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.Create(ctx, link); err != nil {
		return nil, err
	}
	return &ShareLinkWithURL{Link: *link, URL: s.URL(link)}, nil
}

func (s *ShareLinkService) List(ctx context.Context, workspaceID uuid.UUID) ([]ShareLinkWithURL, error) {
	links, err := s.repo.List(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]ShareLinkWithURL, len(links))
	for i := range links {
		out[i] = ShareLinkWithURL{Link: links[i], URL: s.URL(&links[i])}
	}
	return out, nil
}

// Revoke invalida o link imediatamente.
func (s *ShareLinkService) Revoke(ctx context.Context, workspaceID, id uuid.UUID) error {
	link, err := s.repo.GetByID(ctx, workspaceID, id)
	if err != nil {
		return err
	}
	if link.RevokedAt != nil {
		return nil // idempotente
	}
	now := time.Now().UTC()
	link.RevokedAt = &now
	link.UpdatedAt = now
	return s.repo.Update(ctx, link)
}

// Resolve valida um token público e devolve o link + o membro. Registra a
// visualização. Inexistente, expirado e revogado retornam o MESMO erro
// (anti-enumeração, spec).
func (s *ShareLinkService) Resolve(ctx context.Context, token string) (*dom.ShareLink, *dom.FamilyMember, error) {
	token = strings.TrimSpace(token)
	if len(token) != 64 {
		return nil, nil, &dom.ShareLinkGoneError{}
	}
	link, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return nil, nil, &dom.ShareLinkGoneError{}
	}
	now := time.Now().UTC()
	if !link.Active(now) {
		return nil, nil, &dom.ShareLinkGoneError{}
	}
	member, err := s.members.GetByID(ctx, link.WorkspaceID, link.FamilyMemberID)
	if err != nil {
		return nil, nil, &dom.ShareLinkGoneError{}
	}
	// Falha ao registrar view não bloqueia o acesso do médico.
	_ = s.repo.RegisterView(ctx, link.ID, now)
	return link, member, nil
}

// generateShareToken: 32 bytes de crypto/rand → 64 chars hex (256 bits).
func generateShareToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("gerar token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
