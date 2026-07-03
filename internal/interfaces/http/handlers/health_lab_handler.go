package handlers

import (
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

type HealthLabHandler struct {
	svc *app.LabService
}

func NewHealthLabHandler(svc *app.LabService) *HealthLabHandler {
	return &HealthLabHandler{svc: svc}
}

type labCreateJSON struct {
	Name           string  `json:"name" binding:"required"`
	WebsiteURL     *string `json:"website_url"`
	ExamResultsURL *string `json:"exam_results_url"`
	ContactPhone   *string `json:"contact_phone"`
	Address        *string `json:"address"`
	Notes          *string `json:"notes"`
	Active         *bool   `json:"active"`
}

type labUpdateJSON struct {
	Name           string  `json:"name" binding:"required"`
	WebsiteURL     *string `json:"website_url"`
	ExamResultsURL *string `json:"exam_results_url"`
	ContactPhone   *string `json:"contact_phone"`
	Address        *string `json:"address"`
	Notes          *string `json:"notes"`
	Active         *bool   `json:"active"`
}

type labResponse struct {
	ID             uuid.UUID `json:"id"`
	WorkspaceID    uuid.UUID `json:"workspace_id"`
	Name           string    `json:"name"`
	WebsiteURL     *string   `json:"website_url"`
	ExamResultsURL *string   `json:"exam_results_url"`
	ContactPhone   *string   `json:"contact_phone"`
	Address        *string   `json:"address"`
	Notes          *string   `json:"notes"`
	Active         bool      `json:"active"`
	CreatedAt      string    `json:"created_at"`
	UpdatedAt      string    `json:"updated_at"`
}

func mapLab(l *dom.Lab) labResponse {
	return labResponse{
		ID:             l.ID,
		WorkspaceID:    l.WorkspaceID,
		Name:           l.Name,
		WebsiteURL:     l.WebsiteURL,
		ExamResultsURL: l.ExamResultsURL,
		ContactPhone:   l.ContactPhone,
		Address:        l.Address,
		Notes:          l.Notes,
		Active:         l.Active,
		CreatedAt:      l.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:      l.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (h *HealthLabHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body labCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	active := true
	if body.Active != nil {
		active = *body.Active
	}
	l, err := h.svc.Create(c.Request.Context(), app.CreateLabInput{
		WorkspaceID:    ws,
		Name:           body.Name,
		WebsiteURL:     body.WebsiteURL,
		ExamResultsURL: body.ExamResultsURL,
		ContactPhone:   body.ContactPhone,
		Address:        body.Address,
		Notes:          body.Notes,
		Active:         active,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapLab(l))
}

func (h *HealthLabHandler) Get(c *gin.Context) {
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
	l, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapLab(l))
}

func (h *HealthLabHandler) Update(c *gin.Context) {
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
	var body labUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	active := true
	if body.Active != nil {
		active = *body.Active
	}
	l, err := h.svc.Update(c.Request.Context(), app.UpdateLabInput{
		WorkspaceID:    ws,
		ID:             id,
		Name:           body.Name,
		WebsiteURL:     body.WebsiteURL,
		ExamResultsURL: body.ExamResultsURL,
		ContactPhone:   body.ContactPhone,
		Address:        body.Address,
		Notes:          body.Notes,
		Active:         active,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapLab(l))
}

func (h *HealthLabHandler) Delete(c *gin.Context) {
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

func (h *HealthLabHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	limit, offset := pagination(c)
	filter := dom.LabFilter{
		Query:  strings.TrimSpace(c.Query("query")),
		Active: boolQuery(c, "active"),
	}
	res, err := h.svc.List(c.Request.Context(), ws, filter, limit, offset)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]labResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapLab(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}
