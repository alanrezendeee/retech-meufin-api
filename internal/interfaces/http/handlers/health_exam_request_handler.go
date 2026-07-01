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

const examRequestDateLayout = "2006-01-02"

type HealthExamRequestHandler struct {
	svc *app.ExamRequestService
}

func NewHealthExamRequestHandler(svc *app.ExamRequestService) *HealthExamRequestHandler {
	return &HealthExamRequestHandler{svc: svc}
}

// --- payloads ---

type examRequestItemCreateJSON struct {
	MarkerID *uuid.UUID `json:"marker_id"`
	ExamName string     `json:"exam_name" binding:"required"`
	ExamCode *string    `json:"exam_code"`
	BodyArea *string    `json:"body_area"`
	Notes    *string    `json:"notes"`
	Status   string     `json:"status"`
}

type examRequestCreateJSON struct {
	FamilyMemberID uuid.UUID                   `json:"family_member_id" binding:"required"`
	LabID          *uuid.UUID                  `json:"lab_id"`
	RequestedBy    *string                     `json:"requested_by"`
	RequestDate    *string                     `json:"request_date"`
	Status         string                      `json:"status"`
	Notes          *string                     `json:"notes"`
	Items          []examRequestItemCreateJSON `json:"items"`
}

type examRequestUpdateJSON struct {
	FamilyMemberID uuid.UUID  `json:"family_member_id" binding:"required"`
	LabID          *uuid.UUID `json:"lab_id"`
	RequestedBy    *string    `json:"requested_by"`
	RequestDate    *string    `json:"request_date"`
	Status         string     `json:"status"`
	Notes          *string    `json:"notes"`
}

type examRequestItemUpsertJSON struct {
	MarkerID *uuid.UUID `json:"marker_id"`
	ExamName string     `json:"exam_name" binding:"required"`
	ExamCode *string    `json:"exam_code"`
	BodyArea *string    `json:"body_area"`
	Notes    *string    `json:"notes"`
	Status   string     `json:"status"`
}

// --- responses ---

type examRequestItemResponse struct {
	ID            uuid.UUID  `json:"id"`
	WorkspaceID   uuid.UUID  `json:"workspace_id"`
	ExamRequestID uuid.UUID  `json:"exam_request_id"`
	MarkerID      *uuid.UUID `json:"marker_id"`
	ExamName      string     `json:"exam_name"`
	ExamCode      *string    `json:"exam_code"`
	BodyArea      *string    `json:"body_area"`
	Notes         *string    `json:"notes"`
	Status        string     `json:"status"`
	CreatedAt     string     `json:"created_at"`
	UpdatedAt     string     `json:"updated_at"`
}

type examRequestResponse struct {
	ID             uuid.UUID                 `json:"id"`
	WorkspaceID    uuid.UUID                 `json:"workspace_id"`
	FamilyMemberID uuid.UUID                 `json:"family_member_id"`
	LabID          *uuid.UUID                `json:"lab_id"`
	RequestedBy    *string                   `json:"requested_by"`
	RequestDate    string                    `json:"request_date"`
	Status         string                    `json:"status"`
	Notes          *string                   `json:"notes"`
	Items          []examRequestItemResponse `json:"items"`
	CreatedAt      string                    `json:"created_at"`
	UpdatedAt      string                    `json:"updated_at"`
}

func mapExamRequestItem(it *dom.ExamRequestItem) examRequestItemResponse {
	return examRequestItemResponse{
		ID:            it.ID,
		WorkspaceID:   it.WorkspaceID,
		ExamRequestID: it.ExamRequestID,
		MarkerID:      it.MarkerID,
		ExamName:      it.ExamName,
		ExamCode:      it.ExamCode,
		BodyArea:      it.BodyArea,
		Notes:         it.Notes,
		Status:        string(it.Status),
		CreatedAt:     it.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:     it.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func mapExamRequest(r *dom.ExamRequest) examRequestResponse {
	items := make([]examRequestItemResponse, len(r.Items))
	for i := range r.Items {
		items[i] = mapExamRequestItem(&r.Items[i])
	}
	return examRequestResponse{
		ID:             r.ID,
		WorkspaceID:    r.WorkspaceID,
		FamilyMemberID: r.FamilyMemberID,
		LabID:          r.LabID,
		RequestedBy:    r.RequestedBy,
		RequestDate:    r.RequestDate.UTC().Format(examRequestDateLayout),
		Status:         string(r.Status),
		Notes:          r.Notes,
		Items:          items,
		CreatedAt:      r.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:      r.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

// parseExamRequestDate interpreta a data no formato "2006-01-02".
// Retorna (nil, nil) quando ausente e (nil, err) quando inválida.
func parseExamRequestDate(raw *string) (*time.Time, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	t, err := time.Parse(examRequestDateLayout, *raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// --- exam request endpoints ---

func (h *HealthExamRequestHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body examRequestCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	reqDate, err := parseExamRequestDate(body.RequestDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "request_date inválido (use YYYY-MM-DD)")
		return
	}

	items := make([]app.CreateExamRequestItemInput, len(body.Items))
	for i := range body.Items {
		items[i] = app.CreateExamRequestItemInput{
			MarkerID: body.Items[i].MarkerID,
			ExamName: body.Items[i].ExamName,
			ExamCode: body.Items[i].ExamCode,
			BodyArea: body.Items[i].BodyArea,
			Notes:    body.Items[i].Notes,
			Status:   dom.ExamRequestItemStatus(body.Items[i].Status),
		}
	}

	r, err := h.svc.Create(c.Request.Context(), app.CreateExamRequestInput{
		WorkspaceID:    ws,
		FamilyMemberID: body.FamilyMemberID,
		LabID:          body.LabID,
		RequestedBy:    body.RequestedBy,
		RequestDate:    reqDate,
		Status:         dom.ExamRequestStatus(body.Status),
		Notes:          body.Notes,
		Items:          items,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapExamRequest(r))
}

func (h *HealthExamRequestHandler) Get(c *gin.Context) {
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
	r, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapExamRequest(r))
}

func (h *HealthExamRequestHandler) List(c *gin.Context) {
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
	items := make([]examRequestResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapExamRequest(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}

func (h *HealthExamRequestHandler) Update(c *gin.Context) {
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
	var body examRequestUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	reqDate, err := parseExamRequestDate(body.RequestDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "request_date inválido (use YYYY-MM-DD)")
		return
	}
	r, err := h.svc.Update(c.Request.Context(), app.UpdateExamRequestInput{
		WorkspaceID:    ws,
		ID:             id,
		FamilyMemberID: body.FamilyMemberID,
		LabID:          body.LabID,
		RequestedBy:    body.RequestedBy,
		RequestDate:    reqDate,
		Status:         dom.ExamRequestStatus(body.Status),
		Notes:          body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapExamRequest(r))
}

func (h *HealthExamRequestHandler) Delete(c *gin.Context) {
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

// --- exam request item endpoints ---

func (h *HealthExamRequestHandler) AddItem(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	reqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	var body examRequestItemUpsertJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	it, err := h.svc.AddItem(c.Request.Context(), app.AddExamRequestItemInput{
		WorkspaceID:   ws,
		ExamRequestID: reqID,
		MarkerID:      body.MarkerID,
		ExamName:      body.ExamName,
		ExamCode:      body.ExamCode,
		BodyArea:      body.BodyArea,
		Notes:         body.Notes,
		Status:        dom.ExamRequestItemStatus(body.Status),
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapExamRequestItem(it))
}

func (h *HealthExamRequestHandler) UpdateItem(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	reqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "itemId inválido")
		return
	}
	var body examRequestItemUpsertJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	it, err := h.svc.UpdateItem(c.Request.Context(), app.UpdateExamRequestItemInput{
		WorkspaceID:   ws,
		ExamRequestID: reqID,
		ItemID:        itemID,
		MarkerID:      body.MarkerID,
		ExamName:      body.ExamName,
		ExamCode:      body.ExamCode,
		BodyArea:      body.BodyArea,
		Notes:         body.Notes,
		Status:        dom.ExamRequestItemStatus(body.Status),
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapExamRequestItem(it))
}

func (h *HealthExamRequestHandler) DeleteItem(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	reqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "itemId inválido")
		return
	}
	if err := h.svc.DeleteItem(c.Request.Context(), ws, reqID, itemID); err != nil {
		errrespond.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
