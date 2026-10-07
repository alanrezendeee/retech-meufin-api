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

// FinanceAttachmentHandler expõe os anexos de apoio de um lançamento
// (boleto, QR Code Pix, nota, contrato…) — distintos dos comprovantes.
type FinanceAttachmentHandler struct {
	docSvc   *app.FinanceDocumentService
	entrySvc *app.FinancialEntryService
}

func NewFinanceAttachmentHandler(docSvc *app.FinanceDocumentService, entrySvc *app.FinancialEntryService) *FinanceAttachmentHandler {
	return &FinanceAttachmentHandler{docSvc: docSvc, entrySvc: entrySvc}
}

// Upload anexa um documento (multipart: 'file' obrigatório; 'attachment_type'
// obrigatório; 'payment_code', 'note' e 'apply_to' opcionais) ao lançamento :id.
// apply_to=future replica o anexo às parcelas futuras previstas da série —
// só para tipos replicáveis (pix_qrcode, contrato); boleto/nota/fatura/outro
// ficam só no lançamento alvo. A resposta traz replicated_to.
func (h *FinanceAttachmentHandler) Upload(c *gin.Context) {
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

	res, err := h.docSvc.UploadAttachment(c.Request.Context(), app.UploadAttachmentInput{
		WorkspaceID:      ws,
		UploadedByUserID: userID,
		EntryID:          entryID,
		Type:             dom.AttachmentType(c.PostForm("attachment_type")),
		PaymentCode:      c.PostForm("payment_code"),
		Note:             c.PostForm("note"),
		OriginalFileName: fileHeader.Filename,
		MimeType:         fileHeader.Header.Get("Content-Type"),
		Size:             fileHeader.Size,
		Content:          f,
		ApplyToFuture:    c.PostForm("apply_to") == "future",
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	out := mapFinanceDocument(res.Doc)
	n := res.ReplicatedTo
	out.ReplicatedTo = &n
	c.JSON(http.StatusCreated, out)
}

// List lista os anexos do lançamento :id.
func (h *FinanceAttachmentHandler) List(c *gin.Context) {
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
	res, err := h.docSvc.ListAttachments(c.Request.Context(), ws, entryID, limit, offset)
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

// getAttachment carrega :attachmentId garantindo que é um anexo do lançamento :id.
func (h *FinanceAttachmentHandler) getAttachment(c *gin.Context, ws uuid.UUID) (*dom.FinanceDocument, bool) {
	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return nil, false
	}
	id, err := uuid.Parse(c.Param("attachmentId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "attachmentId inválido")
		return nil, false
	}
	doc, err := h.docSvc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return nil, false
	}
	if doc.Kind != dom.DocumentAttachment || doc.EntryID == nil || *doc.EntryID != entryID {
		errrespond.Message(c, http.StatusNotFound, errrespond.CodeBadRequest, "anexo não encontrado")
		return nil, false
	}
	return doc, true
}

// DownloadURL retorna a URL presignada do anexo :attachmentId.
func (h *FinanceAttachmentHandler) DownloadURL(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	doc, ok := h.getAttachment(c, ws)
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

// Delete remove (soft) o anexo :attachmentId.
func (h *FinanceAttachmentHandler) Delete(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	doc, ok := h.getAttachment(c, ws)
	if !ok {
		return
	}
	if err := h.docSvc.Delete(c.Request.Context(), ws, doc.ID); err != nil {
		errrespond.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
