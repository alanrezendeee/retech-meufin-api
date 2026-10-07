// Package session é o domínio do gateway de autenticação do browser.
//
// Padrão "token handler" (BFF): o browser recebe apenas um token aleatório em
// cookie HttpOnly; os access/refresh tokens emitidos pelo retech-auth-api
// ficam no servidor, cifrados em repouso. Nenhum JWT chega ao JavaScript, o
// que elimina o roubo de sessão por XSS via localStorage. O middleware da API
// resolve cookie → sessão → access token e segue validando o JWT via JWKS
// exatamente como antes.
package session

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Erros do domínio.
var (
	ErrInvalidCredentials = errors.New("session: credenciais inválidas")
	ErrUserInactive       = errors.New("session: usuário ou aplicação inativos")
	ErrNotFound           = errors.New("session: sessão não encontrada, expirada ou revogada")
	ErrRefreshRejected    = errors.New("session: refresh recusado pelo auth")
	ErrAuthUnavailable    = errors.New("session: auth indisponível")
)

// Tokens emitidos pelo auth central.
type Tokens struct {
	Access          string
	Refresh         string
	AccessExpiresAt time.Time
}

// UserInfo é o usuário autenticado, conforme devolvido pelo auth.
type UserInfo struct {
	ID    uuid.UUID
	Email string
	Name  string
}

// Authenticated é o resultado de uma autenticação bem-sucedida.
type Authenticated struct {
	User   UserInfo
	Tokens Tokens
}

// Session é a sessão do browser.
type Session struct {
	ID         string // sha256(hex) do token do cookie — nunca o token em si
	User       UserInfo
	TenantID   *uuid.UUID
	Tokens     Tokens
	ExpiresAt  time.Time
	CreatedAt  time.Time
	LastSeenAt time.Time
	RevokedAt  *time.Time
	IP         string
	UserAgent  string
}

// Active informa se a sessão ainda vale em `now`.
func (s Session) Active(now time.Time) bool {
	return s.RevokedAt == nil && s.ExpiresAt.After(now)
}

// Authenticator fala com o provedor de identidade (retech-auth-api).
type Authenticator interface {
	Authenticate(ctx context.Context, email, password string) (Authenticated, error)
	Refresh(ctx context.Context, refreshToken string) (Tokens, error)
}

// Store persiste sessões. Implementações cifram os tokens em repouso.
type Store interface {
	Create(ctx context.Context, s Session) error
	// Get devolve ErrNotFound se ausente. Não filtra expiração/revogação —
	// isso é decisão do serviço (que também limpa o cookie).
	Get(ctx context.Context, id string) (Session, error)
	Touch(ctx context.Context, id string, at time.Time) error
	UpdateTokens(ctx context.Context, id string, t Tokens) error
	Revoke(ctx context.Context, id string, at time.Time) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID, at time.Time) (int64, error)
	PurgeExpired(ctx context.Context, before time.Time) (int64, error)
}
