package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/finance"
	"github.com/retechfin/retechfin-api/internal/infrastructure/pdfutil"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// financeExtractTriggerJSON é o body opcional do trigger de extração.
type financeExtractTriggerJSON struct {
	// PDFPassword: senha do PDF protegido. Usada só em memória para remover
	// a criptografia antes do LLM; nunca é persistida.
	PDFPassword string `json:"pdf_password"`
}

// FinanceExtractTriggerHandler dispara a extração de uma fatura: carrega o
// conteúdo via FinanceDocumentService (storage) e chama a
// FinanceExtractionService (async).
type FinanceExtractTriggerHandler struct {
	docs *app.FinanceDocumentService
	ext  *app.FinanceExtractionService
}

func NewFinanceExtractTriggerHandler(docs *app.FinanceDocumentService, ext *app.FinanceExtractionService) *FinanceExtractTriggerHandler {
	return &FinanceExtractTriggerHandler{docs: docs, ext: ext}
}

func (h *FinanceExtractTriggerHandler) Extract(c *gin.Context) {
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

	var body financeExtractTriggerJSON
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
			return
		}
	}

	doc, content, err := h.docs.LoadContent(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}

	inputType := "image"
	if strings.EqualFold(doc.MimeType, "application/pdf") {
		inputType = "pdf"
		// PDF protegido não passa no provedor LLM: remove a criptografia em
		// memória (senha nunca persiste). Sem senha em PDF protegido, erro
		// claro ANTES de gastar chamada de LLM.
		content, err = pdfutil.EnsureDecrypted(content, body.PDFPassword)
		if err != nil {
			switch {
			case errors.Is(err, pdfutil.ErrPasswordRequired):
				errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest,
					"Este PDF é protegido por senha — informe a senha do arquivo para extrair.")
			case errors.Is(err, pdfutil.ErrWrongPassword):
				errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest,
					"Senha do PDF incorreta — confira e tente novamente.")
			default:
				errrespond.Write(c, err)
			}
			return
		}
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
