package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appv "github.com/retechfin/retechfin-api/internal/application/vehicle"
	dom "github.com/retechfin/retechfin-api/internal/domain/vehicle"
	"github.com/retechfin/retechfin-api/internal/infrastructure/fipe"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// VehicleHandler agrupa todos os endpoints do módulo de frota familiar.
type VehicleHandler struct {
	svc *appv.Service
}

func NewVehicleHandler(svc *appv.Service) *VehicleHandler {
	return &VehicleHandler{svc: svc}
}

// ─── Response types ────────────────────────────────────────────────────────────

type memberResponse struct {
	MemberID string `json:"member_id"`
	Role     string `json:"role"`
}

type vehicleResponse struct {
	ID               string           `json:"id"`
	Nickname         *string          `json:"nickname"`
	Make             string           `json:"make"`
	Model            string           `json:"model"`
	YearManufacture  int              `json:"year_manufacture"`
	YearModel        int              `json:"year_model"`
	Color            *string          `json:"color"`
	Plate            *string          `json:"plate"`
	FuelType         string           `json:"fuel_type"`
	FipeVehicleType  string           `json:"fipe_vehicle_type"`
	FipeCode         *string          `json:"fipe_code"`
	FipeBrandCode    *string          `json:"fipe_brand_code"`
	FipeModelCode    *string          `json:"fipe_model_code"`
	FipeYearCode     *string          `json:"fipe_year_code"`
	AcquisitionDate  *string          `json:"acquisition_date"`
	AcquisitionPrice *float64         `json:"acquisition_price"`
	CurrentOdometer  int              `json:"current_odometer"`
	Status           string           `json:"status"`
	SoldAt           *string          `json:"sold_at"`
	SoldPrice        *float64         `json:"sold_price"`
	Notes            *string          `json:"notes"`
	Members          []memberResponse `json:"members"`
	CreatedAt        string           `json:"created_at"`
	UpdatedAt        string           `json:"updated_at"`
}

func mapVehicle(v *dom.Vehicle) vehicleResponse {
	members := make([]memberResponse, len(v.Members))
	for i, m := range v.Members {
		members[i] = memberResponse{MemberID: m.MemberID.String(), Role: string(m.Role)}
	}
	r := vehicleResponse{
		ID:               v.ID.String(),
		Nickname:         v.Nickname,
		Make:             v.Make,
		Model:            v.Model,
		YearManufacture:  v.YearManufacture,
		YearModel:        v.YearModel,
		Color:            v.Color,
		Plate:            v.Plate,
		FuelType:         string(v.FuelType),
		FipeVehicleType:  v.FipeVehicleType,
		FipeCode:         v.FipeCode,
		FipeBrandCode:    v.FipeBrandCode,
		FipeModelCode:    v.FipeModelCode,
		FipeYearCode:     v.FipeYearCode,
		AcquisitionPrice: v.AcquisitionPrice,
		CurrentOdometer:  v.CurrentOdometer,
		Status:           string(v.Status),
		SoldPrice:        v.SoldPrice,
		Notes:            v.Notes,
		Members:          members,
		CreatedAt:        v.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        v.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if v.AcquisitionDate != nil {
		s := v.AcquisitionDate.Format("2006-01-02")
		r.AcquisitionDate = &s
	}
	if v.SoldAt != nil {
		s := v.SoldAt.Format("2006-01-02")
		r.SoldAt = &s
	}
	return r
}

type maintenanceResponse struct {
	ID                  string   `json:"id"`
	VehicleID           string   `json:"vehicle_id"`
	TemplateID          *string  `json:"template_id"`
	Type                string   `json:"type"`
	Title               string   `json:"title"`
	Description         *string  `json:"description"`
	OdometerAtService   *int     `json:"odometer_at_service"`
	ServiceDate         string   `json:"service_date"`
	Cost                *float64 `json:"cost"`
	SupplierID          *string  `json:"supplier_id"`
	NextServiceOdometer *int     `json:"next_service_odometer"`
	NextServiceDate     *string  `json:"next_service_date"`
	Notes               *string  `json:"notes"`
	CreatedAt           string   `json:"created_at"`
}

func mapMaintenance(m *dom.VehicleMaintenance) maintenanceResponse {
	r := maintenanceResponse{
		ID:                  m.ID.String(),
		VehicleID:           m.VehicleID.String(),
		Type:                m.Type,
		Title:               m.Title,
		Description:         m.Description,
		OdometerAtService:   m.OdometerAtService,
		ServiceDate:         m.ServiceDate.Format("2006-01-02"),
		Cost:                m.Cost,
		NextServiceOdometer: m.NextServiceOdometer,
		Notes:               m.Notes,
		CreatedAt:           m.CreatedAt.UTC().Format(time.RFC3339),
	}
	if m.TemplateID != nil {
		s := m.TemplateID.String()
		r.TemplateID = &s
	}
	if m.SupplierID != nil {
		s := m.SupplierID.String()
		r.SupplierID = &s
	}
	if m.NextServiceDate != nil {
		s := m.NextServiceDate.Format("2006-01-02")
		r.NextServiceDate = &s
	}
	return r
}

type planResponse struct {
	ID           *string `json:"id,omitempty"`
	TemplateID   string  `json:"template_id"`
	Type         string  `json:"type"`
	Name         string  `json:"name"`
	IntervalKM   *int    `json:"interval_km"`
	IntervalDays *int    `json:"interval_days"`
	Enabled      bool    `json:"enabled"`
	IsCustomized bool    `json:"is_customized"`
}

func mapPlan(p dom.VehicleMaintenancePlan) planResponse {
	r := planResponse{
		TemplateID:   p.TemplateID.String(),
		IntervalKM:   p.IntervalKM,
		IntervalDays: p.IntervalDays,
		Enabled:      p.Enabled,
		IsCustomized: p.ID != uuid.Nil,
	}
	if p.ID != uuid.Nil {
		s := p.ID.String()
		r.ID = &s
	}
	if p.Template != nil {
		r.Type = p.Template.Type
		r.Name = p.Template.Name
	}
	return r
}

type alertResponse struct {
	TemplateID    string   `json:"template_id"`
	Type          string   `json:"type"`
	Title         string   `json:"title"`
	Status        string   `json:"status"`
	DueAtKM       *int     `json:"due_at_km"`
	DueAtDate     *string  `json:"due_at_date"`
	KMRemaining   *int     `json:"km_remaining"`
	DaysRemaining *int     `json:"days_remaining"`
	LastOdometer  *int     `json:"last_odometer"`
	LastDate      *string  `json:"last_date"`
}

func mapAlert(a dom.MaintenanceAlert) alertResponse {
	r := alertResponse{
		TemplateID:    a.TemplateID.String(),
		Type:          a.Type,
		Title:         a.Title,
		Status:        string(a.Status),
		DueAtKM:       a.DueAtKM,
		KMRemaining:   a.KMRemaining,
		DaysRemaining: a.DaysRemaining,
		LastOdometer:  a.LastOdometer,
	}
	if a.DueAtDate != nil {
		s := a.DueAtDate.Format("2006-01-02")
		r.DueAtDate = &s
	}
	if a.LastDate != nil {
		s := a.LastDate.Format("2006-01-02")
		r.LastDate = &s
	}
	return r
}

type depreciationResponse struct {
	AcquisitionPrice     *float64            `json:"acquisition_price"`
	CurrentFipeValue     *float64            `json:"current_fipe_value"`
	TotalDepreciationPct *float64            `json:"total_depreciation_pct"`
	TotalDepreciationR   *float64            `json:"total_depreciation_r"`
	MonthsOwned          int                 `json:"months_owned"`
	MonthlyAvgDeprecR    *float64            `json:"monthly_avg_deprec_r"`
	AnnualAvgDeprecR     *float64            `json:"annual_avg_deprec_r"`
	Trend6MonthsR        *float64            `json:"trend_6months_r"`
	History              []fipeHistoryPoint  `json:"history"`
}

type fipeHistoryPoint struct {
	ReferenceMonth string  `json:"reference_month"`
	FipeValue      float64 `json:"fipe_value"`
	FipeFuel       *string `json:"fipe_fuel"`
}

// ─── Input types ──────────────────────────────────────────────────────────────

type memberInput struct {
	MemberID string `json:"member_id"`
	Role     string `json:"role"`
}

type vehicleCreateJSON struct {
	Nickname         *string       `json:"nickname"`
	Make             string        `json:"make" binding:"required"`
	Model            string        `json:"model" binding:"required"`
	YearManufacture  int           `json:"year_manufacture" binding:"required"`
	YearModel        int           `json:"year_model" binding:"required"`
	Color            *string       `json:"color"`
	Plate            *string       `json:"plate"`
	FuelType         string        `json:"fuel_type" binding:"required"`
	FipeVehicleType  string        `json:"fipe_vehicle_type"`
	FipeCode         *string       `json:"fipe_code"`
	FipeBrandCode    *string       `json:"fipe_brand_code"`
	FipeModelCode    *string       `json:"fipe_model_code"`
	FipeYearCode     *string       `json:"fipe_year_code"`
	AcquisitionDate  *string       `json:"acquisition_date"`
	AcquisitionPrice *float64      `json:"acquisition_price"`
	CurrentOdometer  int           `json:"current_odometer"`
	Notes            *string       `json:"notes"`
	Members          []memberInput `json:"members"`
}

type vehicleUpdateJSON struct {
	Nickname         *string       `json:"nickname"`
	Make             string        `json:"make" binding:"required"`
	Model            string        `json:"model" binding:"required"`
	YearManufacture  int           `json:"year_manufacture" binding:"required"`
	YearModel        int           `json:"year_model" binding:"required"`
	Color            *string       `json:"color"`
	Plate            *string       `json:"plate"`
	FuelType         string        `json:"fuel_type" binding:"required"`
	FipeVehicleType  string        `json:"fipe_vehicle_type"`
	FipeCode         *string       `json:"fipe_code"`
	FipeBrandCode    *string       `json:"fipe_brand_code"`
	FipeModelCode    *string       `json:"fipe_model_code"`
	FipeYearCode     *string       `json:"fipe_year_code"`
	AcquisitionDate  *string       `json:"acquisition_date"`
	AcquisitionPrice *float64      `json:"acquisition_price"`
	CurrentOdometer  int           `json:"current_odometer"`
	Status           string        `json:"status"`
	SoldAt           *string       `json:"sold_at"`
	SoldPrice        *float64      `json:"sold_price"`
	Notes            *string       `json:"notes"`
	Members          []memberInput `json:"members"`
}

type maintenanceCreateJSON struct {
	TemplateID          *string  `json:"template_id"`
	Type                string   `json:"type" binding:"required"`
	Title               string   `json:"title" binding:"required"`
	Description         *string  `json:"description"`
	OdometerAtService   *int     `json:"odometer_at_service"`
	ServiceDate         string   `json:"service_date" binding:"required"`
	Cost                *float64 `json:"cost"`
	SupplierID          *string  `json:"supplier_id"`
	NextServiceOdometer *int     `json:"next_service_odometer"`
	NextServiceDate     *string  `json:"next_service_date"`
	Notes               *string  `json:"notes"`
}

type maintenanceUpdateJSON struct {
	Title               string   `json:"title" binding:"required"`
	Description         *string  `json:"description"`
	OdometerAtService   *int     `json:"odometer_at_service"`
	ServiceDate         string   `json:"service_date" binding:"required"`
	Cost                *float64 `json:"cost"`
	SupplierID          *string  `json:"supplier_id"`
	NextServiceOdometer *int     `json:"next_service_odometer"`
	NextServiceDate     *string  `json:"next_service_date"`
	Notes               *string  `json:"notes"`
}

type planUpdateJSON struct {
	IntervalKM   *int `json:"interval_km"`
	IntervalDays *int `json:"interval_days"`
	Enabled      bool `json:"enabled"`
}

type odometerUpdateJSON struct {
	Odometer int `json:"odometer" binding:"min=0"`
}

// ─── Handlers: vehicles ───────────────────────────────────────────────────────

func (h *VehicleHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	result, err := h.svc.List(c.Request.Context(), ws, status, limit, offset)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]vehicleResponse, len(result.Items))
	for i := range result.Items {
		items[i] = mapVehicle(&result.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": result.Total})
}

func (h *VehicleHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body vehicleCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}

	acqDate, err := parseOptionalDate(body.AcquisitionDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "acquisition_date inválido (use YYYY-MM-DD)")
		return
	}

	members, err := parseMemberInputs(body.Members)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, err.Error())
		return
	}

	v, err := h.svc.Create(c.Request.Context(), appv.CreateVehicleInput{
		WorkspaceID:      ws,
		Nickname:         body.Nickname,
		Make:             body.Make,
		Model:            body.Model,
		YearManufacture:  body.YearManufacture,
		YearModel:        body.YearModel,
		Color:            body.Color,
		Plate:            body.Plate,
		FuelType:         body.FuelType,
		FipeVehicleType:  body.FipeVehicleType,
		FipeCode:         body.FipeCode,
		FipeBrandCode:    body.FipeBrandCode,
		FipeModelCode:    body.FipeModelCode,
		FipeYearCode:     body.FipeYearCode,
		AcquisitionDate:  acqDate,
		AcquisitionPrice: body.AcquisitionPrice,
		CurrentOdometer:  body.CurrentOdometer,
		Notes:            body.Notes,
		MemberIDs:        members,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapVehicle(v))
}

func (h *VehicleHandler) Get(c *gin.Context) {
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
	v, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapVehicle(v))
}

func (h *VehicleHandler) Update(c *gin.Context) {
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
	var body vehicleUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}

	acqDate, err := parseOptionalDate(body.AcquisitionDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "acquisition_date inválido")
		return
	}
	soldAt, err := parseOptionalDate(body.SoldAt)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "sold_at inválido")
		return
	}

	members, err := parseMemberInputs(body.Members)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, err.Error())
		return
	}

	v, err := h.svc.Update(c.Request.Context(), appv.UpdateVehicleInput{
		WorkspaceID:      ws,
		ID:               id,
		Nickname:         body.Nickname,
		Make:             body.Make,
		Model:            body.Model,
		YearManufacture:  body.YearManufacture,
		YearModel:        body.YearModel,
		Color:            body.Color,
		Plate:            body.Plate,
		FuelType:         body.FuelType,
		FipeVehicleType:  body.FipeVehicleType,
		FipeCode:         body.FipeCode,
		FipeBrandCode:    body.FipeBrandCode,
		FipeModelCode:    body.FipeModelCode,
		FipeYearCode:     body.FipeYearCode,
		AcquisitionDate:  acqDate,
		AcquisitionPrice: body.AcquisitionPrice,
		CurrentOdometer:  body.CurrentOdometer,
		Status:           body.Status,
		SoldAt:           soldAt,
		SoldPrice:        body.SoldPrice,
		Notes:            body.Notes,
		MemberIDs:        members,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapVehicle(v))
}

func (h *VehicleHandler) Delete(c *gin.Context) {
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

func (h *VehicleHandler) UpdateOdometer(c *gin.Context) {
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
	var body odometerUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	v, err := h.svc.UpdateOdometer(c.Request.Context(), ws, id, body.Odometer)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapVehicle(v))
}

// ─── Handlers: maintenance ────────────────────────────────────────────────────

func (h *VehicleHandler) ListMaintenance(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	vehicleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	items, err := h.svc.ListMaintenance(c.Request.Context(), ws, vehicleID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	resp := make([]maintenanceResponse, len(items))
	for i := range items {
		resp[i] = mapMaintenance(&items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": resp})
}

func (h *VehicleHandler) CreateMaintenance(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	vehicleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	var body maintenanceCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}

	serviceDate, err := time.Parse("2006-01-02", body.ServiceDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "service_date inválido (use YYYY-MM-DD)")
		return
	}
	nextDate, err := parseOptionalDate(body.NextServiceDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "next_service_date inválido")
		return
	}

	in := appv.CreateMaintenanceInput{
		WorkspaceID:         ws,
		VehicleID:           vehicleID,
		Type:                body.Type,
		Title:               body.Title,
		Description:         body.Description,
		OdometerAtService:   body.OdometerAtService,
		ServiceDate:         serviceDate,
		Cost:                body.Cost,
		NextServiceOdometer: body.NextServiceOdometer,
		NextServiceDate:     nextDate,
		Notes:               body.Notes,
	}
	if body.TemplateID != nil {
		tid, err := uuid.Parse(*body.TemplateID)
		if err == nil {
			in.TemplateID = &tid
		}
	}
	if body.SupplierID != nil {
		sid, err := uuid.Parse(*body.SupplierID)
		if err == nil {
			in.SupplierID = &sid
		}
	}

	m, err := h.svc.LogMaintenance(c.Request.Context(), in)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapMaintenance(m))
}

func (h *VehicleHandler) UpdateMaintenance(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	mID, err := uuid.Parse(c.Param("mainId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "maintenance id inválido")
		return
	}
	var body maintenanceUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	serviceDate, err := time.Parse("2006-01-02", body.ServiceDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "service_date inválido")
		return
	}
	nextDate, _ := parseOptionalDate(body.NextServiceDate)

	in := appv.UpdateMaintenanceInput{
		WorkspaceID:         ws,
		ID:                  mID,
		Title:               body.Title,
		Description:         body.Description,
		OdometerAtService:   body.OdometerAtService,
		ServiceDate:         serviceDate,
		Cost:                body.Cost,
		NextServiceOdometer: body.NextServiceOdometer,
		NextServiceDate:     nextDate,
		Notes:               body.Notes,
	}
	if body.SupplierID != nil {
		sid, err := uuid.Parse(*body.SupplierID)
		if err == nil {
			in.SupplierID = &sid
		}
	}

	m, err := h.svc.UpdateMaintenance(c.Request.Context(), in)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapMaintenance(m))
}

func (h *VehicleHandler) DeleteMaintenance(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	mID, err := uuid.Parse(c.Param("mainId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "maintenance id inválido")
		return
	}
	if err := h.svc.DeleteMaintenance(c.Request.Context(), ws, mID); err != nil {
		errrespond.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ─── Handlers: plans ──────────────────────────────────────────────────────────

func (h *VehicleHandler) ListPlans(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	vehicleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	plans, err := h.svc.GetPlans(c.Request.Context(), ws, vehicleID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	resp := make([]planResponse, len(plans))
	for i, p := range plans {
		resp[i] = mapPlan(p)
	}
	c.JSON(http.StatusOK, gin.H{"items": resp})
}

func (h *VehicleHandler) UpdatePlan(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	vehicleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "template_id inválido")
		return
	}
	var body planUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	p, err := h.svc.UpdatePlan(c.Request.Context(), appv.UpdatePlanInput{
		WorkspaceID:  ws,
		VehicleID:    vehicleID,
		TemplateID:   templateID,
		IntervalKM:   body.IntervalKM,
		IntervalDays: body.IntervalDays,
		Enabled:      body.Enabled,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapPlan(*p))
}

// ─── Handlers: alerts ─────────────────────────────────────────────────────────

func (h *VehicleHandler) GetAlerts(c *gin.Context) {
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
	alerts, err := h.svc.CalculateAlerts(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	resp := make([]alertResponse, len(alerts))
	for i, a := range alerts {
		resp[i] = mapAlert(a)
	}
	c.JSON(http.StatusOK, gin.H{"items": resp})
}

// ─── Handlers: FIPE search ───────────────────────────────────────────────────

func (h *VehicleHandler) FipeBrands(c *gin.Context) {
	vt := c.DefaultQuery("type", fipe.VehicleCarros)
	brands, err := h.svc.FipeListBrands(c.Request.Context(), vt)
	if err != nil {
		errrespond.Message(c, http.StatusBadGateway, errrespond.CodeInternal, "erro ao consultar FIPE: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, brands)
}

func (h *VehicleHandler) FipeModels(c *gin.Context) {
	vt := c.DefaultQuery("type", fipe.VehicleCarros)
	brandCode := c.Query("brand_code")
	if brandCode == "" {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "brand_code obrigatório")
		return
	}
	models, err := h.svc.FipeListModels(c.Request.Context(), vt, brandCode)
	if err != nil {
		errrespond.Message(c, http.StatusBadGateway, errrespond.CodeInternal, "erro ao consultar FIPE: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, models)
}

func (h *VehicleHandler) FipeYears(c *gin.Context) {
	vt := c.DefaultQuery("type", fipe.VehicleCarros)
	brandCode := c.Query("brand_code")
	modelCode := c.Query("model_code")
	if brandCode == "" || modelCode == "" {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "brand_code e model_code obrigatórios")
		return
	}
	years, err := h.svc.FipeListYears(c.Request.Context(), vt, brandCode, modelCode)
	if err != nil {
		errrespond.Message(c, http.StatusBadGateway, errrespond.CodeInternal, "erro ao consultar FIPE: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, years)
}

func (h *VehicleHandler) FipePrice(c *gin.Context) {
	vt := c.DefaultQuery("type", fipe.VehicleCarros)
	brandCode := c.Query("brand_code")
	modelCode := c.Query("model_code")
	yearCode := c.Query("year_code")
	if brandCode == "" || modelCode == "" || yearCode == "" {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "brand_code, model_code e year_code obrigatórios")
		return
	}
	price, err := h.svc.FipeGetPrice(c.Request.Context(), vt, brandCode, modelCode, yearCode)
	if err != nil {
		errrespond.Message(c, http.StatusBadGateway, errrespond.CodeInternal, "erro ao consultar FIPE: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, price)
}

// ─── Handlers: depreciation & FIPE history ───────────────────────────────────

func (h *VehicleHandler) GetDepreciation(c *gin.Context) {
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
	report, err := h.svc.GetDepreciation(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	history := make([]fipeHistoryPoint, len(report.History))
	for i, h := range report.History {
		history[i] = fipeHistoryPoint{
			ReferenceMonth: h.ReferenceMonth,
			FipeValue:      h.FipeValue,
			FipeFuel:       h.FipeFuel,
		}
	}
	c.JSON(http.StatusOK, depreciationResponse{
		AcquisitionPrice:     report.AcquisitionPrice,
		CurrentFipeValue:     report.CurrentFipeValue,
		TotalDepreciationPct: report.TotalDepreciationPct,
		TotalDepreciationR:   report.TotalDepreciationR,
		MonthsOwned:          report.MonthsOwned,
		MonthlyAvgDeprecR:    report.MonthlyAvgDeprecR,
		AnnualAvgDeprecR:     report.AnnualAvgDeprecR,
		Trend6MonthsR:        report.Trend6MonthsR,
		History:              history,
	})
}

func (h *VehicleHandler) GetFipeHistory(c *gin.Context) {
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
	history, err := h.svc.ListFipeHistory(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	resp := make([]fipeHistoryPoint, len(history))
	for i, h := range history {
		resp[i] = fipeHistoryPoint{
			ReferenceMonth: h.ReferenceMonth,
			FipeValue:      h.FipeValue,
			FipeFuel:       h.FipeFuel,
		}
	}
	c.JSON(http.StatusOK, gin.H{"items": resp})
}

func (h *VehicleHandler) GetFipeAllYears(c *gin.Context) {
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
	v, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	if v.FipeBrandCode == nil || v.FipeModelCode == nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "veículo sem códigos FIPE configurados")
		return
	}
	prices, err := h.svc.FipeGetAllYearPrices(c.Request.Context(), v.FipeVehicleType, *v.FipeBrandCode, *v.FipeModelCode)
	if err != nil {
		errrespond.Message(c, http.StatusBadGateway, errrespond.CodeInternal, "erro ao consultar FIPE: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": prices})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func parseOptionalDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func parseMemberInputs(inputs []memberInput) ([]appv.MemberInput, error) {
	out := make([]appv.MemberInput, 0, len(inputs))
	for _, m := range inputs {
		id, err := uuid.Parse(m.MemberID)
		if err != nil {
			return nil, &dom.ValidationError{Msg: "member_id inválido: " + m.MemberID}
		}
		out = append(out, appv.MemberInput{MemberID: id, Role: m.Role})
	}
	return out, nil
}
