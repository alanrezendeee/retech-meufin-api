// Package session orquestra o gateway de autenticação do browser: login,
// logout e resolução do cookie em access token válido (renovando no auth
// central quando necessário).
package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/session"
)

const (
	// refreshSkew: renova o access token quando faltar menos que isto para
	// expirar, para a requisição em curso não chegar ao JWKS com token vencido.
	refreshSkew = 30 * time.Second
	// touchInterval limita escritas de last_seen_at.
	touchInterval = time.Minute
)

// ClientMeta são metadados do browser registrados na sessão (auditoria).
type ClientMeta struct {
	IP        string
	UserAgent string
}

// Clock permite controlar o tempo nos testes.
type Clock func() time.Time

// Service é o caso de uso do gateway.
type Service struct {
	auth  dom.Authenticator
	store dom.Store
	ttl   time.Duration
	now   Clock
	log   *slog.Logger
}

// NewService cria o serviço. ttl é a validade absoluta da sessão.
func NewService(auth dom.Authenticator, store dom.Store, ttl time.Duration, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{auth: auth, store: store, ttl: ttl, now: time.Now, log: log}
}

// WithClock troca o relógio (testes).
func (s *Service) WithClock(c Clock) *Service { s.now = c; return s }

// TTL devolve a validade absoluta da sessão (também é o MaxAge do cookie).
func (s *Service) TTL() time.Duration { return s.ttl }

// Login autentica no auth central e cria a sessão. Devolve o token do cookie
// (nunca persistido em claro) e a sessão criada.
func (s *Service) Login(ctx context.Context, email, password string, meta ClientMeta) (string, dom.Session, error) {
	authn, err := s.auth.Authenticate(ctx, email, password)
	if err != nil {
		return "", dom.Session{}, err
	}
	raw, id, err := dom.NewToken()
	if err != nil {
		return "", dom.Session{}, err
	}
	now := s.now()
	sess := dom.Session{
		ID:         id,
		User:       authn.User,
		Tokens:     authn.Tokens,
		ExpiresAt:  now.Add(s.ttl),
		CreatedAt:  now,
		LastSeenAt: now,
		IP:         meta.IP,
		UserAgent:  meta.UserAgent,
	}
	if err := s.store.Create(ctx, sess); err != nil {
		return "", dom.Session{}, fmt.Errorf("session: criar sessão: %w", err)
	}
	s.log.InfoContext(ctx, "session.login", slog.String("user_id", authn.User.ID.String()))
	return raw, sess, nil
}

// Logout revoga a sessão do cookie. Idempotente: sem cookie ou sessão
// inexistente não é erro.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	if err := s.store.Revoke(ctx, dom.HashToken(rawToken), s.now()); err != nil {
		return fmt.Errorf("session: revogar: %w", err)
	}
	return nil
}

// Resolve transforma o token do cookie em sessão ativa com access token
// válido. Renova no auth central quando o access token está para expirar;
// se o auth recusar o refresh, a sessão é revogada. Erros de domínio
// (ErrNotFound, ErrRefreshRejected) significam "limpe o cookie, 401".
func (s *Service) Resolve(ctx context.Context, rawToken string) (dom.Session, error) {
	if rawToken == "" {
		return dom.Session{}, dom.ErrNotFound
	}
	id := dom.HashToken(rawToken)
	sess, err := s.store.Get(ctx, id)
	if err != nil {
		if errors.Is(err, dom.ErrNotFound) {
			return dom.Session{}, dom.ErrNotFound
		}
		return dom.Session{}, fmt.Errorf("session: buscar sessão: %w", err)
	}
	now := s.now()
	if !sess.Active(now) {
		return dom.Session{}, dom.ErrNotFound
	}

	if !sess.Tokens.AccessExpiresAt.After(now.Add(refreshSkew)) {
		tokens, err := s.auth.Refresh(ctx, sess.Tokens.Refresh)
		if err != nil {
			if errors.Is(err, dom.ErrRefreshRejected) {
				_ = s.store.Revoke(ctx, id, now)
				s.log.WarnContext(ctx, "session.refresh_rejected", slog.String("user_id", sess.User.ID.String()))
				return dom.Session{}, dom.ErrRefreshRejected
			}
			// Auth fora do ar: não derruba a sessão; a requisição falha com 503.
			return dom.Session{}, err
		}
		if err := s.store.UpdateTokens(ctx, id, tokens); err != nil {
			return dom.Session{}, fmt.Errorf("session: atualizar tokens: %w", err)
		}
		sess.Tokens = tokens
	}

	if now.Sub(sess.LastSeenAt) >= touchInterval {
		if err := s.store.Touch(ctx, id, now); err != nil {
			s.log.WarnContext(ctx, "session.touch_failed", slog.String("error", err.Error()))
		}
	}
	return sess, nil
}

// RevokeAllForUser encerra todas as sessões do usuário (troca de senha,
// bloqueio). Devolve quantas foram revogadas.
func (s *Service) RevokeAllForUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.store.RevokeAllForUser(ctx, userID, s.now())
}

// PurgeExpired remove sessões expiradas/revogadas há mais de `grace`.
func (s *Service) PurgeExpired(ctx context.Context, grace time.Duration) (int64, error) {
	return s.store.PurgeExpired(ctx, s.now().Add(-grace))
}
