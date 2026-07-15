package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appp "github.com/retechfin/retechfin-api/internal/application/patrimony"
	dom "github.com/retechfin/retechfin-api/internal/domain/patrimony"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

const patrimonyDateLayout = "2006-01-02"

// PatrimonyHandler agrupa os endpoints de imóveis e impostos de bens.
type PatrimonyHandler struct {
	svc *appp.Service
}

func NewPatrimonyHandler(svc *appp.Service) *PatrimonyHandler {
	return &PatrimonyHandler{svc: svc}
}

// ─── Response types ────────────────────────────────────────────────────────────

type propertyResponse struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	PropertyType       string   `json:"property_type"`
	Address            *string  `json:"address"`
	City               *string  `json:"city"`
	State              *string  `json:"state"`
	ZipCode            *string  `json:"zip_code"`
	RegistrationNumber *string  `json:"registration_number"`
	AreaM2             *float64 `json:"area_m2"`
	PurchaseDate       *string  `json:"purchase_date"`
	PurchaseValueCents *int64   `json:"purchase_value_cents"`
	CurrentValueCents  *int64   `json:"current_value_cents"`
	Notes              *string  `json:"notes"`
	Active             bool     `json:"active"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

func mapProperty(p *dom.Property) propertyResponse {
	r := propertyResponse{
		ID:                 p.ID.String(),
		Name:               p.Name,
		PropertyType:       string(p.PropertyType),
		Address:            p.Address,
		City:               p.City,
		State:              p.State,
		ZipCode:            p.ZipCode,
		RegistrationNumber: p.RegistrationNumber,
		AreaM2:             p.AreaM2,
		PurchaseValueCents: p.PurchaseValueCents,
		CurrentValueCents:  p.CurrentValueCents,
		Notes:              p.Notes,
		Active:             p.Active,
		CreatedAt:          p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:          p.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if p.PurchaseDate != nil {
		s := p.PurchaseDate.Format(patrimonyDateLayout)
		r.PurchaseDate = &s
	}
	return r
}

type taxResponse struct {
	ID                string  `json:"id"`
	AssetType         string  `json:"asset_type"`
	PropertyID        *string `json:"property_id"`
	VehicleID         *string `json:"vehicle_id"`
	TaxType           string  `json:"tax_type"`
	ReferenceYear     int     `json:"reference_year"`
	Description       *string `json:"description"`
	DueDate           *string `json:"due_date"`
	AmountCents       int64   `json:"amount_cents"`
	PaidCents         int64   `json:"paid_cents"`
	PaidDate          *string `json:"paid_date"`
	Status            string  `json:"status"`
	InstallmentsTotal int     `json:"installments_total"`
	InstallmentNumber int     `json:"installment_number"`
	Notes             *string `json:"notes"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

func mapTax(t *dom.AssetTax) taxResponse {
	r := taxResponse{
		ID:                t.ID.String(),
		AssetType:         string(t.AssetType),
		TaxType:           string(t.TaxType),
		ReferenceYear:     t.ReferenceYear,
		Description:       t.Description,
		AmountCents:       t.AmountCents,
		PaidCents:         t.PaidCents,
		Status:            string(t.Status),
		InstallmentsTotal: t.InstallmentsTotal,
		InstallmentNumber: t.InstallmentNumber,
		Notes:             t.Notes,
		CreatedAt:         t.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         t.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if t.PropertyID != nil {
		s := t.PropertyID.String()
		r.PropertyID = &s
	}
	if t.VehicleID != nil {
		s := t.VehicleID.String()
		r.VehicleID = &s
	}
	if t.DueDate != nil {
		s := t.DueDate.Format(patrimonyDateLayout)
		r.DueDate = &s
	}
	if t.PaidDate != nil {
		s := t.PaidDate.Format(patrimonyDateLayout)
		r.PaidDate = &s
	}
	return r
}

// ─── Input types ──────────────────────────────────────────────────────────────

type propertyJSON struct {
	Name               string   `json:"name" binding:"required"`
	PropertyType       string   `json:"property_type"`
	Address            *string  `json:"address"`
	City               *string  `json:"city"`
	State              *string  `json:"state"`
	ZipCode            *string  `json:"zip_code"`
	RegistrationNumber *string  `json:"registration_number"`
	AreaM2             *float64 `json:"area_m2"`
	PurchaseDate       *string  `json:"purchase_date"`
	PurchaseValueCents *int64   `json:"purchase_value_cents"`
	CurrentValueCents  *int64   `json:"current_value_cents"`
	Notes              *string  `json:"notes"`
	Active             *bool    `json:"active"`
}

type taxJSON struct {
	AssetType         string  `json:"asset_type" binding:"required"`
	PropertyID        *string `json:"property_id"`
	VehicleID         *string `json:"vehicle_id"`
	TaxType           string  `json:"tax_type" binding:"required"`
	ReferenceYear     int     `json:"reference_year" binding:"required"`
	Description       *string `json:"description"`
	DueDate           *string `json:"due_date"`
	AmountCents       int64   `json:"amount_cents"`
	PaidCents         int64   `json:"paid_cents"`
	PaidDate          *string `json:"paid_date"`
	Status            string  `json:"status"`
	InstallmentsTotal int     `json:"installments_total"`
	InstallmentNumber int     `json:"installment_number"`
	Notes             *string `json:"notes"`
}

type payTaxJSON struct {
	PaidCents int64   `json:"paid_cents"`
	PaidDate  *string `json:"paid_date"`
}

// ─── Handlers: properties ──────────────────────────────────────────────────────

func (h *PatrimonyHandler) ListProperties(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	onlyActive := c.Query("active") == "true"
	limit, offset := pagination(c)
	result, err := h.svc.ListProperties(c.Request.Context(), ws, onlyActive, limit, offset)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]propertyResponse, len(result.Items))
	for i := range result.Items {
		items[i] = mapProperty(&result.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": result.Total})
}

func (h *PatrimonyHandler) CreateProperty(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body propertyJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	in, err := h.propertyInput(ws, body)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, err.Error())
		return
	}
	p, err := h.svc.CreateProperty(c.Request.Context(), in)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapProperty(p))
}

func (h *PatrimonyHandler) GetProperty(c *gin.Context) {
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
	p, err := h.svc.GetProperty(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapProperty(p))
}

func (h *PatrimonyHandler) UpdateProperty(c *gin.Context) {
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
	var body propertyJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	in, err := h.propertyInput(ws, body)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, err.Error())
		return
	}
	in.ID = id
	p, err := h.svc.UpdateProperty(c.Request.Context(), in)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapProperty(p))
}

func (h *PatrimonyHandler) DeleteProperty(c *gin.Context) {
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
	if err := h.svc.DeleteProperty(c.Request.Context(), ws, id); err != nil {
		errrespond.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PatrimonyHandler) propertyInput(ws uuid.UUID, body propertyJSON) (appp.PropertyInput, error) {
	purchaseDate, err := parsePatrimonyDate(body.PurchaseDate)
	if err != nil {
		return appp.PropertyInput{}, &dom.ValidationError{Msg: "purchase_date inválido (use YYYY-MM-DD)"}
	}
	return appp.PropertyInput{
		WorkspaceID:        ws,
		Name:               body.Name,
		PropertyType:       body.PropertyType,
		Address:            body.Address,
		City:               body.City,
		State:              body.State,
		ZipCode:            body.ZipCode,
		RegistrationNumber: body.RegistrationNumber,
		AreaM2:             body.AreaM2,
		PurchaseDate:       purchaseDate,
		PurchaseValueCents: body.PurchaseValueCents,
		CurrentValueCents:  body.CurrentValueCents,
		Notes:              body.Notes,
		Active:             body.Active,
	}, nil
}

// ─── Handlers: taxes ───────────────────────────────────────────────────────────

func (h *PatrimonyHandler) ListTaxes(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	limit, offset := pagination(c)
	params := dom.ListTaxesParams{
		AssetType: c.Query("asset_type"),
		TaxType:   c.Query("tax_type"),
		Status:    c.Query("status"),
		Limit:     limit,
		Offset:    offset,
	}
	if y := c.Query("reference_year"); y != "" {
		params.ReferenceYear, _ = strconv.Atoi(y)
	}
	if pid := c.Query("property_id"); pid != "" {
		if id, err := uuid.Parse(pid); err == nil {
			params.PropertyID = &id
		}
	}
	if vid := c.Query("vehicle_id"); vid != "" {
		if id, err := uuid.Parse(vid); err == nil {
			params.VehicleID = &id
		}
	}
	result, err := h.svc.ListTaxes(c.Request.Context(), ws, params)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]taxResponse, len(result.Items))
	for i := range result.Items {
		items[i] = mapTax(&result.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": result.Total})
}

func (h *PatrimonyHandler) CreateTax(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body taxJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	in, err := h.taxInput(ws, body)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, err.Error())
		return
	}
	t, err := h.svc.CreateTax(c.Request.Context(), in)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapTax(t))
}

func (h *PatrimonyHandler) GetTax(c *gin.Context) {
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
	t, err := h.svc.GetTax(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapTax(t))
}

func (h *PatrimonyHandler) UpdateTax(c *gin.Context) {
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
	var body taxJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	in, err := h.taxInput(ws, body)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, err.Error())
		return
	}
	in.ID = id
	t, err := h.svc.UpdateTax(c.Request.Context(), in)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapTax(t))
}

func (h *PatrimonyHandler) DeleteTax(c *gin.Context) {
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
	if err := h.svc.DeleteTax(c.Request.Context(), ws, id); err != nil {
		errrespond.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PatrimonyHandler) PayTax(c *gin.Context) {
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
	var body payTaxJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	paidDate, err := parsePatrimonyDate(body.PaidDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "paid_date inválido (use YYYY-MM-DD)")
		return
	}
	t, err := h.svc.PayTax(c.Request.Context(), appp.PayTaxInput{
		WorkspaceID: ws,
		ID:          id,
		PaidCents:   body.PaidCents,
		PaidDate:    paidDate,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapTax(t))
}

func (h *PatrimonyHandler) taxInput(ws uuid.UUID, body taxJSON) (appp.TaxInput, error) {
	dueDate, err := parsePatrimonyDate(body.DueDate)
	if err != nil {
		return appp.TaxInput{}, &dom.ValidationError{Msg: "due_date inválido (use YYYY-MM-DD)"}
	}
	paidDate, err := parsePatrimonyDate(body.PaidDate)
	if err != nil {
		return appp.TaxInput{}, &dom.ValidationError{Msg: "paid_date inválido (use YYYY-MM-DD)"}
	}
	in := appp.TaxInput{
		WorkspaceID:       ws,
		AssetType:         body.AssetType,
		TaxType:           body.TaxType,
		ReferenceYear:     body.ReferenceYear,
		Description:       body.Description,
		DueDate:           dueDate,
		AmountCents:       body.AmountCents,
		PaidCents:         body.PaidCents,
		PaidDate:          paidDate,
		Status:            body.Status,
		InstallmentsTotal: body.InstallmentsTotal,
		InstallmentNumber: body.InstallmentNumber,
		Notes:             body.Notes,
	}
	if body.PropertyID != nil && *body.PropertyID != "" {
		id, err := uuid.Parse(*body.PropertyID)
		if err != nil {
			return appp.TaxInput{}, &dom.ValidationError{Msg: "property_id inválido"}
		}
		in.PropertyID = &id
	}
	if body.VehicleID != nil && *body.VehicleID != "" {
		id, err := uuid.Parse(*body.VehicleID)
		if err != nil {
			return appp.TaxInput{}, &dom.ValidationError{Msg: "vehicle_id inválido"}
		}
		in.VehicleID = &id
	}
	return in, nil
}

// ─── Handlers: overview ────────────────────────────────────────────────────────

func (h *PatrimonyHandler) Overview(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	ov, err := h.svc.Overview(c.Request.Context(), ws)
	if err != nil {
		errrespond.Write(c, err)
		return
	}

	byYear := make([]gin.H, len(ov.ByYear))
	for i, y := range ov.ByYear {
		byYear[i] = gin.H{"year": y.Year, "planned_cents": y.PlannedCents, "paid_cents": y.PaidCents}
	}
	byTaxTypeYear := make([]gin.H, len(ov.ByTaxTypeYear))
	for i, t := range ov.ByTaxTypeYear {
		byTaxTypeYear[i] = gin.H{
			"tax_type": string(t.TaxType), "year": t.Year,
			"planned_cents": t.PlannedCents, "paid_cents": t.PaidCents,
		}
	}
	inflation := make([]gin.H, len(ov.Inflation))
	for i, inf := range ov.Inflation {
		inflation[i] = gin.H{
			"tax_type": string(inf.TaxType), "year": inf.Year, "previous_year": inf.PreviousYear,
			"amount_cents": inf.AmountCents, "previous_cents": inf.PreviousCents,
			"variation_pct": inf.VariationPct,
		}
	}
	upcoming := make([]taxResponse, len(ov.Upcoming))
	for i := range ov.Upcoming {
		upcoming[i] = mapTax(&ov.Upcoming[i])
	}
	overdue := make([]taxResponse, len(ov.Overdue))
	for i := range ov.Overdue {
		overdue[i] = mapTax(&ov.Overdue[i])
	}

	c.JSON(http.StatusOK, gin.H{
		"total_properties":     ov.TotalProperties,
		"total_property_value": ov.TotalPropertyValue,
		"by_year":              byYear,
		"by_tax_type_year":     byTaxTypeYear,
		"inflation":            inflation,
		"upcoming":             upcoming,
		"overdue":              overdue,
	})
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

func parsePatrimonyDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(patrimonyDateLayout, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
