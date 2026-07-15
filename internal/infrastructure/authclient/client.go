// Package authclient é o cliente HTTP interno do retech-auth-api.
// Autentica via HMAC-SHA256(body+timestamp) com o mesmo BOOTSTRAP_SECRET do
// authsync (headers X-Signature / X-Timestamp).
//
// Envs:
//
//	AUTH_API_BASE_URL      base do auth (ex.: https://retechauth-api-production.up.railway.app)
//	AUTH_BOOTSTRAP_SECRET  secret compartilhado do HMAC (o mesmo já usado pelo authsync)
package authclient

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Erros sentinela traduzidos das respostas do auth.
var (
	ErrUserNotFound = errors.New("authclient: usuário não encontrado")
	ErrTokenInvalid = errors.New("authclient: token inválido, expirado ou já utilizado")
	ErrWeakPassword = errors.New("authclient: senha fraca")
)

// Config do cliente interno do auth.
type Config struct {
	BaseURL string
	Secret  string
}

// ConfigFromEnv lê a configuração das variáveis de ambiente.
func ConfigFromEnv() Config {
	return Config{
		BaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("AUTH_API_BASE_URL")), "/"),
		Secret:  strings.TrimSpace(os.Getenv("AUTH_BOOTSTRAP_SECRET")),
	}
}

// Complete informa se o cliente está configurado.
func (c Config) Complete() bool { return c.BaseURL != "" && c.Secret != "" }

// Client fala com os endpoints internos (HMAC) do retech-auth-api.
type Client struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: 15 * time.Second}}
}

// ResetRequestResult espelha a resposta de POST /v1/password-reset/request.
type ResetRequestResult struct {
	Token     string    `json:"token"`
	UserName  string    `json:"user_name"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

// post assina o body com HMAC e executa a chamada.
func (c *Client) post(ctx context.Context, path string, payload any) (*http.Response, error) {
	if !c.cfg.Complete() {
		return nil, fmt.Errorf("authclient: configure AUTH_API_BASE_URL e AUTH_BOOTSTRAP_SECRET")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("authclient: serializar payload: %w", err)
	}

	timestamp := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(c.cfg.Secret))
	mac.Write(body)
	mac.Write([]byte(fmt.Sprintf("%d", timestamp)))
	signature := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("authclient: montar requisição: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", timestamp))

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("authclient: chamada ao auth: %w", err)
	}
	return resp, nil
}

// PasswordResetRequest pede um token de redefinição ao auth.
// Retorna ErrUserNotFound quando o e-mail não pertence a um usuário ativo.
func (c *Client) PasswordResetRequest(ctx context.Context, email string) (*ResetRequestResult, error) {
	resp, err := c.post(ctx, "/v1/password-reset/request", map[string]string{"email": email})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var out ResetRequestResult
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, fmt.Errorf("authclient: decodificar resposta: %w", err)
		}
		return &out, nil
	case http.StatusNotFound:
		return nil, ErrUserNotFound
	default:
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("authclient: auth respondeu %d: %s", resp.StatusCode, string(raw))
	}
}

// PasswordResetConfirm consome o token e define a nova senha no auth.
func (c *Client) PasswordResetConfirm(ctx context.Context, token, newPassword string) error {
	resp, err := c.post(ctx, "/v1/password-reset/confirm", map[string]string{
		"token":        token,
		"new_password": newPassword,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusBadRequest:
		return ErrWeakPassword
	case http.StatusUnprocessableEntity:
		return ErrTokenInvalid
	default:
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("authclient: auth respondeu %d: %s", resp.StatusCode, string(raw))
	}
}
