package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/health"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

const appointmentDateLayout = "2006-01-02"

type HealthAppointmentHandler struct {
	svc *app.AppointmentService
}

func NewHealthAppointmentHandler(svc *app.AppointmentService) *HealthAppointmentHandler {
	return &HealthAppointmentHandler{svc: svc}
}

// --- payloads ---

type appointmentCreateJSON struct {
	FamilyMemberID   uuid.UUID  `json:"family_member_id" binding:"required"`
	Kind             string     `json:"kind"`
	Specialty        *string    `json:"specialty"`
	ProfessionalName *string    `json:"professional_name"`
	LabID            *uuid.UUID `json:"lab_id"`
	ExamRequestID    *uuid.UUID `json:"exam_request_id"`
	PlanID           *uuid.UUID `json:"plan_id"`
	ScheduledAt      string     `json:"scheduled_at" binding:"required"`
	Status           string     `json:"status"`
	Reason           *string    `json:"reason"`
	Outcome          *string    `json:"outcome"`
	PriceCents       int64      `json:"price_cents"`
	CoveredByPlan    bool       `json:"covered_by_plan"`
	Notes            *string    `json:"notes"`
}

type appointmentUpdateJSON struct {
	FamilyMemberID   uuid.UUID  `json:"family_member_id" binding:"required"`
	Kind             string     `json:"kind"`
	Specialty        *string    `json:"specialty"`
	ProfessionalName *string    `json:"professional_name"`
	LabID            *uuid.UUID `json:"lab_id"`
	ExamRequestID    *uuid.UUID `json:"exam_request_id"`
	PlanID           *uuid.UUID `json:"plan_id"`
	ScheduledAt      string     `json:"scheduled_at" binding:"required"`
	Status           string     `json:"status"`
	Reason           *string    `json:"reason"`
	Outcome          *string    `json:"outcome"`
	PriceCents       int64      `json:"price_cents"`
	CoveredByPlan    bool       `json:"covered_by_plan"`
	Notes            *string    `json:"notes"`
}

type appointmentCompleteJSON struct {
	Outcome    *string `json:"outcome"`
	PriceCents *int64  `json:"price_cents"`
}

// --- responses ---

type appointmentResponse struct {
	ID               uuid.UUID  `json:"id"`
	WorkspaceID      uuid.UUID  `json:"workspace_id"`
	FamilyMemberID   uuid.UUID  `json:"family_member_id"`
	MemberName       string     `json:"member_name,omitempty"`
	Kind             string     `json:"kind"`
	Specialty        *string    `json:"specialty"`
	ProfessionalName *string    `json:"professional_name"`
	LabID            *uuid.UUID `json:"lab_id"`
	LabName          *string    `json:"lab_name,omitempty"`
	ExamRequestID    *uuid.UUID `json:"exam_request_id"`
	PlanID           *uuid.UUID `json:"plan_id"`
	PlanName         *string    `json:"plan_name,omitempty"`
	ScheduledAt      string     `json:"scheduled_at"`
	Status           string     `json:"status"`
	Reason           *string    `json:"reason"`
	Outcome          *string    `json:"outcome"`
	PriceCents       int64      `json:"price_cents"`
	CoveredByPlan    bool       `json:"covered_by_plan"`
	Notes            *string    `json:"notes"`
	CreatedAt        string     `json:"created_at"`
	UpdatedAt        string     `json:"updated_at"`
}

func specialtyPtr(s *dom.Specialty) *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

func mapAppointment(a *dom.Appointment) appointmentResponse {
	return appointmentResponse{
		ID:               a.ID,
		WorkspaceID:      a.WorkspaceID,
		FamilyMemberID:   a.FamilyMemberID,
		Kind:             string(a.Kind),
		Specialty:        specialtyPtr(a.Specialty),
		ProfessionalName: a.ProfessionalName,
		LabID:            a.LabID,
		ExamRequestID:    a.ExamRequestID,
		PlanID:           a.PlanID,
		ScheduledAt:      a.ScheduledAt.UTC().Format(time.RFC3339Nano),
		Status:           string(a.Status),
		Reason:           a.Reason,
		Outcome:          a.Outcome,
		PriceCents:       a.PriceCents,
		CoveredByPlan:    a.CoveredByPlan,
		Notes:            a.Notes,
		CreatedAt:        a.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:        a.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func mapAppointmentEnriched(a *dom.AppointmentEnriched) appointmentResponse {
	r := mapAppointment(&a.Appointment)
	r.MemberName = a.MemberName
	r.LabName = a.LabName
	r.PlanName = a.PlanName
	return r
}

// parseDateTime aceita RFC3339, "2006-01-02T15:04:05" e "2006-01-02T15:04".
func parseDateTime(raw string) (time.Time, error) {
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04"}
	var lastErr error
	for _, l := range layouts {
		if t, err := time.Parse(l, raw); err == nil {
			return t.UTC(), nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}

// --- endpoints ---

func (h *HealthAppointmentHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body appointmentCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	scheduled, err := parseDateTime(body.ScheduledAt)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "scheduled_at inválido (use ISO 8601)")
		return
	}
	a, err := h.svc.Create(c.Request.Context(), app.CreateAppointmentInput{
		WorkspaceID:      ws,
		FamilyMemberID:   body.FamilyMemberID,
		Kind:             dom.AppointmentKind(body.Kind),
		Specialty:        specialtyFromPtr(body.Specialty),
		ProfessionalName: body.ProfessionalName,
		LabID:            body.LabID,
		ExamRequestID:    body.ExamRequestID,
		PlanID:           body.PlanID,
		ScheduledAt:      scheduled,
		Status:           dom.AppointmentStatus(body.Status),
		Reason:           body.Reason,
		Outcome:          body.Outcome,
		PriceCents:       body.PriceCents,
		CoveredByPlan:    body.CoveredByPlan,
		Notes:            body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapAppointment(a))
}

func specialtyFromPtr(s *string) *dom.Specialty {
	if s == nil {
		return nil
	}
	sp := dom.Specialty(*s)
	return &sp
}

func (h *HealthAppointmentHandler) Get(c *gin.Context) {
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
	a, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapAppointment(a))
}

func (h *HealthAppointmentHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	filter, err := appointmentFilterFromQuery(c)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, err.Error())
		return
	}
	limit, offset := pagination(c)
	res, err := h.svc.List(c.Request.Context(), ws, filter, limit, offset)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]appointmentResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapAppointmentEnriched(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}

func appointmentFilterFromQuery(c *gin.Context) (dom.AppointmentFilter, error) {
	f := dom.AppointmentFilter{
		Status: dom.AppointmentStatus(c.Query("status")),
		Kind:   dom.AppointmentKind(c.Query("kind")),
	}
	if v := c.Query("family_member_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return f, errInvalid("family_member_id inválido")
		}
		f.FamilyMemberID = &id
	}
	if v := c.Query("lab_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return f, errInvalid("lab_id inválido")
		}
		f.LabID = &id
	}
	if v := c.Query("plan_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return f, errInvalid("plan_id inválido")
		}
		f.PlanID = &id
	}
	if v := c.Query("from"); v != "" {
		t, err := time.Parse(appointmentDateLayout, v)
		if err != nil {
			return f, errInvalid("from inválido (use YYYY-MM-DD)")
		}
		from := t.UTC()
		f.From = &from
	}
	if v := c.Query("to"); v != "" {
		t, err := time.Parse(appointmentDateLayout, v)
		if err != nil {
			return f, errInvalid("to inválido (use YYYY-MM-DD)")
		}
		// inclui o dia inteiro do "to".
		to := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.UTC)
		f.To = &to
	}
	return f, nil
}

type simpleErr string

func (e simpleErr) Error() string { return string(e) }
func errInvalid(msg string) error { return simpleErr(msg) }

func (h *HealthAppointmentHandler) Update(c *gin.Context) {
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
	var body appointmentUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	scheduled, err := parseDateTime(body.ScheduledAt)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "scheduled_at inválido (use ISO 8601)")
		return
	}
	a, err := h.svc.Update(c.Request.Context(), app.UpdateAppointmentInput{
		WorkspaceID:      ws,
		ID:               id,
		FamilyMemberID:   body.FamilyMemberID,
		Kind:             dom.AppointmentKind(body.Kind),
		Specialty:        specialtyFromPtr(body.Specialty),
		ProfessionalName: body.ProfessionalName,
		LabID:            body.LabID,
		ExamRequestID:    body.ExamRequestID,
		PlanID:           body.PlanID,
		ScheduledAt:      scheduled,
		Status:           dom.AppointmentStatus(body.Status),
		Reason:           body.Reason,
		Outcome:          body.Outcome,
		PriceCents:       body.PriceCents,
		CoveredByPlan:    body.CoveredByPlan,
		Notes:            body.Notes,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapAppointment(a))
}

func (h *HealthAppointmentHandler) Delete(c *gin.Context) {
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

func (h *HealthAppointmentHandler) Confirm(c *gin.Context) {
	h.runTransition(c, h.svc.Confirm)
}

func (h *HealthAppointmentHandler) Cancel(c *gin.Context) {
	h.runTransition(c, h.svc.Cancel)
}

func (h *HealthAppointmentHandler) NoShow(c *gin.Context) {
	h.runTransition(c, h.svc.NoShow)
}

// runTransition executa uma ação de transição simples (confirm/cancel/no-show).
func (h *HealthAppointmentHandler) runTransition(c *gin.Context, fn func(ctx context.Context, ws, id uuid.UUID) (*dom.Appointment, error)) {
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
	a, err := fn(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapAppointment(a))
}

func (h *HealthAppointmentHandler) Complete(c *gin.Context) {
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
	var body appointmentCompleteJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	a, err := h.svc.Complete(c.Request.Context(), ws, id, app.CompleteAppointmentInput{
		Outcome:    body.Outcome,
		PriceCents: body.PriceCents,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapAppointment(a))
}

func (h *HealthAppointmentHandler) Agenda(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	ag, err := h.svc.GetAgenda(c.Request.Context(), ws)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapAgenda(ag))
}

// --- agenda response ---

type agendaStatusCountResponse struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type agendaSpecialtyCountResponse struct {
	Specialty string `json:"specialty"`
	Count     int64  `json:"count"`
}

type agendaMemberCountResponse struct {
	MemberID   uuid.UUID `json:"member_id"`
	MemberName string    `json:"member_name"`
	Count      int64     `json:"count"`
}

type agendaResponse struct {
	Year                 int                            `json:"year"`
	Upcoming             []appointmentResponse          `json:"upcoming"`
	Next7Count           int                            `json:"next_7_count"`
	Next30Count          int                            `json:"next_30_count"`
	StatusCounts         []agendaStatusCountResponse    `json:"status_counts"`
	YearSpendCents       int64                          `json:"year_spend_cents"`
	PlansMonthlyFeeCents int64                          `json:"plans_monthly_fee_cents"`
	PlansAnnualFeeCents  int64                          `json:"plans_annual_fee_cents"`
	BySpecialty          []agendaSpecialtyCountResponse `json:"by_specialty"`
	ByMember             []agendaMemberCountResponse    `json:"by_member"`
}

func mapAgenda(ag *app.Agenda) agendaResponse {
	upcoming := make([]appointmentResponse, len(ag.Upcoming))
	for i := range ag.Upcoming {
		upcoming[i] = mapAppointmentEnriched(&ag.Upcoming[i])
	}
	statusCounts := make([]agendaStatusCountResponse, len(ag.StatusCounts))
	for i := range ag.StatusCounts {
		statusCounts[i] = agendaStatusCountResponse{Status: ag.StatusCounts[i].Status, Count: ag.StatusCounts[i].Count}
	}
	bySpecialty := make([]agendaSpecialtyCountResponse, len(ag.BySpecialty))
	for i := range ag.BySpecialty {
		bySpecialty[i] = agendaSpecialtyCountResponse{Specialty: ag.BySpecialty[i].Specialty, Count: ag.BySpecialty[i].Count}
	}
	byMember := make([]agendaMemberCountResponse, len(ag.ByMember))
	for i := range ag.ByMember {
		byMember[i] = agendaMemberCountResponse{MemberID: ag.ByMember[i].MemberID, MemberName: ag.ByMember[i].MemberName, Count: ag.ByMember[i].Count}
	}
	return agendaResponse{
		Year:                 ag.Year,
		Upcoming:             upcoming,
		Next7Count:           ag.Next7Count,
		Next30Count:          ag.Next30Count,
		StatusCounts:         statusCounts,
		YearSpendCents:       ag.YearSpendCents,
		PlansMonthlyFeeCents: ag.PlansMonthlyFeeCents,
		PlansAnnualFeeCents:  ag.PlansAnnualFeeCents,
		BySpecialty:          bySpecialty,
		ByMember:             byMember,
	}
}
