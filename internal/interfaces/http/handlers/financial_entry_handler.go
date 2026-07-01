package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/finance"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

const entryDateLayout = "2006-01-02"

type FinancialEntryHandler struct {
	svc *app.FinancialEntryService
}

func NewFinancialEntryHandler(svc *app.FinancialEntryService) *FinancialEntryHandler {
	return &FinancialEntryHandler{svc: svc}
}

type financialEntryResponse struct {
	ID                uuid.UUID  `json:"id"`
	WorkspaceID       uuid.UUID  `json:"workspace_id"`
	Kind              string     `json:"kind"`
	Status            string     `json:"status"`
	AmountCents       int64      `json:"amount_cents"`
	DueDate           string     `json:"due_date"`
	FamilyMemberID    *uuid.UUID `json:"family_member_id"`
	SourceID          *uuid.UUID `json:"source_id"`
	Type              *string    `json:"type"`
	Description       string     `json:"description"`
	Recurrence        string     `json:"recurrence"`
	RecurrenceGroupID *uuid.UUID `json:"recurrence_group_id"`
	Notes             *string    `json:"notes"`
	CreatedAt         string     `json:"created_at"`
	UpdatedAt         string     `json:"updated_at"`
}

func mapFinancialEntry(e *dom.FinancialEntry) financialEntryResponse {
	return financialEntryResponse{
		ID:                e.ID,
		WorkspaceID:       e.WorkspaceID,
		Kind:              string(e.Kind),
		Status:            string(e.Status),
		AmountCents:       e.AmountCents,
		DueDate:           e.DueDate.Format(entryDateLayout),
		FamilyMemberID:    e.FamilyMemberID,
		SourceID:          e.SourceID,
		Type:              e.Type,
		Description:       e.Description,
		Recurrence:        string(e.Recurrence),
		RecurrenceGroupID: e.RecurrenceGroupID,
		Notes:             e.Notes,
		CreatedAt:         e.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:         e.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

type financialEntryCreateJSON struct {
	Kind           string     `json:"kind" binding:"required"`
	Status         string     `json:"status"`
	AmountCents    int64      `json:"amount_cents"`
	DueDate        string     `json:"due_date" binding:"required"`
	FamilyMemberID *uuid.UUID `json:"family_member_id"`
	SourceID       *uuid.UUID `json:"source_id"`
	Type           *string    `json:"type"`
	Description    string     `json:"description"`
	Recurrence     string     `json:"recurrence"`
	Notes          *string    `json:"notes"`
}

func (h *FinancialEntryHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body financialEntryCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	due, err := time.Parse(entryDateLayout, body.DueDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "due_date inválida (use YYYY-MM-DD)")
		return
	}
	entries, err := h.svc.Create(c.Request.Context(), app.CreateEntryInput{
		WorkspaceID: ws, Kind: body.Kind, Status: body.Status, AmountCents: body.AmountCents,
		DueDate: due, FamilyMemberID: body.FamilyMemberID, SourceID: body.SourceID,
		Type: body.Type, Description: body.Description, Recurrence: body.Recurrence, Notes: body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]financialEntryResponse, len(entries))
	for i := range entries {
		items[i] = mapFinancialEntry(&entries[i])
	}
	c.JSON(http.StatusCreated, gin.H{"items": items, "total": len(items)})
}

func (h *FinancialEntryHandler) Get(c *gin.Context) {
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
	e, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapFinancialEntry(e))
}

func (h *FinancialEntryHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	limit, offset := pagination(c)

	var filter dom.FinancialEntryFilter
	if v := c.Query("kind"); v != "" {
		filter.Kind = &v
	}
	if v := c.Query("status"); v != "" {
		filter.Status = &v
	}
	if v := c.Query("type"); v != "" {
		filter.Type = &v
	}
	if v := c.Query("family_member_id"); v != "" {
		fmID, err := uuid.Parse(v)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "family_member_id inválido")
			return
		}
		filter.FamilyMemberID = &fmID
	}
	if v := c.Query("year"); v != "" {
		y, err := strconv.Atoi(v)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "year inválido")
			return
		}
		filter.Year = &y
	}
	if v := c.Query("month"); v != "" {
		m, err := strconv.Atoi(v)
		if err != nil || m < 1 || m > 12 {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "month inválido")
			return
		}
		filter.Month = &m
	}

	res, err := h.svc.List(c.Request.Context(), ws, filter, limit, offset)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]financialEntryResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapFinancialEntry(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}

type financialEntryUpdateJSON struct {
	Kind           string     `json:"kind" binding:"required"`
	Status         string     `json:"status"`
	AmountCents    int64      `json:"amount_cents"`
	DueDate        string     `json:"due_date" binding:"required"`
	FamilyMemberID *uuid.UUID `json:"family_member_id"`
	SourceID       *uuid.UUID `json:"source_id"`
	Type           *string    `json:"type"`
	Description    string     `json:"description"`
	Recurrence     string     `json:"recurrence"`
	Notes          *string    `json:"notes"`
}

func (h *FinancialEntryHandler) Update(c *gin.Context) {
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
	var body financialEntryUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	due, err := time.Parse(entryDateLayout, body.DueDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "due_date inválida (use YYYY-MM-DD)")
		return
	}
	e, err := h.svc.Update(c.Request.Context(), app.UpdateEntryInput{
		WorkspaceID: ws, ID: id, Kind: body.Kind, Status: body.Status, AmountCents: body.AmountCents,
		DueDate: due, FamilyMemberID: body.FamilyMemberID, SourceID: body.SourceID,
		Type: body.Type, Description: body.Description, Recurrence: body.Recurrence, Notes: body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapFinancialEntry(e))
}

func (h *FinancialEntryHandler) Delete(c *gin.Context) {
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

func (h *FinancialEntryHandler) Confirm(c *gin.Context) {
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
	e, err := h.svc.Confirm(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapFinancialEntry(e))
}

func (h *FinancialEntryHandler) Cancel(c *gin.Context) {
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
	e, err := h.svc.Cancel(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapFinancialEntry(e))
}
