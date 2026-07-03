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

// FinanceAccountHandler expõe o CRUD de contas (corrente/poupança/carteira/digital).
type FinanceAccountHandler struct {
	svc *app.AccountService
}

func NewFinanceAccountHandler(svc *app.AccountService) *FinanceAccountHandler {
	return &FinanceAccountHandler{svc: svc}
}

type financeAccountResponse struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	BankName    *string   `json:"bank_name"`
	Active      bool      `json:"active"`
	Notes       *string   `json:"notes"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

func mapFinanceAccount(a *dom.Account) financeAccountResponse {
	return financeAccountResponse{
		ID:          a.ID,
		WorkspaceID: a.WorkspaceID,
		Name:        a.Name,
		Kind:        string(a.Kind),
		BankName:    a.BankName,
		Active:      a.Active,
		Notes:       a.Notes,
		CreatedAt:   a.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   a.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

type financeAccountCreateJSON struct {
	Name     string  `json:"name" binding:"required"`
	Kind     string  `json:"kind" binding:"required"`
	BankName *string `json:"bank_name"`
	Active   *bool   `json:"active"`
	Notes    *string `json:"notes"`
}

func (h *FinanceAccountHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body financeAccountCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	acc, err := h.svc.Create(c.Request.Context(), app.CreateAccountInput{
		WorkspaceID: ws, Name: body.Name, Kind: body.Kind, BankName: body.BankName, Active: body.Active, Notes: body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapFinanceAccount(acc))
}

func (h *FinanceAccountHandler) Get(c *gin.Context) {
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
	acc, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapFinanceAccount(acc))
}

func (h *FinanceAccountHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	limit, offset := pagination(c)
	filter := dom.AccountFilter{
		Query:  strings.TrimSpace(c.Query("query")),
		Kind:   c.Query("kind"),
		Active: boolQuery(c, "active"),
	}
	res, err := h.svc.List(c.Request.Context(), ws, filter, limit, offset)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]financeAccountResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapFinanceAccount(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}

type financeAccountUpdateJSON struct {
	Name     string  `json:"name" binding:"required"`
	Kind     string  `json:"kind" binding:"required"`
	BankName *string `json:"bank_name"`
	Active   *bool   `json:"active"`
	Notes    *string `json:"notes"`
}

func (h *FinanceAccountHandler) Update(c *gin.Context) {
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
	var body financeAccountUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	acc, err := h.svc.Update(c.Request.Context(), app.UpdateAccountInput{
		WorkspaceID: ws, ID: id, Name: body.Name, Kind: body.Kind, BankName: body.BankName, Active: body.Active, Notes: body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapFinanceAccount(acc))
}

func (h *FinanceAccountHandler) Delete(c *gin.Context) {
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
