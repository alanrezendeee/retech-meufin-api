package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/health"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

type HealthPlanHandler struct {
	svc *app.PlanService
}

func NewHealthPlanHandler(svc *app.PlanService) *HealthPlanHandler {
	return &HealthPlanHandler{svc: svc}
}

// --- payloads ---

type healthPlanMemberJSON struct {
	MemberID   uuid.UUID `json:"member_id" binding:"required"`
	CardNumber *string   `json:"card_number"`
	Holder     bool      `json:"holder"`
}

type healthPlanCreateJSON struct {
	Name            string           `json:"name" binding:"required"`
	Operator        *string          `json:"operator"`
	PlanType        string           `json:"plan_type"`
	AnsCode         *string          `json:"ans_code"`
	MonthlyFeeCents int64            `json:"monthly_fee_cents"`
	CoverageNotes   *string          `json:"coverage_notes"`
	Active          *bool            `json:"active"`
	Members         []healthPlanMemberJSON `json:"members"`
}

type healthPlanUpdateJSON struct {
	Name            string  `json:"name" binding:"required"`
	Operator        *string `json:"operator"`
	PlanType        string  `json:"plan_type"`
	AnsCode         *string `json:"ans_code"`
	MonthlyFeeCents int64   `json:"monthly_fee_cents"`
	CoverageNotes   *string `json:"coverage_notes"`
	Active          *bool   `json:"active"`
}

type healthPlanMembersReplaceJSON struct {
	Members []healthPlanMemberJSON `json:"members"`
}

// --- responses ---

type healthPlanMemberResponse struct {
	ID         uuid.UUID `json:"id"`
	PlanID     uuid.UUID `json:"plan_id"`
	MemberID   uuid.UUID `json:"member_id"`
	CardNumber *string   `json:"card_number"`
	Holder     bool      `json:"holder"`
	CreatedAt  string    `json:"created_at"`
}

type healthPlanResponse struct {
	ID              uuid.UUID            `json:"id"`
	WorkspaceID     uuid.UUID            `json:"workspace_id"`
	Name            string               `json:"name"`
	Operator        *string              `json:"operator"`
	PlanType        string               `json:"plan_type"`
	AnsCode         *string              `json:"ans_code"`
	MonthlyFeeCents int64                `json:"monthly_fee_cents"`
	CoverageNotes   *string              `json:"coverage_notes"`
	Active          bool                 `json:"active"`
	Members         []healthPlanMemberResponse `json:"members"`
	CreatedAt       string               `json:"created_at"`
	UpdatedAt       string               `json:"updated_at"`
}

func mapHealthPlanMember(m *dom.PlanMember) healthPlanMemberResponse {
	return healthPlanMemberResponse{
		ID:         m.ID,
		PlanID:     m.PlanID,
		MemberID:   m.MemberID,
		CardNumber: m.CardNumber,
		Holder:     m.Holder,
		CreatedAt:  m.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func mapHealthPlan(p *dom.Plan) healthPlanResponse {
	members := make([]healthPlanMemberResponse, len(p.Members))
	for i := range p.Members {
		members[i] = mapHealthPlanMember(&p.Members[i])
	}
	return healthPlanResponse{
		ID:              p.ID,
		WorkspaceID:     p.WorkspaceID,
		Name:            p.Name,
		Operator:        p.Operator,
		PlanType:        string(p.PlanType),
		AnsCode:         p.AnsCode,
		MonthlyFeeCents: p.MonthlyFeeCents,
		CoverageNotes:   p.CoverageNotes,
		Active:          p.Active,
		Members:         members,
		CreatedAt:       p.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:       p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toHealthPlanMemberInputs(in []healthPlanMemberJSON) []app.PlanMemberInput {
	out := make([]app.PlanMemberInput, len(in))
	for i := range in {
		out[i] = app.PlanMemberInput{
			MemberID:   in[i].MemberID,
			CardNumber: in[i].CardNumber,
			Holder:     in[i].Holder,
		}
	}
	return out
}

// --- endpoints ---

func (h *HealthPlanHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body healthPlanCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	active := true
	if body.Active != nil {
		active = *body.Active
	}
	p, err := h.svc.Create(c.Request.Context(), app.CreatePlanInput{
		WorkspaceID:     ws,
		Name:            body.Name,
		Operator:        body.Operator,
		PlanType:        dom.PlanType(body.PlanType),
		AnsCode:         body.AnsCode,
		MonthlyFeeCents: body.MonthlyFeeCents,
		CoverageNotes:   body.CoverageNotes,
		Active:          active,
		Members:         toHealthPlanMemberInputs(body.Members),
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapHealthPlan(p))
}

func (h *HealthPlanHandler) Get(c *gin.Context) {
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
	p, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapHealthPlan(p))
}

func (h *HealthPlanHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	limit, offset := pagination(c)
	res, err := h.svc.List(c.Request.Context(), ws, limit, offset)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]healthPlanResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapHealthPlan(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}

func (h *HealthPlanHandler) Update(c *gin.Context) {
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
	var body healthPlanUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	active := true
	if body.Active != nil {
		active = *body.Active
	}
	p, err := h.svc.Update(c.Request.Context(), app.UpdatePlanInput{
		WorkspaceID:     ws,
		ID:              id,
		Name:            body.Name,
		Operator:        body.Operator,
		PlanType:        dom.PlanType(body.PlanType),
		AnsCode:         body.AnsCode,
		MonthlyFeeCents: body.MonthlyFeeCents,
		CoverageNotes:   body.CoverageNotes,
		Active:          active,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapHealthPlan(p))
}

func (h *HealthPlanHandler) Delete(c *gin.Context) {
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

func (h *HealthPlanHandler) ReplaceMembers(c *gin.Context) {
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
	var body healthPlanMembersReplaceJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	p, err := h.svc.ReplaceMembers(c.Request.Context(), ws, id, toHealthPlanMemberInputs(body.Members))
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapHealthPlan(p))
}
