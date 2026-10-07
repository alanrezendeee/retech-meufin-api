package configs

import (
	"strings"
	"testing"
)

func TestLoadSessionHostPrefixExigeSecureSemDomain(t *testing.T) {
	cases := []struct {
		name, cookie, secure, domain string
		wantErr                      bool
	}{
		{"__Host- ok", "__Host-meufin_session", "true", "", false},
		{"__Host- sem Secure", "__Host-meufin_session", "false", "", true},
		{"__Host- com Domain", "__Host-meufin_session", "true", ".meufin.app", true},
		{"sem prefixo, dev http", "meufin_session", "false", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SESSION_COOKIE_NAME", tc.cookie)
			t.Setenv("SESSION_COOKIE_SECURE", tc.secure)
			t.Setenv("SESSION_COOKIE_DOMAIN", tc.domain)
			t.Setenv("SESSION_ENCRYPTION_KEY", "x")
			cfg := &Config{AppEnv: "development"}
			err := loadSession(cfg)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "__Host-") {
				t.Fatalf("erro não explica o prefixo: %v", err)
			}
		})
	}
}

func TestLoadSessionExigeChaveEmQualquerAmbienteESecureEmProducao(t *testing.T) {
	t.Setenv("SESSION_COOKIE_NAME", "")
	t.Setenv("SESSION_COOKIE_DOMAIN", "")
	t.Setenv("SESSION_ENCRYPTION_KEY", "")
	t.Setenv("SESSION_COOKIE_SECURE", "false")
	if err := loadSession(&Config{AppEnv: "development"}); err == nil {
		t.Fatal("sem SESSION_ENCRYPTION_KEY não há como autenticar; devia falhar mesmo em dev")
	}
	t.Setenv("SESSION_COOKIE_SECURE", "true")
	if err := loadSession(&Config{AppEnv: "production"}); err == nil {
		t.Fatal("produção sem SESSION_ENCRYPTION_KEY devia falhar")
	}
	t.Setenv("SESSION_ENCRYPTION_KEY", "x")
	t.Setenv("SESSION_COOKIE_SECURE", "false")
	if err := loadSession(&Config{AppEnv: "production"}); err == nil {
		t.Fatal("produção com cookie inseguro devia falhar")
	}
	t.Setenv("SESSION_COOKIE_SECURE", "true")
	cfg := &Config{AppEnv: "production"}
	if err := loadSession(cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.SessionTTL.Hours() != 12 || cfg.SessionCookieName != "meufin_session" || cfg.AppApplicationCode != "meufin" {
		t.Fatalf("defaults: ttl=%s cookie=%s app=%s", cfg.SessionTTL, cfg.SessionCookieName, cfg.AppApplicationCode)
	}
}
