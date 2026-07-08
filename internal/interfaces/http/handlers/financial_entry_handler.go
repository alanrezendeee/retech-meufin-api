package handlers

import (
	"net/http"
	"strconv"
	"strings"
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
	CardID            *uuid.UUID `json:"card_id"`
	ParentID          *uuid.UUID `json:"parent_id"`
	InstallmentNumber *int       `json:"installment_number"`
	InstallmentTotal  *int       `json:"installment_total"`
	Notes             *string    `json:"notes"`
	PaidAt            *string    `json:"paid_at"`
	PaidAmountCents   *int64     `json:"paid_amount_cents"`
	PaymentMethod     *string    `json:"payment_method"`
	PaymentAccountID  *uuid.UUID `json:"payment_account_id"`
	PaymentCardID     *uuid.UUID `json:"payment_card_id"`
	DiscountCents     *int64     `json:"discount_cents"`
	DiscountReason    *string    `json:"discount_reason"`
	ResidualOfID      *uuid.UUID `json:"residual_of_id"`
	PurchaseDate      *string    `json:"purchase_date"`
	FiscalDocumentID  *uuid.UUID `json:"fiscal_document_id"`
	SupplierID        *uuid.UUID `json:"supplier_id"`
	CreatedAt         string     `json:"created_at"`
	UpdatedAt         string     `json:"updated_at"`
}

func mapFinancialEntry(e *dom.FinancialEntry) financialEntryResponse {
	var paidAt *string
	if e.PaidAt != nil {
		v := e.PaidAt.UTC().Format(time.RFC3339Nano)
		paidAt = &v
	}
	var paymentMethod *string
	if e.PaymentMethod != nil {
		v := string(*e.PaymentMethod)
		paymentMethod = &v
	}
	var purchaseDate *string
	if e.PurchaseDate != nil {
		v := e.PurchaseDate.Format(entryDateLayout)
		purchaseDate = &v
	}
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
		CardID:            e.CardID,
		ParentID:          e.ParentID,
		InstallmentNumber: e.InstallmentNumber,
		InstallmentTotal:  e.InstallmentTotal,
		Notes:             e.Notes,
		PaidAt:            paidAt,
		PaidAmountCents:   e.PaidAmountCents,
		PaymentMethod:     paymentMethod,
		PaymentAccountID:  e.PaymentAccountID,
		PaymentCardID:     e.PaymentCardID,
		DiscountCents:     e.DiscountCents,
		DiscountReason:    e.DiscountReason,
		ResidualOfID:      e.ResidualOfID,
		PurchaseDate:      purchaseDate,
		FiscalDocumentID:  e.FiscalDocumentID,
		SupplierID:        e.SupplierID,
		CreatedAt:         e.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:         e.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

type financialEntryCreateJSON struct {
	Kind              string     `json:"kind" binding:"required"`
	Status            string     `json:"status"`
	AmountCents       int64      `json:"amount_cents"`
	DueDate           string     `json:"due_date" binding:"required"`
	FamilyMemberID    *uuid.UUID `json:"family_member_id"`
	SourceID          *uuid.UUID `json:"source_id"`
	Type              *string    `json:"type"`
	Description       string     `json:"description"`
	Recurrence        string     `json:"recurrence"`
	Notes             *string    `json:"notes"`
	CardID            *uuid.UUID `json:"card_id"`
	ParentID          *uuid.UUID `json:"parent_id"`
	InstallmentsTotal *int       `json:"installments_total"`
	SupplierID        *uuid.UUID `json:"supplier_id"`
	PurchaseDate      *string    `json:"purchase_date"` // YYYY-MM-DD; data da compra (itens de fatura)
	// Lançamento retroativo: ocorrências vencidas nascem realizadas.
	ConfirmPastOccurrences bool `json:"confirm_past_occurrences"`
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
	var purchaseDate *time.Time
	if body.PurchaseDate != nil && *body.PurchaseDate != "" {
		d, derr := time.Parse(entryDateLayout, *body.PurchaseDate)
		if derr != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "purchase_date inválida (use YYYY-MM-DD)")
			return
		}
		purchaseDate = &d
	}
	entries, err := h.svc.Create(c.Request.Context(), app.CreateEntryInput{
		WorkspaceID: ws, Kind: body.Kind, Status: body.Status, AmountCents: body.AmountCents,
		DueDate: due, FamilyMemberID: body.FamilyMemberID, SourceID: body.SourceID,
		Type: body.Type, Description: body.Description, Recurrence: body.Recurrence, Notes: body.Notes,
		CardID: body.CardID, ParentID: body.ParentID, InstallmentsTotal: body.InstallmentsTotal,
		SupplierID: body.SupplierID, PurchaseDate: purchaseDate, ConfirmPastOccurrences: body.ConfirmPastOccurrences,
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
	filter.Query = strings.TrimSpace(c.Query("query"))
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
	if v := c.Query("card_id"); v != "" {
		cardID, err := uuid.Parse(v)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "card_id inválido")
			return
		}
		filter.CardID = &cardID
	}
	if v := c.Query("parent_id"); v != "" {
		parentID, err := uuid.Parse(v)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "parent_id inválido")
			return
		}
		filter.ParentID = &parentID
	}
	if v := c.Query("top_level"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "top_level inválido")
			return
		}
		filter.TopLevelOnly = b
	}
	if v := c.Query("due_on"); v != "" {
		d, err := time.Parse(entryDateLayout, v)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "due_on inválida (use YYYY-MM-DD)")
			return
		}
		filter.DueOn = &d
	}
	if v := c.Query("due_from"); v != "" {
		d, err := time.Parse(entryDateLayout, v)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "due_from inválida (use YYYY-MM-DD)")
			return
		}
		filter.DueFrom = &d
	}
	if v := c.Query("due_to"); v != "" {
		d, err := time.Parse(entryDateLayout, v)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "due_to inválida (use YYYY-MM-DD)")
			return
		}
		filter.DueTo = &d
	}
	if v := c.Query("supplier_id"); v != "" {
		sID, err := uuid.Parse(v)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "supplier_id inválido")
			return
		}
		filter.SupplierID = &sID
	}
	if v := c.Query("overdue"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "overdue inválido")
			return
		}
		filter.Overdue = b
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
	SupplierID     *uuid.UUID `json:"supplier_id"`
	PurchaseDate   *string    `json:"purchase_date"` // YYYY-MM-DD; ausente preserva a atual
	// Parcela da compra em fatura: ausente preserva; 0 limpa.
	InstallmentNumber *int `json:"installment_number"`
	InstallmentTotal  *int `json:"installment_total"`
	// ApplyTo: "one" (default) edita só este lançamento; "future" propaga
	// dia do vencimento/valor/descrição/categoria às ocorrências previstas
	// futuras da mesma série recorrente.
	ApplyTo string `json:"apply_to"`
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
	var purchaseDate *time.Time
	if body.PurchaseDate != nil && *body.PurchaseDate != "" {
		d, derr := time.Parse(entryDateLayout, *body.PurchaseDate)
		if derr != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "purchase_date inválida (use YYYY-MM-DD)")
			return
		}
		purchaseDate = &d
	}
	applyToFuture := false
	switch body.ApplyTo {
	case "", "one":
	case "future":
		applyToFuture = true
	default:
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "apply_to inválido (use 'one' ou 'future')")
		return
	}
	e, err := h.svc.Update(c.Request.Context(), app.UpdateEntryInput{
		WorkspaceID: ws, ID: id, Kind: body.Kind, Status: body.Status, AmountCents: body.AmountCents,
		DueDate: due, FamilyMemberID: body.FamilyMemberID, SourceID: body.SourceID,
		Type: body.Type, Description: body.Description, Recurrence: body.Recurrence, Notes: body.Notes,
		SupplierID: body.SupplierID, PurchaseDate: purchaseDate,
		InstallmentNumber: body.InstallmentNumber, InstallmentTotal: body.InstallmentTotal,
		ApplyToFuture: applyToFuture,
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
	// Body opcional: desconto obtido na liquidação.
	var body financialEntryConfirmJSON
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
			return
		}
	}
	var residualDue *time.Time
	if body.ResidualDueDate != nil && *body.ResidualDueDate != "" {
		t, err := time.Parse(entryDateLayout, *body.ResidualDueDate)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "residual_due_date inválida (use YYYY-MM-DD)")
			return
		}
		residualDue = &t
	}
	var paidAt *time.Time
	if body.PaidAt != nil && *body.PaidAt != "" {
		t, err := time.Parse(entryDateLayout, *body.PaidAt)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "paid_at inválida (use YYYY-MM-DD)")
			return
		}
		paidAt = &t
	}
	e, err := h.svc.Confirm(c.Request.Context(), app.ConfirmEntryInput{
		WorkspaceID: ws, ID: id,
		DiscountCents: body.DiscountCents, DiscountReason: body.DiscountReason,
		PaidAmountCents: body.PaidAmountCents, ResidualDueDate: residualDue,
		PaidAt: paidAt,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapFinancialEntry(e))
}

type financialEntryConfirmJSON struct {
	DiscountCents  *int64  `json:"discount_cents"`
	DiscountReason *string `json:"discount_reason"`
	// Pagamento parcial: valor efetivamente pago; a diferença vira residual.
	PaidAmountCents *int64  `json:"paid_amount_cents"`
	ResidualDueDate *string `json:"residual_due_date"` // YYYY-MM-DD; default: vencimento original
	PaidAt          *string `json:"paid_at"`           // YYYY-MM-DD; default: agora
}

// Installments responde GET /finance/installments: projeção de compromissos
// parcelados dentro de faturas (calculada, não são lançamentos).
func (h *FinancialEntryHandler) Installments(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	proj, err := h.svc.InstallmentsProjection(c.Request.Context(), ws)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	groups := make([]gin.H, len(proj.Groups))
	for i, g := range proj.Groups {
		groups[i] = gin.H{
			"description":       g.Description,
			"card_id":           g.CardID,
			"category":          g.Category,
			"installment_cents": g.InstallmentCents,
			"installment_total": g.InstallmentTotal,
			"last_known_number": g.LastKnownNumber,
			"remaining_count":   g.RemainingCount,
			"remaining_cents":   g.RemainingCents,
			"last_due_date":     g.LastDueDate.Format(entryDateLayout),
			"ends_at":           g.EndsAt.Format("2006-01"),
		}
	}
	monthly := make([]gin.H, len(proj.Monthly))
	for i, m := range proj.Monthly {
		monthly[i] = gin.H{"month": m.Month, "total_cents": m.TotalCents, "count": m.Count}
	}
	c.JSON(http.StatusOK, gin.H{
		"groups":                groups,
		"monthly":               monthly,
		"remaining_total_cents": proj.RemainingTotalCents,
	})
}

// DiscountReasons lista o catálogo global de motivos de desconto.
func (h *FinancialEntryHandler) DiscountReasons(c *gin.Context) {
	items := make([]gin.H, len(dom.DiscountReasons))
	for i, r := range dom.DiscountReasons {
		items[i] = gin.H{"slug": r.Slug, "name": r.Name, "description": r.Description}
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

// Reopen desfaz a liquidação: realizada volta a prevista, pagamento limpo.
func (h *FinancialEntryHandler) Reopen(c *gin.Context) {
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
	e, err := h.svc.Reopen(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapFinancialEntry(e))
}

type financialEntrySettleJSON struct {
	PaidAt          *string    `json:"paid_at"`           // RFC3339 ou YYYY-MM-DD; default: agora
	PaidAmountCents *int64     `json:"paid_amount_cents"` // default: amount_cents
	PaymentMethod   string     `json:"payment_method" binding:"required"`
	AccountID       *uuid.UUID `json:"account_id"`
	CardID          *uuid.UUID `json:"card_id"`
	Notes           *string    `json:"notes"`
	DiscountCents   *int64     `json:"discount_cents"`  // desconto obtido; abate do valor pago
	DiscountReason  *string    `json:"discount_reason"` // slug do catálogo /finance/discount-reasons
}

// Settle liquida o lançamento com forma de pagamento, valor e data.
func (h *FinancialEntryHandler) Settle(c *gin.Context) {
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
	var body financialEntrySettleJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	var paidAt *time.Time
	if body.PaidAt != nil && *body.PaidAt != "" {
		t, err := time.Parse(time.RFC3339, *body.PaidAt)
		if err != nil {
			t, err = time.Parse(entryDateLayout, *body.PaidAt)
			if err != nil {
				errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "paid_at inválida (use RFC3339 ou YYYY-MM-DD)")
				return
			}
		}
		paidAt = &t
	}
	e, err := h.svc.Settle(c.Request.Context(), app.SettleEntryInput{
		WorkspaceID: ws, ID: id, PaidAt: paidAt, PaidAmountCents: body.PaidAmountCents,
		PaymentMethod: body.PaymentMethod, AccountID: body.AccountID, CardID: body.CardID, Notes: body.Notes,
		DiscountCents: body.DiscountCents, DiscountReason: body.DiscountReason,
	})
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
