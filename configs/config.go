package configs

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config agrega todas as variáveis obrigatórias. Qualquer ausência impede o startup.
type Config struct {
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBSSLMode        string
	AppPort          string
	AppEnv           string
	LogLevel         string
	MigrationsPath   string
	AuthJWKSURL      string
	CORSOrigins      []string
	AppApplicationID string
	// Gateway de sessão (cookie HttpOnly) — docs/auth-session-gateway.md
	SessionEncryptionKey string        // SESSION_ENCRYPTION_KEY: base64 de 32 bytes; vazio = gateway desabilitado (só Bearer)
	SessionCookieName    string        // SESSION_COOKIE_NAME (padrão meufin_session)
	SessionCookieSecure  bool          // SESSION_COOKIE_SECURE (padrão true; false só fora de produção)
	SessionCookieDomain  string        // SESSION_COOKIE_DOMAIN (opcional; vazio = host da API)
	SessionTTL           time.Duration // SESSION_TTL_HOURS (padrão 168h = validade do refresh token do auth)
	AppApplicationCode   string        // APP_APPLICATION_CODE: application_code no auth (padrão meufin)
	// Integrações opcionais
	FipeBaseURL string // padrão: https://parallelum.com.br/fipe/api/v1
	RedisURL    string // ex: redis://localhost:6379 (opcional; sem Redis = sem cache FIPE)
}

func Load() (*Config, error) {
	log.Println("🔍 Carregando configurações obrigatórias...")

	keys := []string{
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"APP_PORT", "APP_ENV", "LOG_LEVEL", "MIGRATIONS_PATH",
		"AUTH_JWKS_URL", "CORS_ALLOWED_ORIGINS",
	}
	missing := make([]string, 0)
	for _, k := range keys {
		if strings.TrimSpace(os.Getenv(k)) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("variáveis de ambiente obrigatórias ausentes: %s", strings.Join(missing, ", "))
	}

	cfg := &Config{
		DBHost:           os.Getenv("DB_HOST"),
		DBPort:           os.Getenv("DB_PORT"),
		DBUser:           os.Getenv("DB_USER"),
		DBPassword:       os.Getenv("DB_PASSWORD"),
		DBName:           os.Getenv("DB_NAME"),
		DBSSLMode:        os.Getenv("DB_SSLMODE"),
		AppPort:          os.Getenv("APP_PORT"),
		AppEnv:           os.Getenv("APP_ENV"),
		LogLevel:         os.Getenv("LOG_LEVEL"),
		MigrationsPath:   os.Getenv("MIGRATIONS_PATH"),
		AuthJWKSURL:      os.Getenv("AUTH_JWKS_URL"),
		CORSOrigins:      splitAndTrim(os.Getenv("CORS_ALLOWED_ORIGINS")),
		AppApplicationID: strings.TrimSpace(os.Getenv("APP_APPLICATION_ID")),
		FipeBaseURL:      strings.TrimSpace(os.Getenv("FIPE_BASE_URL")),
		RedisURL:         strings.TrimSpace(os.Getenv("REDIS_URL")),
	}

	if cfg.AppEnv != "development" && cfg.AppEnv != "production" && cfg.AppEnv != "test" {
		return nil, fmt.Errorf("APP_ENV inválido: use development, production ou test")
	}

	if err := loadSession(cfg); err != nil {
		return nil, err
	}

	log.Println("✅ Todas as configurações carregadas com sucesso!")

	return cfg, nil
}

// loadSession lê e valida as variáveis do gateway de sessão.
func loadSession(cfg *Config) error {
	cfg.SessionEncryptionKey = strings.TrimSpace(os.Getenv("SESSION_ENCRYPTION_KEY"))
	cfg.SessionCookieName = strings.TrimSpace(os.Getenv("SESSION_COOKIE_NAME"))
	if cfg.SessionCookieName == "" {
		cfg.SessionCookieName = "meufin_session"
	}
	cfg.SessionCookieDomain = strings.TrimSpace(os.Getenv("SESSION_COOKIE_DOMAIN"))
	cfg.AppApplicationCode = strings.TrimSpace(os.Getenv("APP_APPLICATION_CODE"))
	if cfg.AppApplicationCode == "" {
		cfg.AppApplicationCode = "meufin"
	}

	cfg.SessionCookieSecure = true
	if v := strings.TrimSpace(os.Getenv("SESSION_COOKIE_SECURE")); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("SESSION_COOKIE_SECURE inválido: %q", v)
		}
		cfg.SessionCookieSecure = b
	}

	cfg.SessionTTL = 168 * time.Hour
	if v := strings.TrimSpace(os.Getenv("SESSION_TTL_HOURS")); v != "" {
		h, err := strconv.Atoi(v)
		if err != nil || h < 1 {
			return fmt.Errorf("SESSION_TTL_HOURS inválido: %q (inteiro >= 1)", v)
		}
		cfg.SessionTTL = time.Duration(h) * time.Hour
	}

	if cfg.AppEnv == "production" {
		if cfg.SessionEncryptionKey == "" {
			return fmt.Errorf("SESSION_ENCRYPTION_KEY é obrigatória em produção (openssl rand -base64 32)")
		}
		if !cfg.SessionCookieSecure {
			return fmt.Errorf("SESSION_COOKIE_SECURE=false não é permitido em produção")
		}
	}
	return nil
}

// SessionEnabled informa se o gateway de sessão (cookie) está ativo.
func (c *Config) SessionEnabled() bool { return c.SessionEncryptionKey != "" }

// splitAndTrim quebra uma lista separada por vírgula, ignorando itens vazios.
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}
