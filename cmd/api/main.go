package main

import (
	"errors"
	"fmt"
	stdlog "log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/retechfin/retechfin-api/configs"
	appb "github.com/retechfin/retechfin-api/internal/application/budget"
	appl "github.com/retechfin/retechfin-api/internal/application/ledger"
	"github.com/retechfin/retechfin-api/internal/infrastructure/persistence"
	httprouter "github.com/retechfin/retechfin-api/internal/interfaces/http"
	"github.com/retechfin/retechfin-api/pkg/logger"
	gormlogger "gorm.io/gorm/logger"
)

// loadDotEnvFiles carrega o primeiro .env encontrado (raiz do repo ou um nível acima se o cwd for cmd/api).
func loadDotEnvFiles() error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("working directory: %w", err)
	}
	candidates := []string{
		filepath.Join(wd, ".env"),
		filepath.Join(wd, "..", ".env"),
		filepath.Join(wd, "..", "..", ".env"),
	}
	for _, p := range candidates {
		err := godotenv.Load(p)
		if err == nil {
			return nil
		}
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		return fmt.Errorf("carregar %q: %w", p, err)
	}
	return nil
}

func httpPublicBase(port string) string {
	p := strings.TrimSpace(port)
	if p == "" {
		return "http://0.0.0.0:8002"
	}
	return "http://0.0.0.0:" + p
}

func listenDisplay(port string) (portLabel, listenAddr string) {
	p := strings.TrimSpace(port)
	if p == "" {
		p = "8002"
	}
	return p, "0.0.0.0:" + p
}

func main() {
	if err := loadDotEnvFiles(); err != nil {
		stdlog.Fatalf("❌ Erro ao carregar arquivo .env: %v", err)
	}

	cfg, err := configs.Load()
	if err != nil {
		stdlog.Fatalf("❌ Erro ao carregar configuração: %v", err)
	}

	log := logger.New(cfg.AppEnv, cfg.LogLevel)
	slog.SetDefault(log)

	// Gin em release evita o dump enorme de rotas no startup; use LOG_LEVEL=debug para ver rotas em dev.
	if strings.EqualFold(cfg.LogLevel, "debug") && cfg.AppEnv != "production" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	dbLog := gormlogger.Silent
	if strings.EqualFold(cfg.LogLevel, "debug") {
		dbLog = gormlogger.Info
	}

	db, err := persistence.OpenPostgres(cfg.DatabaseURL(), dbLog)
	if err != nil {
		log.Error("❌ Erro ao conectar ao banco de dados", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("✅ Conectado ao banco de dados com sucesso!")

	log.Info("🔄 Verificando migrations...")
	if err := persistence.RunMigrations(cfg.DatabaseURL(), cfg.MigrationsPath); err != nil {
		log.Error("❌ Falha ao aplicar migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("✅ Migrations verificadas e aplicadas!")

	accRepo := persistence.NewAccountRepository(db)
	catRepo := persistence.NewCategoryRepository(db)
	txRepo := persistence.NewTransactionRepository(db)
	budRepo := persistence.NewBudgetRepository(db)

	accSvc := appl.NewAccountService(accRepo)
	catSvc := appl.NewCategoryService(catRepo)
	txSvc := appl.NewTransactionService(txRepo, accRepo, catRepo)
	budSvc := appb.NewService(budRepo, catRepo, txRepo)

	r := httprouter.NewRouter(httprouter.RouterDeps{
		Log:                log,
		DB:                 db,
		Env:                cfg.AppEnv,
		AccountService:     accSvc,
		CategoryService:    catSvc,
		TransactionService: txSvc,
		BudgetService:      budSvc,
	})

	addr := ":" + cfg.AppPort
	base := httpPublicBase(cfg.AppPort)
	portLabel, listenAddr := listenDisplay(cfg.AppPort)

	log.Info(fmt.Sprintf("🚀 Servidor iniciado na porta %s (escutando em %s)", portLabel, listenAddr))
	log.Info(fmt.Sprintf("📝 Ambiente: %s", cfg.AppEnv))
	log.Info(fmt.Sprintf("📝 Nível de log: %s", cfg.LogLevel))
	log.Info(fmt.Sprintf("🔗 Health check: %s/health", base))
	log.Info(fmt.Sprintf("📚 API base: %s/api/v1", base))

	if err := r.Run(addr); err != nil {
		log.Error("❌ Erro no servidor HTTP", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
