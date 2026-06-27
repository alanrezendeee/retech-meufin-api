package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/retechfin/retechfin-api/internal/version"
)

type healthResponse struct {
	Service     string `json:"service"`
	Status      string `json:"status"`
	DataBase    string `json:"dataBase"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type Health struct {
	DB  *gorm.DB
	Env string
}

func (h *Health) Get(c *gin.Context) {
	dataBase := "up"
	status := "ok"
	code := http.StatusOK

	if h.DB == nil {
		dataBase = "down"
		status = "degraded"
		code = http.StatusServiceUnavailable
	} else {
		sqlDB, err := h.DB.DB()
		if err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
			dataBase = "down"
			status = "degraded"
			code = http.StatusServiceUnavailable
		}
	}

	c.JSON(code, healthResponse{
		Service:     version.Service,
		Status:      status,
		DataBase:    dataBase,
		Version:     version.Version,
		Environment: h.Env,
	})
}
