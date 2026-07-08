package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/finance"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// FinanceDocumentHandler expõe upload/listagem/download de documentos
// financeiros (faturas de cartão).
type FinanceDocumentHandler struct {
	svc *app.FinanceDocumentService
}

func NewFinanceDocumentHandler(svc *app.FinanceDocumentService) *FinanceDocumentHandler {
	return &FinanceDocumentHandler{svc: svc}
}

// financeDocumentResponse omite o object_key/bucket crus, expondo apenas o essencial.
type financeDocumentResponse struct {
	ID               uuid.UUID  `json:"id"`
	CardID           *uuid.UUID `json:"card_id,omitempty"`
	EntryID          *uuid.UUID `json:"entry_id,omitempty"`
	FileName         string     `json:"file_name"`
	OriginalFileName string     `json:"original_file_name"`
	MimeType         string     `json:"mime_type"`
	SizeBytes        int64      `json:"size_bytes"`
	ExtractionStatus string     `json:"extraction_status"`
	CreatedAt        string     `json:"created_at"`
	UpdatedAt        string     `json:"updated_at"`
}

func mapFinanceDocument(d *dom.FinanceDocument) financeDocumentResponse {
	return financeDocumentResponse{
		ID:               d.ID,
		CardID:           d.CardID,
		EntryID:          d.EntryID,
		FileName:         d.FileName,
		OriginalFileName: d.OriginalFileName,
		MimeType:         d.MimeType,
		SizeBytes:        d.SizeBytes,
		ExtractionStatus: string(d.ExtractionStatus),
		CreatedAt:        d.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:        d.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (h *FinanceDocumentHandler) Upload(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	userID, ok := userIDFromCtx(c)
	if !ok {
		errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "usuário inválido no token")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "campo 'file' obrigatório")
		return
	}

	cardID, err := optionalUUIDForm(c, "card_id")
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "card_id inválido")
		return
	}

	kind := dom.DocumentKind(c.PostForm("kind"))
	switch kind {
	case "", dom.DocumentImport:
		kind = dom.DocumentImport
	case dom.DocumentFiscal:
	default:
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "kind inválido (use 'import' ou 'fiscal')")
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "não foi possível ler o arquivo")
		return
	}
	defer f.Close()

	mimeType := fileHeader.Header.Get("Content-Type")

	doc, err := h.svc.Upload(c.Request.Context(), app.UploadFinanceDocInput{
		WorkspaceID:      ws,
		UploadedByUserID: userID,
		CardID:           cardID,
		Kind:             kind,
		OriginalFileName: fileHeader.Filename,
		MimeType:         mimeType,
		Size:             fileHeader.Size,
		Content:          f,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapFinanceDocument(doc))
}

func (h *FinanceDocumentHandler) Get(c *gin.Context) {
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
	doc, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapFinanceDocument(doc))
}

func (h *FinanceDocumentHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	limit, offset := pagination(c)
	kind := dom.DocumentKind(c.Query("kind"))
	filter := dom.FinanceDocumentFilter{
		Kind:  &kind,
		Query: c.Query("q"),
	}
	if raw := c.Query("status"); raw != "" {
		st := dom.ExtractionStatus(raw)
		switch st {
		case dom.ExtractionPending, dom.ExtractionProcessing, dom.ExtractionExtracted,
			dom.ExtractionFailed, dom.ExtractionNotRequired:
			filter.Status = &st
		default:
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "status inválido")
			return
		}
	}
	if raw := c.Query("linked"); raw != "" {
		linked := raw == "true" || raw == "1"
		filter.Linked = &linked
	}
	res, err := h.svc.List(c.Request.Context(), ws, filter, limit, offset)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]financeDocumentResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapFinanceDocument(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}

func (h *FinanceDocumentHandler) Delete(c *gin.Context) {
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
	if err := h.svc.Delete(c.Request.Context(), ws, id); err != nil {
		errrespond.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *FinanceDocumentHandler) DownloadURL(c *gin.Context) {
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
	url, err := h.svc.DownloadURL(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}
