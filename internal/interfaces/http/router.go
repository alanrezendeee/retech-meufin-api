package http

import (
	"log/slog"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/gin-gonic/gin"
	appacc "github.com/retechfin/retechfin-api/internal/application/account"
	appb "github.com/retechfin/retechfin-api/internal/application/budget"
	appedu "github.com/retechfin/retechfin-api/internal/application/education"
	appf "github.com/retechfin/retechfin-api/internal/application/finance"
	apph "github.com/retechfin/retechfin-api/internal/application/health"
	apphs "github.com/retechfin/retechfin-api/internal/application/homesafety"
	appl "github.com/retechfin/retechfin-api/internal/application/ledger"
	appp "github.com/retechfin/retechfin-api/internal/application/patrimony"
	appv "github.com/retechfin/retechfin-api/internal/application/vehicle"
	appw "github.com/retechfin/retechfin-api/internal/application/warranty"
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
	FinanceAccountService    *appf.AccountService
	FinanceCategoryService   *appf.ExpenseCategoryService
	FinanceDashboardService  *appf.FinanceDashboardService
	FinanceFiscalService     *appf.FiscalService
	SupplierService          *appf.SupplierService
	MemberDocumentService    *apph.MemberDocumentService
	VehicleService           *appv.Service
	PermsEnforcement         middleware.EnforcementMode

	// Módulos da apresentação (2026-07)
	FinanceFiscalDashboardService *appf.FiscalDashboardService
	PatrimonyService              *appp.Service
	PatrimonyDocumentService      *appp.DocumentService
	WarrantyService               *appw.Service
	WarrantyDocumentService       *appw.DocumentService
	EducationService              *appedu.Service
	HomeSafetyService             *apphs.Service
	PasswordResetService          *appacc.PasswordResetService
	HealthAppointmentService      *apph.AppointmentService
	HealthPlanService             *apph.PlanService
}

func NewRouter(d RouterDeps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(d.CORSOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.AccessLog(d.Log))

	hHealth := &handlers.Health{DB: d.DB, Env: d.Env}
	r.GET("/health", hHealth.Get)

	// Rotas públicas (usuário deslogado) — fluxo "esqueci a senha"
	resetH := handlers.NewPasswordResetHandler(d.PasswordResetService)
	public := r.Group("/api/v1/public")
	{
		public.POST("/password-reset/request", resetH.Request)
		public.POST("/password-reset/confirm", resetH.Confirm)
	}

	accH := handlers.NewAccountHandler(d.AccountService)
	catH := handlers.NewCategoryHandler(d.CategoryService)
	txH := handlers.NewTransactionHandler(d.TransactionService)
	budH := handlers.NewBudgetHandler(d.BudgetService)

	v1 := r.Group("/api/v1")
	v1.Use(middleware.RequireAuth(d.JWKS, d.ApplicationID))
	{
		// Módulo legado (ledger/budget) — subjects retechfin.*
		legacy := v1.Group("", middleware.RequireModule("retechfin", d.PermsEnforcement))
		legacy.POST("/accounts", accH.Create)
		legacy.GET("/accounts", accH.List)
		legacy.GET("/accounts/:id", accH.Get)
		legacy.PUT("/accounts/:id", accH.Update)
		legacy.DELETE("/accounts/:id", accH.Delete)

		legacy.POST("/categories", catH.Create)
		legacy.GET("/categories", catH.List)
		legacy.GET("/categories/:id", catH.Get)
		legacy.PUT("/categories/:id", catH.Update)
		legacy.DELETE("/categories/:id", catH.Delete)

		legacy.POST("/transactions", txH.Create)
		legacy.GET("/transactions", txH.List)
		legacy.GET("/transactions/:id", txH.Get)
		legacy.PUT("/transactions/:id", txH.Update)
		legacy.DELETE("/transactions/:id", txH.Delete)

		legacy.POST("/budgets", budH.Create)
		legacy.GET("/budgets", budH.List)
		legacy.POST("/budgets/validate", budH.Validate)
		legacy.GET("/budgets/:id", budH.Get)
		legacy.PUT("/budgets/:id", budH.Update)
		legacy.DELETE("/budgets/:id", budH.Delete)

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
		health := v1.Group("/health", middleware.RequireModule("health", d.PermsEnforcement))
		{
			health.GET("/markers", mkH.List)
			health.POST("/markers", mkH.Create)
			health.POST("/markers/resolve", mkH.Resolve)
			health.GET("/markers/:id", mkH.Get)
			health.PUT("/markers/:id", mkH.Update)
			health.DELETE("/markers/:id", mkH.Delete)

			health.GET("/family-members", fmH.List)
			health.POST("/family-members", fmH.Create)
			health.GET("/family-members/birthdays", fmH.Birthdays)
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

			// Planos de saúde
			planH := handlers.NewHealthPlanHandler(d.HealthPlanService)
			health.GET("/plans", planH.List)
			health.POST("/plans", planH.Create)
			health.GET("/plans/:id", planH.Get)
			health.PUT("/plans/:id", planH.Update)
			health.DELETE("/plans/:id", planH.Delete)
			health.PUT("/plans/:id/members", planH.ReplaceMembers)

			// Consultas & agenda (rotas estáticas ANTES de /appointments/:id)
			apptH := handlers.NewHealthAppointmentHandler(d.HealthAppointmentService)
			health.GET("/appointments/agenda", apptH.Agenda)
			health.GET("/appointments", apptH.List)
			health.POST("/appointments", apptH.Create)
			health.GET("/appointments/:id", apptH.Get)
			health.PUT("/appointments/:id", apptH.Update)
			health.DELETE("/appointments/:id", apptH.Delete)
			health.POST("/appointments/:id/confirm", apptH.Confirm)
			health.POST("/appointments/:id/complete", apptH.Complete)
			health.POST("/appointments/:id/cancel", apptH.Cancel)
			health.POST("/appointments/:id/no-show", apptH.NoShow)

			// Documentos pessoais dos membros (cpf, rg, cnh, ...)
			memberDocH := handlers.NewHealthMemberDocumentHandler(d.MemberDocumentService)
			health.POST("/family-members/:id/documents", memberDocH.Upload)
			health.GET("/family-members/:id/documents", memberDocH.List)
			health.GET("/family-members/:id/documents/:docId/download-url", memberDocH.DownloadURL)
			health.DELETE("/family-members/:id/documents/:docId", memberDocH.Delete)

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
	supplierH := handlers.NewSupplierHandler(d.SupplierService)
	srcH := handlers.NewIncomeSourceHandler(d.IncomeSourceService)
	entH := handlers.NewFinancialEntryHandler(d.FinancialEntryService)
	cardH := handlers.NewCreditCardHandler(d.CreditCardService)
	finDocH := handlers.NewFinanceDocumentHandler(d.FinanceDocumentService)
	finExtTrigH := handlers.NewFinanceExtractTriggerHandler(d.FinanceDocumentService, d.FinanceExtractionService)
	finExtH := handlers.NewFinanceExtractionHandler(d.FinanceExtractionService, d.FinanceDocumentService, d.FinancialEntryService)
	finFiscalH := handlers.NewFinanceFiscalHandler(d.FinanceFiscalService)
	finance := v1.Group("/finance", middleware.RequireModule("finance", d.PermsEnforcement))
	{
		finance.GET("/suppliers", supplierH.List)
		finance.POST("/suppliers", supplierH.Create)
		finance.GET("/suppliers/:id", supplierH.Get)
		finance.PUT("/suppliers/:id", supplierH.Update)
		finance.DELETE("/suppliers/:id", supplierH.Delete)

		finance.GET("/income-sources", srcH.List)
		finance.POST("/income-sources", srcH.Create)
		finance.GET("/income-sources/:id", srcH.Get)
		finance.PUT("/income-sources/:id", srcH.Update)
		finance.DELETE("/income-sources/:id", srcH.Delete)

		finance.GET("/card-brands", cardH.Brands)
		finance.GET("/cards", cardH.List)
		finance.POST("/cards", cardH.Create)
		finance.GET("/cards/:id", cardH.Get)
		finance.PUT("/cards/:id", cardH.Update)
		finance.DELETE("/cards/:id", cardH.Delete)

		// Contas (corrente/poupança/carteira/digital) usadas na liquidação.
		finAccH := handlers.NewFinanceAccountHandler(d.FinanceAccountService)
		finance.GET("/accounts", finAccH.List)
		finance.POST("/accounts", finAccH.Create)
		finance.GET("/accounts/:id", finAccH.Get)
		finance.PUT("/accounts/:id", finAccH.Update)
		finance.DELETE("/accounts/:id", finAccH.Delete)

		// Categorias de despesa (gerenciadas por workspace, grupo canônico curado)
		catFinH := handlers.NewFinanceCategoryHandler(d.FinanceCategoryService)
		finance.GET("/expense-categories", catFinH.List)
		finance.POST("/expense-categories", catFinH.Create)
		finance.PUT("/expense-categories/:id", catFinH.Update)
		finance.DELETE("/expense-categories/:id", catFinH.Delete)

		finance.GET("/discount-reasons", entH.DiscountReasons)
		finance.GET("/installments", entH.Installments)
		finance.GET("/entries", entH.List)
		finance.POST("/entries", entH.Create)
		finance.GET("/entries/year-bounds", entH.YearBounds)
		finance.GET("/entries/:id", entH.Get)
		finance.PUT("/entries/:id", entH.Update)
		finance.DELETE("/entries/:id", entH.Delete)
		finance.POST("/entries/:id/confirm", entH.Confirm)
		finance.POST("/entries/:id/reopen", entH.Reopen)
		finance.POST("/entries/:id/cancel", entH.Cancel)
		finance.POST("/entries/:id/settle", entH.Settle)
		finance.POST("/entries/:id/resize-installments", entH.ResizeInstallments)

		// Comprovantes de pagamento anexados a lançamentos.
		receiptH := handlers.NewFinanceReceiptHandler(d.FinanceDocumentService, d.FinancialEntryService)
		finance.POST("/entries/:id/receipts", receiptH.Upload)
		finance.GET("/entries/:id/receipts", receiptH.List)
		finance.GET("/entries/:id/receipts/:receiptId/download-url", receiptH.DownloadURL)
		finance.DELETE("/entries/:id/receipts/:receiptId", receiptH.Delete)

		// Dashboard financeira (agregados; valores em cents).
		finDashH := handlers.NewFinanceDashboardHandler(d.FinanceDashboardService)
		finance.GET("/dashboard", finDashH.Summary)
		finance.GET("/dashboard/monthly", finDashH.Monthly)

		finance.POST("/documents", finDocH.Upload)
		finance.GET("/documents", finDocH.List)
		finance.GET("/documents/:id", finDocH.Get)
		finance.DELETE("/documents/:id", finDocH.Delete)
		finance.GET("/documents/:id/download-url", finDocH.DownloadURL)
		finance.POST("/documents/:id/extract", finExtTrigH.Extract)
		finance.GET("/documents/:id/extraction-status", finExtH.Status)
		finance.POST("/documents/:id/confirm", finExtH.Confirm)
		finance.POST("/documents/:id/fiscal-confirm", finFiscalH.Confirm)
		finance.GET("/entries/:id/fiscal-items", finFiscalH.ListByEntry)

		// Dashboard fiscal (inflação pessoal por item comprado)
		finFiscalDashH := handlers.NewFinanceFiscalDashboardHandler(d.FinanceFiscalDashboardService)
		finance.GET("/fiscal/dashboard", finFiscalDashH.Summary)
		finance.GET("/fiscal/products", finFiscalDashH.Products)
		finance.GET("/fiscal/products/price-history", finFiscalDashH.PriceHistory)
		finance.GET("/fiscal/inflation", finFiscalDashH.Inflation)
	}

	// Frota Familiar
	vehicleH := handlers.NewVehicleHandler(d.VehicleService)
	vehicles := v1.Group("/vehicles", middleware.RequireModule("vehicles", d.PermsEnforcement))
	{
		// FIPE search (estático antes de /:id para não conflitar)
		vehicles.GET("/fipe/brands", vehicleH.FipeBrands)
		vehicles.GET("/fipe/models", vehicleH.FipeModels)
		vehicles.GET("/fipe/years", vehicleH.FipeYears)
		vehicles.GET("/fipe/price", vehicleH.FipePrice)

		// Veículos
		vehicles.GET("", vehicleH.List)
		vehicles.POST("", vehicleH.Create)
		vehicles.GET("/:id", vehicleH.Get)
		vehicles.PUT("/:id", vehicleH.Update)
		vehicles.DELETE("/:id", vehicleH.Delete)
		vehicles.PATCH("/:id/odometer", vehicleH.UpdateOdometer)

		// Manutenções (unificadas: inclui itens, status de OS, etc.)
		vehicles.GET("/:id/maintenances", vehicleH.ListMaintenance)
		vehicles.POST("/:id/maintenances", vehicleH.CreateMaintenance)
		vehicles.GET("/:id/maintenances/:mId", vehicleH.GetMaintenance)
		vehicles.PUT("/:id/maintenances/:mId", vehicleH.UpdateMaintenance)
		vehicles.DELETE("/:id/maintenances/:mId", vehicleH.DeleteMaintenance)

		// Itens de manutenção
		vehicles.POST("/:id/maintenances/:mId/items", vehicleH.AddMaintenanceItem)
		vehicles.PUT("/:id/maintenances/:mId/items/:itemId", vehicleH.UpdateMaintenanceItem)
		vehicles.DELETE("/:id/maintenances/:mId/items/:itemId", vehicleH.DeleteMaintenanceItem)

		// Planos de manutenção por veículo
		vehicles.GET("/:id/plans", vehicleH.ListPlans)
		vehicles.PUT("/:id/plans/:templateId", vehicleH.UpdatePlan)

		// Alertas e depreciação
		vehicles.GET("/:id/alerts", vehicleH.GetAlerts)
		vehicles.GET("/:id/depreciation", vehicleH.GetDepreciation)
		vehicles.GET("/:id/fipe-history", vehicleH.GetFipeHistory)
		vehicles.GET("/:id/fipe-all-years", vehicleH.GetFipeAllYears)

		// Agendamentos de manutenção
		vehicles.GET("/:id/schedules", vehicleH.ListSchedules)
		vehicles.POST("/:id/schedules", vehicleH.CreateSchedule)
		vehicles.PUT("/:id/schedules/:schedId", vehicleH.UpdateSchedule)
		vehicles.DELETE("/:id/schedules/:schedId", vehicleH.DeleteSchedule)

		// Analytics
		vehicles.GET("/:id/analytics", vehicleH.GetAnalytics)

		// Catálogo global de manutenção (sem /:id)
		vehicles.GET("/maintenance/catalog", vehicleH.SearchCatalog)
	}

	// Patrimônio — imóveis + impostos de bens
	patrimonyH := handlers.NewPatrimonyHandler(d.PatrimonyService)
	patrimonyDocH := handlers.NewPropertyDocumentHandler(d.PatrimonyDocumentService)
	patrimony := v1.Group("/patrimony", middleware.RequireModule("patrimony", d.PermsEnforcement))
	{
		patrimony.GET("/properties", patrimonyH.ListProperties)
		patrimony.POST("/properties", patrimonyH.CreateProperty)
		patrimony.GET("/properties/:id", patrimonyH.GetProperty)
		patrimony.PUT("/properties/:id", patrimonyH.UpdateProperty)
		patrimony.DELETE("/properties/:id", patrimonyH.DeleteProperty)

		patrimony.POST("/properties/:id/documents", patrimonyDocH.Upload)
		patrimony.GET("/properties/:id/documents", patrimonyDocH.List)
		patrimony.GET("/properties/:id/documents/:docId/download-url", patrimonyDocH.DownloadURL)
		patrimony.DELETE("/properties/:id/documents/:docId", patrimonyDocH.Delete)

		// /taxes/overview ANTES de /taxes/:id (conflito de wildcard do Gin)
		patrimony.GET("/taxes/overview", patrimonyH.Overview)
		patrimony.GET("/taxes", patrimonyH.ListTaxes)
		patrimony.POST("/taxes", patrimonyH.CreateTax)
		patrimony.GET("/taxes/:id", patrimonyH.GetTax)
		patrimony.PUT("/taxes/:id", patrimonyH.UpdateTax)
		patrimony.DELETE("/taxes/:id", patrimonyH.DeleteTax)
		patrimony.POST("/taxes/:id/pay", patrimonyH.PayTax)
	}

	// Garantias de bens
	warrantyH := handlers.NewWarrantyHandler(d.WarrantyService, d.WarrantyDocumentService)
	warranties := v1.Group("/warranties", middleware.RequireModule("warranties", d.PermsEnforcement))
	{
		warranties.GET("/summary", warrantyH.Summary)
		warranties.GET("", warrantyH.List)
		warranties.POST("", warrantyH.Create)
		warranties.GET("/:id", warrantyH.Get)
		warranties.PUT("/:id", warrantyH.Update)
		warranties.DELETE("/:id", warrantyH.Delete)

		warranties.POST("/:id/documents", warrantyH.UploadDocument)
		warranties.GET("/:id/documents", warrantyH.ListDocuments)
		warranties.GET("/:id/documents/:docId/download-url", warrantyH.DocumentDownloadURL)
		warranties.DELETE("/:id/documents/:docId", warrantyH.DeleteDocument)
	}

	// Educação / Material Escolar
	educationH := handlers.NewEducationHandler(d.EducationService)
	education := v1.Group("/education", middleware.RequireModule("education", d.PermsEnforcement))
	{
		education.GET("/dashboard", educationH.Dashboard)

		education.GET("/enrollments", educationH.ListEnrollments)
		education.POST("/enrollments", educationH.CreateEnrollment)
		education.GET("/enrollments/:id", educationH.GetEnrollment)
		education.PUT("/enrollments/:id", educationH.UpdateEnrollment)
		education.DELETE("/enrollments/:id", educationH.DeleteEnrollment)

		education.GET("/supply-lists", educationH.ListSupplyLists)
		education.POST("/supply-lists", educationH.CreateSupplyList)
		education.GET("/supply-lists/:id", educationH.GetSupplyList)
		education.PUT("/supply-lists/:id", educationH.UpdateSupplyList)
		education.DELETE("/supply-lists/:id", educationH.DeleteSupplyList)

		education.POST("/supply-lists/:id/items", educationH.AddItem)
		education.PUT("/supply-lists/:id/items/:itemId", educationH.UpdateItem)
		education.DELETE("/supply-lists/:id/items/:itemId", educationH.DeleteItem)
		education.POST("/supply-lists/:id/items/:itemId/purchase", educationH.PurchaseItem)
	}

	// Segurança do Lar
	homeSafetyH := handlers.NewHomeSafetyHandler(d.HomeSafetyService)
	homeSafety := v1.Group("/home-safety", middleware.RequireModule("homesafety", d.PermsEnforcement))
	{
		homeSafety.GET("/dashboard", homeSafetyH.Dashboard)
		homeSafety.GET("/catalog", homeSafetyH.Catalog)

		homeSafety.GET("/items", homeSafetyH.ListItems)
		homeSafety.POST("/items", homeSafetyH.CreateItem)
		homeSafety.GET("/items/:id", homeSafetyH.GetItem)
		homeSafety.PUT("/items/:id", homeSafetyH.UpdateItem)
		homeSafety.DELETE("/items/:id", homeSafetyH.DeleteItem)

		homeSafety.GET("/items/:id/events", homeSafetyH.ListEvents)
		homeSafety.POST("/items/:id/events", homeSafetyH.CreateEvent)
		homeSafety.DELETE("/items/:id/events/:eventId", homeSafetyH.DeleteEvent)
	}

	return r
}
