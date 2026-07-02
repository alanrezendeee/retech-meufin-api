package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/finance"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// FinanceReceiptHandler expõe upload/listagem/download de comprovantes de
// pagamento anexados a lançamentos.
type FinanceReceiptHandler struct {
	docSvc   *app.FinanceDocumentService
	entrySvc *app.FinancialEntryService
}

func NewFinanceReceiptHandler(docSvc *app.FinanceDocumentService, entrySvc *app.FinancialEntryService) *FinanceReceiptHandler {
	return &FinanceReceiptHandler{docSvc: docSvc, entrySvc: entrySvc}
}

// Upload anexa um comprovante (multipart, campo 'file') ao lançamento :id.
func (h *FinanceReceiptHandler) Upload(c *gin.Context) {
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
	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	// Garante que o lançamento existe neste workspace antes de aceitar o arquivo.
	if _, err := h.entrySvc.Get(c.Request.Context(), ws, entryID); err != nil {
		errrespond.Write(c, err)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "campo 'file' obrigatório")
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "não foi possível ler o arquivo")
		return
	}
	defer f.Close()

	doc, err := h.docSvc.UploadReceipt(c.Request.Context(), app.UploadFinanceDocInput{
		WorkspaceID:      ws,
		UploadedByUserID: userID,
		OriginalFileName: fileHeader.Filename,
		MimeType:         fileHeader.Header.Get("Content-Type"),
		Size:             fileHeader.Size,
		Content:          f,
	}, entryID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapFinanceDocument(doc))
}

// List lista os comprovantes do lançamento :id.
func (h *FinanceReceiptHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	limit, offset := pagination(c)
	res, err := h.docSvc.ListReceipts(c.Request.Context(), ws, entryID, limit, offset)
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

// getReceipt carrega o documento :receiptId garantindo que é um comprovante.
func (h *FinanceReceiptHandler) getReceipt(c *gin.Context, ws uuid.UUID) (*dom.FinanceDocument, bool) {
	id, err := uuid.Parse(c.Param("receiptId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "receiptId inválido")
		return nil, false
	}
	doc, err := h.docSvc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return nil, false
	}
	if doc.Kind != dom.DocumentReceipt {
		errrespond.Message(c, http.StatusNotFound, errrespond.CodeBadRequest, "comprovante não encontrado")
		return nil, false
	}
	return doc, true
}

// DownloadURL retorna a URL presignada do comprovante :receiptId.
func (h *FinanceReceiptHandler) DownloadURL(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	doc, ok := h.getReceipt(c, ws)
	if !ok {
		return
	}
	url, err := h.docSvc.DownloadURL(c.Request.Context(), ws, doc.ID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

// Delete remove (soft) o comprovante :receiptId.
func (h *FinanceReceiptHandler) Delete(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	doc, ok := h.getReceipt(c, ws)
	if !ok {
		return
	}
	if err := h.docSvc.Delete(c.Request.Context(), ws, doc.ID); err != nil {
		errrespond.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
