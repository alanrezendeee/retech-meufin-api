package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/finance"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// ReconciliationHandler expõe a conciliação cupom × fatura de cartão.
// Só SUGERE; a aplicação exige confirmação explícita do usuário.
type ReconciliationHandler struct {
	svc *app.ReconciliationService
}

func NewReconciliationHandler(svc *app.ReconciliationService) *ReconciliationHandler {
	return &ReconciliationHandler{svc: svc}
}

type reconcileMatchJSON struct {
	// Compra da fatura
	PurchaseEntryID     string `json:"purchase_entry_id"`
	PurchaseDescription string `json:"purchase_description"`
	PurchaseDate        string `json:"purchase_date"`
	AmountCents         int64  `json:"amount_cents"`
	// Cupom (despesa avulsa) que casa com a compra
	CupomEntryID string `json:"cupom_entry_id"`
	DocumentID   string `json:"document_id"`
	CupomMerchant string `json:"cupom_merchant"`
	CupomDate     string `json:"cupom_date"`
	DaysDiff      int    `json:"days_diff"`
}

// Suggest responde GET /finance/entries/:id/reconciliation — sugestões de
// conciliação para uma fatura (:id). Não altera nada.
func (h *ReconciliationHandler) Suggest(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	matches, err := h.svc.SuggestForInvoice(c.Request.Context(), ws, invoiceID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	out := make([]reconcileMatchJSON, len(matches))
	for i, m := range matches {
		out[i] = reconcileMatchJSON{
			PurchaseEntryID:     m.Purchase.EntryID.String(),
			PurchaseDescription: m.Purchase.Description,
			PurchaseDate:        m.Purchase.Date.UTC().Format(entryDateLayout),
			AmountCents:         m.Purchase.AmountCents,
			CupomEntryID:        m.Cupom.CupomEntryID.String(),
			DocumentID:          m.Cupom.DocumentID.String(),
			CupomMerchant:       m.Cupom.Merchant,
			CupomDate:           m.Cupom.Date.UTC().Format(entryDateLayout),
			DaysDiff:            m.DaysDiff,
		}
	}
	c.JSON(http.StatusOK, gin.H{"matches": out, "total": len(out)})
}

type reconcileRequest struct {
	CupomEntryID  string `json:"cupom_entry_id" binding:"required"`
	TargetEntryID string `json:"target_entry_id" binding:"required"`
}

// Reconcile responde POST /finance/reconcile — aplica a conciliação escolhida:
// move o detalhamento do cupom para a compra da fatura e remove a despesa avulsa.
func (h *ReconciliationHandler) Reconcile(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body reconcileRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	cupomID, err := uuid.Parse(body.CupomEntryID)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "cupom_entry_id inválido")
		return
	}
	targetID, err := uuid.Parse(body.TargetEntryID)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "target_entry_id inválido")
		return
	}
	entry, err := h.svc.Reconcile(c.Request.Context(), ws, cupomID, targetID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"entry": mapInvoiceEntry(entry)})
}
