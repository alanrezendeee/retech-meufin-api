// Package session orquestra o gateway de autenticação do browser sobre o
// retech-authkit/session: login, logout e resolução do cookie em sessão com
// access token válido e verificado.
package session

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/theretechlabs/retech-authkit/jwtverify"
	"github.com/theretechlabs/retech-authkit/session"

	dom "github.com/retechfin/retechfin-api/internal/domain/session"
)

// ClientMeta são metadados do browser registrados na sessão (auditoria).
type ClientMeta = session.ClientMeta

// Resolved é a sessão autenticada desta requisição (sessão + claims do JWT).
type Resolved = session.Resolved

// Service é o caso de uso do gateway.
type Service struct {
	kit *session.Service
	ttl time.Duration
}

// NewService cria o serviço. verifier nil só em testes sem JWT.
func NewService(auth dom.Authenticator, store dom.Store, verifier *jwtverify.Verifier, ttl time.Duration, log *slog.Logger) *Service {
	return &Service{kit: session.New(auth, store, verifier, session.Config{TTL: ttl, Log: log}), ttl: ttl}
}

// TTL devolve a validade absoluta da sessão (também é o MaxAge do cookie).
func (s *Service) TTL() time.Duration { return s.ttl }

// Login autentica no auth central e cria a sessão. O workspace vem do
// tenant_id do JWT. Devolve o token do cookie (nunca persistido em claro).
func (s *Service) Login(ctx context.Context, email, password string, meta ClientMeta) (string, dom.Session, error) {
	return s.kit.Login(ctx, email, password, meta, nil)
}

// Logout revoga a sessão local e o refresh token no auth. Idempotente.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	return s.kit.Logout(ctx, rawToken)
}

// Resolve transforma o token do cookie em sessão ativa com access token
// verificado (renovado no auth quando necessário).
func (s *Service) Resolve(ctx context.Context, rawToken string) (Resolved, error) {
	return s.kit.Resolve(ctx, rawToken)
}

// RevokeAllForUser encerra todas as sessões do usuário (troca de senha, bloqueio).
func (s *Service) RevokeAllForUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.kit.RevokeAllForUser(ctx, userID)
}

// PurgeExpired remove sessões expiradas/revogadas há mais de `grace`.
func (s *Service) PurgeExpired(ctx context.Context, grace time.Duration) (int64, error) {
	return s.kit.PurgeExpired(ctx, grace)
}
