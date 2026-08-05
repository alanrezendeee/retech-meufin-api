package health

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
)

// ExamImportService transforma o extracted_json de um documento de saúde em
// SUGESTÕES revisáveis (Suggest) e materializa a revisão do usuário em um
// resultado de exame (Confirm) — espelho do fluxo de importação de fatura do
// finance, adaptado ao domínio de exames (marcadores + laboratório).
type ExamImportService struct {
	docs    dom.DocumentRepository
	markers *MarkerService
	labs    *LabService
	results *ExamResultService
}

func NewExamImportService(docs dom.DocumentRepository, markers *MarkerService, labs *LabService, results *ExamResultService) *ExamImportService {
	return &ExamImportService{docs: docs, markers: markers, labs: labs, results: results}
}

// --- shape do extracted_json (tool registrar_exame, exam-extract-v1) ---

type extractedExamItem struct {
	ExamName      string   `json:"exam_name"`
	MarkerName    string   `json:"marker_name"`
	ResultValue   string   `json:"result_value"`
	NumericValue  *float64 `json:"numeric_value"`
	Unit          string   `json:"unit"`
	ReferenceMin  *float64 `json:"reference_min"`
	ReferenceMax  *float64 `json:"reference_max"`
	ReferenceText string   `json:"reference_text"`
	Material      string   `json:"material"`
	Method        string   `json:"method"`
	Interp        string   `json:"interpretation"`
	RawText       string   `json:"raw_text"`
}

type extractedExamJSON struct {
	PatientName    string              `json:"patient_name"`
	ExamDate       string              `json:"exam_date"`
	CollectionDate string              `json:"collection_date"`
	ReleaseDate    string              `json:"release_date"`
	LaboratoryName string              `json:"laboratory_name"`
	DoctorName     string              `json:"doctor_name"`
	Exams          []extractedExamItem `json:"exams"`
	Summary        string              `json:"summary"`
	Warnings       []string            `json:"warnings"`
}

// --- sugestões (saída do Suggest) ---

// MarkerCandidateSuggestion é um candidato fuzzy do catálogo para um item.
type MarkerCandidateSuggestion struct {
	MarkerID      uuid.UUID
	CanonicalName string
	CanonicalUnit *string
	Similarity    float64
}

// ExamItemSuggestion é um item do laudo com a resolução de marcador anexada.
type ExamItemSuggestion struct {
	RawName        string
	ResultValue    string
	ResultNumeric  *float64
	Unit           *string
	ReferenceMin   *float64
	ReferenceMax   *float64
	ReferenceText  *string
	Material       *string
	Method         *string
	Interpretation *string
	RawText        *string

	// Resolução contra o catálogo: matched|ambiguous|unresolved.
	MarkerStatus string
	MarkerID     *uuid.UUID
	MarkerName   *string
	Candidates   []MarkerCandidateSuggestion
	// MarkerIsNew: sem match exato — a UI oferece criar o marcador.
	MarkerIsNew bool
}

// ExamSuggestion é o laudo estruturado pronto para revisão na UI.
type ExamSuggestion struct {
	PatientName    string
	ExamDate       string // YYYY-MM-DD ("" quando ilegível)
	CollectionDate string
	ReleaseDate    string
	LaboratoryName string
	DoctorName     string
	Summary        string
	Warnings       []string
	Items          []ExamItemSuggestion

	// Match do laboratório por nome (case-insensitive) no cadastro da tenant.
	LabID    *uuid.UUID
	LabIsNew bool
}

// Suggest deriva as sugestões do extracted_json do documento. Exige que a
// extração já tenha concluído (extraction_status=extracted).
func (s *ExamImportService) Suggest(ctx context.Context, workspaceID uuid.UUID, doc *dom.Document) (*ExamSuggestion, error) {
	if len(doc.ExtractedJSON) == 0 {
		return nil, &dom.ValidationError{Msg: "documento ainda não tem dados extraídos"}
	}
	var raw extractedExamJSON
	if err := json.Unmarshal(doc.ExtractedJSON, &raw); err != nil {
		return nil, &dom.ValidationError{Msg: "extracted_json inválido"}
	}

	out := &ExamSuggestion{
		PatientName:    strings.TrimSpace(raw.PatientName),
		ExamDate:       normalizeExamDate(raw.ExamDate),
		CollectionDate: normalizeExamDate(raw.CollectionDate),
		ReleaseDate:    normalizeExamDate(raw.ReleaseDate),
		LaboratoryName: strings.TrimSpace(raw.LaboratoryName),
		DoctorName:     strings.TrimSpace(raw.DoctorName),
		Summary:        strings.TrimSpace(raw.Summary),
		Warnings:       raw.Warnings,
	}

	// Resolução de marcadores em lote (nome do marcador; fallback nome do exame).
	resolveIn := make([]ResolveItemInput, 0, len(raw.Exams))
	for _, e := range raw.Exams {
		name := strings.TrimSpace(e.MarkerName)
		if name == "" {
			name = strings.TrimSpace(e.ExamName)
		}
		resolveIn = append(resolveIn, ResolveItemInput{RawName: name})
	}
	resolved, err := s.markers.Resolve(ctx, workspaceID, resolveIn)
	if err != nil {
		return nil, err
	}

	out.Items = make([]ExamItemSuggestion, 0, len(raw.Exams))
	for i, e := range raw.Exams {
		it := ExamItemSuggestion{
			RawName:        resolveIn[i].RawName,
			ResultValue:    strings.TrimSpace(e.ResultValue),
			ResultNumeric:  e.NumericValue,
			Unit:           optStr(e.Unit),
			ReferenceMin:   e.ReferenceMin,
			ReferenceMax:   e.ReferenceMax,
			ReferenceText:  optStr(e.ReferenceText),
			Material:       optStr(e.Material),
			Method:         optStr(e.Method),
			Interpretation: optStr(e.Interp),
			RawText:        optStr(e.RawText),
			MarkerStatus:   string(ResolveUnresolved),
			MarkerIsNew:    true,
		}
		if i < len(resolved) {
			r := resolved[i]
			it.MarkerStatus = string(r.Status)
			switch r.Status {
			case ResolveMatched:
				id := r.Matched.ID
				name := r.Matched.CanonicalName
				it.MarkerID = &id
				it.MarkerName = &name
				it.MarkerIsNew = false
			case ResolveAmbiguous:
				for _, c := range r.Candidates {
					it.Candidates = append(it.Candidates, MarkerCandidateSuggestion{
						MarkerID:      c.Marker.ID,
						CanonicalName: c.Marker.CanonicalName,
						CanonicalUnit: c.Marker.CanonicalUnit,
						Similarity:    c.Similarity,
					})
				}
			}
		}
		out.Items = append(out.Items, it)
	}

	// Laboratório: match por nome no cadastro da tenant.
	if out.LaboratoryName != "" {
		if lab, err := s.findLabByName(ctx, workspaceID, out.LaboratoryName); err == nil && lab != nil {
			id := lab.ID
			out.LabID = &id
		} else {
			out.LabIsNew = true
		}
	}

	return out, nil
}

// --- confirmação (entrada revisada pelo usuário) ---

// ConfirmExamNewMarker descreve um marcador a criar no catálogo da tenant.
type ConfirmExamNewMarker struct {
	Name          string
	Category      string
	CanonicalUnit *string
	RefMin        *float64
	RefMax        *float64
	RefText       *string
	Aliases       []string
}

type ConfirmExamItemInput struct {
	MarkerID      *uuid.UUID
	NewMarker     *ConfirmExamNewMarker
	RawMarkerName *string
	ResultValue   string
	ResultNumeric *float64
	Unit          *string
	ReferenceMin  *float64
	ReferenceMax  *float64
	ReferenceText *string
	Method        *string
	Material      *string
	RawText       *string
}

type ConfirmExamInput struct {
	WorkspaceID    uuid.UUID
	DocumentID     uuid.UUID
	FamilyMemberID uuid.UUID
	ExamDate       time.Time
	CollectionDate *time.Time
	ReleaseDate    *time.Time
	LabID          *uuid.UUID
	NewLabName     *string
	Summary        *string
	Notes          *string
	Items          []ConfirmExamItemInput
}

type ConfirmExamResult struct {
	ExamResult     *dom.ExamResult
	CreatedMarkers []dom.Marker
	CreatedLab     *dom.Lab
}

// Confirm materializa a revisão: cria marcadores/lab novos quando pedido,
// cria o resultado (source=llm, status=extracted) e vincula o documento.
func (s *ExamImportService) Confirm(ctx context.Context, in ConfirmExamInput) (*ConfirmExamResult, error) {
	if len(in.Items) == 0 {
		return nil, &dom.ValidationError{Msg: "o exame precisa de ao menos um item"}
	}
	doc, err := s.docs.GetByID(ctx, in.WorkspaceID, in.DocumentID)
	if err != nil {
		return nil, err
	}

	out := &ConfirmExamResult{}

	// Laboratório: get-or-create por nome quando o usuário pediu criação.
	labID := in.LabID
	if labID == nil && in.NewLabName != nil && strings.TrimSpace(*in.NewLabName) != "" {
		name := strings.TrimSpace(*in.NewLabName)
		if existing, err := s.findLabByName(ctx, in.WorkspaceID, name); err == nil && existing != nil {
			id := existing.ID
			labID = &id
		} else {
			lab, err := s.labs.Create(ctx, CreateLabInput{
				WorkspaceID: in.WorkspaceID,
				Name:        name,
				Kind:        dom.LabKindLaboratorio,
				Active:      true,
			})
			if err != nil {
				return nil, err
			}
			id := lab.ID
			labID = &id
			out.CreatedLab = lab
		}
	}

	// Marcadores novos: cria (idempotente — duplicado reaproveita o existente).
	items := make([]CreateExamResultItemInput, 0, len(in.Items))
	for _, it := range in.Items {
		markerID := it.MarkerID
		if markerID == nil && it.NewMarker != nil && strings.TrimSpace(it.NewMarker.Name) != "" {
			category := strings.TrimSpace(it.NewMarker.Category)
			if category == "" {
				category = "outros"
			}
			m, err := s.markers.Create(ctx, CreateMarkerInput{
				WorkspaceID:    in.WorkspaceID,
				CanonicalName:  strings.TrimSpace(it.NewMarker.Name),
				Category:       category,
				CanonicalUnit:  it.NewMarker.CanonicalUnit,
				DefaultRefMin:  it.NewMarker.RefMin,
				DefaultRefMax:  it.NewMarker.RefMax,
				DefaultRefText: it.NewMarker.RefText,
				Aliases:        it.NewMarker.Aliases,
			})
			if err != nil {
				var dup *dom.DuplicateError
				if errors.As(err, &dup) && dup.Existing != nil {
					id := dup.Existing.ID
					markerID = &id
				} else {
					return nil, err
				}
			} else {
				id := m.ID
				markerID = &id
				out.CreatedMarkers = append(out.CreatedMarkers, *m)
			}
		}
		items = append(items, CreateExamResultItemInput{
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
		})
	}

	result, err := s.results.Create(ctx, CreateExamResultInput{
		WorkspaceID:    in.WorkspaceID,
		FamilyMemberID: in.FamilyMemberID,
		LabID:          labID,
		ExamDate:       in.ExamDate,
		CollectionDate: in.CollectionDate,
		ReleaseDate:    in.ReleaseDate,
		SourceType:     string(dom.SourceLLM),
		Status:         string(dom.ResultStatusExtracted),
		Summary:        in.Summary,
		Notes:          in.Notes,
		Items:          items,
	})
	if err != nil {
		return nil, err
	}
	out.ExamResult = result

	// Vincula o documento ao resultado (falha aqui não desfaz o resultado).
	resultID := result.ID
	doc.ExamResultID = &resultID
	if doc.FamilyMemberID == nil {
		fm := in.FamilyMemberID
		doc.FamilyMemberID = &fm
	}
	if doc.LabID == nil {
		doc.LabID = labID
	}
	doc.UpdatedAt = time.Now().UTC()
	_ = s.docs.LinkExamResult(ctx, doc)

	return out, nil
}

// findLabByName procura um laboratório ativo pelo nome (case-insensitive).
func (s *ExamImportService) findLabByName(ctx context.Context, workspaceID uuid.UUID, name string) (*dom.Lab, error) {
	res, err := s.labs.List(ctx, workspaceID, dom.LabFilter{Query: name}, 50, 0)
	if err != nil {
		return nil, err
	}
	want := strings.ToLower(strings.TrimSpace(name))
	for i := range res.Items {
		if strings.ToLower(strings.TrimSpace(res.Items[i].Name)) == want {
			return &res.Items[i], nil
		}
	}
	return nil, nil
}

// normalizeExamDate converte datas do laudo para YYYY-MM-DD. O prompt pede ISO,
// mas laudos brasileiros imprimem DD/MM/YYYY e o LLM pode transcrever
// literalmente. Retorna "" quando não consegue interpretar.
func normalizeExamDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	for _, layout := range []string{"2006-01-02", "02/01/2006", "02-01-2006", "02.01.2006", "02/01/06"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return ""
}

func optStr(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
