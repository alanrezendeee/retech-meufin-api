package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/finance"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// FinanceCategoryHandler expõe o CRUD das categorias de despesa do workspace.
type FinanceCategoryHandler struct {
	svc *app.ExpenseCategoryService
}

func NewFinanceCategoryHandler(svc *app.ExpenseCategoryService) *FinanceCategoryHandler {
	return &FinanceCategoryHandler{svc: svc}
}

type expenseCategoryResponse struct {
	ID        uuid.UUID `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	GroupSlug string    `json:"group_slug"`
	GroupName string    `json:"group_name"`
	Active    bool      `json:"active"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

func mapExpenseCategory(c *dom.ExpenseCategory) expenseCategoryResponse {
	return expenseCategoryResponse{
		ID:        c.ID,
		Slug:      c.Slug,
		Name:      c.Name,
		GroupSlug: c.GroupSlug,
		GroupName: dom.ExpenseGroups[c.GroupSlug],
		Active:    c.Active,
		CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: c.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

// List retorna as categorias do workspace (semeia as padrão no primeiro uso)
// e o catálogo curado de grupos (pro select do front — fonte única).
func (h *FinanceCategoryHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	cats, err := h.svc.List(c.Request.Context(), ws)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]expenseCategoryResponse, len(cats))
	for i := range cats {
		items[i] = mapExpenseCategory(&cats[i])
	}
	groups := make([]gin.H, 0, len(dom.ExpenseGroups))
	for slug, name := range dom.ExpenseGroups {
		groups = append(groups, gin.H{"slug": slug, "name": name})
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items), "groups": groups})
}

type expenseCategoryCreateJSON struct {
	Name      string `json:"name" binding:"required"`
	GroupSlug string `json:"group_slug" binding:"required"`
}

func (h *FinanceCategoryHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body expenseCategoryCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	cat, err := h.svc.Create(c.Request.Context(), app.CreateExpenseCategoryInput{
		WorkspaceID: ws, Name: body.Name, GroupSlug: body.GroupSlug,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapExpenseCategory(cat))
}

type expenseCategoryUpdateJSON struct {
	Name      string `json:"name" binding:"required"`
	GroupSlug string `json:"group_slug" binding:"required"`
	Active    *bool  `json:"active"`
}

func (h *FinanceCategoryHandler) Update(c *gin.Context) {
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
	var body expenseCategoryUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	cat, err := h.svc.Update(c.Request.Context(), app.UpdateExpenseCategoryInput{
		WorkspaceID: ws, ID: id, Name: body.Name, GroupSlug: body.GroupSlug, Active: body.Active,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapExpenseCategory(cat))
}

func (h *FinanceCategoryHandler) Delete(c *gin.Context) {
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
