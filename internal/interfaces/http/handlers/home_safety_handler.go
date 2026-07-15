package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apphs "github.com/retechfin/retechfin-api/internal/application/homesafety"
	dom "github.com/retechfin/retechfin-api/internal/domain/homesafety"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// HomeSafetyHandler agrupa os endpoints do módulo de segurança do lar.
type HomeSafetyHandler struct {
	svc *apphs.Service
}

func NewHomeSafetyHandler(svc *apphs.Service) *HomeSafetyHandler {
	return &HomeSafetyHandler{svc: svc}
}

// ─── Response types ───────────────────────────────────────────────────────────

type homeSafetyItemResponse struct {
	ID                    string  `json:"id"`
	Name                  string  `json:"name"`
	Category              string  `json:"category"`
	RiskType              string  `json:"risk_type"`
	Location              *string `json:"location"`
	Brand                 *string `json:"brand"`
	Model                 *string `json:"model"`
	InstalledAt           *string `json:"installed_at"`
	LifespanMonths        *int    `json:"lifespan_months"`
	ServiceIntervalMonths *int    `json:"service_interval_months"`
	LastServiceAt         *string `json:"last_service_at"`
	NextDueDate           *string `json:"next_due_date"`
	Priority              string  `json:"priority"`
	Responsible           *string `json:"responsible"`
	LastCostCents         int64   `json:"last_cost_cents"`
	Active                bool    `json:"active"`
	Notes                 *string `json:"notes"`
	Status                string  `json:"status"`
	DaysUntilDue          *int    `json:"days_until_due"`
	CreatedAt             string  `json:"created_at"`
	UpdatedAt             string  `json:"updated_at"`
}

func mapHomeSafetyItem(i *dom.Item) homeSafetyItemResponse {
	now := time.Now().UTC()
	r := homeSafetyItemResponse{
		ID:                    i.ID.String(),
		Name:                  i.Name,
		Category:              string(i.Category),
		RiskType:              string(i.RiskType),
		Location:              i.Location,
		Brand:                 i.Brand,
		Model:                 i.Model,
		LifespanMonths:        i.LifespanMonths,
		ServiceIntervalMonths: i.ServiceIntervalMonths,
		Priority:              string(i.Priority),
		Responsible:           i.Responsible,
		LastCostCents:         i.LastCostCents,
		Active:                i.Active,
		Notes:                 i.Notes,
		Status:                string(i.Status(now)),
		DaysUntilDue:          i.DaysUntilDue(now),
		CreatedAt:             i.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:             i.UpdatedAt.UTC().Format(time.RFC3339),
	}
	r.InstalledAt = formatDatePtr(i.InstalledAt)
	r.LastServiceAt = formatDatePtr(i.LastServiceAt)
	r.NextDueDate = formatDatePtr(i.NextDueDate)
	return r
}

type homeSafetyEventResponse struct {
	ID        string  `json:"id"`
	ItemID    string  `json:"item_id"`
	EventType string  `json:"event_type"`
	EventDate string  `json:"event_date"`
	CostCents int64   `json:"cost_cents"`
	Provider  *string `json:"provider"`
	Notes     *string `json:"notes"`
	CreatedAt string  `json:"created_at"`
}

func mapHomeSafetyEvent(e *dom.Event) homeSafetyEventResponse {
	return homeSafetyEventResponse{
		ID:        e.ID.String(),
		ItemID:    e.ItemID.String(),
		EventType: string(e.EventType),
		EventDate: e.EventDate.Format("2006-01-02"),
		CostCents: e.CostCents,
		Provider:  e.Provider,
		Notes:     e.Notes,
		CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func formatDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

// ─── Input types ──────────────────────────────────────────────────────────────

type homeSafetyItemJSON struct {
	Name                  string  `json:"name" binding:"required"`
	Category              string  `json:"category"`
	RiskType              string  `json:"risk_type"`
	Location              *string `json:"location"`
	Brand                 *string `json:"brand"`
	Model                 *string `json:"model"`
	InstalledAt           *string `json:"installed_at"`
	LifespanMonths        *int    `json:"lifespan_months"`
	ServiceIntervalMonths *int    `json:"service_interval_months"`
	LastServiceAt         *string `json:"last_service_at"`
	Priority              string  `json:"priority"`
	Responsible           *string `json:"responsible"`
	LastCostCents         int64   `json:"last_cost_cents"`
	Active                *bool   `json:"active"`
	Notes                 *string `json:"notes"`
}

type homeSafetyEventJSON struct {
	EventType string  `json:"event_type"`
	EventDate string  `json:"event_date" binding:"required"`
	CostCents int64   `json:"cost_cents"`
	Provider  *string `json:"provider"`
	Notes     *string `json:"notes"`
}

// ─── Handlers: items ──────────────────────────────────────────────────────────

func (h *HomeSafetyHandler) ListItems(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	items, err := h.svc.List(c.Request.Context(), ws,
		c.Query("category"), c.Query("status"), c.Query("location"), c.Query("q"))
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	resp := make([]homeSafetyItemResponse, len(items))
	for i := range items {
		resp[i] = mapHomeSafetyItem(&items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": resp})
}

func (h *HomeSafetyHandler) CreateItem(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body homeSafetyItemJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	in, err := h.buildCreateInput(ws, body)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, err.Error())
		return
	}
	item, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapHomeSafetyItem(item))
}

func (h *HomeSafetyHandler) GetItem(c *gin.Context) {
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
	item, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapHomeSafetyItem(item))
}

func (h *HomeSafetyHandler) UpdateItem(c *gin.Context) {
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
	var body homeSafetyItemJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	base, err := h.buildCreateInput(ws, body)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, err.Error())
		return
	}
	item, err := h.svc.Update(c.Request.Context(), apphs.UpdateItemInput{
		WorkspaceID:           ws,
		ID:                    id,
		Name:                  base.Name,
		Category:              base.Category,
		RiskType:              base.RiskType,
		Location:              base.Location,
		Brand:                 base.Brand,
		Model:                 base.Model,
		InstalledAt:           base.InstalledAt,
		LifespanMonths:        base.LifespanMonths,
		ServiceIntervalMonths: base.ServiceIntervalMonths,
		LastServiceAt:         base.LastServiceAt,
		Priority:              base.Priority,
		Responsible:           base.Responsible,
		LastCostCents:         base.LastCostCents,
		Active:                base.Active,
		Notes:                 base.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapHomeSafetyItem(item))
}

func (h *HomeSafetyHandler) DeleteItem(c *gin.Context) {
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

func (h *HomeSafetyHandler) buildCreateInput(ws uuid.UUID, body homeSafetyItemJSON) (apphs.CreateItemInput, error) {
	installedAt, err := parseOptionalDate(body.InstalledAt)
	if err != nil {
		return apphs.CreateItemInput{}, &dom.ValidationError{Msg: "installed_at inválido (use YYYY-MM-DD)"}
	}
	lastServiceAt, err := parseOptionalDate(body.LastServiceAt)
	if err != nil {
		return apphs.CreateItemInput{}, &dom.ValidationError{Msg: "last_service_at inválido (use YYYY-MM-DD)"}
	}
	return apphs.CreateItemInput{
		WorkspaceID:           ws,
		Name:                  body.Name,
		Category:              body.Category,
		RiskType:              body.RiskType,
		Location:              body.Location,
		Brand:                 body.Brand,
		Model:                 body.Model,
		InstalledAt:           installedAt,
		LifespanMonths:        body.LifespanMonths,
		ServiceIntervalMonths: body.ServiceIntervalMonths,
		LastServiceAt:         lastServiceAt,
		Priority:              body.Priority,
		Responsible:           body.Responsible,
		LastCostCents:         body.LastCostCents,
		Active:                body.Active,
		Notes:                 body.Notes,
	}, nil
}

// ─── Handlers: events ─────────────────────────────────────────────────────────

func (h *HomeSafetyHandler) ListEvents(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	events, err := h.svc.ListEvents(c.Request.Context(), ws, itemID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	resp := make([]homeSafetyEventResponse, len(events))
	for i := range events {
		resp[i] = mapHomeSafetyEvent(&events[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": resp})
}

func (h *HomeSafetyHandler) CreateEvent(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	var body homeSafetyEventJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	eventDate, err := parseOptionalDate(&body.EventDate)
	if err != nil || eventDate == nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "event_date inválido (use YYYY-MM-DD)")
		return
	}
	ev, item, err := h.svc.CreateEvent(c.Request.Context(), apphs.CreateEventInput{
		WorkspaceID: ws,
		ItemID:      itemID,
		EventType:   body.EventType,
		EventDate:   *eventDate,
		CostCents:   body.CostCents,
		Provider:    body.Provider,
		Notes:       body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"event": mapHomeSafetyEvent(ev), "item": mapHomeSafetyItem(item)})
}

func (h *HomeSafetyHandler) DeleteEvent(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	eventID, err := uuid.Parse(c.Param("eventId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "eventId inválido")
		return
	}
	if err := h.svc.DeleteEvent(c.Request.Context(), ws, itemID, eventID); err != nil {
		errrespond.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ─── Handlers: dashboard & catalog ────────────────────────────────────────────

func (h *HomeSafetyHandler) Dashboard(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	dash, err := h.svc.Dashboard(c.Request.Context(), ws)
	if err != nil {
		errrespond.Write(c, err)
		return
	}

	statusCounts := make([]gin.H, len(dash.StatusCounts))
	for i, sc := range dash.StatusCounts {
		statusCounts[i] = gin.H{"status": string(sc.Status), "count": sc.Count}
	}
	byCategory := make([]gin.H, len(dash.ByCategory))
	for i, cc := range dash.ByCategory {
		byCategory[i] = gin.H{"category": string(cc.Category), "count": cc.Count}
	}
	byRisk := make([]gin.H, len(dash.ByRisk))
	for i, rc := range dash.ByRisk {
		byRisk[i] = gin.H{"risk_type": string(rc.RiskType), "count": rc.Count}
	}
	costByYear := make([]gin.H, len(dash.CostByYear))
	for i, yc := range dash.CostByYear {
		costByYear[i] = gin.H{"year": yc.Year, "cost_cents": yc.CostCents}
	}
	costByCategory := make([]gin.H, len(dash.CostByCategory))
	for i, cc := range dash.CostByCategory {
		costByCategory[i] = gin.H{"category": string(cc.Category), "cost_cents": cc.CostCents}
	}
	mapItems := func(items []dom.Item) []homeSafetyItemResponse {
		out := make([]homeSafetyItemResponse, len(items))
		for i := range items {
			out[i] = mapHomeSafetyItem(&items[i])
		}
		return out
	}

	c.JSON(http.StatusOK, gin.H{
		"total_items":       dash.TotalItems,
		"annual_cost_cents": dash.AnnualCostCents,
		"status_counts":     statusCounts,
		"by_category":       byCategory,
		"by_risk":           byRisk,
		"cost_by_year":      costByYear,
		"cost_by_category":  costByCategory,
		"overdue":           mapItems(dash.Overdue),
		"due_next_30":       mapItems(dash.DueNext30),
		"due_next_90":       mapItems(dash.DueNext90),
	})
}

func (h *HomeSafetyHandler) Catalog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": h.svc.Catalog()})
}
