package health

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
)

type FamilyMemberService struct {
	repo dom.FamilyMemberRepository
}

func NewFamilyMemberService(repo dom.FamilyMemberRepository) *FamilyMemberService {
	return &FamilyMemberService{repo: repo}
}

type CreateFamilyMemberInput struct {
	WorkspaceID  uuid.UUID
	FullName     string
	Relationship string
	BirthDate    *time.Time
	Gender       *string
	Document     *string
	Notes        *string
	Active       *bool
}

type UpdateFamilyMemberInput struct {
	WorkspaceID  uuid.UUID
	ID           uuid.UUID
	FullName     string
	Relationship string
	BirthDate    *time.Time
	Gender       *string
	Document     *string
	Notes        *string
	Active       *bool
}

func (s *FamilyMemberService) Create(ctx context.Context, in CreateFamilyMemberInput) (*dom.FamilyMember, error) {
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	f := &dom.FamilyMember{
		ID:           uuid.New(),
		WorkspaceID:  in.WorkspaceID,
		FullName:     in.FullName,
		Relationship: in.Relationship,
		BirthDate:    in.BirthDate,
		Gender:       in.Gender,
		Document:     in.Document,
		Notes:        in.Notes,
		Active:       active,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := f.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *FamilyMemberService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.FamilyMember, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

func (s *FamilyMemberService) Update(ctx context.Context, in UpdateFamilyMemberInput) (*dom.FamilyMember, error) {
	f, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	f.FullName = in.FullName
	f.Relationship = in.Relationship
	f.BirthDate = in.BirthDate
	f.Gender = in.Gender
	f.Document = in.Document
	f.Notes = in.Notes
	if in.Active != nil {
		f.Active = *in.Active
	}
	f.UpdatedAt = time.Now().UTC()
	if err := f.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *FamilyMemberService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, workspaceID, id)
}

type ListFamilyMembersResult struct {
	Items []dom.FamilyMember
	Total int64
}

func (s *FamilyMemberService) List(ctx context.Context, workspaceID uuid.UUID, filter dom.FamilyMemberFilter, limit, offset int) (*ListFamilyMembersResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListFamilyMembersResult{Items: items, Total: total}, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, dom.ErrNotFound)
}
