package configs

import (
	"fmt"
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
}

func Load() (*Config, error) {
	keys := []string{
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"APP_PORT", "APP_ENV", "LOG_LEVEL", "MIGRATIONS_PATH",
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
	}

	if cfg.AppEnv != "development" && cfg.AppEnv != "production" && cfg.AppEnv != "test" {
		return nil, fmt.Errorf("APP_ENV inválido: use development, production ou test")
	}

	return cfg, nil
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}
