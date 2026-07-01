package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/health"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

type HealthMarkerHandler struct {
	svc *app.MarkerService
}

func NewHealthMarkerHandler(svc *app.MarkerService) *HealthMarkerHandler {
	return &HealthMarkerHandler{svc: svc}
}

type markerAliasResponse struct {
	Alias           string `json:"alias"`
	NormalizedAlias string `json:"normalized_alias"`
}

type markerResponse struct {
	ID             uuid.UUID             `json:"id"`
	Scope          string                `json:"scope"`
	WorkspaceID    *uuid.UUID            `json:"workspace_id"`
	CanonicalName  string                `json:"canonical_name"`
	NormalizedKey  string                `json:"normalized_key"`
	LoincCode      *string               `json:"loinc_code"`
	Category       string                `json:"category"`
	Comparability  string                `json:"comparability_class"`
	CanonicalUnit  *string               `json:"canonical_unit"`
	DefaultRefMin  *float64              `json:"default_ref_min"`
	DefaultRefMax  *float64              `json:"default_ref_max"`
	DefaultRefText *string               `json:"default_ref_text"`
	Active         bool                  `json:"active"`
	Aliases        []markerAliasResponse `json:"aliases"`
	CreatedAt      string                `json:"created_at"`
	UpdatedAt      string                `json:"updated_at"`
}

func mapMarker(m *dom.Marker) markerResponse {
	aliases := make([]markerAliasResponse, len(m.Aliases))
	for i := range m.Aliases {
		aliases[i] = markerAliasResponse{Alias: m.Aliases[i].Alias, NormalizedAlias: m.Aliases[i].NormalizedAlias}
	}
	return markerResponse{
		ID: m.ID, Scope: string(m.Scope), WorkspaceID: m.WorkspaceID,
		CanonicalName: m.CanonicalName, NormalizedKey: m.NormalizedKey, LoincCode: m.LoincCode,
		Category: m.Category, Comparability: string(m.Comparability), CanonicalUnit: m.CanonicalUnit,
		DefaultRefMin: m.DefaultRefMin, DefaultRefMax: m.DefaultRefMax, DefaultRefText: m.DefaultRefText,
		Active: m.Active, Aliases: aliases,
		CreatedAt: m.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: m.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

// writeDuplicate responde 409 com o marcador existente para sugestão.
func writeDuplicate(c *gin.Context, dup *dom.DuplicateError) {
	c.JSON(http.StatusConflict, gin.H{
		"error": gin.H{
			"code":    "MARKER_DUPLICATE",
			"message": "marcador já existe no catálogo",
		},
		"suggestion": mapMarker(dup.Existing),
	})
}

type markerCreateJSON struct {
	CanonicalName  string   `json:"canonical_name" binding:"required"`
	Category       string   `json:"category" binding:"required"`
	Comparability  string   `json:"comparability_class"`
	CanonicalUnit  *string  `json:"canonical_unit"`
	LoincCode      *string  `json:"loinc_code"`
	DefaultRefMin  *float64 `json:"default_ref_min"`
	DefaultRefMax  *float64 `json:"default_ref_max"`
	DefaultRefText *string  `json:"default_ref_text"`
	Aliases        []string `json:"aliases"`
}

func (h *HealthMarkerHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body markerCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	m, err := h.svc.Create(c.Request.Context(), app.CreateMarkerInput{
		WorkspaceID: ws, CanonicalName: body.CanonicalName, Category: body.Category,
		Comparability: body.Comparability, CanonicalUnit: body.CanonicalUnit, LoincCode: body.LoincCode,
		DefaultRefMin: body.DefaultRefMin, DefaultRefMax: body.DefaultRefMax, DefaultRefText: body.DefaultRefText,
		Aliases: body.Aliases,
	})
	if err != nil {
		var dup *dom.DuplicateError
		if errors.As(err, &dup) {
			writeDuplicate(c, dup)
			return
		}
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapMarker(m))
}

func (h *HealthMarkerHandler) Get(c *gin.Context) {
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
	m, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapMarker(m))
}

func (h *HealthMarkerHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	limit, offset := pagination(c)
	res, err := h.svc.List(c.Request.Context(), ws, c.Query("query"), c.Query("category"), limit, offset)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]markerResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapMarker(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}

type markerUpdateJSON struct {
	CanonicalName  string   `json:"canonical_name" binding:"required"`
	Category       string   `json:"category" binding:"required"`
	Comparability  string   `json:"comparability_class"`
	CanonicalUnit  *string  `json:"canonical_unit"`
	LoincCode      *string  `json:"loinc_code"`
	DefaultRefMin  *float64 `json:"default_ref_min"`
	DefaultRefMax  *float64 `json:"default_ref_max"`
	DefaultRefText *string  `json:"default_ref_text"`
	Active         *bool    `json:"active"`
}

func (h *HealthMarkerHandler) Update(c *gin.Context) {
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
	var body markerUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	m, err := h.svc.Update(c.Request.Context(), app.UpdateMarkerInput{
		WorkspaceID: ws, ID: id, CanonicalName: body.CanonicalName, Category: body.Category,
		Comparability: body.Comparability, CanonicalUnit: body.CanonicalUnit, LoincCode: body.LoincCode,
		DefaultRefMin: body.DefaultRefMin, DefaultRefMax: body.DefaultRefMax, DefaultRefText: body.DefaultRefText,
		Active: body.Active,
	})
	if err != nil {
		var dup *dom.DuplicateError
		if errors.As(err, &dup) {
			writeDuplicate(c, dup)
			return
		}
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapMarker(m))
}

func (h *HealthMarkerHandler) Delete(c *gin.Context) {
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

type resolveJSON struct {
	Items []struct {
		RawName string  `json:"raw_name" binding:"required"`
		Unit    *string `json:"unit"`
	} `json:"items" binding:"required"`
}

type resolveCandidateResponse struct {
	Marker     markerResponse `json:"marker"`
	Similarity float64        `json:"similarity"`
}

type resolveItemResponse struct {
	RawName    string                     `json:"raw_name"`
	Status     string                     `json:"status"`
	Matched    *markerResponse            `json:"matched"`
	Candidates []resolveCandidateResponse `json:"candidates"`
}

func (h *HealthMarkerHandler) Resolve(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body resolveJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	in := make([]app.ResolveItemInput, len(body.Items))
	for i := range body.Items {
		in[i] = app.ResolveItemInput{RawName: body.Items[i].RawName, Unit: body.Items[i].Unit}
	}
	results, err := h.svc.Resolve(c.Request.Context(), ws, in)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	out := make([]resolveItemResponse, len(results))
	for i := range results {
		r := results[i]
		item := resolveItemResponse{RawName: r.RawName, Status: string(r.Status)}
		if r.Matched != nil {
			mr := mapMarker(r.Matched)
			item.Matched = &mr
		}
		for _, cand := range r.Candidates {
			cm := cand
			item.Candidates = append(item.Candidates, resolveCandidateResponse{
				Marker: mapMarker(&cm.Marker), Similarity: cm.Similarity,
			})
		}
		out[i] = item
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}
