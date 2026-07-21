package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	stdlog "log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/retechfin/retechfin-api/configs"
	appacc "github.com/retechfin/retechfin-api/internal/application/account"
	appb "github.com/retechfin/retechfin-api/internal/application/budget"
	appedu "github.com/retechfin/retechfin-api/internal/application/education"
	appent "github.com/retechfin/retechfin-api/internal/application/entitlement"
	appf "github.com/retechfin/retechfin-api/internal/application/finance"
	apph "github.com/retechfin/retechfin-api/internal/application/health"
	apphs "github.com/retechfin/retechfin-api/internal/application/homesafety"
	appl "github.com/retechfin/retechfin-api/internal/application/ledger"
	appp "github.com/retechfin/retechfin-api/internal/application/patrimony"
	appv "github.com/retechfin/retechfin-api/internal/application/vehicle"
	appw "github.com/retechfin/retechfin-api/internal/application/warranty"
	domfinance "github.com/retechfin/retechfin-api/internal/domain/finance"
	"github.com/retechfin/retechfin-api/internal/infrastructure/authclient"
	"github.com/retechfin/retechfin-api/internal/infrastructure/authsync"
	"github.com/retechfin/retechfin-api/internal/infrastructure/cache"
	"github.com/retechfin/retechfin-api/internal/infrastructure/extraction"
	"github.com/retechfin/retechfin-api/internal/infrastructure/fipe"
	"github.com/retechfin/retechfin-api/internal/infrastructure/infosimples"
	"github.com/retechfin/retechfin-api/internal/infrastructure/notification"
	"github.com/retechfin/retechfin-api/internal/infrastructure/persistence"
	"github.com/retechfin/retechfin-api/internal/infrastructure/qrdecode"
	infraqueue "github.com/retechfin/retechfin-api/internal/infrastructure/queue"
	"github.com/retechfin/retechfin-api/internal/infrastructure/storage"
	httprouter "github.com/retechfin/retechfin-api/internal/interfaces/http"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
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

	log.Info("🔐 Carregando JWKS do auth...", slog.String("url", cfg.AuthJWKSURL))
	jwks, err := keyfunc.Get(cfg.AuthJWKSURL, keyfunc.Options{
		RefreshInterval:   time.Hour,
		RefreshUnknownKID: true,
		RefreshErrorHandler: func(err error) {
			log.Error("⚠️ Falha ao atualizar JWKS", slog.String("error", err.Error()))
		},
	})
	if err != nil {
		log.Error("❌ Falha ao carregar JWKS do auth", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("✅ JWKS carregado!")

	// Autorização por módulo: lê o claim perms do token (emitido pelo auth).
	permsMode := middleware.EnforcementModeFromEnv()
	log.Info(fmt.Sprintf("🛡️ Autorização por módulo: %s (PERMS_ENFORCEMENT)", permsMode))

	// Manifesto de permissions → auth (SyncManifest): telas novas viram
	// permissions no banco do auth automaticamente a cada deploy.
	syncCfg := authsync.ConfigFromEnv()
	if syncCfg.Enabled() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if summary, err := authsync.Sync(ctx, syncCfg); err != nil {
				log.Warn("⚠️ Sync do manifesto de permissions falhou", slog.String("error", err.Error()))
			} else {
				log.Info(fmt.Sprintf("✅ Manifesto de permissions sincronizado com o auth! %s", summary))
			}
		}()
	} else {
		log.Warn("⚠️ Sync do manifesto de permissions desabilitado (AUTH_SYNC_URL, AUTH_BOOTSTRAP_SECRET)")
	}

	accRepo := persistence.NewAccountRepository(db)
	catRepo := persistence.NewCategoryRepository(db)
	txRepo := persistence.NewTransactionRepository(db)
	budRepo := persistence.NewBudgetRepository(db)

	markerRepo := persistence.NewHealthMarkerRepository(db)
	familyRepo := persistence.NewHealthFamilyMemberRepository(db)
	labRepo := persistence.NewHealthLabRepository(db)
	examReqRepo := persistence.NewHealthExamRequestRepository(db)
	examResRepo := persistence.NewHealthExamResultRepository(db)
	dashboardRepo := persistence.NewHealthDashboardRepository(db)
	docRepo := persistence.NewHealthDocumentRepository(db)
	extJobRepo := persistence.NewHealthExtractionJobRepository(db)

	storageCfg := storage.ConfigFromEnv()
	objStorage := storage.New(storageCfg)
	if objStorage.Enabled() {
		// NewMinioStorage valida a conexão no boot (BucketExists) — aqui já está operacional.
		log.Info(fmt.Sprintf("✅ MinIO conectado! endpoint=%s bucket=%s ssl=%t", storageCfg.Endpoint, storageCfg.Bucket, storageCfg.UseSSL))
	} else {
		log.Warn("⚠️ MinIO não configurado — upload/download de documentos indisponível (MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY, MINIO_BUCKET_HEALTH)")
	}

	extractionCfg := extraction.ConfigFromEnv()
	extractor := extraction.New(extractionCfg)
	if extractor.Enabled() {
		model := extractionCfg.Model
		if model == "" {
			model = extraction.DefaultModel
		}
		log.Info(fmt.Sprintf("✅ Extração LLM ativa! provider=%s model=%s", extractor.Provider(), model))
	} else {
		log.Warn("⚠️ Extração LLM desabilitada — import de fatura/exames por PDF indisponível (EXTRACTION_PROVIDER, EXTRACTION_API_KEY)")
	}

	// Redis (opcional — sem Redis o cache FIPE simplesmente não existe)
	var redisCache *cache.Cache
	if cfg.RedisURL != "" {
		var rErr error
		redisCache, rErr = cache.New(cfg.RedisURL)
		if rErr != nil {
			log.Warn("⚠️ Redis não disponível — cache FIPE desabilitado", slog.String("error", rErr.Error()))
		} else {
			log.Info("✅ Redis conectado!")
		}
	} else {
		log.Warn("⚠️ REDIS_URL não configurado — cache FIPE desabilitado")
	}

	// Infosimples — consulta SEFAZ de NFC-e (cupom fiscal). Sem token, a
	// ingestão fiscal opera só pelo caminho IA (fallback).
	infosimplesClient := infosimples.New(infosimples.ConfigFromEnv())
	if infosimplesClient.Enabled() {
		log.Info("✅ Infosimples (SEFAZ/NFC-e) configurado!")
	} else {
		log.Warn("⚠️ Infosimples desabilitado — ingestão fiscal só por IA (defina INFOSIMPLES_TOKEN)")
	}

	// FIPE
	fipeClient := fipe.New(cfg.FipeBaseURL, redisCache)
	log.Info(fmt.Sprintf("✅ Cliente FIPE configurado! base_url=%s", func() string {
		if cfg.FipeBaseURL != "" {
			return cfg.FipeBaseURL
		}
		return fipe.DefaultBaseURL
	}()))

	supplierRepo := persistence.NewSupplierRepository(db)
	incomeSourceRepo := persistence.NewIncomeSourceRepository(db)
	financialEntryRepo := persistence.NewFinancialEntryRepository(db)
	creditCardRepo := persistence.NewCreditCardRepository(db)
	finDocRepo := persistence.NewFinanceDocumentRepository(db)
	finExtJobRepo := persistence.NewFinanceExtractionJobRepository(db)
	finAccountRepo := persistence.NewFinanceAccountRepository(db)
	finCategoryRepo := persistence.NewFinanceExpenseCategoryRepository(db)
	finDashRepo := persistence.NewFinanceDashboardRepository(db)
	finFiscalItemRepo := persistence.NewFiscalItemRepository(db)
	entitlementRepo := persistence.NewEntitlementRepository(db)
	memberDocRepo := persistence.NewHealthMemberDocumentRepository(db)

	accSvc := appl.NewAccountService(accRepo)
	catSvc := appl.NewCategoryService(catRepo)
	txSvc := appl.NewTransactionService(txRepo, accRepo, catRepo)
	budSvc := appb.NewService(budRepo, catRepo, txRepo)
	markerSvc := apph.NewMarkerService(markerRepo)
	familySvc := apph.NewFamilyMemberService(familyRepo, objStorage)
	labSvc := apph.NewLabService(labRepo)
	examReqSvc := apph.NewExamRequestService(examReqRepo)
	examResSvc := apph.NewExamResultService(examResRepo)
	dashboardSvc := apph.NewDashboardService(dashboardRepo, markerRepo)
	docSvc := apph.NewDocumentService(docRepo, objStorage, storageCfg.MaxUploadMB)
	extractionSvc := apph.NewExtractionService(extJobRepo, extractor)
	supplierSvc := appf.NewSupplierService(supplierRepo)
	incomeSourceSvc := appf.NewIncomeSourceService(incomeSourceRepo)
	financialEntrySvc := appf.NewFinancialEntryService(financialEntryRepo, finCategoryRepo)
	finCategorySvc := appf.NewExpenseCategoryService(finCategoryRepo)
	creditCardSvc := appf.NewCreditCardService(creditCardRepo)
	entitlementSvc := appent.NewService(entitlementRepo, redisCache)
	finDocSvc := appf.NewFinanceDocumentService(finDocRepo, objStorage, storageCfg.MaxUploadMB)
	fiscalCategorizer := appf.NewFiscalCategorizer(finCategorySvc, extraction.NewCategorizer(extractionCfg), redisCache)
	// Fila de ingestão fiscal: in-process agora (atrás da interface Publisher);
	// trocar por RabbitMQ é só um novo adaptador. QR decoder server-side lê a
	// chave da imagem quando o usuário não a informa.
	fiscalQueue := infraqueue.NewInProcess(6, 512, 3, 5*time.Second, log)
	qrDecoder := qrdecode.New()
	fiscalKeyReader := extraction.NewKeyReader(extractionCfg)
	finExtSvc := appf.NewFinanceExtractionService(finExtJobRepo, finDocRepo, extractor, infosimplesClient, entitlementSvc, redisCache, fiscalCategorizer, fiscalQueue, qrDecoder, fiscalKeyReader)

	// Worker: consome mensagens de ingestão fiscal, recarrega o conteúdo do
	// storage e processa. Registrado antes de Start.
	fiscalQueue.Register(appf.MessageTypeFiscalIngestion, func(ctx context.Context, msg infraqueue.Message) error {
		var m appf.FiscalIngestionMessage
		if err := json.Unmarshal(msg.Body, &m); err != nil {
			return nil // mensagem inválida: descarta
		}
		doc, content, err := finDocSvc.LoadContent(ctx, m.WorkspaceID, m.DocumentID)
		if err != nil {
			if errors.Is(err, domfinance.ErrNotFound) {
				return nil // documento removido: descarta
			}
			return err // transitório: reenfileira
		}
		return finExtSvc.ProcessFiscal(ctx, m.JobID, m.WorkspaceID, m.DocumentID, m.InputType, doc.MimeType, content, m.Chave)
	})
	fiscalQueue.Start(context.Background())
	finAccountSvc := appf.NewAccountService(finAccountRepo)
	finDashSvc := appf.NewFinanceDashboardService(finDashRepo)
	finFiscalSvc := appf.NewFiscalService(finFiscalItemRepo, financialEntryRepo, finDocRepo, financialEntrySvc, finCategorySvc)
	reconciliationSvc := appf.NewReconciliationService(persistence.NewReconciliationRepository(db), financialEntryRepo, finFiscalItemRepo, finDocRepo)
	memberDocSvc := apph.NewMemberDocumentService(memberDocRepo, familyRepo, objStorage, storageCfg.MaxUploadMB)

	vehicleRepo := persistence.NewVehicleRepository(db)
	vehicleSvc := appv.NewService(vehicleRepo, fipeClient, log)

	finFiscalDashRepo := persistence.NewFiscalDashboardRepository(db)
	finFiscalDashSvc := appf.NewFiscalDashboardService(finFiscalDashRepo)

	patrimonyRepo := persistence.NewPatrimonyRepository(db)
	patrimonyDocRepo := persistence.NewPropertyDocumentRepository(db)
	patrimonySvc := appp.NewService(patrimonyRepo)
	patrimonyDocSvc := appp.NewDocumentService(patrimonyDocRepo, patrimonyRepo, objStorage, storageCfg.MaxUploadMB)

	warrantyRepo := persistence.NewWarrantyRepository(db)
	warrantySvc := appw.NewService(warrantyRepo)
	warrantyDocSvc := appw.NewDocumentService(warrantyRepo, objStorage, storageCfg.MaxUploadMB)

	educationRepo := persistence.NewEducationRepository(db)
	educationSvc := appedu.NewService(educationRepo)

	homeSafetyRepo := persistence.NewHomeSafetyRepository(db)
	homeSafetySvc := apphs.NewService(homeSafetyRepo)

	healthApptRepo := persistence.NewHealthAppointmentRepository(db)
	healthApptSvc := apph.NewAppointmentService(healthApptRepo)
	healthPlanRepo := persistence.NewHealthPlanRepository(db)
	healthPlanSvc := apph.NewPlanService(healthPlanRepo)

	userProfileRepo := persistence.NewUserProfileRepository(db)
	profileSvc := appacc.NewProfileService(userProfileRepo, objStorage)

	// Notificações (e-mail via useSend) + fluxo "esqueci a senha"
	mailer := notification.New(notification.ConfigFromEnv())
	if mailer.Enabled() {
		log.Info("✅ Canal de e-mail (useSend) configurado!")
	} else {
		log.Warn("⚠️ Canal de e-mail desabilitado — configure USESEND_BASE_URL, USESEND_API_KEY e MAIL_FROM_EMAIL")
	}
	authClient := authclient.New(authclient.ConfigFromEnv())
	passwordResetSvc := appacc.NewPasswordResetService(authClient, mailer, log)

	r := httprouter.NewRouter(httprouter.RouterDeps{
		Log:                      log,
		DB:                       db,
		Env:                      cfg.AppEnv,
		JWKS:                     jwks,
		ApplicationID:            cfg.AppApplicationID,
		CORSOrigins:              cfg.CORSOrigins,
		AccountService:           accSvc,
		CategoryService:          catSvc,
		TransactionService:       txSvc,
		BudgetService:            budSvc,
		HealthMarkerService:      markerSvc,
		FamilyMemberService:      familySvc,
		LabService:               labSvc,
		ExamRequestService:       examReqSvc,
		ExamResultService:        examResSvc,
		DashboardService:         dashboardSvc,
		DocumentService:          docSvc,
		ExtractionService:        extractionSvc,
		IncomeSourceService:      incomeSourceSvc,
		FinancialEntryService:    financialEntrySvc,
		CreditCardService:        creditCardSvc,
		FinanceDocumentService:   finDocSvc,
		FinanceExtractionService: finExtSvc,
		FinanceAccountService:    finAccountSvc,
		FinanceCategoryService:   finCategorySvc,
		FinanceDashboardService:  finDashSvc,
		FinanceFiscalService:     finFiscalSvc,
		ReconciliationService:    reconciliationSvc,
		EntitlementService:       entitlementSvc,
		SupplierService:          supplierSvc,
		MemberDocumentService:    memberDocSvc,
		VehicleService:           vehicleSvc,
		PermsEnforcement:         permsMode,

		FinanceFiscalDashboardService: finFiscalDashSvc,
		PatrimonyService:              patrimonySvc,
		PatrimonyDocumentService:      patrimonyDocSvc,
		WarrantyService:               warrantySvc,
		WarrantyDocumentService:       warrantyDocSvc,
		EducationService:              educationSvc,
		HomeSafetyService:             homeSafetySvc,
		PasswordResetService:          passwordResetSvc,
		HealthAppointmentService:      healthApptSvc,
		HealthPlanService:             healthPlanSvc,
		ProfileService:                profileSvc,
	})

	// Sweeper de ingestão fiscal: reenfileira jobs travados (pending/processing
	// antigos) no boot e periodicamente — recuperação após restart/crash.
	go func() {
		sweep := func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			if _, err := finExtSvc.RecoverStaleFiscalJobs(ctx, 5*time.Minute); err != nil {
				log.Warn("⚠️ recuperação de jobs fiscais falhou", slog.String("error", err.Error()))
			}
		}
		sweep()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			sweep()
		}
	}()

	// Recorrências rolling: completa o horizonte de 12 meses no boot e diariamente.
	go func() {
		run := func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			if n, err := financialEntrySvc.ExtendRecurrences(ctx); err != nil {
				log.Warn("⚠️ Extensão de recorrências falhou", slog.String("error", err.Error()))
			} else if n > 0 {
				log.Info(fmt.Sprintf("🔁 Recorrências estendidas: %d ocorrências previstas criadas", n))
			}
		}
		run()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			run()
		}
	}()

	// FIPE history: atualiza o valor FIPE de todos os veículos ativos uma vez ao mês.
	go func() {
		refresh := func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			month := time.Now().Format("01/2006")
			if n, err := vehicleSvc.RefreshFipeHistory(ctx, month); err != nil {
				log.Warn("⚠️ Atualização FIPE mensal falhou", slog.String("error", err.Error()))
			} else {
				log.Info(fmt.Sprintf("📈 FIPE atualizado: %d veículos | %s", n, month))
			}
		}
		refresh()
		ticker := time.NewTicker(30 * 24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			refresh()
		}
	}()

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
