package finance

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

type IncomeSourceService struct {
	repo dom.IncomeSourceRepository
}

func NewIncomeSourceService(repo dom.IncomeSourceRepository) *IncomeSourceService {
	return &IncomeSourceService{repo: repo}
}

type CreateIncomeSourceInput struct {
	WorkspaceID uuid.UUID
	Name        string
	Kind        string
	Active      *bool
	Notes       *string
}

type UpdateIncomeSourceInput struct {
	WorkspaceID uuid.UUID
	ID          uuid.UUID
	Name        string
	Kind        string
	Active      *bool
	Notes       *string
}

func (s *IncomeSourceService) Create(ctx context.Context, in CreateIncomeSourceInput) (*dom.IncomeSource, error) {
	now := time.Now().UTC()
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	src := &dom.IncomeSource{
		ID:          uuid.New(),
		WorkspaceID: in.WorkspaceID,
		Name:        in.Name,
		Kind:        dom.IncomeSourceKind(in.Kind),
		Active:      active,
		Notes:       in.Notes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := src.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, src); err != nil {
		return nil, err
	}
	return src, nil
}

func (s *IncomeSourceService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.IncomeSource, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

type ListIncomeSourcesResult struct {
	Items []dom.IncomeSource
	Total int64
}

func (s *IncomeSourceService) List(ctx context.Context, workspaceID uuid.UUID, filter dom.IncomeSourceFilter, limit, offset int) (*ListIncomeSourcesResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListIncomeSourcesResult{Items: items, Total: total}, nil
}

func (s *IncomeSourceService) Update(ctx context.Context, in UpdateIncomeSourceInput) (*dom.IncomeSource, error) {
	src, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	src.Name = in.Name
	src.Kind = dom.IncomeSourceKind(in.Kind)
	if in.Active != nil {
		src.Active = *in.Active
	}
	src.Notes = in.Notes
	src.UpdatedAt = time.Now().UTC()
	if err := src.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, src); err != nil {
		return nil, err
	}
	return src, nil
}

func (s *IncomeSourceService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, workspaceID, id)
}
