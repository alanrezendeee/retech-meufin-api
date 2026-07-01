package finance

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

type FinancialEntryService struct {
	repo dom.FinancialEntryRepository
}

func NewFinancialEntryService(repo dom.FinancialEntryRepository) *FinancialEntryService {
	return &FinancialEntryService{repo: repo}
}

type CreateEntryInput struct {
	WorkspaceID       uuid.UUID
	Kind              string
	Status            string // opcional, default prevista
	AmountCents       int64
	DueDate           time.Time
	FamilyMemberID    *uuid.UUID
	SourceID          *uuid.UUID
	Type              *string
	Description       string
	Recurrence        string
	Notes             *string
	CardID            *uuid.UUID
	ParentID          *uuid.UUID
	InstallmentsTotal *int
}

type UpdateEntryInput struct {
	WorkspaceID    uuid.UUID
	ID             uuid.UUID
	Kind           string
	Status         string
	AmountCents    int64
	DueDate        time.Time
	FamilyMemberID *uuid.UUID
	SourceID       *uuid.UUID
	Type           *string
	Description    string
	Recurrence     string
	Notes          *string
}

// Create monta o lançamento base, gera as ocorrências recorrentes e persiste em lote.
func (s *FinancialEntryService) Create(ctx context.Context, in CreateEntryInput) ([]dom.FinancialEntry, error) {
	now := time.Now().UTC()
	status := dom.Status(in.Status)
	if status == "" {
		status = dom.StatusPrevista
	}
	recurrence := dom.Recurrence(in.Recurrence)
	if recurrence == "" {
		recurrence = dom.RecurrenceNone
	}
	base := dom.FinancialEntry{
		ID:             uuid.New(),
		WorkspaceID:    in.WorkspaceID,
		Kind:           dom.Kind(in.Kind),
		Status:         status,
		AmountCents:    in.AmountCents,
		DueDate:        in.DueDate,
		FamilyMemberID: in.FamilyMemberID,
		SourceID:       in.SourceID,
		Type:           in.Type,
		Description:    in.Description,
		Recurrence:     recurrence,
		CardID:         in.CardID,
		ParentID:       in.ParentID,
		Notes:          in.Notes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Três caminhos mutuamente exclusivos: parcelado, recorrente ou único.
	var occurrences []dom.FinancialEntry
	switch {
	case in.InstallmentsTotal != nil && *in.InstallmentsTotal > 1:
		// Parcelado: N lançamentos mensais, recurrence forçada para none.
		base.Recurrence = dom.RecurrenceNone
		base.Status = dom.StatusPrevista
		if err := base.Validate(); err != nil {
			return nil, err
		}
		occurrences = dom.GenerateInstallments(base, *in.InstallmentsTotal)
	default:
		// Recorrente (recurrence != none) ou único.
		if err := base.Validate(); err != nil {
			return nil, err
		}
		occurrences = dom.GenerateOccurrences(base)
	}
	batch := make([]*dom.FinancialEntry, len(occurrences))
	for i := range occurrences {
		occ := occurrences[i]
		occ.CreatedAt = now
		occ.UpdatedAt = now
		batch[i] = &occ
	}
	if err := s.repo.CreateBatch(ctx, batch); err != nil {
		return nil, err
	}
	out := make([]dom.FinancialEntry, len(batch))
	for i := range batch {
		out[i] = *batch[i]
	}
	return out, nil
}

func (s *FinancialEntryService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.FinancialEntry, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

type ListEntriesResult struct {
	Items []dom.FinancialEntry
	Total int64
}

func (s *FinancialEntryService) List(ctx context.Context, workspaceID uuid.UUID, filter dom.FinancialEntryFilter, limit, offset int) (*ListEntriesResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListEntriesResult{Items: items, Total: total}, nil
}

func (s *FinancialEntryService) Update(ctx context.Context, in UpdateEntryInput) (*dom.FinancialEntry, error) {
	e, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	e.Kind = dom.Kind(in.Kind)
	if in.Status != "" {
		e.Status = dom.Status(in.Status)
	}
	e.AmountCents = in.AmountCents
	e.DueDate = in.DueDate
	e.FamilyMemberID = in.FamilyMemberID
	e.SourceID = in.SourceID
	e.Type = in.Type
	e.Description = in.Description
	if in.Recurrence != "" {
		e.Recurrence = dom.Recurrence(in.Recurrence)
	}
	e.Notes = in.Notes
	e.UpdatedAt = time.Now().UTC()
	if err := e.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *FinancialEntryService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, workspaceID, id)
}

// Confirm marca o lançamento como realizado.
func (s *FinancialEntryService) Confirm(ctx context.Context, workspaceID, id uuid.UUID) (*dom.FinancialEntry, error) {
	return s.setStatus(ctx, workspaceID, id, dom.StatusRealizada)
}

// Cancel marca o lançamento como cancelado.
func (s *FinancialEntryService) Cancel(ctx context.Context, workspaceID, id uuid.UUID) (*dom.FinancialEntry, error) {
	return s.setStatus(ctx, workspaceID, id, dom.StatusCancelada)
}

func (s *FinancialEntryService) setStatus(ctx context.Context, workspaceID, id uuid.UUID, status dom.Status) (*dom.FinancialEntry, error) {
	e, err := s.repo.GetByID(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}
	e.Status = status
	e.UpdatedAt = time.Now().UTC()
	if err := e.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}
