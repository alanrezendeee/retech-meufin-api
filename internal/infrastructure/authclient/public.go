package authclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	domsess "github.com/retechfin/retechfin-api/internal/domain/session"
)

// Endpoints PÚBLICOS do retech-auth-api (sem HMAC): autenticação por senha,
// renovação de tokens e perfil do usuário logado. Usados pelo gateway de
// sessão — o browser nunca fala com o auth diretamente.
//
// Só precisam de AUTH_API_BASE_URL (o secret HMAC é exclusivo dos endpoints
// internos em client.go).

const maxAuthResponseBytes = 1 << 20

// PublicAuthenticator implementa domsess.Authenticator sobre o auth central.
type PublicAuthenticator struct {
	c               *Client
	applicationCode string
}

// NewPublicAuthenticator cria o autenticador para a aplicação informada
// (application_code do POST /v1/authenticate, ex.: "meufin").
func NewPublicAuthenticator(c *Client, applicationCode string) *PublicAuthenticator {
	return &PublicAuthenticator{c: c, applicationCode: applicationCode}
}

var _ domsess.Authenticator = (*PublicAuthenticator)(nil)

// Configured informa se a base do auth está definida.
func (a *PublicAuthenticator) Configured() bool { return a.c.cfg.BaseURL != "" }

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"user"`
}

func (a *PublicAuthenticator) postPublic(ctx context.Context, path string, payload any, out any) (int, error) {
	if a.c.cfg.BaseURL == "" {
		return 0, fmt.Errorf("authclient: configure AUTH_API_BASE_URL")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("authclient: serializar payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.c.cfg.BaseURL+path, strings.NewReader(string(body)))
	if err != nil {
		return 0, fmt.Errorf("authclient: montar requisição: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := a.c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", domsess.ErrAuthUnavailable, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxAuthResponseBytes))
	if err != nil {
		return resp.StatusCode, fmt.Errorf("authclient: ler resposta: %w", err)
	}
	if resp.StatusCode == http.StatusOK && out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("authclient: decodificar resposta: %w", err)
		}
	}
	return resp.StatusCode, nil
}

func toTokens(tr tokenResponse, now time.Time) (domsess.Tokens, error) {
	if tr.AccessToken == "" || tr.RefreshToken == "" {
		return domsess.Tokens{}, fmt.Errorf("authclient: resposta do auth sem tokens")
	}
	exp := tr.ExpiresIn
	if exp <= 0 {
		exp = 300 // defensivo: força refresh cedo se o auth não informar
	}
	return domsess.Tokens{
		Access:          tr.AccessToken,
		Refresh:         tr.RefreshToken,
		AccessExpiresAt: now.Add(time.Duration(exp) * time.Second),
	}, nil
}

// Authenticate valida e-mail/senha no auth (POST /v1/authenticate).
func (a *PublicAuthenticator) Authenticate(ctx context.Context, email, password string) (domsess.Authenticated, error) {
	var tr tokenResponse
	status, err := a.postPublic(ctx, "/v1/authenticate", map[string]string{
		"email":            email,
		"password":         password,
		"application_code": a.applicationCode,
	}, &tr)
	if err != nil {
		return domsess.Authenticated{}, err
	}
	switch status {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusBadRequest:
		return domsess.Authenticated{}, domsess.ErrInvalidCredentials
	case http.StatusForbidden:
		return domsess.Authenticated{}, domsess.ErrUserInactive
	default:
		return domsess.Authenticated{}, fmt.Errorf("%w: auth respondeu %d", domsess.ErrAuthUnavailable, status)
	}
	tokens, err := toTokens(tr, time.Now())
	if err != nil {
		return domsess.Authenticated{}, err
	}
	uid, err := uuid.Parse(strings.TrimSpace(tr.User.ID))
	if err != nil {
		return domsess.Authenticated{}, fmt.Errorf("authclient: user.id inválido na resposta do auth")
	}
	return domsess.Authenticated{
		User:   domsess.UserInfo{ID: uid, Email: tr.User.Email, Name: tr.User.Name},
		Tokens: tokens,
	}, nil
}

// Refresh renova o par de tokens (POST /v1/refresh).
func (a *PublicAuthenticator) Refresh(ctx context.Context, refreshToken string) (domsess.Tokens, error) {
	var tr tokenResponse
	status, err := a.postPublic(ctx, "/v1/refresh", map[string]string{"refresh_token": refreshToken}, &tr)
	if err != nil {
		return domsess.Tokens{}, err
	}
	switch status {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusBadRequest:
		return domsess.Tokens{}, domsess.ErrRefreshRejected
	default:
		return domsess.Tokens{}, fmt.Errorf("%w: auth respondeu %d", domsess.ErrAuthUnavailable, status)
	}
	return toTokens(tr, time.Now())
}

// Me devolve o corpo bruto de GET /v1/me para o access token informado
// (usuário, roles, permissions, abilities CASL). O gateway repassa ao admin
// sem reinterpretar — o contrato é do auth.
func (a *PublicAuthenticator) Me(ctx context.Context, accessToken string) (status int, body []byte, err error) {
	if a.c.cfg.BaseURL == "" {
		return 0, nil, fmt.Errorf("authclient: configure AUTH_API_BASE_URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.c.cfg.BaseURL+"/v1/me", nil)
	if err != nil {
		return 0, nil, fmt.Errorf("authclient: montar requisição: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := a.c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", domsess.ErrAuthUnavailable, err)
	}
	defer resp.Body.Close()
	body, err = io.ReadAll(io.LimitReader(resp.Body, maxAuthResponseBytes))
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("authclient: ler resposta: %w", err)
	}
	return resp.StatusCode, body, nil
}

// BaseURL expõe a base do auth (usada pelo proxy IAM).
func (c *Client) BaseURL() string { return c.cfg.BaseURL }
