package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/health"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// HealthExamConfirmHandler materializa a revisão de um documento extraído em
// um resultado de exame (POST /health/documents/:id/confirm) — espelho do
// confirm de fatura do finance.
type HealthExamConfirmHandler struct {
	imp *app.ExamImportService
}

func NewHealthExamConfirmHandler(imp *app.ExamImportService) *HealthExamConfirmHandler {
	return &HealthExamConfirmHandler{imp: imp}
}

type confirmExamNewMarkerJSON struct {
	Name          string   `json:"name"`
	Category      string   `json:"category"`
	CanonicalUnit *string  `json:"canonical_unit"`
	RefMin        *float64 `json:"ref_min"`
	RefMax        *float64 `json:"ref_max"`
	RefText       *string  `json:"ref_text"`
	Aliases       []string `json:"aliases"`
}

type confirmExamItemJSON struct {
	MarkerID      *string                   `json:"marker_id"`
	NewMarker     *confirmExamNewMarkerJSON `json:"new_marker"`
	RawMarkerName *string                   `json:"raw_marker_name"`
	ResultValue   string                    `json:"result_value"`
	ResultNumeric *float64                  `json:"result_numeric"`
	Unit          *string                   `json:"unit"`
	ReferenceMin  *float64                  `json:"reference_min"`
	ReferenceMax  *float64                  `json:"reference_max"`
	ReferenceText *string                   `json:"reference_text"`
	Method        *string                   `json:"method"`
	Material      *string                   `json:"material"`
	RawText       *string                   `json:"raw_text"`
}

type confirmExamRequest struct {
	FamilyMemberID string                `json:"family_member_id"`
	ExamDate       string                `json:"exam_date"` // YYYY-MM-DD, obrigatório
	CollectionDate *string               `json:"collection_date"`
	ReleaseDate    *string               `json:"release_date"`
	LabID          *string               `json:"lab_id"`
	NewLabName     *string               `json:"new_lab_name"`
	Summary        *string               `json:"summary"`
	Notes          *string               `json:"notes"`
	Items          []confirmExamItemJSON `json:"items"`
}

type createdMarkerResponse struct {
	ID            uuid.UUID `json:"id"`
	CanonicalName string    `json:"canonical_name"`
	Category      string    `json:"category"`
}

// Confirm responde POST /health/documents/:id/confirm.
func (h *HealthExamConfirmHandler) Confirm(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}

	var body confirmExamRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	if len(body.Items) == 0 {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "o exame precisa de ao menos um item")
		return
	}
	memberID, err := uuid.Parse(body.FamilyMemberID)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "family_member_id inválido")
		return
	}
	examDate, err := time.Parse("2006-01-02", body.ExamDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "exam_date inválida (use YYYY-MM-DD)")
		return
	}
	collectionDate, err := parseOptionalDate(body.CollectionDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "collection_date inválida (use YYYY-MM-DD)")
		return
	}
	releaseDate, err := parseOptionalDate(body.ReleaseDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "release_date inválida (use YYYY-MM-DD)")
		return
	}
	labID, err := parseOptionalUUID(body.LabID)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "lab_id inválido")
		return
	}

	in := app.ConfirmExamInput{
		WorkspaceID:    ws,
		DocumentID:     documentID,
		FamilyMemberID: memberID,
		ExamDate:       examDate,
		CollectionDate: collectionDate,
		ReleaseDate:    releaseDate,
		LabID:          labID,
		NewLabName:     body.NewLabName,
		Summary:        body.Summary,
		Notes:          body.Notes,
	}
	for _, it := range body.Items {
		markerID, err := parseOptionalUUID(it.MarkerID)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "marker_id inválido")
			return
		}
		item := app.ConfirmExamItemInput{
			MarkerID:      markerID,
			RawMarkerName: it.RawMarkerName,
			ResultValue:   it.ResultValue,
			ResultNumeric: it.ResultNumeric,
			Unit:          it.Unit,
			ReferenceMin:  it.ReferenceMin,
			ReferenceMax:  it.ReferenceMax,
			ReferenceText: it.ReferenceText,
			Method:        it.Method,
			Material:      it.Material,
			RawText:       it.RawText,
		}
		if it.NewMarker != nil {
			item.NewMarker = &app.ConfirmExamNewMarker{
				Name:          it.NewMarker.Name,
				Category:      it.NewMarker.Category,
				CanonicalUnit: it.NewMarker.CanonicalUnit,
				RefMin:        it.NewMarker.RefMin,
				RefMax:        it.NewMarker.RefMax,
				RefText:       it.NewMarker.RefText,
				Aliases:       it.NewMarker.Aliases,
			}
		}
		in.Items = append(in.Items, item)
	}

	res, err := h.imp.Confirm(c.Request.Context(), in)
	if err != nil {
		errrespond.Write(c, err)
		return
	}

	createdMarkers := make([]createdMarkerResponse, 0, len(res.CreatedMarkers))
	for _, m := range res.CreatedMarkers {
		createdMarkers = append(createdMarkers, createdMarkerResponse{
			ID:            m.ID,
			CanonicalName: m.CanonicalName,
			Category:      m.Category,
		})
	}

	resp := gin.H{
		"exam_result_id":  res.ExamResult.ID,
		"document_id":     documentID,
		"items_total":     len(res.ExamResult.Items),
		"created_markers": createdMarkers,
	}
	if res.CreatedLab != nil {
		resp["created_lab"] = gin.H{"id": res.CreatedLab.ID, "name": res.CreatedLab.Name}
	}
	c.JSON(http.StatusCreated, resp)
}
