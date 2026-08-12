package health

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
)

type ExamResultService struct {
	repo    dom.ExamResultRepository
	markers dom.MarkerRepository
}

func NewExamResultService(repo dom.ExamResultRepository, markers dom.MarkerRepository) *ExamResultService {
	return &ExamResultService{repo: repo, markers: markers}
}

type CreateExamResultItemInput struct {
	MarkerID       *uuid.UUID
	RawMarkerName  *string
	ResultValue    string
	ResultNumeric  *float64
	Unit           *string
	ReferenceMin   *float64
	ReferenceMax   *float64
	ReferenceText  *string
	Interpretation *string
	Method         *string
	Material       *string
	RawText        *string
}

type CreateExamResultInput struct {
	WorkspaceID    uuid.UUID
	FamilyMemberID uuid.UUID
	LabID          *uuid.UUID
	ExamRequestID  *uuid.UUID
	ExamDate       time.Time
	CollectionDate *time.Time
	ReleaseDate    *time.Time
	SourceType     string
	Status         string
	Summary        *string
	Notes          *string
	Items          []CreateExamResultItemInput
}

type UpdateExamResultInput struct {
	WorkspaceID    uuid.UUID
	ID             uuid.UUID
	FamilyMemberID uuid.UUID
	LabID          *uuid.UUID
	ExamRequestID  *uuid.UUID
	ExamDate       time.Time
	CollectionDate *time.Time
	ReleaseDate    *time.Time
	SourceType     string
	Status         string
	Summary        *string
	Notes          *string
}

type ExamResultItemInput struct {
	MarkerID       *uuid.UUID
	RawMarkerName  *string
	ResultValue    string
	ResultNumeric  *float64
	Unit           *string
	ReferenceMin   *float64
	ReferenceMax   *float64
	ReferenceText  *string
	Interpretation *string
	Method         *string
	Material       *string
	RawText        *string
}

type AddExamResultItemInput struct {
	WorkspaceID  uuid.UUID
	ExamResultID uuid.UUID
	Item         ExamResultItemInput
}

type UpdateExamResultItemInput struct {
	WorkspaceID  uuid.UUID
	ExamResultID uuid.UUID
	ItemID       uuid.UUID
	Item         ExamResultItemInput
}

// applyItemDerived preenche result_numeric (quando ausente) e sempre recalcula
// interpretation_computed a partir do valor e da faixa de referência. Quando o
// laudo não traz faixa (ex.: VLDL — "não dispomos de valor de referência"),
// usa a faixa de CURADORIA do catálogo (default_ref do marcador) como
// fallback — a faixa do item permanece nula, fiel ao laudo. Metas condicionais
// (tiers, ex.: LDL por risco) NÃO geram veredito único: são confrontadas
// informativamente na UI, linha a linha.
func (s *ExamResultService) applyItemDerived(ctx context.Context, workspaceID uuid.UUID, it *dom.ExamResultItem) {
	if it.ResultNumeric == nil {
		it.ResultNumeric = dom.ParseResultNumeric(it.ResultValue)
	}
	refMin, refMax := it.ReferenceMin, it.ReferenceMax
	if refMin == nil && refMax == nil && it.MarkerID != nil {
		if m, err := s.markers.GetByID(ctx, workspaceID, *it.MarkerID); err == nil {
			refMin, refMax = m.DefaultRefMin, m.DefaultRefMax
		}
	}
	it.InterpretationComputed = dom.ComputeInterpretation(it.ResultNumeric, refMin, refMax)
}

func (s *ExamResultService) Create(ctx context.Context, in CreateExamResultInput) (*dom.ExamResult, error) {
	// Resultado descritivo (laudo de imagem sem medidas): permitido sem itens,
	// desde que o resumo carregue a informação.
	if len(in.Items) == 0 && (in.Summary == nil || strings.TrimSpace(*in.Summary) == "") {
		return nil, &dom.ValidationError{Msg: "resultado sem itens precisa de um resumo (laudos descritivos)"}
	}
	now := time.Now().UTC()
	r := &dom.ExamResult{
		ID:             uuid.New(),
		WorkspaceID:    in.WorkspaceID,
		FamilyMemberID: in.FamilyMemberID,
		LabID:          in.LabID,
		ExamRequestID:  in.ExamRequestID,
		ExamDate:       in.ExamDate,
		CollectionDate: in.CollectionDate,
		ReleaseDate:    in.ReleaseDate,
		SourceType:     dom.SourceType(in.SourceType),
		Status:         dom.ExamResultStatus(in.Status),
		Summary:        in.Summary,
		Notes:          in.Notes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	for _, ii := range in.Items {
		item := dom.ExamResultItem{
			ID:             uuid.New(),
			WorkspaceID:    in.WorkspaceID,
			ExamResultID:   r.ID,
			MarkerID:       ii.MarkerID,
			RawMarkerName:  ii.RawMarkerName,
			ResultValue:    ii.ResultValue,
			ResultNumeric:  ii.ResultNumeric,
			Unit:           ii.Unit,
			ReferenceMin:   ii.ReferenceMin,
			ReferenceMax:   ii.ReferenceMax,
			ReferenceText:  ii.ReferenceText,
			Interpretation: ii.Interpretation,
			Method:         ii.Method,
			Material:       ii.Material,
			RawText:        ii.RawText,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		s.applyItemDerived(ctx, in.WorkspaceID, &item)
		r.Items = append(r.Items, item)
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *ExamResultService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.ExamResult, error) {
	r, err := s.repo.GetByID(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}
	s.attachMarkerInfo(ctx, workspaceID, r)
	return r, nil
}

// attachMarkerInfo anexa a curadoria do marcador (nome, texto de referência e
// tiers) aos itens — a UI usa para tooltip e para o confronto informativo com
// a tabela (ex.: LDL por risco). Falha em um marcador não derruba a leitura.
func (s *ExamResultService) attachMarkerInfo(ctx context.Context, workspaceID uuid.UUID, r *dom.ExamResult) {
	cache := map[uuid.UUID]*dom.MarkerInfo{}
	for i := range r.Items {
		id := r.Items[i].MarkerID
		if id == nil {
			continue
		}
		info, seen := cache[*id]
		if !seen {
			if m, err := s.markers.GetByID(ctx, workspaceID, *id); err == nil {
				info = &dom.MarkerInfo{
					CanonicalName: m.CanonicalName,
					RefText:       m.DefaultRefText,
					RefTiers:      m.DefaultRefTiers,
				}
			}
			cache[*id] = info
		}
		r.Items[i].Marker = info
	}
}

func (s *ExamResultService) Update(ctx context.Context, in UpdateExamResultInput) (*dom.ExamResult, error) {
	r, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	r.FamilyMemberID = in.FamilyMemberID
	r.LabID = in.LabID
	r.ExamRequestID = in.ExamRequestID
	r.ExamDate = in.ExamDate
	r.CollectionDate = in.CollectionDate
	r.ReleaseDate = in.ReleaseDate
	r.SourceType = dom.SourceType(in.SourceType)
	r.Status = dom.ExamResultStatus(in.Status)
	r.Summary = in.Summary
	r.Notes = in.Notes
	r.UpdatedAt = time.Now().UTC()
	// não revalida itens já persistidos; validação de cabeçalho apenas.
	r.Items = nil
	if err := r.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, r); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
}

func (s *ExamResultService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, workspaceID, id)
}

type ListExamResultsResult struct {
	Items []dom.ExamResult
	Total int64
}

func (s *ExamResultService) List(ctx context.Context, workspaceID uuid.UUID, filter dom.ExamResultFilter, limit, offset int) (*ListExamResultsResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListExamResultsResult{Items: items, Total: total}, nil
}

func (s *ExamResultService) AddItem(ctx context.Context, in AddExamResultItemInput) (*dom.ExamResultItem, error) {
	now := time.Now().UTC()
	item := &dom.ExamResultItem{
		ID:             uuid.New(),
		WorkspaceID:    in.WorkspaceID,
		ExamResultID:   in.ExamResultID,
		MarkerID:       in.Item.MarkerID,
		RawMarkerName:  in.Item.RawMarkerName,
		ResultValue:    in.Item.ResultValue,
		ResultNumeric:  in.Item.ResultNumeric,
		Unit:           in.Item.Unit,
		ReferenceMin:   in.Item.ReferenceMin,
		ReferenceMax:   in.Item.ReferenceMax,
		ReferenceText:  in.Item.ReferenceText,
		Interpretation: in.Item.Interpretation,
		Method:         in.Item.Method,
		Material:       in.Item.Material,
		RawText:        in.Item.RawText,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	s.applyItemDerived(ctx, in.WorkspaceID, item)
	if err := item.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.AddItem(ctx, in.WorkspaceID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ExamResultService) UpdateItem(ctx context.Context, in UpdateExamResultItemInput) (*dom.ExamResultItem, error) {
	// garante que o resultado pai existe/pertence ao workspace e localiza o item.
	r, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ExamResultID)
	if err != nil {
		return nil, err
	}
	var current *dom.ExamResultItem
	for i := range r.Items {
		if r.Items[i].ID == in.ItemID {
			current = &r.Items[i]
			break
		}
	}
	if current == nil {
		return nil, dom.ErrNotFound
	}
	current.MarkerID = in.Item.MarkerID
	current.RawMarkerName = in.Item.RawMarkerName
	current.ResultValue = in.Item.ResultValue
	current.ResultNumeric = in.Item.ResultNumeric
	current.Unit = in.Item.Unit
	current.ReferenceMin = in.Item.ReferenceMin
	current.ReferenceMax = in.Item.ReferenceMax
	current.ReferenceText = in.Item.ReferenceText
	current.Interpretation = in.Item.Interpretation
	current.Method = in.Item.Method
	current.Material = in.Item.Material
	current.RawText = in.Item.RawText
	current.UpdatedAt = time.Now().UTC()
	s.applyItemDerived(ctx, in.WorkspaceID, current)
	if err := current.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateItem(ctx, current); err != nil {
		return nil, err
	}
	return current, nil
}

func (s *ExamResultService) DeleteItem(ctx context.Context, workspaceID, resultID, itemID uuid.UUID) error {
	return s.repo.SoftDeleteItem(ctx, workspaceID, resultID, itemID)
}
