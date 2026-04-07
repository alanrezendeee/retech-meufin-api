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

type TransactionHandler struct {
	svc *app.TransactionService
}

func NewTransactionHandler(svc *app.TransactionService) *TransactionHandler {
	return &TransactionHandler{svc: svc}
}

type transactionCreateJSON struct {
	AccountID   uuid.UUID `json:"account_id" binding:"required"`
	CategoryID  uuid.UUID `json:"category_id" binding:"required"`
	AmountCents int64     `json:"amount_cents" binding:"required"`
	Flow        string    `json:"flow" binding:"required"`
	Description string    `json:"description"`
	OccurredAt  string    `json:"occurred_at" binding:"required"`
}

type transactionUpdateJSON struct {
	AccountID   uuid.UUID `json:"account_id" binding:"required"`
	CategoryID  uuid.UUID `json:"category_id" binding:"required"`
	AmountCents int64     `json:"amount_cents" binding:"required"`
	Flow        string    `json:"flow" binding:"required"`
	Description string    `json:"description"`
	OccurredAt  string    `json:"occurred_at" binding:"required"`
}

type transactionResponse struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	AccountID   uuid.UUID `json:"account_id"`
	CategoryID  uuid.UUID `json:"category_id"`
	AmountCents int64     `json:"amount_cents"`
	Flow        string    `json:"flow"`
	Description string    `json:"description"`
	OccurredAt  string    `json:"occurred_at"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

func mapTransaction(t *dom.Transaction) transactionResponse {
	return transactionResponse{
		ID: t.ID, WorkspaceID: t.WorkspaceID, AccountID: t.AccountID, CategoryID: t.CategoryID,
		AmountCents: t.AmountCents, Flow: string(t.Flow), Description: t.Description,
		OccurredAt: t.OccurredAt.UTC().Format(time.RFC3339Nano),
		CreatedAt:  t.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:  t.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func parseRFC3339(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}

func (h *TransactionHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body transactionCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	occ, err := parseRFC3339(body.OccurredAt)
	if err != nil {
		occ, err = time.Parse(time.RFC3339, body.OccurredAt)
	}
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "occurred_at deve ser RFC3339")
		return
	}
	tx, err := h.svc.Create(c.Request.Context(), app.CreateTransactionInput{
		WorkspaceID: ws,
		AccountID:   body.AccountID,
		CategoryID:  body.CategoryID,
		AmountCents: body.AmountCents,
		Flow:        dom.Flow(body.Flow),
		Description: body.Description,
		OccurredAt:  occ,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapTransaction(tx))
}

func (h *TransactionHandler) Get(c *gin.Context) {
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
	tx, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapTransaction(tx))
}

func (h *TransactionHandler) Update(c *gin.Context) {
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
	var body transactionUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	occ, err := parseRFC3339(body.OccurredAt)
	if err != nil {
		occ, err = time.Parse(time.RFC3339, body.OccurredAt)
	}
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "occurred_at deve ser RFC3339")
		return
	}
	tx, err := h.svc.Update(c.Request.Context(), app.UpdateTransactionInput{
		WorkspaceID: ws,
		ID:          id,
		AccountID:   body.AccountID,
		CategoryID:  body.CategoryID,
		AmountCents: body.AmountCents,
		Flow:        dom.Flow(body.Flow),
		Description: body.Description,
		OccurredAt:  occ,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapTransaction(tx))
}

func (h *TransactionHandler) Delete(c *gin.Context) {
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

func (h *TransactionHandler) List(c *gin.Context) {
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
	items := make([]transactionResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapTransaction(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}
