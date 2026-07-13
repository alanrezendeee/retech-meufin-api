package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appv "github.com/retechfin/retechfin-api/internal/application/vehicle"
	dom "github.com/retechfin/retechfin-api/internal/domain/vehicle"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// ─── Response types ────────────────────────────────────────────────────────────

type serviceOrderItemResponse struct {
	ID                        string   `json:"id"`
	ServiceOrderID            string   `json:"service_order_id"`
	ItemType                  string   `json:"item_type"`
	Category                  string   `json:"category"`
	Description               string   `json:"description"`
	Quantity                  float64  `json:"quantity"`
	UnitPriceCents            int64    `json:"unit_price_cents"`
	TotalPriceCents           int64    `json:"total_price_cents"`
	KMAtInstallation          *int     `json:"km_at_installation"`
	ReplacementIntervalKM     *int     `json:"replacement_interval_km"`
	ReplacementIntervalMonths *int     `json:"replacement_interval_months"`
	NextDueKM                 *int     `json:"next_due_km"`
	NextDueDate               *string  `json:"next_due_date"`
	WarrantyExpiresDate       *string  `json:"warranty_expires_date"`
	WarrantyExpiresKM         *int     `json:"warranty_expires_km"`
	Notes                     *string  `json:"notes"`
	CreatedAt                 string   `json:"created_at"`
}

func mapOSItem(item *dom.ServiceOrderItem) serviceOrderItemResponse {
	r := serviceOrderItemResponse{
		ID:                        item.ID.String(),
		ServiceOrderID:            item.ServiceOrderID.String(),
		ItemType:                  string(item.ItemType),
		Category:                  string(item.Category),
		Description:               item.Description,
		Quantity:                  item.Quantity,
		UnitPriceCents:            item.UnitPriceCents,
		TotalPriceCents:           item.TotalPriceCents,
		KMAtInstallation:          item.KMAtInstallation,
		ReplacementIntervalKM:     item.ReplacementIntervalKM,
		ReplacementIntervalMonths: item.ReplacementIntervalMonths,
		NextDueKM:                 item.NextDueKM,
		WarrantyExpiresKM:         item.WarrantyExpiresKM,
		Notes:                     item.Notes,
		CreatedAt:                 item.CreatedAt.UTC().Format(time.RFC3339),
	}
	if item.NextDueDate != nil {
		s := item.NextDueDate.Format("2006-01-02")
		r.NextDueDate = &s
	}
	if item.WarrantyExpiresDate != nil {
		s := item.WarrantyExpiresDate.Format("2006-01-02")
		r.WarrantyExpiresDate = &s
	}
	return r
}

type serviceOrderResponse struct {
	ID                 string                     `json:"id"`
	VehicleID          string                     `json:"vehicle_id"`
	SupplierID         *string                    `json:"supplier_id"`
	OSNumber           *string                    `json:"os_number"`
	ServiceDate        string                     `json:"service_date"`
	KMAtService        int                        `json:"km_at_service"`
	TotalProductsCents int64                      `json:"total_products_cents"`
	TotalServicesCents int64                      `json:"total_services_cents"`
	TotalCents         int64                      `json:"total_cents"`
	PaymentMethod      *string                    `json:"payment_method"`
	Technician         *string                    `json:"technician"`
	Notes              *string                    `json:"notes"`
	Status             string                     `json:"status"`
	Items              []serviceOrderItemResponse `json:"items"`
	CreatedAt          string                     `json:"created_at"`
	UpdatedAt          string                     `json:"updated_at"`
}

func mapServiceOrder(o *dom.ServiceOrder) serviceOrderResponse {
	r := serviceOrderResponse{
		ID:                 o.ID.String(),
		VehicleID:          o.VehicleID.String(),
		OSNumber:           o.OSNumber,
		ServiceDate:        o.ServiceDate.Format("2006-01-02"),
		KMAtService:        o.KMAtService,
		TotalProductsCents: o.TotalProductsCents,
		TotalServicesCents: o.TotalServicesCents,
		TotalCents:         o.TotalCents,
		PaymentMethod:      o.PaymentMethod,
		Technician:         o.Technician,
		Notes:              o.Notes,
		Status:             string(o.Status),
		CreatedAt:          o.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:          o.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if o.SupplierID != nil {
		s := o.SupplierID.String()
		r.SupplierID = &s
	}
	r.Items = make([]serviceOrderItemResponse, len(o.Items))
	for i := range o.Items {
		r.Items[i] = mapOSItem(&o.Items[i])
	}
	return r
}

type catalogItemResponse struct {
	ID                    string  `json:"id"`
	Category              string  `json:"category"`
	ItemType              string  `json:"item_type"`
	Name                  string  `json:"name"`
	Description           *string `json:"description"`
	DefaultIntervalKM     *int    `json:"default_interval_km"`
	DefaultIntervalMonths *int    `json:"default_interval_months"`
	DefaultWarrantyMonths *int    `json:"default_warranty_months"`
}

func mapCatalogItem(c dom.MaintenanceCatalogItem) catalogItemResponse {
	return catalogItemResponse{
		ID:                    c.ID.String(),
		Category:              string(c.Category),
		ItemType:              string(c.ItemType),
		Name:                  c.Name,
		Description:           c.Description,
		DefaultIntervalKM:     c.DefaultIntervalKM,
		DefaultIntervalMonths: c.DefaultIntervalMonths,
		DefaultWarrantyMonths: c.DefaultWarrantyMonths,
	}
}

type scheduleResponse struct {
	ID                 string  `json:"id"`
	VehicleID          string  `json:"vehicle_id"`
	ServiceOrderItemID *string `json:"service_order_item_id"`
	Description        string  `json:"description"`
	Category           string  `json:"category"`
	ScheduledKM        *int    `json:"scheduled_km"`
	ScheduledDate      *string `json:"scheduled_date"`
	AlertStatus        string  `json:"alert_status"`
	CompletedAt        *string `json:"completed_at"`
	Notes              *string `json:"notes"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

func mapSchedule(s *dom.MaintenanceSchedule) scheduleResponse {
	r := scheduleResponse{
		ID:          s.ID.String(),
		VehicleID:   s.VehicleID.String(),
		Description: s.Description,
		Category:    string(s.Category),
		ScheduledKM: s.ScheduledKM,
		AlertStatus: string(s.AlertStatus),
		Notes:       s.Notes,
		CreatedAt:   s.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   s.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if s.ServiceOrderItemID != nil {
		v := s.ServiceOrderItemID.String()
		r.ServiceOrderItemID = &v
	}
	if s.ScheduledDate != nil {
		v := s.ScheduledDate.Format("2006-01-02")
		r.ScheduledDate = &v
	}
	if s.CompletedAt != nil {
		v := s.CompletedAt.Format("2006-01-02")
		r.CompletedAt = &v
	}
	return r
}

type analyticsResponse struct {
	TotalSpentCents    int64                    `json:"total_spent_cents"`
	TotalProductsCents int64                    `json:"total_products_cents"`
	TotalServicesCents int64                    `json:"total_services_cents"`
	CostPerKM          *float64                 `json:"cost_per_km"`
	TotalOSCount       int                      `json:"total_os_count"`
	AvgCostPerOSCents  int64                    `json:"avg_cost_per_os_cents"`
	SpendingByCategory []categorySpendingResp   `json:"spending_by_category"`
	SpendingBySupplier []supplierSpendingResp   `json:"spending_by_supplier"`
	MonthlySpending    []monthlySpendingResp    `json:"monthly_spending"`
}

type categorySpendingResp struct {
	Category   string `json:"category"`
	TotalCents int64  `json:"total_cents"`
}
type supplierSpendingResp struct {
	SupplierID   string `json:"supplier_id"`
	SupplierName string `json:"supplier_name"`
	TotalCents   int64  `json:"total_cents"`
}
type monthlySpendingResp struct {
	Month      string `json:"month"`
	TotalCents int64  `json:"total_cents"`
}

func mapAnalytics(a *dom.VehicleAnalytics) analyticsResponse {
	r := analyticsResponse{
		TotalSpentCents:    a.TotalSpentCents,
		TotalProductsCents: a.TotalProductsCents,
		TotalServicesCents: a.TotalServicesCents,
		CostPerKM:          a.CostPerKM,
		TotalOSCount:       a.TotalOSCount,
		AvgCostPerOSCents:  a.AvgCostPerOSCents,
	}
	r.SpendingByCategory = make([]categorySpendingResp, len(a.SpendingByCategory))
	for i, v := range a.SpendingByCategory {
		r.SpendingByCategory[i] = categorySpendingResp{Category: v.Category, TotalCents: v.TotalCents}
	}
	r.SpendingBySupplier = make([]supplierSpendingResp, len(a.SpendingBySupplier))
	for i, v := range a.SpendingBySupplier {
		r.SpendingBySupplier[i] = supplierSpendingResp{SupplierID: v.SupplierID, SupplierName: v.SupplierName, TotalCents: v.TotalCents}
	}
	r.MonthlySpending = make([]monthlySpendingResp, len(a.MonthlySpending))
	for i, v := range a.MonthlySpending {
		r.MonthlySpending[i] = monthlySpendingResp{Month: v.Month, TotalCents: v.TotalCents}
	}
	return r
}

// ─── Input types ──────────────────────────────────────────────────────────────

type osItemJSON struct {
	ItemType                  string   `json:"item_type" binding:"required"`
	Category                  string   `json:"category"`
	Description               string   `json:"description" binding:"required"`
	Quantity                  float64  `json:"quantity"`
	UnitPriceCents            int64    `json:"unit_price_cents"`
	KMAtInstallation          *int     `json:"km_at_installation"`
	ReplacementIntervalKM     *int     `json:"replacement_interval_km"`
	ReplacementIntervalMonths *int     `json:"replacement_interval_months"`
	WarrantyExpiresDate       *string  `json:"warranty_expires_date"`
	WarrantyExpiresKM         *int     `json:"warranty_expires_km"`
	Notes                     *string  `json:"notes"`
}

type serviceOrderCreateJSON struct {
	SupplierID    *string      `json:"supplier_id"`
	OSNumber      *string      `json:"os_number"`
	ServiceDate   string       `json:"service_date" binding:"required"`
	KMAtService   int          `json:"km_at_service"`
	PaymentMethod *string      `json:"payment_method"`
	Technician    *string      `json:"technician"`
	Notes         *string      `json:"notes"`
	Status        string       `json:"status"`
	Items         []osItemJSON `json:"items"`
}

type serviceOrderUpdateJSON struct {
	SupplierID    *string `json:"supplier_id"`
	OSNumber      *string `json:"os_number"`
	ServiceDate   string  `json:"service_date" binding:"required"`
	KMAtService   int     `json:"km_at_service"`
	PaymentMethod *string `json:"payment_method"`
	Technician    *string `json:"technician"`
	Notes         *string `json:"notes"`
	Status        string  `json:"status"`
}

type scheduleCreateJSON struct {
	ServiceOrderItemID *string `json:"service_order_item_id"`
	Description        string  `json:"description" binding:"required"`
	Category           string  `json:"category"`
	ScheduledKM        *int    `json:"scheduled_km"`
	ScheduledDate      *string `json:"scheduled_date"`
	Notes              *string `json:"notes"`
}

type scheduleUpdateJSON struct {
	Description   string  `json:"description" binding:"required"`
	Category      string  `json:"category"`
	ScheduledKM   *int    `json:"scheduled_km"`
	ScheduledDate *string `json:"scheduled_date"`
	AlertStatus   string  `json:"alert_status"`
	CompletedAt   *string `json:"completed_at"`
	Notes         *string `json:"notes"`
}

// ─── Handlers: Service Orders ─────────────────────────────────────────────────

func (h *VehicleHandler) ListServiceOrders(c *gin.Context) {
	ws, vid, ok := vehicleContext(c)
	if !ok {
		return
	}
	orders, err := h.svc.ListServiceOrders(c.Request.Context(), ws, vid)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]serviceOrderResponse, len(orders))
	for i := range orders {
		items[i] = mapServiceOrder(&orders[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *VehicleHandler) CreateServiceOrder(c *gin.Context) {
	ws, vid, ok := vehicleContext(c)
	if !ok {
		return
	}
	var body serviceOrderCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	serviceDate, err := time.Parse("2006-01-02", body.ServiceDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "service_date inválido")
		return
	}

	var supplierID *uuid.UUID
	if body.SupplierID != nil {
		sid, err := uuid.Parse(*body.SupplierID)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "supplier_id inválido")
			return
		}
		supplierID = &sid
	}

	items, err := parseOSItems(body.Items)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, err.Error())
		return
	}

	o, err := h.svc.CreateServiceOrder(c.Request.Context(), appv.CreateServiceOrderInput{
		WorkspaceID:   ws,
		VehicleID:     vid,
		SupplierID:    supplierID,
		OSNumber:      body.OSNumber,
		ServiceDate:   serviceDate,
		KMAtService:   body.KMAtService,
		PaymentMethod: body.PaymentMethod,
		Technician:    body.Technician,
		Notes:         body.Notes,
		Status:        body.Status,
		Items:         items,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapServiceOrder(o))
}

func (h *VehicleHandler) GetServiceOrder(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	osID, err := uuid.Parse(c.Param("osId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "osId inválido")
		return
	}
	o, err := h.svc.GetServiceOrder(c.Request.Context(), ws, osID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapServiceOrder(o))
}

func (h *VehicleHandler) UpdateServiceOrder(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	osID, err := uuid.Parse(c.Param("osId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "osId inválido")
		return
	}
	var body serviceOrderUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	serviceDate, err := time.Parse("2006-01-02", body.ServiceDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "service_date inválido")
		return
	}
	var supplierID *uuid.UUID
	if body.SupplierID != nil {
		sid, err := uuid.Parse(*body.SupplierID)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "supplier_id inválido")
			return
		}
		supplierID = &sid
	}
	o, err := h.svc.UpdateServiceOrder(c.Request.Context(), appv.UpdateServiceOrderInput{
		WorkspaceID:   ws,
		ID:            osID,
		SupplierID:    supplierID,
		OSNumber:      body.OSNumber,
		ServiceDate:   serviceDate,
		KMAtService:   body.KMAtService,
		PaymentMethod: body.PaymentMethod,
		Technician:    body.Technician,
		Notes:         body.Notes,
		Status:        body.Status,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapServiceOrder(o))
}

func (h *VehicleHandler) DeleteServiceOrder(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	osID, err := uuid.Parse(c.Param("osId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "osId inválido")
		return
	}
	if err := h.svc.DeleteServiceOrder(c.Request.Context(), ws, osID); err != nil {
		errrespond.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ─── Handlers: OS Items ───────────────────────────────────────────────────────

func (h *VehicleHandler) AddOSItem(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	osID, err := uuid.Parse(c.Param("osId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "osId inválido")
		return
	}
	var body osItemJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	warrantyDate, err := parseOptionalDate(body.WarrantyExpiresDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "warranty_expires_date inválido")
		return
	}
	item, err := h.svc.AddServiceOrderItem(c.Request.Context(), appv.AddServiceOrderItemInput{
		WorkspaceID:               ws,
		ServiceOrderID:            osID,
		ItemType:                  body.ItemType,
		Category:                  body.Category,
		Description:               body.Description,
		Quantity:                  body.Quantity,
		UnitPriceCents:            body.UnitPriceCents,
		KMAtInstallation:          body.KMAtInstallation,
		ReplacementIntervalKM:     body.ReplacementIntervalKM,
		ReplacementIntervalMonths: body.ReplacementIntervalMonths,
		WarrantyExpiresDate:       warrantyDate,
		WarrantyExpiresKM:         body.WarrantyExpiresKM,
		Notes:                     body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapOSItem(item))
}

func (h *VehicleHandler) UpdateOSItem(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "itemId inválido")
		return
	}
	var body osItemJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	warrantyDate, err := parseOptionalDate(body.WarrantyExpiresDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "warranty_expires_date inválido")
		return
	}
	item, err := h.svc.UpdateServiceOrderItem(c.Request.Context(), ws, itemID, appv.ServiceOrderItemInput{
		ItemType:                  body.ItemType,
		Category:                  body.Category,
		Description:               body.Description,
		Quantity:                  body.Quantity,
		UnitPriceCents:            body.UnitPriceCents,
		KMAtInstallation:          body.KMAtInstallation,
		ReplacementIntervalKM:     body.ReplacementIntervalKM,
		ReplacementIntervalMonths: body.ReplacementIntervalMonths,
		WarrantyExpiresDate:       warrantyDate,
		WarrantyExpiresKM:         body.WarrantyExpiresKM,
		Notes:                     body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapOSItem(item))
}

func (h *VehicleHandler) DeleteOSItem(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "itemId inválido")
		return
	}
	if err := h.svc.DeleteServiceOrderItem(c.Request.Context(), ws, itemID); err != nil {
		errrespond.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ─── Handlers: Catalog ────────────────────────────────────────────────────────

func (h *VehicleHandler) SearchCatalog(c *gin.Context) {
	query := c.Query("q")
	category := c.Query("category")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	items, err := h.svc.SearchCatalog(c.Request.Context(), query, category, limit)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	out := make([]catalogItemResponse, len(items))
	for i, it := range items {
		out[i] = mapCatalogItem(it)
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

// ─── Handlers: Schedules ──────────────────────────────────────────────────────

func (h *VehicleHandler) ListSchedules(c *gin.Context) {
	ws, vid, ok := vehicleContext(c)
	if !ok {
		return
	}
	scheds, err := h.svc.ListSchedules(c.Request.Context(), ws, vid)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]scheduleResponse, len(scheds))
	for i := range scheds {
		items[i] = mapSchedule(&scheds[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *VehicleHandler) CreateSchedule(c *gin.Context) {
	ws, vid, ok := vehicleContext(c)
	if !ok {
		return
	}
	var body scheduleCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	schedDate, err := parseOptionalDate(body.ScheduledDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "scheduled_date inválido")
		return
	}
	var osItemID *uuid.UUID
	if body.ServiceOrderItemID != nil {
		id, err := uuid.Parse(*body.ServiceOrderItemID)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "service_order_item_id inválido")
			return
		}
		osItemID = &id
	}
	sched, err := h.svc.CreateSchedule(c.Request.Context(), appv.CreateScheduleInput{
		WorkspaceID:        ws,
		VehicleID:          vid,
		ServiceOrderItemID: osItemID,
		Description:        body.Description,
		Category:           body.Category,
		ScheduledKM:        body.ScheduledKM,
		ScheduledDate:      schedDate,
		Notes:              body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapSchedule(sched))
}

func (h *VehicleHandler) UpdateSchedule(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	schedID, err := uuid.Parse(c.Param("schedId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "schedId inválido")
		return
	}
	var body scheduleUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	schedDate, err := parseOptionalDate(body.ScheduledDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "scheduled_date inválido")
		return
	}
	completedAt, err := parseOptionalDate(body.CompletedAt)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "completed_at inválido")
		return
	}
	sched, err := h.svc.UpdateSchedule(c.Request.Context(), appv.UpdateScheduleInput{
		WorkspaceID:   ws,
		ID:            schedID,
		Description:   body.Description,
		Category:      body.Category,
		ScheduledKM:   body.ScheduledKM,
		ScheduledDate: schedDate,
		AlertStatus:   body.AlertStatus,
		CompletedAt:   completedAt,
		Notes:         body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapSchedule(sched))
}

func (h *VehicleHandler) DeleteSchedule(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	schedID, err := uuid.Parse(c.Param("schedId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "schedId inválido")
		return
	}
	if err := h.svc.DeleteSchedule(c.Request.Context(), ws, schedID); err != nil {
		errrespond.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ─── Handlers: Analytics ──────────────────────────────────────────────────────

func (h *VehicleHandler) GetAnalytics(c *gin.Context) {
	ws, vid, ok := vehicleContext(c)
	if !ok {
		return
	}
	months, _ := strconv.Atoi(c.DefaultQuery("months", "12"))
	analytics, err := h.svc.GetAnalytics(c.Request.Context(), ws, vid, months)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapAnalytics(analytics))
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func vehicleContext(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return uuid.Nil, uuid.Nil, false
	}
	vid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "vehicle id inválido")
		return uuid.Nil, uuid.Nil, false
	}
	return ws, vid, true
}

func parseOSItems(raw []osItemJSON) ([]appv.ServiceOrderItemInput, error) {
	items := make([]appv.ServiceOrderItemInput, len(raw))
	for i, r := range raw {
		warrantyDate, err := parseOptionalDate(r.WarrantyExpiresDate)
		if err != nil {
			return nil, err
		}
		items[i] = appv.ServiceOrderItemInput{
			ItemType:                  r.ItemType,
			Category:                  r.Category,
			Description:               r.Description,
			Quantity:                  r.Quantity,
			UnitPriceCents:            r.UnitPriceCents,
			KMAtInstallation:          r.KMAtInstallation,
			ReplacementIntervalKM:     r.ReplacementIntervalKM,
			ReplacementIntervalMonths: r.ReplacementIntervalMonths,
			WarrantyExpiresDate:       warrantyDate,
			WarrantyExpiresKM:         r.WarrantyExpiresKM,
			Notes:                     r.Notes,
		}
	}
	return items, nil
}
