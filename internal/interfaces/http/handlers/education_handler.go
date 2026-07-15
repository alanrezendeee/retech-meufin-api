package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appe "github.com/retechfin/retechfin-api/internal/application/education"
	dom "github.com/retechfin/retechfin-api/internal/domain/education"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// EducationHandler agrupa os endpoints do módulo de Educação / Material Escolar.
type EducationHandler struct {
	svc *appe.Service
}

func NewEducationHandler(svc *appe.Service) *EducationHandler {
	return &EducationHandler{svc: svc}
}

// writeErr mapeia os erros do domínio de educação para respostas HTTP.
func writeEduErr(c *gin.Context, err error) {
	var ve *dom.ValidationError
	if errors.Is(err, dom.ErrNotFound) {
		errrespond.Message(c, http.StatusNotFound, errrespond.CodeNotFound, err.Error())
		return
	}
	if errors.As(err, &ve) {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeValidation, ve.Msg)
		return
	}
	errrespond.Write(c, err)
}

// ─── Response mappers ────────────────────────────────────────────────────────

type enrollmentResponse struct {
	ID                 string  `json:"id"`
	MemberID           string  `json:"member_id"`
	MemberName         *string `json:"member_name"`
	SchoolYear         int     `json:"school_year"`
	Stage              string  `json:"stage"`
	SchoolName         *string `json:"school_name"`
	Grade              *string `json:"grade"`
	Shift              *string `json:"shift"`
	MonthlyFeeCents    int64   `json:"monthly_fee_cents"`
	EnrollmentFeeCents int64   `json:"enrollment_fee_cents"`
	Notes              *string `json:"notes"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

func mapEnrollment(e *dom.SchoolEnrollment) enrollmentResponse {
	r := enrollmentResponse{
		ID:                 e.ID.String(),
		MemberID:           e.MemberID.String(),
		MemberName:         e.MemberName,
		SchoolYear:         e.SchoolYear,
		Stage:              string(e.Stage),
		SchoolName:         e.SchoolName,
		Grade:              e.Grade,
		MonthlyFeeCents:    e.MonthlyFeeCents,
		EnrollmentFeeCents: e.EnrollmentFeeCents,
		Notes:              e.Notes,
		CreatedAt:          e.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:          e.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if e.Shift != nil {
		s := string(*e.Shift)
		r.Shift = &s
	}
	return r
}

type supplyItemResponse struct {
	ID                  string   `json:"id"`
	ListID              string   `json:"list_id"`
	Name                string   `json:"name"`
	Category            string   `json:"category"`
	Quantity            float64  `json:"quantity"`
	ReferencePriceCents int64    `json:"reference_price_cents"`
	Purchased           bool     `json:"purchased"`
	PaidPriceCents      int64    `json:"paid_price_cents"`
	PurchasedAt         *string  `json:"purchased_at"`
	Store               *string  `json:"store"`
	Notes               *string  `json:"notes"`
	CreatedAt           string   `json:"created_at"`
	UpdatedAt           string   `json:"updated_at"`
}

func mapItem(i *dom.SchoolSupplyItem) supplyItemResponse {
	r := supplyItemResponse{
		ID:                  i.ID.String(),
		ListID:              i.ListID.String(),
		Name:                i.Name,
		Category:            string(i.Category),
		Quantity:            i.Quantity,
		ReferencePriceCents: i.ReferencePriceCents,
		Purchased:           i.Purchased,
		PaidPriceCents:      i.PaidPriceCents,
		Store:               i.Store,
		Notes:               i.Notes,
		CreatedAt:           i.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:           i.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if i.PurchasedAt != nil {
		s := i.PurchasedAt.Format("2006-01-02")
		r.PurchasedAt = &s
	}
	return r
}

type supplyListResponse struct {
	ID           string               `json:"id"`
	EnrollmentID string               `json:"enrollment_id"`
	MemberID     *string              `json:"member_id"`
	MemberName   *string              `json:"member_name"`
	SchoolYear   *int                 `json:"school_year"`
	Title        string               `json:"title"`
	Status       string               `json:"status"`
	Notes        *string              `json:"notes"`
	Items        []supplyItemResponse `json:"items"`
	CreatedAt    string               `json:"created_at"`
	UpdatedAt    string               `json:"updated_at"`
}

func mapList(l *dom.SchoolSupplyList) supplyListResponse {
	r := supplyListResponse{
		ID:           l.ID.String(),
		EnrollmentID: l.EnrollmentID.String(),
		MemberName:   l.MemberName,
		SchoolYear:   l.SchoolYear,
		Title:        l.Title,
		Status:       string(l.Status),
		Notes:        l.Notes,
		CreatedAt:    l.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    l.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if l.MemberID != nil {
		s := l.MemberID.String()
		r.MemberID = &s
	}
	r.Items = make([]supplyItemResponse, len(l.Items))
	for i := range l.Items {
		r.Items[i] = mapItem(&l.Items[i])
	}
	return r
}

// ─── Input types ─────────────────────────────────────────────────────────────

type enrollmentJSON struct {
	MemberID           string  `json:"member_id" binding:"required"`
	SchoolYear         int     `json:"school_year" binding:"required"`
	Stage              string  `json:"stage" binding:"required"`
	SchoolName         *string `json:"school_name"`
	Grade              *string `json:"grade"`
	Shift              *string `json:"shift"`
	MonthlyFeeCents    int64   `json:"monthly_fee_cents"`
	EnrollmentFeeCents int64   `json:"enrollment_fee_cents"`
	Notes              *string `json:"notes"`
}

type listJSON struct {
	EnrollmentID string  `json:"enrollment_id"`
	Title        string  `json:"title" binding:"required"`
	Status       string  `json:"status"`
	Notes        *string `json:"notes"`
}

type itemJSON struct {
	Name                string   `json:"name" binding:"required"`
	Category            string   `json:"category"`
	Quantity            float64  `json:"quantity"`
	ReferencePriceCents int64    `json:"reference_price_cents"`
	Purchased           bool     `json:"purchased"`
	PaidPriceCents      int64    `json:"paid_price_cents"`
	PurchasedAt         *string  `json:"purchased_at"`
	Store               *string  `json:"store"`
	Notes               *string  `json:"notes"`
}

type purchaseJSON struct {
	PaidPriceCents int64   `json:"paid_price_cents"`
	PurchasedAt    *string `json:"purchased_at"`
	Store          *string `json:"store"`
}

// ─── Handlers: enrollments ───────────────────────────────────────────────────

func (h *EducationHandler) ListEnrollments(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var memberID *uuid.UUID
	if s := c.Query("member_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "member_id inválido")
			return
		}
		memberID = &id
	}
	var schoolYear *int
	if s := c.Query("school_year"); s != "" {
		y, err := strconv.Atoi(s)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "school_year inválido")
			return
		}
		schoolYear = &y
	}
	items, err := h.svc.ListEnrollments(c.Request.Context(), ws, memberID, schoolYear)
	if err != nil {
		writeEduErr(c, err)
		return
	}
	resp := make([]enrollmentResponse, len(items))
	for i := range items {
		resp[i] = mapEnrollment(&items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": resp})
}

func (h *EducationHandler) CreateEnrollment(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body enrollmentJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	memberID, err := uuid.Parse(body.MemberID)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "member_id inválido")
		return
	}
	e, err := h.svc.CreateEnrollment(c.Request.Context(), appe.CreateEnrollmentInput{
		WorkspaceID:        ws,
		MemberID:           memberID,
		SchoolYear:         body.SchoolYear,
		Stage:              body.Stage,
		SchoolName:         body.SchoolName,
		Grade:              body.Grade,
		Shift:              body.Shift,
		MonthlyFeeCents:    body.MonthlyFeeCents,
		EnrollmentFeeCents: body.EnrollmentFeeCents,
		Notes:              body.Notes,
	})
	if err != nil {
		writeEduErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapEnrollment(e))
}

func (h *EducationHandler) GetEnrollment(c *gin.Context) {
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
	e, err := h.svc.GetEnrollment(c.Request.Context(), ws, id)
	if err != nil {
		writeEduErr(c, err)
		return
	}
	c.JSON(http.StatusOK, mapEnrollment(e))
}

func (h *EducationHandler) UpdateEnrollment(c *gin.Context) {
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
	var body enrollmentJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	memberID, err := uuid.Parse(body.MemberID)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "member_id inválido")
		return
	}
	e, err := h.svc.UpdateEnrollment(c.Request.Context(), appe.UpdateEnrollmentInput{
		WorkspaceID:        ws,
		ID:                 id,
		MemberID:           memberID,
		SchoolYear:         body.SchoolYear,
		Stage:              body.Stage,
		SchoolName:         body.SchoolName,
		Grade:              body.Grade,
		Shift:              body.Shift,
		MonthlyFeeCents:    body.MonthlyFeeCents,
		EnrollmentFeeCents: body.EnrollmentFeeCents,
		Notes:              body.Notes,
	})
	if err != nil {
		writeEduErr(c, err)
		return
	}
	c.JSON(http.StatusOK, mapEnrollment(e))
}

func (h *EducationHandler) DeleteEnrollment(c *gin.Context) {
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
	if err := h.svc.DeleteEnrollment(c.Request.Context(), ws, id); err != nil {
		writeEduErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ─── Handlers: supply lists ──────────────────────────────────────────────────

func (h *EducationHandler) ListSupplyLists(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var enrollmentID *uuid.UUID
	if s := c.Query("enrollment_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "enrollment_id inválido")
			return
		}
		enrollmentID = &id
	}
	var schoolYear *int
	if s := c.Query("school_year"); s != "" {
		y, err := strconv.Atoi(s)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "school_year inválido")
			return
		}
		schoolYear = &y
	}
	status := c.Query("status")
	items, err := h.svc.ListSupplyLists(c.Request.Context(), ws, enrollmentID, schoolYear, status)
	if err != nil {
		writeEduErr(c, err)
		return
	}
	resp := make([]supplyListResponse, len(items))
	for i := range items {
		resp[i] = mapList(&items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": resp})
}

func (h *EducationHandler) CreateSupplyList(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body listJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	enrollmentID, err := uuid.Parse(body.EnrollmentID)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "enrollment_id inválido")
		return
	}
	l, err := h.svc.CreateList(c.Request.Context(), appe.CreateListInput{
		WorkspaceID:  ws,
		EnrollmentID: enrollmentID,
		Title:        body.Title,
		Status:       body.Status,
		Notes:        body.Notes,
	})
	if err != nil {
		writeEduErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapList(l))
}

func (h *EducationHandler) GetSupplyList(c *gin.Context) {
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
	l, err := h.svc.GetList(c.Request.Context(), ws, id)
	if err != nil {
		writeEduErr(c, err)
		return
	}
	c.JSON(http.StatusOK, mapList(l))
}

func (h *EducationHandler) UpdateSupplyList(c *gin.Context) {
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
	var body listJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	l, err := h.svc.UpdateList(c.Request.Context(), appe.UpdateListInput{
		WorkspaceID: ws,
		ID:          id,
		Title:       body.Title,
		Status:      body.Status,
		Notes:       body.Notes,
	})
	if err != nil {
		writeEduErr(c, err)
		return
	}
	c.JSON(http.StatusOK, mapList(l))
}

func (h *EducationHandler) DeleteSupplyList(c *gin.Context) {
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
	if err := h.svc.DeleteList(c.Request.Context(), ws, id); err != nil {
		writeEduErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ─── Handlers: supply items ──────────────────────────────────────────────────

func (h *EducationHandler) AddItem(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	listID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	var body itemJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	purchasedAt, err := parseOptionalDate(body.PurchasedAt)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "purchased_at inválido (use YYYY-MM-DD)")
		return
	}
	item, err := h.svc.AddItem(c.Request.Context(), appe.AddItemInput{
		WorkspaceID:         ws,
		ListID:              listID,
		Name:                body.Name,
		Category:            body.Category,
		Quantity:            body.Quantity,
		ReferencePriceCents: body.ReferencePriceCents,
		Purchased:           body.Purchased,
		PaidPriceCents:      body.PaidPriceCents,
		PurchasedAt:         purchasedAt,
		Store:               body.Store,
		Notes:               body.Notes,
	})
	if err != nil {
		writeEduErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapItem(item))
}

func (h *EducationHandler) UpdateItem(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	listID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "itemId inválido")
		return
	}
	var body itemJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	purchasedAt, err := parseOptionalDate(body.PurchasedAt)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "purchased_at inválido (use YYYY-MM-DD)")
		return
	}
	item, err := h.svc.UpdateItem(c.Request.Context(), appe.UpdateItemInput{
		WorkspaceID:         ws,
		ListID:              listID,
		ItemID:              itemID,
		Name:                body.Name,
		Category:            body.Category,
		Quantity:            body.Quantity,
		ReferencePriceCents: body.ReferencePriceCents,
		Purchased:           body.Purchased,
		PaidPriceCents:      body.PaidPriceCents,
		PurchasedAt:         purchasedAt,
		Store:               body.Store,
		Notes:               body.Notes,
	})
	if err != nil {
		writeEduErr(c, err)
		return
	}
	c.JSON(http.StatusOK, mapItem(item))
}

func (h *EducationHandler) DeleteItem(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	listID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "itemId inválido")
		return
	}
	if err := h.svc.DeleteItem(c.Request.Context(), ws, listID, itemID); err != nil {
		writeEduErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *EducationHandler) PurchaseItem(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	listID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "itemId inválido")
		return
	}
	var body purchaseJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	purchasedAt, err := parseOptionalDate(body.PurchasedAt)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "purchased_at inválido (use YYYY-MM-DD)")
		return
	}
	item, err := h.svc.PurchaseItem(c.Request.Context(), appe.PurchaseItemInput{
		WorkspaceID:    ws,
		ListID:         listID,
		ItemID:         itemID,
		PaidPriceCents: body.PaidPriceCents,
		PurchasedAt:    purchasedAt,
		Store:          body.Store,
	})
	if err != nil {
		writeEduErr(c, err)
		return
	}
	c.JSON(http.StatusOK, mapItem(item))
}

// ─── Handlers: dashboard ─────────────────────────────────────────────────────

type memberSpendResponse struct {
	MemberID       string  `json:"member_id"`
	MemberName     string  `json:"member_name"`
	TotalPaidCents int64   `json:"total_paid_cents"`
	ItemCount      int     `json:"item_count"`
	PurchasedCount int     `json:"purchased_count"`
	PurchasedPct   float64 `json:"purchased_pct"`
}

type categoryAvgResponse struct {
	Category       string `json:"category"`
	ItemCount      int    `json:"item_count"`
	PurchasedCount int    `json:"purchased_count"`
	TotalPaidCents int64  `json:"total_paid_cents"`
	AvgPaidCents   int64  `json:"avg_paid_cents"`
}

type yearSpendResponse struct {
	SchoolYear          int   `json:"school_year"`
	MonthlyFeesCents    int64 `json:"monthly_fees_cents"`
	EnrollmentFeesCents int64 `json:"enrollment_fees_cents"`
	SuppliesPaidCents   int64 `json:"supplies_paid_cents"`
	TotalCents          int64 `json:"total_cents"`
}

type dashboardResponse struct {
	SchoolYear          int                   `json:"school_year"`
	TotalReferenceCents int64                 `json:"total_reference_cents"`
	TotalPaidCents      int64                 `json:"total_paid_cents"`
	ListCount           int                   `json:"list_count"`
	ItemCount           int                   `json:"item_count"`
	PurchasedCount      int                   `json:"purchased_count"`
	PurchasedPct        float64               `json:"purchased_pct"`
	SavingsCents        int64                 `json:"savings_cents"`
	SavingsPct          float64               `json:"savings_pct"`
	MonthlyFeesCents    int64                 `json:"monthly_fees_cents"`
	EnrollmentFeesCents int64                 `json:"enrollment_fees_cents"`
	ByMember            []memberSpendResponse `json:"by_member"`
	ByCategory          []categoryAvgResponse `json:"by_category"`
	AnnualEvolution     []yearSpendResponse   `json:"annual_evolution"`
}

func (h *EducationHandler) Dashboard(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	schoolYear, _ := strconv.Atoi(c.Query("school_year"))
	d, err := h.svc.Dashboard(c.Request.Context(), ws, schoolYear)
	if err != nil {
		writeEduErr(c, err)
		return
	}
	resp := dashboardResponse{
		SchoolYear:          d.SchoolYear,
		TotalReferenceCents: d.TotalReferenceCents,
		TotalPaidCents:      d.TotalPaidCents,
		ListCount:           d.ListCount,
		ItemCount:           d.ItemCount,
		PurchasedCount:      d.PurchasedCount,
		PurchasedPct:        d.PurchasedPct,
		SavingsCents:        d.SavingsCents,
		SavingsPct:          d.SavingsPct,
		MonthlyFeesCents:    d.MonthlyFeesCents,
		EnrollmentFeesCents: d.EnrollmentFeesCents,
	}
	resp.ByMember = make([]memberSpendResponse, len(d.ByMember))
	for i, m := range d.ByMember {
		resp.ByMember[i] = memberSpendResponse{
			MemberID:       m.MemberID,
			MemberName:     m.MemberName,
			TotalPaidCents: m.TotalPaidCents,
			ItemCount:      m.ItemCount,
			PurchasedCount: m.PurchasedCount,
			PurchasedPct:   m.PurchasedPct,
		}
	}
	resp.ByCategory = make([]categoryAvgResponse, len(d.ByCategory))
	for i, ca := range d.ByCategory {
		resp.ByCategory[i] = categoryAvgResponse{
			Category:       ca.Category,
			ItemCount:      ca.ItemCount,
			PurchasedCount: ca.PurchasedCount,
			TotalPaidCents: ca.TotalPaidCents,
			AvgPaidCents:   ca.AvgPaidCents,
		}
	}
	resp.AnnualEvolution = make([]yearSpendResponse, len(d.AnnualEvolution))
	for i, ys := range d.AnnualEvolution {
		resp.AnnualEvolution[i] = yearSpendResponse{
			SchoolYear:          ys.SchoolYear,
			MonthlyFeesCents:    ys.MonthlyFeesCents,
			EnrollmentFeesCents: ys.EnrollmentFeesCents,
			SuppliesPaidCents:   ys.SuppliesPaidCents,
			TotalCents:          ys.TotalCents,
		}
	}
	c.JSON(http.StatusOK, resp)
}
