package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/finance"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// RenegotiationHandler expõe a apuração e a aplicação dos eventos de dívida:
// renegociação, quitação antecipada e troca de bem financiado.
type RenegotiationHandler struct {
	svc *app.RenegotiationService
}

func NewRenegotiationHandler(svc *app.RenegotiationService) *RenegotiationHandler {
	return &RenegotiationHandler{svc: svc}
}

type openChargeResponse struct {
	ID   uuid.UUID `json:"id"`
	Kind string    `json:"kind"` // installment | residual
	// Status: paid | partially_paid | overdue | upcoming
	Status          string `json:"status"`
	Description     string `json:"description"`
	AmountCents     int64  `json:"amount_cents"`
	PaidAmountCents *int64 `json:"paid_amount_cents,omitempty"`
	DueDate         string `json:"due_date"`
	// Included indica que a cobrança compõe o saldo apurado.
	Included          bool    `json:"included"`
	InstallmentNumber *int    `json:"installment_number,omitempty"`
	OriginDescription *string `json:"origin_description,omitempty"`
}

type renegotiationPreviewResponse struct {
	GroupID          uuid.UUID            `json:"group_id"`
	Description      string               `json:"description"`
	InstallmentTotal int                  `json:"installment_total"`
	PaidCount        int                  `json:"paid_count"`
	PaidCents        int64                `json:"paid_cents"`
	Charges          []openChargeResponse `json:"charges"`
	InstallmentCount int                  `json:"installment_count"`
	InstallmentCents int64                `json:"installment_cents"`
	ResidualCount    int                  `json:"residual_count"`
	ResidualCents    int64                `json:"residual_cents"`
	OpenTotalCents   int64                `json:"open_total_cents"`
	OverdueCount     int                  `json:"overdue_count"`
	OverdueCents     int64                `json:"overdue_cents"`
	NextDueDate      *string              `json:"next_due_date,omitempty"`
	SuggestedDueDate string               `json:"suggested_due_date"`
	TypicalAmount    int64                `json:"typical_amount_cents"`
	AssetType        *string              `json:"asset_type,omitempty"`
	AssetID          *uuid.UUID           `json:"asset_id,omitempty"`
	Category         *string              `json:"category,omitempty"`
}

func mapPreview(p *app.RenegotiationPreview) renegotiationPreviewResponse {
	out := renegotiationPreviewResponse{
		GroupID:          p.GroupID,
		Description:      p.Description,
		InstallmentTotal: p.InstallmentTotal,
		PaidCount:        p.PaidCount,
		PaidCents:        p.PaidCents,
		InstallmentCount: p.InstallmentCount,
		InstallmentCents: p.InstallmentCents,
		ResidualCount:    p.ResidualCount,
		ResidualCents:    p.ResidualCents,
		OpenTotalCents:   p.OpenTotalCents,
		OverdueCount:     p.OverdueCount,
		OverdueCents:     p.OverdueCents,
		SuggestedDueDate: p.SuggestedDueDate.Format("2006-01-02"),
		TypicalAmount:    p.TypicalAmountCent,
		Charges:          make([]openChargeResponse, 0, len(p.Charges)),
		AssetType:        assetTypeStr(p.AssetType),
		AssetID:          p.AssetID,
		Category:         p.Category,
	}
	if p.NextDueDate != nil {
		d := p.NextDueDate.Format("2006-01-02")
		out.NextDueDate = &d
	}
	for _, c := range p.Charges {
		out.Charges = append(out.Charges, openChargeResponse{
			ID:                c.ID,
			Kind:              string(c.Kind),
			Status:            string(c.Status),
			PaidAmountCents:   c.PaidAmountCents,
			Included:          c.Included,
			Description:       c.Description,
			AmountCents:       c.AmountCents,
			DueDate:           c.DueDate.Format("2006-01-02"),
			InstallmentNumber: c.InstallmentNumber,
			OriginDescription: c.OriginDescription,
		})
	}
	return out
}

func mapRenegotiation(r *dom.Renegotiation) gin.H {
	var payer *string
	if r.Payer != nil {
		p := string(*r.Payer)
		payer = &p
	}
	return gin.H{
		"id":                     r.ID,
		"kind":                   string(r.Kind),
		"date":                   r.Date.Format("2006-01-02"),
		"description":            r.Description,
		"settled_amount_cents":   r.SettledAmountCents,
		"new_amount_cents":       r.NewAmountCents,
		"adjustment_cents":       r.AdjustmentCents,
		"origin_count":           r.OriginCount,
		"new_count":              r.NewCount,
		"origin_group_id":        r.OriginGroupID,
		"new_group_id":           r.NewGroupID,
		"notes":                  r.Notes,
		"payoff_cents":           r.PayoffCents,
		"payer":                  payer,
		"payoff_entry_id":        r.PayoffEntryID,
		"asset_type":             assetTypeStr(r.AssetType),
		"asset_id":               r.AssetID,
		"new_asset_id":           r.NewAssetID,
		"trade_in_cents":         r.TradeInCents,
		"trade_in_entry_id":      r.TradeInEntryID,
		"cash_downpayment_cents": r.CashDownpaymentCents,
		"downpayment_entry_id":   r.DownpaymentEntryID,
		"net_downpayment_cents":  r.NetDownpaymentCents(),
		"created_at":             r.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func mapRenegotiationPtr(r *dom.Renegotiation) any {
	if r == nil {
		return nil
	}
	return mapRenegotiation(r)
}

// Preview responde GET /finance/installments/:groupId/renegotiation-preview.
func (h *RenegotiationHandler) Preview(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	groupID, err := uuid.Parse(c.Param("groupId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "group_id inválido")
		return
	}
	p, err := h.svc.PreviewGroup(c.Request.Context(), ws, groupID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapPreview(p))
}

// PreviewByEntry responde GET /finance/entries/:id/renegotiation-preview.
func (h *RenegotiationHandler) PreviewByEntry(c *gin.Context) {
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
	p, err := h.svc.Preview(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapPreview(p))
}

// parseDatePtr converte "YYYY-MM-DD" opcional; vazio devolve nil.
func parseDatePtr(s *string, field string) (*time.Time, string) {
	if s == nil || *s == "" {
		return nil, ""
	}
	d, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, field + " inválida (use YYYY-MM-DD)"
	}
	return &d, ""
}

func paymentMethodPtr(s *string) *dom.PaymentMethod {
	if s == nil || *s == "" {
		return nil
	}
	m := dom.PaymentMethod(*s)
	return &m
}

type renegotiateRequest struct {
	GroupID          string  `json:"group_id" binding:"required"`
	Date             *string `json:"date"` // YYYY-MM-DD; ausente = hoje
	InstallmentCount int     `json:"installment_count" binding:"required"`
	InstallmentCents int64   `json:"installment_cents" binding:"required"`
	FirstDueDate     string  `json:"first_due_date" binding:"required"`
	Description      string  `json:"description"`
	Notes            *string `json:"notes"`
}

// Create responde POST /finance/renegotiations.
func (h *RenegotiationHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body renegotiateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	groupID, err := uuid.Parse(body.GroupID)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "group_id inválido")
		return
	}
	firstDue, err := time.Parse("2006-01-02", body.FirstDueDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "first_due_date inválida (use YYYY-MM-DD)")
		return
	}
	date, msg := parseDatePtr(body.Date, "date")
	if msg != "" {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, msg)
		return
	}

	res, err := h.svc.Renegotiate(c.Request.Context(), app.RenegotiateInput{
		WorkspaceID:      ws,
		GroupID:          groupID,
		Date:             date,
		InstallmentCount: body.InstallmentCount,
		InstallmentCents: body.InstallmentCents,
		FirstDueDate:     firstDue,
		Description:      body.Description,
		Notes:            body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}

	created := make([]financialEntryResponse, 0, len(res.Created))
	for i := range res.Created {
		created = append(created, mapFinancialEntry(&res.Created[i]))
	}
	c.JSON(http.StatusCreated, gin.H{
		"renegotiation": mapRenegotiation(res.Renegotiation),
		"created":       created,
	})
}

type payoffRequest struct {
	Date        *string `json:"date"` // YYYY-MM-DD; ausente = hoje
	PayoffCents int64   `json:"payoff_cents" binding:"required"`
	// Payer: proprio | terceiro (default proprio).
	Payer            string     `json:"payer"`
	PaymentMethod    *string    `json:"payment_method"`
	PaymentAccountID *uuid.UUID `json:"payment_account_id"`
	Description      string     `json:"description"`
	Notes            *string    `json:"notes"`
	AssetType        *string    `json:"asset_type"`
	AssetID          *uuid.UUID `json:"asset_id"`
}

// Payoff responde POST /finance/debts/:groupId/payoff — quitação antecipada.
func (h *RenegotiationHandler) Payoff(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	groupID, err := uuid.Parse(c.Param("groupId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "groupId inválido")
		return
	}
	var body payoffRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	date, msg := parseDatePtr(body.Date, "date")
	if msg != "" {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, msg)
		return
	}
	res, err := h.svc.Payoff(c.Request.Context(), app.PayoffInput{
		WorkspaceID:      ws,
		GroupID:          groupID,
		Date:             date,
		PayoffCents:      body.PayoffCents,
		Payer:            dom.Payer(body.Payer),
		PaymentMethod:    paymentMethodPtr(body.PaymentMethod),
		PaymentAccountID: body.PaymentAccountID,
		Description:      body.Description,
		Notes:            body.Notes,
		AssetType:        assetTypePtr(body.AssetType),
		AssetID:          body.AssetID,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"event":        mapRenegotiation(res.Event),
		"payoff_entry": mapFinancialEntry(res.PayoffEntry),
	})
}

type newContractRequest struct {
	InstallmentCount int        `json:"installment_count" binding:"required"`
	InstallmentCents int64      `json:"installment_cents" binding:"required"`
	FirstDueDate     string     `json:"first_due_date" binding:"required"`
	Description      string     `json:"description"`
	Category         *string    `json:"category"`
	SupplierID       *uuid.UUID `json:"supplier_id"`
	FamilyMemberID   *uuid.UUID `json:"family_member_id"`
}

type assetSwapRequest struct {
	Date          *string `json:"date"`
	AssetType     string  `json:"asset_type"` // default vehicle
	OldAssetID    string  `json:"old_asset_id" binding:"required"`
	NewAssetID    string  `json:"new_asset_id" binding:"required"`
	OldAssetLabel string  `json:"old_asset_label"`
	NewAssetLabel string  `json:"new_asset_label"`
	// Contrato antigo (opcional).
	OldGroupID             *string    `json:"old_group_id"`
	PayoffCents            int64      `json:"payoff_cents"`
	Payer                  string     `json:"payer"` // proprio | terceiro (default terceiro)
	PayoffPaymentMethod    *string    `json:"payoff_payment_method"`
	PayoffPaymentAccountID *uuid.UUID `json:"payoff_payment_account_id"`
	// Troca.
	TradeInCents         int64      `json:"trade_in_cents"`
	CashDownpaymentCents int64      `json:"cash_downpayment_cents"`
	CashPaymentMethod    *string    `json:"cash_payment_method"`
	CashPaymentAccountID *uuid.UUID `json:"cash_payment_account_id"`
	NewAssetPriceCents   *int64     `json:"new_asset_price_cents"`
	// Financiamento novo (opcional).
	NewContract *newContractRequest `json:"new_contract"`
	Description string              `json:"description"`
	Notes       *string             `json:"notes"`
}

// AssetSwap responde POST /finance/asset-swaps — troca de bem financiado.
func (h *RenegotiationHandler) AssetSwap(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body assetSwapRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	oldAsset, err := uuid.Parse(body.OldAssetID)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "old_asset_id inválido")
		return
	}
	newAsset, err := uuid.Parse(body.NewAssetID)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "new_asset_id inválido")
		return
	}
	var oldGroup *uuid.UUID
	if body.OldGroupID != nil && *body.OldGroupID != "" {
		g, gerr := uuid.Parse(*body.OldGroupID)
		if gerr != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "old_group_id inválido")
			return
		}
		oldGroup = &g
	}
	date, msg := parseDatePtr(body.Date, "date")
	if msg != "" {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, msg)
		return
	}
	assetType := dom.AssetType(body.AssetType)
	if assetType == "" {
		assetType = dom.AssetVehicle
	}

	in := app.AssetSwapInput{
		WorkspaceID:            ws,
		Date:                   date,
		AssetType:              assetType,
		OldAssetID:             oldAsset,
		NewAssetID:             newAsset,
		OldAssetLabel:          body.OldAssetLabel,
		NewAssetLabel:          body.NewAssetLabel,
		OldGroupID:             oldGroup,
		PayoffCents:            body.PayoffCents,
		Payer:                  dom.Payer(body.Payer),
		PayoffPaymentMethod:    paymentMethodPtr(body.PayoffPaymentMethod),
		PayoffPaymentAccountID: body.PayoffPaymentAccountID,
		TradeInCents:           body.TradeInCents,
		CashDownpaymentCents:   body.CashDownpaymentCents,
		CashPaymentMethod:      paymentMethodPtr(body.CashPaymentMethod),
		CashPaymentAccountID:   body.CashPaymentAccountID,
		NewAssetPriceCents:     body.NewAssetPriceCents,
		Description:            body.Description,
		Notes:                  body.Notes,
	}
	if nc := body.NewContract; nc != nil {
		firstDue, ferr := time.Parse("2006-01-02", nc.FirstDueDate)
		if ferr != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "new_contract.first_due_date inválida (use YYYY-MM-DD)")
			return
		}
		in.NewContract = &app.NewContractSpec{
			InstallmentCount: nc.InstallmentCount,
			InstallmentCents: nc.InstallmentCents,
			FirstDueDate:     firstDue,
			Description:      nc.Description,
			Category:         nc.Category,
			SupplierID:       nc.SupplierID,
			FamilyMemberID:   nc.FamilyMemberID,
		}
	}

	res, err := h.svc.AssetSwap(c.Request.Context(), in)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	created := make([]financialEntryResponse, 0, len(res.Created))
	for i := range res.Created {
		created = append(created, mapFinancialEntry(&res.Created[i]))
	}
	out := gin.H{
		"event":   mapRenegotiation(res.Event),
		"created": created,
	}
	if res.PayoffEntry != nil {
		out["payoff_entry"] = mapFinancialEntry(res.PayoffEntry)
	}
	if res.TradeInEntry != nil {
		out["trade_in_entry"] = mapFinancialEntry(res.TradeInEntry)
	}
	if res.DownpaymentEntry != nil {
		out["downpayment_entry"] = mapFinancialEntry(res.DownpaymentEntry)
	}
	c.JSON(http.StatusCreated, out)
}

// List responde GET /finance/renegotiations.
func (h *RenegotiationHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	res, err := h.svc.List(c.Request.Context(), ws, limit, offset)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]gin.H, 0, len(res.Items))
	for i := range res.Items {
		items = append(items, mapRenegotiation(&res.Items[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}

// Get responde GET /finance/renegotiations/:id — o evento com os dois lados.
func (h *RenegotiationHandler) Get(c *gin.Context) {
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
	d, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	origins := make([]financialEntryResponse, 0, len(d.Origins))
	for i := range d.Origins {
		origins = append(origins, mapFinancialEntry(&d.Origins[i]))
	}
	created := make([]financialEntryResponse, 0, len(d.Created))
	for i := range d.Created {
		created = append(created, mapFinancialEntry(&d.Created[i]))
	}
	paidBefore := make([]financialEntryResponse, 0, len(d.PaidBefore))
	for i := range d.PaidBefore {
		paidBefore = append(paidBefore, mapFinancialEntry(&d.PaidBefore[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"renegotiation":             mapRenegotiation(d.Renegotiation),
		"origins":                   origins,
		"created":                   created,
		"paid_before":               paidBefore,
		"paid_before_count":         len(paidBefore),
		"paid_before_cents":         d.PaidBeforeCents,
		"previous_renegotiation_id": d.PreviousID,
		"next_renegotiation_id":     d.NextID,
		"root_group_id":             d.RootGroupID,
	})
}

func fmtDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

func mapLineage(l *app.DebtLineage) gin.H {
	stages := make([]gin.H, 0, len(l.Stages))
	for i := range l.Stages {
		st := &l.Stages[i]
		entries := make([]financialEntryResponse, 0, len(st.Entries))
		for j := range st.Entries {
			entries = append(entries, mapFinancialEntry(&st.Entries[j]))
		}
		var payoffEntry any
		if st.PayoffEntry != nil {
			payoffEntry = mapFinancialEntry(st.PayoffEntry)
		}
		stages = append(stages, gin.H{
			"index":             st.Index,
			"group_id":          st.GroupID,
			"description":       st.Description,
			"renegotiation":     mapRenegotiationPtr(st.Renegotiation),
			"settled_by":        mapRenegotiationPtr(st.SettledBy),
			"payoff_entry":      payoffEntry,
			"installment_total": st.InstallmentTotal,
			"total_cents":       st.TotalCents,
			"first_due_date":    fmtDatePtr(st.FirstDueDate),
			"last_due_date":     fmtDatePtr(st.LastDueDate),
			"paid_count":        st.PaidCount,
			"paid_cents":        st.PaidCents,
			"carried_count":     st.CarriedCount,
			"carried_cents":     st.CarriedCents,
			"cancelled_count":   st.CancelledCount,
			"cancelled_cents":   st.CancelledCents,
			"open_count":        st.OpenCount,
			"open_cents":        st.OpenCents,
			"overdue_count":     st.OverdueCount,
			"overdue_cents":     st.OverdueCents,
			"asset_type":        assetTypeStr(st.AssetType),
			"asset_id":          st.AssetID,
			"entries":           entries,
		})
	}
	return gin.H{
		"root_group_id":        l.RootGroupID,
		"current_group_id":     l.CurrentGroupID,
		"description":          l.Description,
		"stages":               stages,
		"original_cents":       l.OriginalCents,
		"interest_cents":       l.InterestCents,
		"discount_cents":       l.DiscountCents,
		"current_total_cents":  l.CurrentTotalCents,
		"paid_cents":           l.PaidCents,
		"paid_count":           l.PaidCount,
		"open_cents":           l.OpenCents,
		"open_count":           l.OpenCount,
		"overdue_cents":        l.OverdueCents,
		"overdue_count":        l.OverdueCount,
		"renegotiation_count":  l.RenegotiationCnt,
		"settled":              l.Settled,
		"closed_by":            mapRenegotiationPtr(l.ClosedBy),
		"successor_group_id":   l.SuccessorGroupID,
		"origin_event":         mapRenegotiationPtr(l.OriginEvent),
		"predecessor_group_id": l.PredecessorGroupID,
		"asset_type":           assetTypeStr(l.AssetType),
		"asset_id":             l.AssetID,
	}
}

// Lineage responde GET /finance/debts/:groupId — a história da dívida
// através das renegociações, a partir de qualquer grupo da cadeia.
func (h *RenegotiationHandler) Lineage(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	groupID, err := uuid.Parse(c.Param("groupId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "groupId inválido")
		return
	}
	l, err := h.svc.Lineage(c.Request.Context(), ws, groupID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapLineage(l))
}

// AssetDebts responde GET /finance/assets/:assetType/:assetId/debts — os
// contratos que financiam o bem (com linhagem) e os eventos em que aparece.
func (h *RenegotiationHandler) AssetDebts(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	assetID, err := uuid.Parse(c.Param("assetId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "assetId inválido")
		return
	}
	assetType := dom.AssetType(c.Param("assetType"))
	res, err := h.svc.AssetDebts(c.Request.Context(), ws, assetType, assetID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	debts := make([]gin.H, 0, len(res.Debts))
	for i := range res.Debts {
		debts = append(debts, mapLineage(&res.Debts[i]))
	}
	events := make([]gin.H, 0, len(res.Events))
	for i := range res.Events {
		events = append(events, mapRenegotiation(&res.Events[i]))
	}
	c.JSON(http.StatusOK, gin.H{"debts": debts, "events": events})
}
