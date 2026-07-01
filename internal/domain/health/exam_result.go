package health

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SourceType indica a origem do resultado do exame.
type SourceType string

const (
	SourceManual SourceType = "manual"
	SourcePDF    SourceType = "pdf"
	SourceImage  SourceType = "image"
	SourceOCR    SourceType = "ocr"
	SourceLLM    SourceType = "llm"
)

// ExamResultStatus modela o ciclo de vida do resultado.
type ExamResultStatus string

const (
	ResultStatusDraft      ExamResultStatus = "draft"
	ResultStatusProcessing ExamResultStatus = "processing"
	ResultStatusExtracted  ExamResultStatus = "extracted"
	ResultStatusReviewed   ExamResultStatus = "reviewed"
	ResultStatusFailed     ExamResultStatus = "failed"
)

// Interpretation classifica um valor frente à sua faixa de referência.
// NÃO é diagnóstico, apenas posição relativa à faixa.
type Interpretation string

const (
	InterpretationLow          Interpretation = "low"
	InterpretationNormal       Interpretation = "normal"
	InterpretationHigh         Interpretation = "high"
	InterpretationCritical     Interpretation = "critical"
	InterpretationInconclusive Interpretation = "inconclusive"
)

// ExamResult é o cabeçalho de um conjunto de resultados de exame de um membro.
type ExamResult struct {
	ID             uuid.UUID
	WorkspaceID    uuid.UUID
	FamilyMemberID uuid.UUID
	LabID          *uuid.UUID
	ExamRequestID  *uuid.UUID
	ExamDate       time.Time
	CollectionDate *time.Time
	ReleaseDate    *time.Time
	SourceType     SourceType
	Status         ExamResultStatus
	Summary        *string
	Notes          *string
	Items          []ExamResultItem
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ExamResultItem é uma linha de resultado (um analito medido).
type ExamResultItem struct {
	ID                     uuid.UUID
	WorkspaceID            uuid.UUID
	ExamResultID           uuid.UUID
	MarkerID               *uuid.UUID
	RawMarkerName          *string
	ResultValue            string
	ResultNumeric          *float64
	Unit                   *string
	ReferenceMin           *float64
	ReferenceMax           *float64
	ReferenceText          *string
	Interpretation         *string
	InterpretationComputed *string
	Method                 *string
	Material               *string
	RawText                *string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// Validate valida invariantes do cabeçalho do resultado.
func (r *ExamResult) Validate() error {
	if r.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	if r.FamilyMemberID == uuid.Nil {
		return &ValidationError{Msg: "family_member_id é obrigatório"}
	}
	if r.ExamDate.IsZero() {
		return &ValidationError{Msg: "exam_date é obrigatório"}
	}
	switch r.SourceType {
	case SourceManual, SourcePDF, SourceImage, SourceOCR, SourceLLM:
	case "":
		r.SourceType = SourceManual
	default:
		return &ValidationError{Msg: "source_type inválido"}
	}
	switch r.Status {
	case ResultStatusDraft, ResultStatusProcessing, ResultStatusExtracted, ResultStatusReviewed, ResultStatusFailed:
	case "":
		r.Status = ResultStatusDraft
	default:
		return &ValidationError{Msg: "status inválido"}
	}
	for i := range r.Items {
		r.Items[i].WorkspaceID = r.WorkspaceID
		if err := r.Items[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Validate valida invariantes de um item de resultado.
func (it *ExamResultItem) Validate() error {
	if it.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	rv := strings.TrimSpace(it.ResultValue)
	if rv == "" {
		return &ValidationError{Msg: "result_value é obrigatório"}
	}
	it.ResultValue = rv
	if err := validateInterpretationPtr(it.Interpretation); err != nil {
		return err
	}
	if err := validateInterpretationPtr(it.InterpretationComputed); err != nil {
		return err
	}
	if it.ReferenceMin != nil && it.ReferenceMax != nil && *it.ReferenceMin > *it.ReferenceMax {
		return &ValidationError{Msg: "reference_min não pode ser maior que reference_max"}
	}
	return nil
}

func validateInterpretationPtr(p *string) error {
	if p == nil || *p == "" {
		return nil
	}
	switch Interpretation(*p) {
	case InterpretationLow, InterpretationNormal, InterpretationHigh, InterpretationCritical, InterpretationInconclusive:
		return nil
	default:
		return &ValidationError{Msg: "interpretation inválida"}
	}
}

// ParseResultNumeric extrai um valor numérico de um result_value em pt-BR.
// Aceita vírgula decimal ("1,23" -> 1.23), ignora prefixos "<"/">" ("<0,1" -> 0.1)
// e retorna nil para valores qualitativos ("Não reagente").
func ParseResultNumeric(s string) *float64 {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil
	}
	// remove prefixos comparativos e símbolos de igualdade/aproximação
	t = strings.TrimLeft(t, "<>=~≈≥≤ ")
	t = strings.TrimSpace(t)
	// normaliza separadores pt-BR: remove separador de milhar "." e troca vírgula por ponto.
	if strings.Contains(t, ",") {
		t = strings.ReplaceAll(t, ".", "")
		t = strings.ReplaceAll(t, ",", ".")
	}
	// captura o primeiro token numérico
	var b strings.Builder
	for i, r := range t {
		if r >= '0' && r <= '9' || r == '.' || (r == '-' && i == 0) {
			b.WriteRune(r)
			continue
		}
		break
	}
	num := b.String()
	if num == "" || num == "." || num == "-" {
		return nil
	}
	v, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return nil
	}
	return &v
}

// ComputeInterpretation classifica um valor frente à faixa [min,max].
// Retorna "low" (<min), "high" (>max) ou "normal". Retorna nil se não há valor.
// NÃO é diagnóstico, só posição na faixa.
func ComputeInterpretation(value *float64, min, max *float64) *string {
	if value == nil {
		return nil
	}
	var res string
	switch {
	case min != nil && *value < *min:
		res = string(InterpretationLow)
	case max != nil && *value > *max:
		res = string(InterpretationHigh)
	default:
		res = string(InterpretationNormal)
	}
	return &res
}

// ExamResultRepository abstrai a persistência de resultados e seus itens, escopo do tenant.
type ExamResultRepository interface {
	Create(ctx context.Context, r *ExamResult) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*ExamResult, error)
	Update(ctx context.Context, r *ExamResult) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]ExamResult, int64, error)

	AddItem(ctx context.Context, workspaceID uuid.UUID, item *ExamResultItem) error
	UpdateItem(ctx context.Context, item *ExamResultItem) error
	SoftDeleteItem(ctx context.Context, workspaceID, resultID, itemID uuid.UUID) error
}
