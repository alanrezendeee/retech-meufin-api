package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/health"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// HealthExtractTriggerHandler dispara a extração de um documento: carrega o
// conteúdo via DocumentService (storage) e chama a ExtractionService (async).
type HealthExtractTriggerHandler struct {
	docs *app.DocumentService
	ext  *app.ExtractionService
}

func NewHealthExtractTriggerHandler(docs *app.DocumentService, ext *app.ExtractionService) *HealthExtractTriggerHandler {
	return &HealthExtractTriggerHandler{docs: docs, ext: ext}
}

func (h *HealthExtractTriggerHandler) Extract(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}

	doc, content, err := h.docs.LoadContent(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}

	inputType := "image"
	if strings.EqualFold(doc.MimeType, "application/pdf") {
		inputType = "pdf"
	}

	job, err := h.ext.StartExtraction(c.Request.Context(), ws, id, inputType, doc.MimeType, content)
	if err != nil {
		errrespond.Write(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"document_id": id,
		"job_id":      job.ID,
		"status":      string(job.Status),
	})
}
