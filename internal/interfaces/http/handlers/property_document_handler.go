package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appp "github.com/retechfin/retechfin-api/internal/application/patrimony"
	dom "github.com/retechfin/retechfin-api/internal/domain/patrimony"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// PropertyDocumentHandler expõe upload/listagem/download de documentos de imóveis.
type PropertyDocumentHandler struct {
	svc *appp.DocumentService
}

func NewPropertyDocumentHandler(svc *appp.DocumentService) *PropertyDocumentHandler {
	return &PropertyDocumentHandler{svc: svc}
}

type propertyDocumentResponse struct {
	ID          uuid.UUID `json:"id"`
	PropertyID  uuid.UUID `json:"property_id"`
	DocType     string    `json:"doc_type"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	Notes       *string   `json:"notes"`
	CreatedAt   string    `json:"created_at"`
}

func mapPropertyDocument(d *dom.PropertyDocument) propertyDocumentResponse {
	return propertyDocumentResponse{
		ID:          d.ID,
		PropertyID:  d.PropertyID,
		DocType:     string(d.DocType),
		FileName:    d.FileName,
		ContentType: d.ContentType,
		SizeBytes:   d.SizeBytes,
		Notes:       d.Notes,
		CreatedAt:   d.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

// Upload anexa um documento (multipart, campo 'file') ao imóvel :id.
// Campos de formulário: doc_type (obrigatório), notes.
func (h *PropertyDocumentHandler) Upload(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	propertyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}

	docType := c.PostForm("doc_type")
	if docType == "" {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "campo 'doc_type' obrigatório")
		return
	}
	var notes *string
	if v := c.PostForm("notes"); v != "" {
		notes = &v
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

	doc, err := h.svc.Upload(c.Request.Context(), appp.UploadDocInput{
		WorkspaceID:      ws,
		PropertyID:       propertyID,
		DocType:          docType,
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
	c.JSON(http.StatusCreated, mapPropertyDocument(doc))
}

// List lista os documentos do imóvel :id.
func (h *PropertyDocumentHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	propertyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	docs, err := h.svc.ListByProperty(c.Request.Context(), ws, propertyID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]propertyDocumentResponse, len(docs))
	for i := range docs {
		items[i] = mapPropertyDocument(&docs[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// DownloadURL retorna a URL presignada do documento :docId.
func (h *PropertyDocumentHandler) DownloadURL(c *gin.Context) {
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

// Delete remove o documento :docId.
func (h *PropertyDocumentHandler) Delete(c *gin.Context) {
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
