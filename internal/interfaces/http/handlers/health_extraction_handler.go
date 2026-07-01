package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/health"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// HealthExtractionHandler expõe a consulta de status de extração de um
// documento de saúde.
//
// NOTA DE FIAÇÃO (a ser feita fora deste arquivo):
// O DISPARO da extração (POST /documents/:id/extract) NÃO vive aqui, porque
// precisa do CONTEÚDO do arquivo, que reside no módulo de documentos (storage).
// A fiação correta é o document handler carregar o conteúdo do documento via
// seu DocumentService/storage e chamar:
//
//	ExtractionService.StartExtraction(ctx, workspaceID, documentID, inputType, mimeType, content)
//
// Este handler cobre apenas o polling de status:
//
//	GET /documents/:id/extraction-status -> job (status, provider, model, error, timestamps)
type HealthExtractionHandler struct {
	svc *app.ExtractionService
}

func NewHealthExtractionHandler(svc *app.ExtractionService) *HealthExtractionHandler {
	return &HealthExtractionHandler{svc: svc}
}

type extractionStatusResponse struct {
	ID            uuid.UUID `json:"id"`
	DocumentID    uuid.UUID `json:"document_id"`
	Provider      string    `json:"provider"`
	Model         *string   `json:"model,omitempty"`
	Status        string    `json:"status"`
	InputType     string    `json:"input_type"`
	PromptVersion *string   `json:"prompt_version,omitempty"`
	ErrorMessage  *string   `json:"error_message,omitempty"`
	StartedAt     *string   `json:"started_at,omitempty"`
	FinishedAt    *string   `json:"finished_at,omitempty"`
	CreatedAt     string    `json:"created_at"`
	UpdatedAt     string    `json:"updated_at"`
}

func mapExtractionStatus(j *dom.ExtractionJob) extractionStatusResponse {
	out := extractionStatusResponse{
		ID:            j.ID,
		DocumentID:    j.DocumentID,
		Provider:      j.Provider,
		Model:         j.Model,
		Status:        string(j.Status),
		InputType:     string(j.InputType),
		PromptVersion: j.PromptVersion,
		ErrorMessage:  j.ErrorMessage,
		CreatedAt:     j.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:     j.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if j.StartedAt != nil {
		s := j.StartedAt.UTC().Format(time.RFC3339Nano)
		out.StartedAt = &s
	}
	if j.FinishedAt != nil {
		f := j.FinishedAt.UTC().Format(time.RFC3339Nano)
		out.FinishedAt = &f
	}
	return out
}

// Status responde GET /documents/:id/extraction-status com o job de extração
// mais recente do documento.
func (h *HealthExtractionHandler) Status(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	job, err := h.svc.GetStatus(c.Request.Context(), ws, documentID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapExtractionStatus(job))
}
