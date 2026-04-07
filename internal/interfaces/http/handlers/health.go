package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Health struct{}

func (h *Health) Get(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
