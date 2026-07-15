package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// UseSendSender envia e-mails via useSend (https://usesend.com — self-hosted).
// API: POST {base}/api/v1/emails com Authorization: Bearer <api key>.
type UseSendSender struct {
	cfg    Config
	client *http.Client
}

// NewUseSendSender cria o sender com timeout curto — envio de e-mail nunca
// deve segurar uma request do usuário por muito tempo.
func NewUseSendSender(cfg Config) *UseSendSender {
	return &UseSendSender{
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *UseSendSender) Enabled() bool { return true }

type useSendPayload struct {
	To      string `json:"to"`
	From    string `json:"from"`
	Subject string `json:"subject"`
	HTML    string `json:"html,omitempty"`
	Text    string `json:"text,omitempty"`
}

// Send envia o e-mail. Erros incluem o corpo da resposta do useSend para
// facilitar diagnóstico (domínio não verificado, key inválida etc.).
func (s *UseSendSender) Send(ctx context.Context, msg Email) error {
	if msg.To == "" {
		return fmt.Errorf("notification: destinatário vazio")
	}

	from := s.cfg.FromEmail
	if s.cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", s.cfg.FromName, s.cfg.FromEmail)
	}

	body, err := json.Marshal(useSendPayload{
		To:      msg.To,
		From:    from,
		Subject: msg.Subject,
		HTML:    msg.HTML,
		Text:    msg.Text,
	})
	if err != nil {
		return fmt.Errorf("notification: serializar payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.BaseURL+"/api/v1/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("notification: montar requisição: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("notification: chamada ao useSend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("notification: useSend respondeu %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}
