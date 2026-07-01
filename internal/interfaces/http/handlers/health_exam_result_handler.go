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

const examResultDateLayout = "2006-01-02"

type HealthExamResultHandler struct {
	svc *app.ExamResultService
}

func NewHealthExamResultHandler(svc *app.ExamResultService) *HealthExamResultHandler {
	return &HealthExamResultHandler{svc: svc}
}

// --- JSON payloads ---

type examResultItemJSON struct {
	MarkerID       *uuid.UUID `json:"marker_id"`
	RawMarkerName  *string    `json:"raw_marker_name"`
	ResultValue    string     `json:"result_value" binding:"required"`
	ResultNumeric  *float64   `json:"result_numeric"`
	Unit           *string    `json:"unit"`
	ReferenceMin   *float64   `json:"reference_min"`
	ReferenceMax   *float64   `json:"reference_max"`
	ReferenceText  *string    `json:"reference_text"`
	Interpretation *string    `json:"interpretation"`
	Method         *string    `json:"method"`
	Material       *string    `json:"material"`
	RawText        *string    `json:"raw_text"`
}

type examResultCreateJSON struct {
	FamilyMemberID uuid.UUID            `json:"family_member_id" binding:"required"`
	LabID          *uuid.UUID           `json:"lab_id"`
	ExamRequestID  *uuid.UUID           `json:"exam_request_id"`
	ExamDate       string               `json:"exam_date" binding:"required"`
	CollectionDate *string              `json:"collection_date"`
	ReleaseDate    *string              `json:"release_date"`
	SourceType     string               `json:"source_type"`
	Status         string               `json:"status"`
	Summary        *string              `json:"summary"`
	Notes          *string              `json:"notes"`
	Items          []examResultItemJSON `json:"items"`
}

type examResultUpdateJSON struct {
	FamilyMemberID uuid.UUID  `json:"family_member_id" binding:"required"`
	LabID          *uuid.UUID `json:"lab_id"`
	ExamRequestID  *uuid.UUID `json:"exam_request_id"`
	ExamDate       string     `json:"exam_date" binding:"required"`
	CollectionDate *string    `json:"collection_date"`
	ReleaseDate    *string    `json:"release_date"`
	SourceType     string     `json:"source_type"`
	Status         string     `json:"status"`
	Summary        *string    `json:"summary"`
	Notes          *string    `json:"notes"`
}

// --- responses ---

type examResultItemResponse struct {
	ID                     uuid.UUID  `json:"id"`
	WorkspaceID            uuid.UUID  `json:"workspace_id"`
	ExamResultID           uuid.UUID  `json:"exam_result_id"`
	MarkerID               *uuid.UUID `json:"marker_id"`
	RawMarkerName          *string    `json:"raw_marker_name"`
	ResultValue            string     `json:"result_value"`
	ResultNumeric          *float64   `json:"result_numeric"`
	Unit                   *string    `json:"unit"`
	ReferenceMin           *float64   `json:"reference_min"`
	ReferenceMax           *float64   `json:"reference_max"`
	ReferenceText          *string    `json:"reference_text"`
	Interpretation         *string    `json:"interpretation"`
	InterpretationComputed *string    `json:"interpretation_computed"`
	Method                 *string    `json:"method"`
	Material               *string    `json:"material"`
	RawText                *string    `json:"raw_text"`
	CreatedAt              string     `json:"created_at"`
	UpdatedAt              string     `json:"updated_at"`
}

type examResultResponse struct {
	ID             uuid.UUID                `json:"id"`
	WorkspaceID    uuid.UUID                `json:"workspace_id"`
	FamilyMemberID uuid.UUID                `json:"family_member_id"`
	LabID          *uuid.UUID               `json:"lab_id"`
	ExamRequestID  *uuid.UUID               `json:"exam_request_id"`
	ExamDate       string                   `json:"exam_date"`
	CollectionDate *string                  `json:"collection_date"`
	ReleaseDate    *string                  `json:"release_date"`
	SourceType     string                   `json:"source_type"`
	Status         string                   `json:"status"`
	Summary        *string                  `json:"summary"`
	Notes          *string                  `json:"notes"`
	Items          []examResultItemResponse `json:"items"`
	CreatedAt      string                   `json:"created_at"`
	UpdatedAt      string                   `json:"updated_at"`
}

func mapExamResultItem(it *dom.ExamResultItem) examResultItemResponse {
	return examResultItemResponse{
		ID:                     it.ID,
		WorkspaceID:            it.WorkspaceID,
		ExamResultID:           it.ExamResultID,
		MarkerID:               it.MarkerID,
		RawMarkerName:          it.RawMarkerName,
		ResultValue:            it.ResultValue,
		ResultNumeric:          it.ResultNumeric,
		Unit:                   it.Unit,
		ReferenceMin:           it.ReferenceMin,
		ReferenceMax:           it.ReferenceMax,
		ReferenceText:          it.ReferenceText,
		Interpretation:         it.Interpretation,
		InterpretationComputed: it.InterpretationComputed,
		Method:                 it.Method,
		Material:               it.Material,
		RawText:                it.RawText,
		CreatedAt:              it.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:              it.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func formatExamDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(examResultDateLayout)
	return &s
}

func mapExamResult(r *dom.ExamResult) examResultResponse {
	items := make([]examResultItemResponse, len(r.Items))
	for i := range r.Items {
		items[i] = mapExamResultItem(&r.Items[i])
	}
	return examResultResponse{
		ID:             r.ID,
		WorkspaceID:    r.WorkspaceID,
		FamilyMemberID: r.FamilyMemberID,
		LabID:          r.LabID,
		ExamRequestID:  r.ExamRequestID,
		ExamDate:       r.ExamDate.UTC().Format(examResultDateLayout),
		CollectionDate: formatExamDatePtr(r.CollectionDate),
		ReleaseDate:    formatExamDatePtr(r.ReleaseDate),
		SourceType:     string(r.SourceType),
		Status:         string(r.Status),
		Summary:        r.Summary,
		Notes:          r.Notes,
		Items:          items,
		CreatedAt:      r.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:      r.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func parseExamDate(c *gin.Context, field, raw string) (time.Time, bool) {
	t, err := time.Parse(examResultDateLayout, raw)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, field+" inválido (use YYYY-MM-DD)")
		return time.Time{}, false
	}
	return t, true
}

func parseExamDatePtr(c *gin.Context, field string, raw *string) (*time.Time, bool) {
	if raw == nil || *raw == "" {
		return nil, true
	}
	t, err := time.Parse(examResultDateLayout, *raw)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, field+" inválido (use YYYY-MM-DD)")
		return nil, false
	}
	return &t, true
}

func toItemInput(j examResultItemJSON) app.ExamResultItemInput {
	return app.ExamResultItemInput{
		MarkerID:       j.MarkerID,
		RawMarkerName:  j.RawMarkerName,
		ResultValue:    j.ResultValue,
		ResultNumeric:  j.ResultNumeric,
		Unit:           j.Unit,
		ReferenceMin:   j.ReferenceMin,
		ReferenceMax:   j.ReferenceMax,
		ReferenceText:  j.ReferenceText,
		Interpretation: j.Interpretation,
		Method:         j.Method,
		Material:       j.Material,
		RawText:        j.RawText,
	}
}

// --- handlers do resultado ---

func (h *HealthExamResultHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body examResultCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	examDate, ok := parseExamDate(c, "exam_date", body.ExamDate)
	if !ok {
		return
	}
	collection, ok := parseExamDatePtr(c, "collection_date", body.CollectionDate)
	if !ok {
		return
	}
	release, ok := parseExamDatePtr(c, "release_date", body.ReleaseDate)
	if !ok {
		return
	}
	items := make([]app.CreateExamResultItemInput, len(body.Items))
	for i, it := range body.Items {
		in := toItemInput(it)
		items[i] = app.CreateExamResultItemInput{
			MarkerID:       in.MarkerID,
			RawMarkerName:  in.RawMarkerName,
			ResultValue:    in.ResultValue,
			ResultNumeric:  in.ResultNumeric,
			Unit:           in.Unit,
			ReferenceMin:   in.ReferenceMin,
			ReferenceMax:   in.ReferenceMax,
			ReferenceText:  in.ReferenceText,
			Interpretation: in.Interpretation,
			Method:         in.Method,
			Material:       in.Material,
			RawText:        in.RawText,
		}
	}
	r, err := h.svc.Create(c.Request.Context(), app.CreateExamResultInput{
		WorkspaceID:    ws,
		FamilyMemberID: body.FamilyMemberID,
		LabID:          body.LabID,
		ExamRequestID:  body.ExamRequestID,
		ExamDate:       examDate,
		CollectionDate: collection,
		ReleaseDate:    release,
		SourceType:     body.SourceType,
		Status:         body.Status,
		Summary:        body.Summary,
		Notes:          body.Notes,
		Items:          items,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapExamResult(r))
}

func (h *HealthExamResultHandler) Get(c *gin.Context) {
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
	c.JSON(http.StatusOK, mapExamResult(r))
}

func (h *HealthExamResultHandler) List(c *gin.Context) {
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
	items := make([]examResultResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapExamResult(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}

func (h *HealthExamResultHandler) Update(c *gin.Context) {
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
	var body examResultUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	examDate, ok := parseExamDate(c, "exam_date", body.ExamDate)
	if !ok {
		return
	}
	collection, ok := parseExamDatePtr(c, "collection_date", body.CollectionDate)
	if !ok {
		return
	}
	release, ok := parseExamDatePtr(c, "release_date", body.ReleaseDate)
	if !ok {
		return
	}
	r, err := h.svc.Update(c.Request.Context(), app.UpdateExamResultInput{
		WorkspaceID:    ws,
		ID:             id,
		FamilyMemberID: body.FamilyMemberID,
		LabID:          body.LabID,
		ExamRequestID:  body.ExamRequestID,
		ExamDate:       examDate,
		CollectionDate: collection,
		ReleaseDate:    release,
		SourceType:     body.SourceType,
		Status:         body.Status,
		Summary:        body.Summary,
		Notes:          body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapExamResult(r))
}

func (h *HealthExamResultHandler) Delete(c *gin.Context) {
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

// --- handlers dos itens ---

func (h *HealthExamResultHandler) AddItem(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	resultID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	var body examResultItemJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	item, err := h.svc.AddItem(c.Request.Context(), app.AddExamResultItemInput{
		WorkspaceID:  ws,
		ExamResultID: resultID,
		Item:         toItemInput(body),
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapExamResultItem(item))
}

func (h *HealthExamResultHandler) UpdateItem(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	resultID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "itemId inválido")
		return
	}
	var body examResultItemJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	item, err := h.svc.UpdateItem(c.Request.Context(), app.UpdateExamResultItemInput{
		WorkspaceID:  ws,
		ExamResultID: resultID,
		ItemID:       itemID,
		Item:         toItemInput(body),
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapExamResultItem(item))
}

func (h *HealthExamResultHandler) DeleteItem(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	resultID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "itemId inválido")
		return
	}
	if err := h.svc.DeleteItem(c.Request.Context(), ws, resultID, itemID); err != nil {
		errrespond.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
