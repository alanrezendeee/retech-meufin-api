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

// HealthExtractionHandler expõe a consulta de status de extração de um
// documento de saúde.
//
// NOTA DE FIAÇÃO (a ser feita fora deste arquivo):
// O DISPARO da extração (POST /documents/:id/extract) NÃO vive aqui, porque
// precisa do CONTEÚDO do arquivo, que reside no módulo de documentos (storage).
// A fiação correta é o document handler carregar o conteúdo do documento via
// seu DocumentService/storage e chamar:
//
//	ExtractionService.StartExtraction(ctx, workspaceID, documentID, inputType, mimeType, content)
//
// Este handler cobre apenas o polling de status:
//
//	GET /documents/:id/extraction-status -> job (status, provider, model, error, timestamps)
type HealthExtractionHandler struct {
	svc  *app.ExtractionService
	docs *app.DocumentService
	imp  *app.ExamImportService
}

func NewHealthExtractionHandler(svc *app.ExtractionService, docs *app.DocumentService, imp *app.ExamImportService) *HealthExtractionHandler {
	return &HealthExtractionHandler{svc: svc, docs: docs, imp: imp}
}

type extractionStatusResponse struct {
	ID            uuid.UUID `json:"id"`
	DocumentID    uuid.UUID `json:"document_id"`
	Provider      string    `json:"provider"`
	Model         *string   `json:"model,omitempty"`
	Status        string    `json:"status"`
	InputType     string    `json:"input_type"`
	PromptVersion *string   `json:"prompt_version,omitempty"`
	ErrorMessage  *string   `json:"error_message,omitempty"`
	StartedAt     *string   `json:"started_at,omitempty"`
	FinishedAt    *string   `json:"finished_at,omitempty"`
	CreatedAt     string    `json:"created_at"`
	UpdatedAt     string    `json:"updated_at"`
	// Exam: sugestões derivadas do extracted_json quando a extração concluiu.
	Exam *examSuggestionResponse `json:"exam,omitempty"`
}

type markerCandidateResponse struct {
	MarkerID      uuid.UUID `json:"marker_id"`
	CanonicalName string    `json:"canonical_name"`
	CanonicalUnit *string   `json:"canonical_unit,omitempty"`
	Similarity    float64   `json:"similarity"`
}

type examItemSuggestionResponse struct {
	RawName        string                    `json:"raw_name"`
	ResultValue    string                    `json:"result_value"`
	ResultNumeric  *float64                  `json:"result_numeric,omitempty"`
	Unit           *string                   `json:"unit,omitempty"`
	ReferenceMin   *float64                  `json:"reference_min,omitempty"`
	ReferenceMax   *float64                  `json:"reference_max,omitempty"`
	ReferenceText  *string                   `json:"reference_text,omitempty"`
	Material       *string                   `json:"material,omitempty"`
	Method         *string                   `json:"method,omitempty"`
	Interpretation *string                   `json:"interpretation,omitempty"`
	RawText        *string                   `json:"raw_text,omitempty"`
	MarkerStatus   string                    `json:"marker_status"` // matched|ambiguous|unresolved
	MarkerID       *uuid.UUID                `json:"marker_id,omitempty"`
	MarkerName     *string                   `json:"marker_name,omitempty"`
	Candidates     []markerCandidateResponse `json:"candidates,omitempty"`
	MarkerIsNew    bool                      `json:"marker_is_new,omitempty"`
	MarkerRefText  *string                   `json:"marker_ref_text,omitempty"`
	MarkerRefTiers []dom.RefTier             `json:"marker_ref_tiers,omitempty"`
}

type examSuggestionResponse struct {
	PatientName    string                       `json:"patient_name,omitempty"`
	ExamDate       string                       `json:"exam_date,omitempty"`
	CollectionDate string                       `json:"collection_date,omitempty"`
	ReleaseDate    string                       `json:"release_date,omitempty"`
	LaboratoryName string                       `json:"laboratory_name,omitempty"`
	DoctorName     string                       `json:"doctor_name,omitempty"`
	Summary        string                       `json:"summary,omitempty"`
	Warnings       []string                     `json:"warnings,omitempty"`
	Items          []examItemSuggestionResponse `json:"items"`
	LabID          *uuid.UUID                   `json:"lab_id,omitempty"`
	LabIsNew       bool                         `json:"lab_is_new,omitempty"`
}

func mapExamSuggestion(s *app.ExamSuggestion) *examSuggestionResponse {
	out := &examSuggestionResponse{
		PatientName:    s.PatientName,
		ExamDate:       s.ExamDate,
		CollectionDate: s.CollectionDate,
		ReleaseDate:    s.ReleaseDate,
		LaboratoryName: s.LaboratoryName,
		DoctorName:     s.DoctorName,
		Summary:        s.Summary,
		Warnings:       s.Warnings,
		LabID:          s.LabID,
		LabIsNew:       s.LabIsNew,
		Items:          make([]examItemSuggestionResponse, 0, len(s.Items)),
	}
	for _, it := range s.Items {
		r := examItemSuggestionResponse{
			RawName:        it.RawName,
			ResultValue:    it.ResultValue,
			ResultNumeric:  it.ResultNumeric,
			Unit:           it.Unit,
			ReferenceMin:   it.ReferenceMin,
			ReferenceMax:   it.ReferenceMax,
			ReferenceText:  it.ReferenceText,
			Material:       it.Material,
			Method:         it.Method,
			Interpretation: it.Interpretation,
			RawText:        it.RawText,
			MarkerStatus:   it.MarkerStatus,
			MarkerID:       it.MarkerID,
			MarkerName:     it.MarkerName,
			MarkerIsNew:    it.MarkerIsNew,
			MarkerRefText:  it.MarkerRefText,
			MarkerRefTiers: it.MarkerRefTiers,
		}
		for _, c := range it.Candidates {
			r.Candidates = append(r.Candidates, markerCandidateResponse{
				MarkerID:      c.MarkerID,
				CanonicalName: c.CanonicalName,
				CanonicalUnit: c.CanonicalUnit,
				Similarity:    c.Similarity,
			})
		}
		out.Items = append(out.Items, r)
	}
	return out
}

func mapExtractionStatus(j *dom.ExtractionJob) extractionStatusResponse {
	out := extractionStatusResponse{
		ID:            j.ID,
		DocumentID:    j.DocumentID,
		Provider:      j.Provider,
		Model:         j.Model,
		Status:        string(j.Status),
		InputType:     string(j.InputType),
		PromptVersion: j.PromptVersion,
		ErrorMessage:  j.ErrorMessage,
		CreatedAt:     j.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:     j.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if j.StartedAt != nil {
		s := j.StartedAt.UTC().Format(time.RFC3339Nano)
		out.StartedAt = &s
	}
	if j.FinishedAt != nil {
		f := j.FinishedAt.UTC().Format(time.RFC3339Nano)
		out.FinishedAt = &f
	}
	return out
}

// Status responde GET /documents/:id/extraction-status com o job de extração
// mais recente do documento.
func (h *HealthExtractionHandler) Status(c *gin.Context) {
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
	job, err := h.svc.GetStatus(c.Request.Context(), ws, documentID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	resp := mapExtractionStatus(job)

	// Extração concluída: anexa as sugestões derivadas do extracted_json
	// (itens com resolução de marcador + match de laboratório).
	if doc, derr := h.docs.Get(c.Request.Context(), ws, documentID); derr == nil &&
		doc.ExtractionStatus == dom.ExtractionExtracted && len(doc.ExtractedJSON) > 0 {
		if sug, serr := h.imp.Suggest(c.Request.Context(), ws, doc); serr == nil {
			resp.Exam = mapExamSuggestion(sug)
		}
	}

	c.JSON(http.StatusOK, resp)
}
