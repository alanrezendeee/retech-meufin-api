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

const familyMemberBirthDateLayout = "2006-01-02"

type HealthFamilyMemberHandler struct {
	svc *app.FamilyMemberService
}

func NewHealthFamilyMemberHandler(svc *app.FamilyMemberService) *HealthFamilyMemberHandler {
	return &HealthFamilyMemberHandler{svc: svc}
}

type familyMemberCreateJSON struct {
	FullName     string   `json:"full_name" binding:"required"`
	Relationship string   `json:"relationship" binding:"required"`
	BirthDate    *string  `json:"birth_date"`
	Gender       *string  `json:"gender"`
	Document     *string  `json:"document"`
	Notes        *string  `json:"notes"`
	HeightCm     *float64 `json:"height_cm"`
	WeightKg     *float64 `json:"weight_kg"`
	Active       *bool    `json:"active"`
}

type familyMemberUpdateJSON struct {
	FullName     string   `json:"full_name" binding:"required"`
	Relationship string   `json:"relationship" binding:"required"`
	BirthDate    *string  `json:"birth_date"`
	Gender       *string  `json:"gender"`
	Document     *string  `json:"document"`
	Notes        *string  `json:"notes"`
	HeightCm     *float64 `json:"height_cm"`
	WeightKg     *float64 `json:"weight_kg"`
	Active       *bool    `json:"active"`
}

type familyMemberResponse struct {
	ID           uuid.UUID `json:"id"`
	WorkspaceID  uuid.UUID `json:"workspace_id"`
	FullName     string    `json:"full_name"`
	Relationship string    `json:"relationship"`
	BirthDate    *string   `json:"birth_date"`
	Gender       *string   `json:"gender"`
	Document     *string   `json:"document"`
	Notes        *string   `json:"notes"`
	HeightCm     *float64  `json:"height_cm"`
	WeightKg     *float64  `json:"weight_kg"`
	Age          *int      `json:"age"`
	Active       bool      `json:"active"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`
}

func mapFamilyMember(f *dom.FamilyMember) familyMemberResponse {
	var birth *string
	if f.BirthDate != nil {
		s := f.BirthDate.UTC().Format(familyMemberBirthDateLayout)
		birth = &s
	}
	return familyMemberResponse{
		ID:           f.ID,
		WorkspaceID:  f.WorkspaceID,
		FullName:     f.FullName,
		Relationship: f.Relationship,
		BirthDate:    birth,
		Gender:       f.Gender,
		Document:     f.Document,
		Notes:        f.Notes,
		HeightCm:     f.HeightCm,
		WeightKg:     f.WeightKg,
		Age:          f.Age(),
		Active:       f.Active,
		CreatedAt:    f.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:    f.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

type birthdayResponse struct {
	ID           uuid.UUID `json:"id"`
	FullName     string    `json:"full_name"`
	Relationship string    `json:"relationship"`
	BirthDate    string    `json:"birth_date"`
	Age          int       `json:"age"`
	Turns        int       `json:"turns"`
	NextBirthday string    `json:"next_birthday"`
	DaysUntil    int       `json:"days_until"`
}

func mapBirthday(b *dom.Birthday) birthdayResponse {
	var birth string
	if b.Member.BirthDate != nil {
		birth = b.Member.BirthDate.UTC().Format(familyMemberBirthDateLayout)
	}
	return birthdayResponse{
		ID:           b.Member.ID,
		FullName:     b.Member.FullName,
		Relationship: b.Member.Relationship,
		BirthDate:    birth,
		Age:          b.Age,
		Turns:        b.Turns,
		NextBirthday: b.NextBirthday.Format(familyMemberBirthDateLayout),
		DaysUntil:    b.DaysUntil,
	}
}

func parseBirthDate(c *gin.Context, raw *string) (*time.Time, bool) {
	if raw == nil || *raw == "" {
		return nil, true
	}
	t, err := time.Parse(familyMemberBirthDateLayout, *raw)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "birth_date inválido (use YYYY-MM-DD)")
		return nil, false
	}
	return &t, true
}

func (h *HealthFamilyMemberHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body familyMemberCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	birth, ok := parseBirthDate(c, body.BirthDate)
	if !ok {
		return
	}
	f, err := h.svc.Create(c.Request.Context(), app.CreateFamilyMemberInput{
		WorkspaceID:  ws,
		FullName:     body.FullName,
		Relationship: body.Relationship,
		BirthDate:    birth,
		Gender:       body.Gender,
		Document:     body.Document,
		Notes:        body.Notes,
		HeightCm:     body.HeightCm,
		WeightKg:     body.WeightKg,
		Active:       body.Active,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapFamilyMember(f))
}

func (h *HealthFamilyMemberHandler) Get(c *gin.Context) {
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
	f, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapFamilyMember(f))
}

func (h *HealthFamilyMemberHandler) Update(c *gin.Context) {
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
	var body familyMemberUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	birth, ok := parseBirthDate(c, body.BirthDate)
	if !ok {
		return
	}
	f, err := h.svc.Update(c.Request.Context(), app.UpdateFamilyMemberInput{
		WorkspaceID:  ws,
		ID:           id,
		FullName:     body.FullName,
		Relationship: body.Relationship,
		BirthDate:    birth,
		Gender:       body.Gender,
		Document:     body.Document,
		Notes:        body.Notes,
		HeightCm:     body.HeightCm,
		WeightKg:     body.WeightKg,
		Active:       body.Active,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapFamilyMember(f))
}

func (h *HealthFamilyMemberHandler) Delete(c *gin.Context) {
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

func (h *HealthFamilyMemberHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	limit, offset := pagination(c)
	filter := dom.FamilyMemberFilter{
		Query:        strings.TrimSpace(c.Query("query")),
		Relationship: c.Query("relationship"),
		Active:       boolQuery(c, "active"),
	}
	res, err := h.svc.List(c.Request.Context(), ws, filter, limit, offset)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]familyMemberResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapFamilyMember(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}

// Birthdays lista os próximos aniversários dos membros ativos (quadro do painel).
func (h *HealthFamilyMemberHandler) Birthdays(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	list, err := h.svc.Birthdays(c.Request.Context(), ws)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]birthdayResponse, len(list))
	for i := range list {
		items[i] = mapBirthday(&list[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}
