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

type maintenanceItemResponse struct {
	ID                        string   `json:"id"`
	MaintenanceID             string   `json:"maintenance_id"`
	CatalogItemID             *string  `json:"catalog_item_id"`
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

func mapMaintenanceItem(item *dom.VehicleMaintenanceItem) maintenanceItemResponse {
	r := maintenanceItemResponse{
		ID:                        item.ID.String(),
		MaintenanceID:             item.MaintenanceID.String(),
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
	if item.CatalogItemID != nil {
		s := item.CatalogItemID.String()
		r.CatalogItemID = &s
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
	ID                string  `json:"id"`
	VehicleID         string  `json:"vehicle_id"`
	MaintenanceID     *string `json:"maintenance_id"`
	MaintenanceItemID *string `json:"maintenance_item_id"`
	Description       string  `json:"description"`
	Category          string  `json:"category"`
	ScheduledKM       *int    `json:"scheduled_km"`
	ScheduledDate     *string `json:"scheduled_date"`
	AlertStatus       string  `json:"alert_status"`
	CompletedAt       *string `json:"completed_at"`
	Notes             *string `json:"notes"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
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
	if s.MaintenanceID != nil {
		v := s.MaintenanceID.String()
		r.MaintenanceID = &v
	}
	if s.MaintenanceItemID != nil {
		v := s.MaintenanceItemID.String()
		r.MaintenanceItemID = &v
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
	TotalSpentCents    int64                  `json:"total_spent_cents"`
	TotalProductsCents int64                  `json:"total_products_cents"`
	TotalServicesCents int64                  `json:"total_services_cents"`
	CostPerKM          *float64               `json:"cost_per_km"`
	TotalCount         int                    `json:"total_count"`
	AvgCostPerOSCents  int64                  `json:"avg_cost_per_os_cents"`
	SpendingByCategory []categorySpendingResp `json:"spending_by_category"`
	SpendingBySupplier []supplierSpendingResp `json:"spending_by_supplier"`
	MonthlySpending    []monthlySpendingResp  `json:"monthly_spending"`
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
		TotalCount:         a.TotalCount,
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

type maintenanceItemJSON struct {
	CatalogItemID             *string `json:"catalog_item_id"`
	ItemType                  string  `json:"item_type" binding:"required"`
	Category                  string  `json:"category"`
	Description               string  `json:"description" binding:"required"`
	Quantity                  float64 `json:"quantity"`
	UnitPriceCents            int64   `json:"unit_price_cents"`
	KMAtInstallation          *int    `json:"km_at_installation"`
	ReplacementIntervalKM     *int    `json:"replacement_interval_km"`
	ReplacementIntervalMonths *int    `json:"replacement_interval_months"`
	WarrantyExpiresDate       *string `json:"warranty_expires_date"`
	WarrantyExpiresKM         *int    `json:"warranty_expires_km"`
	Notes                     *string `json:"notes"`
}

type scheduleCreateJSON struct {
	MaintenanceID     *string `json:"maintenance_id"`
	MaintenanceItemID *string `json:"maintenance_item_id"`
	Description       string  `json:"description" binding:"required"`
	Category          string  `json:"category"`
	ScheduledKM       *int    `json:"scheduled_km"`
	ScheduledDate     *string `json:"scheduled_date"`
	Notes             *string `json:"notes"`
}

type scheduleUpdateJSON struct {
	MaintenanceID *string `json:"maintenance_id"`
	Description   string  `json:"description" binding:"required"`
	Category      string  `json:"category"`
	ScheduledKM   *int    `json:"scheduled_km"`
	ScheduledDate *string `json:"scheduled_date"`
	AlertStatus   string  `json:"alert_status"`
	CompletedAt   *string `json:"completed_at"`
	Notes         *string `json:"notes"`
}

// ─── Handlers: Maintenance Items ──────────────────────────────────────────────

func (h *VehicleHandler) AddMaintenanceItem(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	mID, err := uuid.Parse(c.Param("mId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "mId inválido")
		return
	}
	var body maintenanceItemJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	warrantyDate, err := parseOptionalDate(body.WarrantyExpiresDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "warranty_expires_date inválido")
		return
	}
	addInput := appv.AddMaintenanceItemInput{
		WorkspaceID:               ws,
		MaintenanceID:             mID,
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
	}
	if body.CatalogItemID != nil {
		cid, err := uuid.Parse(*body.CatalogItemID)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "catalog_item_id inválido")
			return
		}
		addInput.CatalogItemID = &cid
	}
	item, err := h.svc.AddMaintenanceItem(c.Request.Context(), addInput)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapMaintenanceItem(item))
}

func (h *VehicleHandler) UpdateMaintenanceItem(c *gin.Context) {
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
	var body maintenanceItemJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	warrantyDate, err := parseOptionalDate(body.WarrantyExpiresDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "warranty_expires_date inválido")
		return
	}
	updateInput := appv.MaintenanceItemInput{
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
	}
	if body.CatalogItemID != nil {
		cid, err := uuid.Parse(*body.CatalogItemID)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "catalog_item_id inválido")
			return
		}
		updateInput.CatalogItemID = &cid
	}
	item, err := h.svc.UpdateMaintenanceItem(c.Request.Context(), ws, itemID, updateInput)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapMaintenanceItem(item))
}

func (h *VehicleHandler) DeleteMaintenanceItem(c *gin.Context) {
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
	if err := h.svc.DeleteMaintenanceItem(c.Request.Context(), ws, itemID); err != nil {
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
	var maintID *uuid.UUID
	if body.MaintenanceID != nil {
		id, err := uuid.Parse(*body.MaintenanceID)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "maintenance_id inválido")
			return
		}
		maintID = &id
	}
	var maintItemID *uuid.UUID
	if body.MaintenanceItemID != nil {
		id, err := uuid.Parse(*body.MaintenanceItemID)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "maintenance_item_id inválido")
			return
		}
		maintItemID = &id
	}
	sched, err := h.svc.CreateSchedule(c.Request.Context(), appv.CreateScheduleInput{
		WorkspaceID:       ws,
		VehicleID:         vid,
		MaintenanceID:     maintID,
		MaintenanceItemID: maintItemID,
		Description:       body.Description,
		Category:          body.Category,
		ScheduledKM:       body.ScheduledKM,
		ScheduledDate:     schedDate,
		Notes:             body.Notes,
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
	var updMaintID *uuid.UUID
	if body.MaintenanceID != nil {
		id, err := uuid.Parse(*body.MaintenanceID)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "maintenance_id inválido")
			return
		}
		updMaintID = &id
	}
	sched, err := h.svc.UpdateSchedule(c.Request.Context(), appv.UpdateScheduleInput{
		WorkspaceID:   ws,
		ID:            schedID,
		MaintenanceID: updMaintID,
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

func parseMaintenanceItems(raw []maintenanceItemJSON) ([]appv.MaintenanceItemInput, error) {
	items := make([]appv.MaintenanceItemInput, len(raw))
	for i, r := range raw {
		warrantyDate, err := parseOptionalDate(r.WarrantyExpiresDate)
		if err != nil {
			return nil, err
		}
		inp := appv.MaintenanceItemInput{
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
		if r.CatalogItemID != nil {
			cid, err := uuid.Parse(*r.CatalogItemID)
			if err != nil {
				return nil, err
			}
			inp.CatalogItemID = &cid
		}
		items[i] = inp
	}
	return items, nil
}
