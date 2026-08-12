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

	c.JSON(http.StatusOK, gin.H{
		"marker":       mapEvolutionMarker(res.Marker),
		"default_mode": res.DefaultMode,
		"points":       mapEvolutionPoints(res.Points),
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

func mapEvolutionPoints(points []dom.EvolutionPoint) []evolutionPointResponse {
	out := make([]evolutionPointResponse, len(points))
	for i := range points {
		p := points[i]
		out[i] = evolutionPointResponse{
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
	return out
}

// Panels responde GET /health/dashboard/panels: marcadores agrupados por
// categoria — com histórico (points) e sem nenhum resultado (missing).
func (h *HealthDashboardHandler) Panels(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
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

	panels, err := h.svc.Panels(c.Request.Context(), ws, familyMemberID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}

	out := make([]gin.H, 0, len(panels))
	for _, p := range panels {
		markers := make([]gin.H, 0, len(p.Markers))
		for i := range p.Markers {
			pm := p.Markers[i]
			markers = append(markers, gin.H{
				"marker":       mapEvolutionMarker(&pm.Marker),
				"default_mode": pm.DefaultMode,
				"points":       mapEvolutionPoints(pm.Points),
			})
		}
		missing := make([]gin.H, 0, len(p.Missing))
		for i := range p.Missing {
			missing = append(missing, mapEvolutionMarker(&p.Missing[i]))
		}
		out = append(out, gin.H{
			"category": p.Category,
			"markers":  markers,
			"missing":  missing,
		})
	}
	c.JSON(http.StatusOK, gin.H{"panels": out})
}
