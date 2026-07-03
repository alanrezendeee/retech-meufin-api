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

type CreditCardHandler struct {
	svc *app.CreditCardService
}

func NewCreditCardHandler(svc *app.CreditCardService) *CreditCardHandler {
	return &CreditCardHandler{svc: svc}
}

type creditCardResponse struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Name        string    `json:"name"`
	Brand       *string   `json:"brand"`
	ClosingDay  *int      `json:"closing_day"`
	DueDay      *int      `json:"due_day"`
	Active      bool      `json:"active"`
	Notes       *string   `json:"notes"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

func mapCreditCard(c *dom.CreditCard) creditCardResponse {
	return creditCardResponse{
		ID:          c.ID,
		WorkspaceID: c.WorkspaceID,
		Name:        c.Name,
		Brand:       c.Brand,
		ClosingDay:  c.ClosingDay,
		DueDay:      c.DueDay,
		Active:      c.Active,
		Notes:       c.Notes,
		CreatedAt:   c.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   c.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

type creditCardCreateJSON struct {
	Name       string  `json:"name" binding:"required"`
	Brand      *string `json:"brand"`
	ClosingDay *int    `json:"closing_day"`
	DueDay     *int    `json:"due_day"`
	Active     *bool   `json:"active"`
	Notes      *string `json:"notes"`
}

func (h *CreditCardHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body creditCardCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	card, err := h.svc.Create(c.Request.Context(), app.CreateCreditCardInput{
		WorkspaceID: ws, Name: body.Name, Brand: body.Brand,
		ClosingDay: body.ClosingDay, DueDay: body.DueDay, Active: body.Active, Notes: body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapCreditCard(card))
}

func (h *CreditCardHandler) Get(c *gin.Context) {
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
	card, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapCreditCard(card))
}

func (h *CreditCardHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	limit, offset := pagination(c)
	filter := dom.CreditCardFilter{
		Query:  strings.TrimSpace(c.Query("query")),
		Active: boolQuery(c, "active"),
	}
	res, err := h.svc.List(c.Request.Context(), ws, filter, limit, offset)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]creditCardResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapCreditCard(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}

type creditCardUpdateJSON struct {
	Name       string  `json:"name" binding:"required"`
	Brand      *string `json:"brand"`
	ClosingDay *int    `json:"closing_day"`
	DueDay     *int    `json:"due_day"`
	Active     *bool   `json:"active"`
	Notes      *string `json:"notes"`
}

func (h *CreditCardHandler) Update(c *gin.Context) {
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
	var body creditCardUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	card, err := h.svc.Update(c.Request.Context(), app.UpdateCreditCardInput{
		WorkspaceID: ws, ID: id, Name: body.Name, Brand: body.Brand,
		ClosingDay: body.ClosingDay, DueDay: body.DueDay, Active: body.Active, Notes: body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapCreditCard(card))
}

func (h *CreditCardHandler) Delete(c *gin.Context) {
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
