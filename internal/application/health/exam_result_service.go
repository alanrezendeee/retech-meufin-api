package health

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
)

type ExamResultService struct {
	repo dom.ExamResultRepository
}

func NewExamResultService(repo dom.ExamResultRepository) *ExamResultService {
	return &ExamResultService{repo: repo}
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
// interpretation_computed a partir do valor e da faixa de referência.
func applyItemDerived(it *dom.ExamResultItem) {
	if it.ResultNumeric == nil {
		it.ResultNumeric = dom.ParseResultNumeric(it.ResultValue)
	}
	it.InterpretationComputed = dom.ComputeInterpretation(it.ResultNumeric, it.ReferenceMin, it.ReferenceMax)
}

func (s *ExamResultService) Create(ctx context.Context, in CreateExamResultInput) (*dom.ExamResult, error) {
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
		applyItemDerived(&item)
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
	return s.repo.GetByID(ctx, workspaceID, id)
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
	applyItemDerived(item)
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
	applyItemDerived(current)
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
