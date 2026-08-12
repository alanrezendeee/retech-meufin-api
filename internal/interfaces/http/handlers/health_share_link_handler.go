package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/health"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// HealthShareLinkHandler cobre o CRUD autenticado dos links e o acesso
// público (médico, sem login) aos painéis do membro compartilhado.
type HealthShareLinkHandler struct {
	links *app.ShareLinkService
	dash  *app.DashboardService
}

func NewHealthShareLinkHandler(links *app.ShareLinkService, dash *app.DashboardService) *HealthShareLinkHandler {
	return &HealthShareLinkHandler{links: links, dash: dash}
}

type shareLinkCreateJSON struct {
	FamilyMemberID string  `json:"family_member_id" binding:"required"`
	Title          *string `json:"title"`
	ExpiresInHours int     `json:"expires_in_hours" binding:"required"`
}

func mapShareLink(l *app.ShareLinkWithURL) gin.H {
	now := time.Now().UTC()
	status := "active"
	switch {
	case l.Link.RevokedAt != nil:
		status = "revoked"
	case !now.Before(l.Link.ExpiresAt):
		status = "expired"
	}
	var lastViewed *string
	if l.Link.LastViewedAt != nil {
		v := l.Link.LastViewedAt.UTC().Format(time.RFC3339)
		lastViewed = &v
	}
	return gin.H{
		"id":               l.Link.ID,
		"url":              l.URL,
		"scope":            l.Link.Scope,
		"family_member_id": l.Link.FamilyMemberID,
		"title":            l.Link.Title,
		"status":           status,
		"expires_at":       l.Link.ExpiresAt.UTC().Format(time.RFC3339),
		"view_count":       l.Link.ViewCount,
		"last_viewed_at":   lastViewed,
		"created_at":       l.Link.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (h *HealthShareLinkHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body shareLinkCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	memberID, err := uuid.Parse(body.FamilyMemberID)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "family_member_id inválido")
		return
	}
	in := app.CreateShareLinkInput{
		WorkspaceID:    ws,
		FamilyMemberID: memberID,
		Title:          body.Title,
		ExpiresInHours: body.ExpiresInHours,
	}
	if uid, ok := c.Get(middleware.CtxUserID); ok {
		if id, err := uuid.Parse(strings.TrimSpace(uid.(string))); err == nil {
			in.CreatedBy = &id
		}
	}
	link, err := h.links.Create(c.Request.Context(), in)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapShareLink(link))
}

func (h *HealthShareLinkHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	links, err := h.links.List(c.Request.Context(), ws)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]gin.H, len(links))
	for i := range links {
		items[i] = mapShareLink(&links[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *HealthShareLinkHandler) Revoke(c *gin.Context) {
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
	if err := h.links.Revoke(c.Request.Context(), ws, id); err != nil {
		errrespond.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// PublicPanels responde GET /api/v1/public/health/:token — sem autenticação.
// Token inválido/expirado/revogado → 410 (mesmo código, anti-enumeração).
func (h *HealthShareLinkHandler) PublicPanels(c *gin.Context) {
	link, member, err := h.links.Resolve(c.Request.Context(), c.Param("token"))
	if err != nil {
		var gone *dom.ShareLinkGoneError
		if errors.As(err, &gone) {
			c.JSON(http.StatusGone, gin.H{"error": gin.H{"code": "LINK_EXPIRED", "message": gone.Error()}})
			return
		}
		errrespond.Write(c, err)
		return
	}

	panels, err := h.dash.Panels(c.Request.Context(), link.WorkspaceID, &link.FamilyMemberID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	out := make([]gin.H, 0, len(panels))
	for _, p := range panels {
		if len(p.Markers) == 0 {
			// Página pública mostra só o que existe — sem catálogo vazio.
			continue
		}
		markers := make([]gin.H, 0, len(p.Markers))
		for i := range p.Markers {
			pm := p.Markers[i]
			markers = append(markers, gin.H{
				"marker":       mapEvolutionMarker(&pm.Marker),
				"default_mode": pm.DefaultMode,
				"points":       mapEvolutionPoints(pm.Points),
			})
		}
		out = append(out, gin.H{"category": p.Category, "markers": markers})
	}

	c.JSON(http.StatusOK, gin.H{
		"link": gin.H{
			"title":       link.Title,
			"scope":       link.Scope,
			"member_name": member.FullName,
			"expires_at":  link.ExpiresAt.UTC().Format(time.RFC3339),
		},
		"panels": out,
	})
}
