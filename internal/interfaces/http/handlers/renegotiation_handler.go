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

// RenegotiationHandler expõe a apuração e a aplicação de renegociações.
type RenegotiationHandler struct {
	svc *app.RenegotiationService
}

func NewRenegotiationHandler(svc *app.RenegotiationService) *RenegotiationHandler {
	return &RenegotiationHandler{svc: svc}
}

type openChargeResponse struct {
	ID                uuid.UUID `json:"id"`
	Kind              string    `json:"kind"` // installment | residual
	Description       string    `json:"description"`
	AmountCents       int64     `json:"amount_cents"`
	DueDate           string    `json:"due_date"`
	InstallmentNumber *int      `json:"installment_number,omitempty"`
	OriginDescription *string   `json:"origin_description,omitempty"`
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
	NextDueDate      *string              `json:"next_due_date,omitempty"`
	SuggestedDueDate string               `json:"suggested_due_date"`
	TypicalAmount    int64                `json:"typical_amount_cents"`
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
		SuggestedDueDate: p.SuggestedDueDate.Format("2006-01-02"),
		TypicalAmount:    p.TypicalAmountCent,
		Charges:          make([]openChargeResponse, 0, len(p.Charges)),
	}
	if p.NextDueDate != nil {
		d := p.NextDueDate.Format("2006-01-02")
		out.NextDueDate = &d
	}
	for _, c := range p.Charges {
		out.Charges = append(out.Charges, openChargeResponse{
			ID:                c.ID,
			Kind:              string(c.Kind),
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
	return gin.H{
		"id":                   r.ID,
		"date":                 r.Date.Format("2006-01-02"),
		"description":          r.Description,
		"settled_amount_cents": r.SettledAmountCents,
		"new_amount_cents":     r.NewAmountCents,
		"adjustment_cents":     r.AdjustmentCents,
		"origin_count":         r.OriginCount,
		"new_count":            r.NewCount,
		"notes":                r.Notes,
		"created_at":           r.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
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
	var date *time.Time
	if body.Date != nil && *body.Date != "" {
		d, derr := time.Parse("2006-01-02", *body.Date)
		if derr != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "date inválida (use YYYY-MM-DD)")
			return
		}
		date = &d
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
	c.JSON(http.StatusOK, gin.H{
		"renegotiation": mapRenegotiation(d.Renegotiation),
		"origins":       origins,
		"created":       created,
	})
}
