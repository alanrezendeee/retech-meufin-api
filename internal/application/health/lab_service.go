package health

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
)

type LabService struct {
	repo dom.LabRepository
}

func NewLabService(repo dom.LabRepository) *LabService {
	return &LabService{repo: repo}
}

type CreateLabInput struct {
	WorkspaceID    uuid.UUID
	Name           string
	WebsiteURL     *string
	ExamResultsURL *string
	ContactPhone   *string
	Address        *string
	Notes          *string
	Active         bool
}

type UpdateLabInput struct {
	WorkspaceID    uuid.UUID
	ID             uuid.UUID
	Name           string
	WebsiteURL     *string
	ExamResultsURL *string
	ContactPhone   *string
	Address        *string
	Notes          *string
	Active         bool
}

func (s *LabService) Create(ctx context.Context, in CreateLabInput) (*dom.Lab, error) {
	now := time.Now().UTC()
	l := &dom.Lab{
		ID:             uuid.New(),
		WorkspaceID:    in.WorkspaceID,
		Name:           in.Name,
		WebsiteURL:     in.WebsiteURL,
		ExamResultsURL: in.ExamResultsURL,
		ContactPhone:   in.ContactPhone,
		Address:        in.Address,
		Notes:          in.Notes,
		Active:         in.Active,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := l.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (s *LabService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Lab, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

func (s *LabService) Update(ctx context.Context, in UpdateLabInput) (*dom.Lab, error) {
	l, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	l.Name = in.Name
	l.WebsiteURL = in.WebsiteURL
	l.ExamResultsURL = in.ExamResultsURL
	l.ContactPhone = in.ContactPhone
	l.Address = in.Address
	l.Notes = in.Notes
	l.Active = in.Active
	l.UpdatedAt = time.Now().UTC()
	if err := l.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (s *LabService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, workspaceID, id)
}

type ListLabsResult struct {
	Items []dom.Lab
	Total int64
}

func (s *LabService) List(ctx context.Context, workspaceID uuid.UUID, filter dom.LabFilter, limit, offset int) (*ListLabsResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListLabsResult{Items: items, Total: total}, nil
}
