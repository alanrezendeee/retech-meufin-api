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
