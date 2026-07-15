package notification

import (
	"context"
	"fmt"
)

// DisabledSender é o fallback quando o useSend não está configurado.
// Send falha com mensagem clara em vez de sumir com o e-mail silenciosamente.
type DisabledSender struct{}

func (DisabledSender) Enabled() bool { return false }

func (DisabledSender) Send(_ context.Context, _ Email) error {
	return fmt.Errorf("notification: envio de e-mail desabilitado — configure USESEND_BASE_URL, USESEND_API_KEY e MAIL_FROM_EMAIL")
}
