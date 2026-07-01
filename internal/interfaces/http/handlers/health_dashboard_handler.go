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

type HealthDashboardHandler struct {
	svc *app.DashboardService
}

func NewHealthDashboardHandler(svc *app.DashboardService) *HealthDashboardHandler {
	return &HealthDashboardHandler{svc: svc}
}

func (h *HealthDashboardHandler) Counts(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	counts, err := h.svc.Counts(c.Request.Context(), ws)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"family_members":           counts.FamilyMembers,
		"exam_results":             counts.ExamResults,
		"tenant_markers":           counts.TenantMarkers,
		"documents_pending_review": counts.DocumentsPendingReview,
	})
}

type evolutionPointResponse struct {
	ExamDate       string     `json:"exam_date"`
	Value          *float64   `json:"value"`
	Unit           *string    `json:"unit"`
	ReferenceMin   *float64   `json:"reference_min"`
	ReferenceMax   *float64   `json:"reference_max"`
	ReferenceText  *string    `json:"reference_text"`
	LabID          *uuid.UUID `json:"lab_id"`
	Interpretation *string    `json:"interpretation"`
	Normalized     *float64   `json:"normalized"`
}

func parseDateParam(v string) *time.Time {
	if v == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil
	}
	return &t
}

func (h *HealthDashboardHandler) MarkerEvolution(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	markerID, err := uuid.Parse(c.Param("markerId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "markerId inválido")
		return
	}
	var familyMemberID *uuid.UUID
	if v := c.Query("family_member_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "family_member_id inválido")
			return
		}
		familyMemberID = &id
	}
	from := parseDateParam(c.Query("from"))
	to := parseDateParam(c.Query("to"))

	res, err := h.svc.MarkerEvolution(c.Request.Context(), ws, markerID, familyMemberID, from, to)
	if err != nil {
		errrespond.Write(c, err)
		return
	}

	points := make([]evolutionPointResponse, len(res.Points))
	for i := range res.Points {
		p := res.Points[i]
		points[i] = evolutionPointResponse{
			ExamDate:       p.ExamDate.UTC().Format("2006-01-02"),
			Value:          p.Value,
			Unit:           p.Unit,
			ReferenceMin:   p.RefMin,
			ReferenceMax:   p.RefMax,
			ReferenceText:  p.RefText,
			LabID:          p.LabID,
			Interpretation: p.Interpretation,
			Normalized:     p.Normalized,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"marker":       mapEvolutionMarker(res.Marker),
		"default_mode": res.DefaultMode,
		"points":       points,
	})
}

func mapEvolutionMarker(m *dom.Marker) gin.H {
	return gin.H{
		"id":                  m.ID,
		"canonical_name":      m.CanonicalName,
		"canonical_unit":      m.CanonicalUnit,
		"comparability_class": string(m.Comparability),
		"category":            m.Category,
	}
}
