package configs

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/theretechlabs/retech-authkit/cookie"
)

// Config agrega todas as variáveis obrigatórias. Qualquer ausência impede o startup.
type Config struct {
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string
	AppPort        string
	AppEnv         string
	LogLevel       string
	MigrationsPath string
	CORSOrigins    []string
	// Auth central (retech-auth-api) — padrão de envs compartilhado com o CashFlowfy.
	AuthAPIBaseURL      string // AUTH_API_BASE_URL (obrigatória)
	AuthJWKSURL         string // AUTH_JWKS_URL (opcional; padrão AUTH_API_BASE_URL/.well-known/jwks.json)
	AuthBootstrapSecret string // AUTH_BOOTSTRAP_SECRET (obrigatória: authsync e password-reset)
	AuthIssuer          string // AUTH_ISSUER (opcional; padrão retech-auth-api)
	// Gateway de sessão (cookie HttpOnly) — docs/auth-session-gateway.md
	SessionEncryptionKey string        // SESSION_ENCRYPTION_KEY: base64 de 32 bytes (obrigatória)
	SessionCookieName    string        // SESSION_COOKIE_NAME (padrão meufin_session)
	SessionCookieSecure  bool          // SESSION_COOKIE_SECURE (padrão true; false só fora de produção)
	SessionCookieDomain  string        // SESSION_COOKIE_DOMAIN (opcional; vazio = host da API)
	SessionTTL           time.Duration // SESSION_TTL (duração; padrão 12h, igual ao CashFlowfy)
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
		"AUTH_API_BASE_URL", "AUTH_BOOTSTRAP_SECRET", "CORS_ALLOWED_ORIGINS",
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
		DBHost:              os.Getenv("DB_HOST"),
		DBPort:              os.Getenv("DB_PORT"),
		DBUser:              os.Getenv("DB_USER"),
		DBPassword:          os.Getenv("DB_PASSWORD"),
		DBName:              os.Getenv("DB_NAME"),
		DBSSLMode:           os.Getenv("DB_SSLMODE"),
		AppPort:             os.Getenv("APP_PORT"),
		AppEnv:              os.Getenv("APP_ENV"),
		LogLevel:            os.Getenv("LOG_LEVEL"),
		MigrationsPath:      os.Getenv("MIGRATIONS_PATH"),
		CORSOrigins:         splitAndTrim(os.Getenv("CORS_ALLOWED_ORIGINS")),
		AuthAPIBaseURL:      strings.TrimRight(strings.TrimSpace(os.Getenv("AUTH_API_BASE_URL")), "/"),
		AuthJWKSURL:         strings.TrimSpace(os.Getenv("AUTH_JWKS_URL")),
		AuthBootstrapSecret: strings.TrimSpace(os.Getenv("AUTH_BOOTSTRAP_SECRET")),
		AuthIssuer:          strings.TrimSpace(os.Getenv("AUTH_ISSUER")),
		FipeBaseURL:         strings.TrimSpace(os.Getenv("FIPE_BASE_URL")),
		RedisURL:            strings.TrimSpace(os.Getenv("REDIS_URL")),
	}

	if cfg.AppEnv != "development" && cfg.AppEnv != "production" && cfg.AppEnv != "test" {
		return nil, fmt.Errorf("APP_ENV inválido: use development, production ou test")
	}
	if cfg.AuthJWKSURL == "" {
		cfg.AuthJWKSURL = cfg.AuthAPIBaseURL + "/.well-known/jwks.json"
	}
	if cfg.AuthIssuer == "" {
		cfg.AuthIssuer = "retech-auth-api"
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

	cfg.SessionTTL = 12 * time.Hour
	if v := strings.TrimSpace(os.Getenv("SESSION_TTL")); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("SESSION_TTL inválido: %q (duração, ex.: 12h)", v)
		}
		cfg.SessionTTL = d
	}

	// Cookie: __Host- só com Secure e sem Domain; Secure obrigatório em produção;
	// TTL mínimo — mesmas regras do CashFlowfy (retech-authkit/cookie).
	if err := cfg.SessionCookie().Validate(cfg.AppEnv == "production"); err != nil {
		return err
	}

	// A única forma de autenticar é o cookie de sessão: sem chave, não há API.
	if cfg.SessionEncryptionKey == "" {
		return fmt.Errorf("SESSION_ENCRYPTION_KEY é obrigatória (openssl rand -base64 32)")
	}
	return nil
}

// SessionCookie devolve a configuração do cookie de sessão.
func (c *Config) SessionCookie() cookie.Config {
	return cookie.Config{Name: c.SessionCookieName, Secure: c.SessionCookieSecure, Domain: c.SessionCookieDomain, TTL: c.SessionTTL}
}

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
