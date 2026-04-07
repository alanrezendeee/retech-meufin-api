package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/ledger"
	dom "github.com/retechfin/retechfin-api/internal/domain/ledger"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

type CategoryHandler struct {
	svc *app.CategoryService
}

func NewCategoryHandler(svc *app.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

type categoryCreateJSON struct {
	Name     string     `json:"name" binding:"required"`
	Kind     string     `json:"kind" binding:"required"`
	ParentID *uuid.UUID `json:"parent_id"`
}

type categoryUpdateJSON struct {
	Name     string     `json:"name" binding:"required"`
	Kind     string     `json:"kind" binding:"required"`
	ParentID *uuid.UUID `json:"parent_id"`
}

type categoryResponse struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	Name        string     `json:"name"`
	Kind        string     `json:"kind"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
}

func mapCategory(c *dom.Category) categoryResponse {
	return categoryResponse{
		ID: c.ID, WorkspaceID: c.WorkspaceID, Name: c.Name, Kind: string(c.Kind),
		ParentID: c.ParentID,
		CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: c.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (h *CategoryHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body categoryCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	cat, err := h.svc.Create(c.Request.Context(), app.CreateCategoryInput{
		WorkspaceID: ws,
		Name:        body.Name,
		Kind:        dom.CategoryKind(body.Kind),
		ParentID:    body.ParentID,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapCategory(cat))
}

func (h *CategoryHandler) Get(c *gin.Context) {
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
	cat, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapCategory(cat))
}

func (h *CategoryHandler) Update(c *gin.Context) {
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
	var body categoryUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	cat, err := h.svc.Update(c.Request.Context(), app.UpdateCategoryInput{
		WorkspaceID: ws,
		ID:          id,
		Name:        body.Name,
		Kind:        dom.CategoryKind(body.Kind),
		ParentID:    body.ParentID,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapCategory(cat))
}

func (h *CategoryHandler) Delete(c *gin.Context) {
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

func (h *CategoryHandler) List(c *gin.Context) {
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
	items := make([]categoryResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapCategory(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}
