// Comando de seed do catálogo base de Saúde Familiar.
// Uso: go run ./cmd/seed   (carrega .env; usa as variáveis DB_*).
package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
	apph "github.com/retechfin/retechfin-api/internal/application/health"
	"github.com/retechfin/retechfin-api/internal/infrastructure/persistence"
	gormlogger "gorm.io/gorm/logger"
)

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func mustEnv(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		log.Fatalf("❌ variável obrigatória ausente: %s", key)
	}
	return v
}

func databaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		url.QueryEscape(mustEnv("DB_USER")),
		url.QueryEscape(mustEnv("DB_PASSWORD")),
		mustEnv("DB_HOST"),
		env("DB_PORT", "5432"),
		mustEnv("DB_NAME"),
		env("DB_SSLMODE", "disable"),
	)
}

func main() {
	_ = godotenv.Load(".env")

	dsn := databaseURL()
	migrationsPath := env("MIGRATIONS_PATH", "./migrations")

	log.Println("🔄 Aplicando migrations...")
	if err := persistence.RunMigrations(dsn, migrationsPath); err != nil {
		log.Fatalf("❌ falha nas migrations: %v", err)
	}

	db, err := persistence.OpenPostgres(dsn, gormlogger.Warn)
	if err != nil {
		log.Fatalf("❌ falha ao conectar: %v", err)
	}

	repo := persistence.NewHealthMarkerRepository(db)
	svc := apph.NewMarkerService(repo)

	log.Println("🌱 Populando catálogo base de marcadores...")
	n, err := svc.SeedSystem(context.Background())
	if err != nil {
		log.Fatalf("❌ falha no seed: %v", err)
	}
	log.Printf("✅ Seed concluído. Novos marcadores inseridos: %d (idempotente).", n)
}
