package health

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
)

type PlanService struct {
	repo dom.PlanRepository
}

func NewPlanService(repo dom.PlanRepository) *PlanService {
	return &PlanService{repo: repo}
}

type PlanMemberInput struct {
	MemberID   uuid.UUID
	CardNumber *string
	Holder     bool
}

type CreatePlanInput struct {
	WorkspaceID     uuid.UUID
	Name            string
	Operator        *string
	PlanType        dom.PlanType
	AnsCode         *string
	MonthlyFeeCents int64
	CoverageNotes   *string
	Active          bool
	Members         []PlanMemberInput
}

type UpdatePlanInput struct {
	WorkspaceID     uuid.UUID
	ID              uuid.UUID
	Name            string
	Operator        *string
	PlanType        dom.PlanType
	AnsCode         *string
	MonthlyFeeCents int64
	CoverageNotes   *string
	Active          bool
}

func (s *PlanService) Create(ctx context.Context, in CreatePlanInput) (*dom.Plan, error) {
	now := time.Now().UTC()
	p := &dom.Plan{
		ID:              uuid.New(),
		WorkspaceID:     in.WorkspaceID,
		Name:            in.Name,
		Operator:        in.Operator,
		PlanType:        in.PlanType,
		AnsCode:         in.AnsCode,
		MonthlyFeeCents: in.MonthlyFeeCents,
		CoverageNotes:   in.CoverageNotes,
		Active:          in.Active,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	for _, min := range in.Members {
		p.Members = append(p.Members, dom.PlanMember{
			ID:          uuid.New(),
			WorkspaceID: in.WorkspaceID,
			PlanID:      p.ID,
			MemberID:    min.MemberID,
			CardNumber:  min.CardNumber,
			Holder:      min.Holder,
			CreatedAt:   now,
		})
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, in.WorkspaceID, p.ID)
}

func (s *PlanService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Plan, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

type ListPlansResult struct {
	Items []dom.Plan
	Total int64
}

func (s *PlanService) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) (*ListPlansResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListPlansResult{Items: items, Total: total}, nil
}

func (s *PlanService) Update(ctx context.Context, in UpdatePlanInput) (*dom.Plan, error) {
	p, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	p.Name = in.Name
	p.Operator = in.Operator
	p.PlanType = in.PlanType
	p.AnsCode = in.AnsCode
	p.MonthlyFeeCents = in.MonthlyFeeCents
	p.CoverageNotes = in.CoverageNotes
	p.Active = in.Active
	p.UpdatedAt = time.Now().UTC()
	// Não revalida vínculos existentes (não são alterados aqui).
	p.Members = nil
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
}

func (s *PlanService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, workspaceID, id)
}

func (s *PlanService) ReplaceMembers(ctx context.Context, workspaceID, planID uuid.UUID, in []PlanMemberInput) (*dom.Plan, error) {
	now := time.Now().UTC()
	members := make([]dom.PlanMember, 0, len(in))
	for _, min := range in {
		m := dom.PlanMember{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			PlanID:      planID,
			MemberID:    min.MemberID,
			CardNumber:  min.CardNumber,
			Holder:      min.Holder,
			CreatedAt:   now,
		}
		if err := m.Validate(); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	if err := s.repo.ReplaceMembers(ctx, workspaceID, planID, members); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, workspaceID, planID)
}
