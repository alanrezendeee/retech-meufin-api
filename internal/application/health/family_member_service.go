package health

import (
	"context"
	"errors"
	"sort"
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
	HeightCm     *float64
	WeightKg     *float64
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
	HeightCm     *float64
	WeightKg     *float64
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
		HeightCm:     in.HeightCm,
		WeightKg:     in.WeightKg,
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
	f.HeightCm = in.HeightCm
	f.WeightKg = in.WeightKg
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

// Birthdays retorna os membros ativos com data de nascimento, ordenados pelo
// próximo aniversário (menos dias restantes primeiro).
func (s *FamilyMemberService) Birthdays(ctx context.Context, workspaceID uuid.UUID) ([]dom.Birthday, error) {
	members, err := s.repo.ListWithBirthDate(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := make([]dom.Birthday, 0, len(members))
	for i := range members {
		b, ok := dom.NextBirthdayOf(members[i], now)
		if !ok {
			continue
		}
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DaysUntil != out[j].DaysUntil {
			return out[i].DaysUntil < out[j].DaysUntil
		}
		return out[i].Member.FullName < out[j].Member.FullName
	})
	return out, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, dom.ErrNotFound)
}
