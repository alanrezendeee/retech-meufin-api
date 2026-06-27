package http

import (
	"log/slog"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/gin-gonic/gin"
	appb "github.com/retechfin/retechfin-api/internal/application/budget"
	appl "github.com/retechfin/retechfin-api/internal/application/ledger"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/handlers"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
	"gorm.io/gorm"
)

type RouterDeps struct {
	Log                 *slog.Logger
	DB                  *gorm.DB
	Env                 string
	JWKS                *keyfunc.JWKS
	ApplicationID       string
	CORSOrigins         []string
	AccountService      *appl.AccountService
	CategoryService     *appl.CategoryService
	TransactionService  *appl.TransactionService
	BudgetService       *appb.Service
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
	}

	return r
}
