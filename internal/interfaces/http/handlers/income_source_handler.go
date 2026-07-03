package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/finance"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

type IncomeSourceHandler struct {
	svc *app.IncomeSourceService
}

func NewIncomeSourceHandler(svc *app.IncomeSourceService) *IncomeSourceHandler {
	return &IncomeSourceHandler{svc: svc}
}

type incomeSourceResponse struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Active      bool      `json:"active"`
	Notes       *string   `json:"notes"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

func mapIncomeSource(s *dom.IncomeSource) incomeSourceResponse {
	return incomeSourceResponse{
		ID:          s.ID,
		WorkspaceID: s.WorkspaceID,
		Name:        s.Name,
		Kind:        string(s.Kind),
		Active:      s.Active,
		Notes:       s.Notes,
		CreatedAt:   s.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   s.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

type incomeSourceCreateJSON struct {
	Name   string  `json:"name" binding:"required"`
	Kind   string  `json:"kind" binding:"required"`
	Active *bool   `json:"active"`
	Notes  *string `json:"notes"`
}

func (h *IncomeSourceHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body incomeSourceCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	src, err := h.svc.Create(c.Request.Context(), app.CreateIncomeSourceInput{
		WorkspaceID: ws, Name: body.Name, Kind: body.Kind, Active: body.Active, Notes: body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapIncomeSource(src))
}

func (h *IncomeSourceHandler) Get(c *gin.Context) {
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
	src, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapIncomeSource(src))
}

func (h *IncomeSourceHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	limit, offset := pagination(c)
	filter := dom.IncomeSourceFilter{
		Query:  strings.TrimSpace(c.Query("query")),
		Kind:   c.Query("kind"),
		Active: boolQuery(c, "active"),
	}
	res, err := h.svc.List(c.Request.Context(), ws, filter, limit, offset)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]incomeSourceResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapIncomeSource(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}

type incomeSourceUpdateJSON struct {
	Name   string  `json:"name" binding:"required"`
	Kind   string  `json:"kind" binding:"required"`
	Active *bool   `json:"active"`
	Notes  *string `json:"notes"`
}

func (h *IncomeSourceHandler) Update(c *gin.Context) {
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
	var body incomeSourceUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	src, err := h.svc.Update(c.Request.Context(), app.UpdateIncomeSourceInput{
		WorkspaceID: ws, ID: id, Name: body.Name, Kind: body.Kind, Active: body.Active, Notes: body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapIncomeSource(src))
}

func (h *IncomeSourceHandler) Delete(c *gin.Context) {
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
