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

const planDocDateLayout = "2006-01-02"

// HealthPlanDocumentHandler expõe upload/listagem/download de documentos de um
// plano de saúde (contrato/apólice, carteirinha, boletos etc.).
type HealthPlanDocumentHandler struct {
	svc *app.PlanDocumentService
}

func NewHealthPlanDocumentHandler(svc *app.PlanDocumentService) *HealthPlanDocumentHandler {
	return &HealthPlanDocumentHandler{svc: svc}
}

type planDocumentResponse struct {
	ID               uuid.UUID `json:"id"`
	PlanID           uuid.UUID `json:"plan_id"`
	DocType          string    `json:"doc_type"`
	Label            *string   `json:"label"`
	DocNumber        *string   `json:"doc_number"`
	ValidUntil       *string   `json:"valid_until"`
	Notes            *string   `json:"notes"`
	FileName         string    `json:"file_name"`
	OriginalFileName string    `json:"original_file_name"`
	MimeType         string    `json:"mime_type"`
	SizeBytes        int64     `json:"size_bytes"`
	CreatedAt        string    `json:"created_at"`
	UpdatedAt        string    `json:"updated_at"`
}

func mapPlanDocument(d *dom.PlanDocument) planDocumentResponse {
	var validUntil *string
	if d.ValidUntil != nil {
		v := d.ValidUntil.Format(planDocDateLayout)
		validUntil = &v
	}
	return planDocumentResponse{
		ID:               d.ID,
		PlanID:           d.PlanID,
		DocType:          string(d.DocType),
		Label:            d.Label,
		DocNumber:        d.DocNumber,
		ValidUntil:       validUntil,
		Notes:            d.Notes,
		FileName:         d.FileName,
		OriginalFileName: d.OriginalFileName,
		MimeType:         d.MimeType,
		SizeBytes:        d.SizeBytes,
		CreatedAt:        d.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:        d.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

// Upload anexa um documento (multipart, campo 'file') ao plano :id.
// Campos: doc_type (obrigatório), label, doc_number, valid_until (YYYY-MM-DD), notes.
func (h *HealthPlanDocumentHandler) Upload(c *gin.Context) {
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
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}

	docType := c.PostForm("doc_type")
	if docType == "" {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "campo 'doc_type' obrigatório")
		return
	}
	var label, docNumber, notes *string
	if v := c.PostForm("label"); v != "" {
		label = &v
	}
	if v := c.PostForm("doc_number"); v != "" {
		docNumber = &v
	}
	if v := c.PostForm("notes"); v != "" {
		notes = &v
	}
	var validUntil *time.Time
	if v := c.PostForm("valid_until"); v != "" {
		t, err := time.Parse(planDocDateLayout, v)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "valid_until inválida (use YYYY-MM-DD)")
			return
		}
		validUntil = &t
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

	doc, err := h.svc.Upload(c.Request.Context(), app.UploadPlanDocInput{
		WorkspaceID:      ws,
		PlanID:           planID,
		UploadedByUserID: userID,
		DocType:          docType,
		Label:            label,
		DocNumber:        docNumber,
		ValidUntil:       validUntil,
		Notes:            notes,
		OriginalFileName: fileHeader.Filename,
		MimeType:         fileHeader.Header.Get("Content-Type"),
		Size:             fileHeader.Size,
		Content:          f,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapPlanDocument(doc))
}

// List lista os documentos do plano :id.
func (h *HealthPlanDocumentHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	limit, offset := pagination(c)
	res, err := h.svc.ListByPlan(c.Request.Context(), ws, planID, limit, offset)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]planDocumentResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapPlanDocument(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}

// DownloadURL retorna a URL presignada do documento :docId.
func (h *HealthPlanDocumentHandler) DownloadURL(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	docID, err := uuid.Parse(c.Param("docId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "docId inválido")
		return
	}
	url, err := h.svc.DownloadURL(c.Request.Context(), ws, docID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

// Delete remove (soft) o documento :docId.
func (h *HealthPlanDocumentHandler) Delete(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	docID, err := uuid.Parse(c.Param("docId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "docId inválido")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), ws, docID); err != nil {
		errrespond.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
