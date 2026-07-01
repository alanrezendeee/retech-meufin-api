package http

import (
	"log/slog"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/gin-gonic/gin"
	appb "github.com/retechfin/retechfin-api/internal/application/budget"
	appf "github.com/retechfin/retechfin-api/internal/application/finance"
	apph "github.com/retechfin/retechfin-api/internal/application/health"
	appl "github.com/retechfin/retechfin-api/internal/application/ledger"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/handlers"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
	"gorm.io/gorm"
)

type RouterDeps struct {
	Log                      *slog.Logger
	DB                       *gorm.DB
	Env                      string
	JWKS                     *keyfunc.JWKS
	ApplicationID            string
	CORSOrigins              []string
	AccountService           *appl.AccountService
	CategoryService          *appl.CategoryService
	TransactionService       *appl.TransactionService
	BudgetService            *appb.Service
	HealthMarkerService      *apph.MarkerService
	FamilyMemberService      *apph.FamilyMemberService
	LabService               *apph.LabService
	ExamRequestService       *apph.ExamRequestService
	ExamResultService        *apph.ExamResultService
	DashboardService         *apph.DashboardService
	DocumentService          *apph.DocumentService
	ExtractionService        *apph.ExtractionService
	IncomeSourceService      *appf.IncomeSourceService
	FinancialEntryService    *appf.FinancialEntryService
	CreditCardService        *appf.CreditCardService
	FinanceDocumentService   *appf.FinanceDocumentService
	FinanceExtractionService *appf.FinanceExtractionService
}

func NewRouter(d RouterDeps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(d.CORSOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.AccessLog(d.Log))

	hHealth := &handlers.Health{DB: d.DB, Env: d.Env}
	r.GET("/health", hHealth.Get)

	accH := handlers.NewAccountHandler(d.AccountService)
	catH := handlers.NewCategoryHandler(d.CategoryService)
	txH := handlers.NewTransactionHandler(d.TransactionService)
	budH := handlers.NewBudgetHandler(d.BudgetService)

	v1 := r.Group("/api/v1")
	v1.Use(middleware.RequireAuth(d.JWKS, d.ApplicationID))
	{
		v1.POST("/accounts", accH.Create)
		v1.GET("/accounts", accH.List)
		v1.GET("/accounts/:id", accH.Get)
		v1.PUT("/accounts/:id", accH.Update)
		v1.DELETE("/accounts/:id", accH.Delete)

		v1.POST("/categories", catH.Create)
		v1.GET("/categories", catH.List)
		v1.GET("/categories/:id", catH.Get)
		v1.PUT("/categories/:id", catH.Update)
		v1.DELETE("/categories/:id", catH.Delete)

		v1.POST("/transactions", txH.Create)
		v1.GET("/transactions", txH.List)
		v1.GET("/transactions/:id", txH.Get)
		v1.PUT("/transactions/:id", txH.Update)
		v1.DELETE("/transactions/:id", txH.Delete)

		v1.POST("/budgets", budH.Create)
		v1.GET("/budgets", budH.List)
		v1.POST("/budgets/validate", budH.Validate)
		v1.GET("/budgets/:id", budH.Get)
		v1.PUT("/budgets/:id", budH.Update)
		v1.DELETE("/budgets/:id", budH.Delete)

		// Saúde Familiar — catálogo de marcadores (Fase 0)
		mkH := handlers.NewHealthMarkerHandler(d.HealthMarkerService)
		fmH := handlers.NewHealthFamilyMemberHandler(d.FamilyMemberService)
		labH := handlers.NewHealthLabHandler(d.LabService)
		reqH := handlers.NewHealthExamRequestHandler(d.ExamRequestService)
		resH := handlers.NewHealthExamResultHandler(d.ExamResultService)
		dashH := handlers.NewHealthDashboardHandler(d.DashboardService)
		docH := handlers.NewHealthDocumentHandler(d.DocumentService)
		extStatusH := handlers.NewHealthExtractionHandler(d.ExtractionService)
		extTrigH := handlers.NewHealthExtractTriggerHandler(d.DocumentService, d.ExtractionService)
		health := v1.Group("/health")
		{
			health.GET("/markers", mkH.List)
			health.POST("/markers", mkH.Create)
			health.POST("/markers/resolve", mkH.Resolve)
			health.GET("/markers/:id", mkH.Get)
			health.PUT("/markers/:id", mkH.Update)
			health.DELETE("/markers/:id", mkH.Delete)

			health.GET("/family-members", fmH.List)
			health.POST("/family-members", fmH.Create)
			health.GET("/family-members/:id", fmH.Get)
			health.PUT("/family-members/:id", fmH.Update)
			health.DELETE("/family-members/:id", fmH.Delete)

			health.GET("/labs", labH.List)
			health.POST("/labs", labH.Create)
			health.GET("/labs/:id", labH.Get)
			health.PUT("/labs/:id", labH.Update)
			health.DELETE("/labs/:id", labH.Delete)

			health.GET("/exam-requests", reqH.List)
			health.POST("/exam-requests", reqH.Create)
			health.GET("/exam-requests/:id", reqH.Get)
			health.PUT("/exam-requests/:id", reqH.Update)
			health.DELETE("/exam-requests/:id", reqH.Delete)
			health.POST("/exam-requests/:id/items", reqH.AddItem)
			health.PUT("/exam-requests/:id/items/:itemId", reqH.UpdateItem)
			health.DELETE("/exam-requests/:id/items/:itemId", reqH.DeleteItem)

			health.GET("/exam-results", resH.List)
			health.POST("/exam-results", resH.Create)
			health.GET("/exam-results/:id", resH.Get)
			health.PUT("/exam-results/:id", resH.Update)
			health.DELETE("/exam-results/:id", resH.Delete)
			health.POST("/exam-results/:id/items", resH.AddItem)
			health.PUT("/exam-results/:id/items/:itemId", resH.UpdateItem)
			health.DELETE("/exam-results/:id/items/:itemId", resH.DeleteItem)

			health.GET("/dashboard", dashH.Counts)
			health.GET("/dashboard/markers/:markerId/evolution", dashH.MarkerEvolution)

			health.POST("/documents", docH.Upload)
			health.GET("/documents", docH.List)
			health.GET("/documents/:id", docH.Get)
			health.DELETE("/documents/:id", docH.Delete)
			health.GET("/documents/:id/download-url", docH.DownloadURL)
			health.POST("/documents/:id/extract", extTrigH.Extract)
			health.GET("/documents/:id/extraction-status", extStatusH.Status)
		}
	}

	// Financeiro — lançamento único (crédito/débito) + fontes de receita
	srcH := handlers.NewIncomeSourceHandler(d.IncomeSourceService)
	entH := handlers.NewFinancialEntryHandler(d.FinancialEntryService)
	cardH := handlers.NewCreditCardHandler(d.CreditCardService)
	finDocH := handlers.NewFinanceDocumentHandler(d.FinanceDocumentService)
	finExtTrigH := handlers.NewFinanceExtractTriggerHandler(d.FinanceDocumentService, d.FinanceExtractionService)
	finExtH := handlers.NewFinanceExtractionHandler(d.FinanceExtractionService, d.FinanceDocumentService, d.FinancialEntryService)
	finance := v1.Group("/finance")
	{
		finance.GET("/income-sources", srcH.List)
		finance.POST("/income-sources", srcH.Create)
		finance.GET("/income-sources/:id", srcH.Get)
		finance.PUT("/income-sources/:id", srcH.Update)
		finance.DELETE("/income-sources/:id", srcH.Delete)

		finance.GET("/cards", cardH.List)
		finance.POST("/cards", cardH.Create)
		finance.GET("/cards/:id", cardH.Get)
		finance.PUT("/cards/:id", cardH.Update)
		finance.DELETE("/cards/:id", cardH.Delete)

		finance.GET("/entries", entH.List)
		finance.POST("/entries", entH.Create)
		finance.GET("/entries/:id", entH.Get)
		finance.PUT("/entries/:id", entH.Update)
		finance.DELETE("/entries/:id", entH.Delete)
		finance.POST("/entries/:id/confirm", entH.Confirm)
		finance.POST("/entries/:id/cancel", entH.Cancel)

		finance.POST("/documents", finDocH.Upload)
		finance.GET("/documents", finDocH.List)
		finance.GET("/documents/:id", finDocH.Get)
		finance.DELETE("/documents/:id", finDocH.Delete)
		finance.GET("/documents/:id/download-url", finDocH.DownloadURL)
		finance.POST("/documents/:id/extract", finExtTrigH.Extract)
		finance.GET("/documents/:id/extraction-status", finExtH.Status)
		finance.POST("/documents/:id/confirm", finExtH.Confirm)
	}

	return r
}
