package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appb "github.com/retechfin/retechfin-api/internal/application/budget"
	domb "github.com/retechfin/retechfin-api/internal/domain/budget"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

type BudgetHandler struct {
	svc *appb.Service
}

func NewBudgetHandler(svc *appb.Service) *BudgetHandler {
	return &BudgetHandler{svc: svc}
}

type budgetCreateJSON struct {
	CategoryID uuid.UUID `json:"category_id" binding:"required"`
	Year       int       `json:"year" binding:"required"`
	Month      int       `json:"month" binding:"required"`
	LimitCents int64     `json:"limit_cents" binding:"required"`
}

type budgetUpdateJSON struct {
	LimitCents int64 `json:"limit_cents" binding:"required"`
}

type budgetResponse struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	CategoryID  uuid.UUID `json:"category_id"`
	Year        int       `json:"year"`
	Month       int       `json:"month"`
	LimitCents  int64     `json:"limit_cents"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

func mapBudget(b *domb.Budget) budgetResponse {
	return budgetResponse{
		ID: b.ID, WorkspaceID: b.WorkspaceID, CategoryID: b.CategoryID,
		Year: b.Year, Month: b.Month, LimitCents: b.LimitCents,
		CreatedAt: b.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: b.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (h *BudgetHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body budgetCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	b, err := h.svc.Create(c.Request.Context(), appb.CreateBudgetInput{
		WorkspaceID: ws,
		CategoryID:  body.CategoryID,
		Year:        body.Year,
		Month:       body.Month,
		LimitCents:  body.LimitCents,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapBudget(b))
}

func (h *BudgetHandler) Get(c *gin.Context) {
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
	b, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapBudget(b))
}

func (h *BudgetHandler) Update(c *gin.Context) {
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
	var body budgetUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	b, err := h.svc.Update(c.Request.Context(), appb.UpdateBudgetInput{
		WorkspaceID: ws, ID: id, LimitCents: body.LimitCents,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapBudget(b))
}

func (h *BudgetHandler) Delete(c *gin.Context) {
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

func (h *BudgetHandler) List(c *gin.Context) {
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
	items := make([]budgetResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapBudget(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}

type validateBudgetJSON struct {
	Year  int `json:"year" binding:"required"`
	Month int `json:"month" binding:"required"`
}

func (h *BudgetHandler) Validate(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body validateBudgetJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	lines, err := h.svc.ValidateBudget(c.Request.Context(), appb.ValidateBudgetInput{
		WorkspaceID: ws, Year: body.Year, Month: body.Month,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"lines": lines})
}
