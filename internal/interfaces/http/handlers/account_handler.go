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

type AccountHandler struct {
	svc *app.AccountService
}

func NewAccountHandler(svc *app.AccountService) *AccountHandler {
	return &AccountHandler{svc: svc}
}

type accountCreateJSON struct {
	Name     string `json:"name" binding:"required"`
	Currency string `json:"currency" binding:"required"`
}

type accountUpdateJSON struct {
	Name     string `json:"name" binding:"required"`
	Currency string `json:"currency" binding:"required"`
}

type accountResponse struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Name        string    `json:"name"`
	Currency    string    `json:"currency"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

func mapAccount(a *dom.Account) accountResponse {
	return accountResponse{
		ID: a.ID, WorkspaceID: a.WorkspaceID, Name: a.Name, Currency: a.Currency,
		CreatedAt: a.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: a.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (h *AccountHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body accountCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	a, err := h.svc.Create(c.Request.Context(), app.CreateAccountInput{
		WorkspaceID: ws, Name: body.Name, Currency: body.Currency,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapAccount(a))
}

func (h *AccountHandler) Get(c *gin.Context) {
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
	a, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapAccount(a))
}

func (h *AccountHandler) Update(c *gin.Context) {
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
	var body accountUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	a, err := h.svc.Update(c.Request.Context(), app.UpdateAccountInput{
		WorkspaceID: ws, ID: id, Name: body.Name, Currency: body.Currency,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapAccount(a))
}

func (h *AccountHandler) Delete(c *gin.Context) {
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

func (h *AccountHandler) List(c *gin.Context) {
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
	items := make([]accountResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapAccount(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}
