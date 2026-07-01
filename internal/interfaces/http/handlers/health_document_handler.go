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

type HealthDocumentHandler struct {
	svc *app.DocumentService
}

func NewHealthDocumentHandler(svc *app.DocumentService) *HealthDocumentHandler {
	return &HealthDocumentHandler{svc: svc}
}

// documentResponse omite o object_key/bucket crus, expondo apenas o essencial.
type documentResponse struct {
	ID               uuid.UUID  `json:"id"`
	DocumentType     string     `json:"document_type"`
	FamilyMemberID   *uuid.UUID `json:"family_member_id,omitempty"`
	LabID            *uuid.UUID `json:"lab_id,omitempty"`
	ExamRequestID    *uuid.UUID `json:"exam_request_id,omitempty"`
	ExamResultID     *uuid.UUID `json:"exam_result_id,omitempty"`
	FileName         string     `json:"file_name"`
	OriginalFileName string     `json:"original_file_name"`
	MimeType         string     `json:"mime_type"`
	SizeBytes        int64      `json:"size_bytes"`
	ExtractionStatus string     `json:"extraction_status"`
	CreatedAt        string     `json:"created_at"`
	UpdatedAt        string     `json:"updated_at"`
}

func mapDocument(d *dom.Document) documentResponse {
	return documentResponse{
		ID:               d.ID,
		DocumentType:     string(d.DocumentType),
		FamilyMemberID:   d.FamilyMemberID,
		LabID:            d.LabID,
		ExamRequestID:    d.ExamRequestID,
		ExamResultID:     d.ExamResultID,
		FileName:         d.FileName,
		OriginalFileName: d.OriginalFileName,
		MimeType:         d.MimeType,
		SizeBytes:        d.SizeBytes,
		ExtractionStatus: string(d.ExtractionStatus),
		CreatedAt:        d.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:        d.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

// userIDFromCtx extrai o user id (string UUID) setado por RequireAuth.
func userIDFromCtx(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(middleware.CtxUserID)
	if !ok {
		return uuid.Nil, false
	}
	s, ok := v.(string)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// optionalUUIDForm lê um campo de formulário opcional como *uuid.UUID.
// Retorna (nil,nil) quando ausente; erro quando presente mas inválido.
func optionalUUIDForm(c *gin.Context, field string) (*uuid.UUID, error) {
	raw := c.PostForm(field)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (h *HealthDocumentHandler) Upload(c *gin.Context) {
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
	docType := c.PostForm("document_type")
	if docType == "" {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "campo 'document_type' obrigatório")
		return
	}

	familyMemberID, err := optionalUUIDForm(c, "family_member_id")
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "family_member_id inválido")
		return
	}
	labID, err := optionalUUIDForm(c, "lab_id")
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "lab_id inválido")
		return
	}
	examRequestID, err := optionalUUIDForm(c, "exam_request_id")
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "exam_request_id inválido")
		return
	}
	examResultID, err := optionalUUIDForm(c, "exam_result_id")
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "exam_result_id inválido")
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "não foi possível ler o arquivo")
		return
	}
	defer f.Close()

	mimeType := fileHeader.Header.Get("Content-Type")

	doc, err := h.svc.Upload(c.Request.Context(), app.UploadDocumentInput{
		WorkspaceID:      ws,
		UploadedByUserID: userID,
		DocumentType:     docType,
		FamilyMemberID:   familyMemberID,
		LabID:            labID,
		ExamRequestID:    examRequestID,
		ExamResultID:     examResultID,
		OriginalFileName: fileHeader.Filename,
		MimeType:         mimeType,
		Size:             fileHeader.Size,
		Content:          f,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapDocument(doc))
}

func (h *HealthDocumentHandler) Get(c *gin.Context) {
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
	c.JSON(http.StatusOK, mapDocument(doc))
}

func (h *HealthDocumentHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	limit, offset := pagination(c)
	res, err := h.svc.List(c.Request.Context(), ws, limit, offset)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]documentResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapDocument(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}

func (h *HealthDocumentHandler) Delete(c *gin.Context) {
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

func (h *HealthDocumentHandler) DownloadURL(c *gin.Context) {
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
