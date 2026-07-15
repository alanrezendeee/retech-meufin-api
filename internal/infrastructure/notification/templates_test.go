package notification

import (
	"strings"
	"testing"
)

func TestPasswordResetEmail(t *testing.T) {
	msg, err := PasswordResetEmail("Fernanda Oliveira", "https://app.meufin.app/reset-password?token=abc123", 60)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if msg.Subject == "" {
		t.Fatal("subject vazio")
	}
	for _, want := range []string{"Fernanda Oliveira", "https://app.meufin.app/reset-password?token=abc123", "60 minutos", "#00e676"} {
		if !strings.Contains(msg.HTML, want) {
			t.Errorf("HTML não contém %q", want)
		}
	}
	if !strings.Contains(msg.Text, "https://app.meufin.app/reset-password?token=abc123") {
		t.Error("texto plain não contém o link")
	}
}

func TestPasswordResetEmailEscapesHTML(t *testing.T) {
	msg, err := PasswordResetEmail("<script>alert(1)</script>", "https://app.meufin.app/reset", 60)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if strings.Contains(msg.HTML, "<script>alert(1)</script>") {
		t.Error("nome do usuário não foi escapado no HTML")
	}
}

func TestFactoryFallsBackToDisabled(t *testing.T) {
	s := New(Config{})
	if s.Enabled() {
		t.Error("factory sem config deveria devolver sender desabilitado")
	}
}

func TestPasswordResetEmailLogo(t *testing.T) {
	t.Setenv("MAIL_LOGO_URL", "https://admin.meufin.app/logo-email.png")
	msg, err := PasswordResetEmail("Alan", "https://x/reset", 60)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !strings.Contains(msg.HTML, `src="https://admin.meufin.app/logo-email.png"`) {
		t.Error("HTML não contém a imagem da logo quando MAIL_LOGO_URL está setada")
	}

	t.Setenv("MAIL_LOGO_URL", "")
	msg2, _ := PasswordResetEmail("Alan", "https://x/reset", 60)
	if strings.Contains(msg2.HTML, "<img") {
		t.Error("HTML não deveria ter <img> sem MAIL_LOGO_URL")
	}
}
