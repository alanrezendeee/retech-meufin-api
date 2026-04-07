package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/retechfin/retechfin-api/configs"
	appb "github.com/retechfin/retechfin-api/internal/application/budget"
	appl "github.com/retechfin/retechfin-api/internal/application/ledger"
	"github.com/retechfin/retechfin-api/internal/infrastructure/persistence"
	httprouter "github.com/retechfin/retechfin-api/internal/interfaces/http"
	"github.com/retechfin/retechfin-api/pkg/logger"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	cfg, err := configs.Load()
	if err != nil {
		slog.Error("config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv, cfg.LogLevel)

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	dbLog := gormlogger.Silent
	if strings.EqualFold(cfg.LogLevel, "debug") {
		dbLog = gormlogger.Info
	}

	db, err := persistence.OpenPostgres(cfg.DatabaseURL(), dbLog)
	if err != nil {
		log.Error("postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err := persistence.RunMigrations(cfg.DatabaseURL(), cfg.MigrationsPath); err != nil {
		log.Error("migrate", slog.String("error", err.Error()))
		os.Exit(1)
	}

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
		AccountService:     accSvc,
		CategoryService:    catSvc,
		TransactionService: txSvc,
		BudgetService:      budSvc,
	})

	addr := ":" + cfg.AppPort
	log.Info("servidor iniciando", slog.String("addr", addr), slog.String("env", cfg.AppEnv))
	if err := r.Run(addr); err != nil {
		log.Error("gin", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
