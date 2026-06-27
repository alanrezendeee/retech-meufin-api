package configs

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// Config agrega todas as variáveis obrigatórias. Qualquer ausência impede o startup.
type Config struct {
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	DBSSLMode       string
	AppPort         string
	AppEnv          string
	LogLevel        string
	MigrationsPath  string
	AuthJWKSURL     string
	CORSOrigins     []string
	AppApplicationID string
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
		DBHost:         os.Getenv("DB_HOST"),
		DBPort:         os.Getenv("DB_PORT"),
		DBUser:         os.Getenv("DB_USER"),
		DBPassword:     os.Getenv("DB_PASSWORD"),
		DBName:         os.Getenv("DB_NAME"),
		DBSSLMode:      os.Getenv("DB_SSLMODE"),
		AppPort:        os.Getenv("APP_PORT"),
		AppEnv:         os.Getenv("APP_ENV"),
		LogLevel:       os.Getenv("LOG_LEVEL"),
		MigrationsPath: os.Getenv("MIGRATIONS_PATH"),
		AuthJWKSURL:    os.Getenv("AUTH_JWKS_URL"),
		CORSOrigins:    splitAndTrim(os.Getenv("CORS_ALLOWED_ORIGINS")),
		// Opcional: se setado, valida que o token pertence a esta aplicação.
		AppApplicationID: strings.TrimSpace(os.Getenv("APP_APPLICATION_ID")),
	}

	if cfg.AppEnv != "development" && cfg.AppEnv != "production" && cfg.AppEnv != "test" {
		return nil, fmt.Errorf("APP_ENV inválido: use development, production ou test")
	}

	log.Println("✅ Todas as configurações carregadas com sucesso!")

	return cfg, nil
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
