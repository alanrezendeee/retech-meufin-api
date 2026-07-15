// Package notification centraliza o envio de notificações do MeuFin.
// Hoje o único canal é e-mail (useSend self-hosted); a factory New devolve a
// implementação configurada ou um sender desabilitado (no-op com erro claro),
// espelhando o padrão do pacote storage.
//
// Envs:
//
//	USESEND_BASE_URL  URL base do useSend (ex.: https://usesend.suaempresa.com)
//	USESEND_API_KEY   API key gerada no painel do useSend (Settings → API Keys)
//	MAIL_FROM_EMAIL   remetente (ex.: nao-responda@meufin.app) — domínio verificado no useSend
//	MAIL_FROM_NAME    nome amigável do remetente (ex.: MeuFin)
package notification

import (
	"context"
	"os"
	"strings"
)

// Email é a mensagem de e-mail canônica do MeuFin.
type Email struct {
	To      string
	ToName  string
	Subject string
	HTML    string
	Text    string // fallback plain-text (clientes sem HTML)
}

// EmailSender envia e-mails. Implementações: useSend (produção) e disabled.
type EmailSender interface {
	Enabled() bool
	Send(ctx context.Context, msg Email) error
}

// Config agrupa a configuração do canal de e-mail.
type Config struct {
	BaseURL   string
	APIKey    string
	FromEmail string
	FromName  string
}

// ConfigFromEnv lê a configuração das variáveis de ambiente.
func ConfigFromEnv() Config {
	return Config{
		BaseURL:   strings.TrimRight(strings.TrimSpace(os.Getenv("USESEND_BASE_URL")), "/"),
		APIKey:    strings.TrimSpace(os.Getenv("USESEND_API_KEY")),
		FromEmail: strings.TrimSpace(os.Getenv("MAIL_FROM_EMAIL")),
		FromName:  strings.TrimSpace(os.Getenv("MAIL_FROM_NAME")),
	}
}

// Complete informa se há configuração suficiente para enviar e-mails.
func (c Config) Complete() bool {
	return c.BaseURL != "" && c.APIKey != "" && c.FromEmail != ""
}

// New é a factory do canal de e-mail: useSend quando configurado, senão disabled.
func New(cfg Config) EmailSender {
	if !cfg.Complete() {
		return DisabledSender{}
	}
	return NewUseSendSender(cfg)
}
