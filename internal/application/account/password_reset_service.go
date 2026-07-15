// Package account orquestra fluxos de conta do usuário no MeuFin.
// O auth (retech-auth-api) é dono das credenciais; aqui coordenamos o fluxo
// e enviamos as notificações (e-mail via useSend).
package account

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/retechfin/retechfin-api/internal/infrastructure/authclient"
	"github.com/retechfin/retechfin-api/internal/infrastructure/notification"
)

// Erros expostos ao handler.
var (
	ErrTokenInvalid = errors.New("token inválido, expirado ou já utilizado")
	ErrWeakPassword = errors.New("a nova senha deve ter no mínimo 8 caracteres")
)

const resetTTLMinutes = 60

// PasswordResetService implementa o "esqueci a senha" ponta a ponta:
// pede o token ao auth (HMAC) e envia o e-mail com o link de redefinição.
type PasswordResetService struct {
	auth     *authclient.Client
	mailer   notification.EmailSender
	adminURL string // base do admin (link do e-mail) — env ADMIN_BASE_URL
	log      *slog.Logger
}

func NewPasswordResetService(auth *authclient.Client, mailer notification.EmailSender, log *slog.Logger) *PasswordResetService {
	return &PasswordResetService{
		auth:     auth,
		mailer:   mailer,
		adminURL: strings.TrimRight(strings.TrimSpace(os.Getenv("ADMIN_BASE_URL")), "/"),
		log:      log,
	}
}

// Request dispara o fluxo para o e-mail informado. Quando o e-mail não
// pertence a um usuário, retorna nil MESMO ASSIM — a API pública nunca revela
// se um e-mail está cadastrado (proteção contra enumeração).
func (s *PasswordResetService) Request(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil
	}

	result, err := s.auth.PasswordResetRequest(ctx, email)
	if err != nil {
		if errors.Is(err, authclient.ErrUserNotFound) {
			s.log.Info("🔐 reset de senha solicitado para e-mail não cadastrado (resposta genérica)",
				slog.String("email", email))
			return nil
		}
		return fmt.Errorf("solicitar token ao auth: %w", err)
	}

	if s.adminURL == "" {
		return fmt.Errorf("ADMIN_BASE_URL não configurada — impossível montar o link de redefinição")
	}
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.adminURL, url.QueryEscape(result.Token))

	msg, err := notification.PasswordResetEmail(result.UserName, resetURL, resetTTLMinutes)
	if err != nil {
		return err
	}
	msg.To = result.Email
	msg.ToName = result.UserName

	if err := s.mailer.Send(ctx, msg); err != nil {
		return fmt.Errorf("enviar e-mail de redefinição: %w", err)
	}

	s.log.Info("📧 e-mail de redefinição de senha enviado", slog.String("email", result.Email))
	return nil
}

// Confirm consome o token no auth e define a nova senha.
func (s *PasswordResetService) Confirm(ctx context.Context, token, newPassword string) error {
	if len(strings.TrimSpace(newPassword)) < 8 {
		return ErrWeakPassword
	}
	err := s.auth.PasswordResetConfirm(ctx, token, newPassword)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, authclient.ErrWeakPassword):
		return ErrWeakPassword
	case errors.Is(err, authclient.ErrTokenInvalid):
		return ErrTokenInvalid
	default:
		return err
	}
}
