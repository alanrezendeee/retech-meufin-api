package finance

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

type AccountService struct {
	repo dom.AccountRepository
}

func NewAccountService(repo dom.AccountRepository) *AccountService {
	return &AccountService{repo: repo}
}

type CreateAccountInput struct {
	WorkspaceID uuid.UUID
	Name        string
	Kind        string
	BankName    *string
	Active      *bool
	Notes       *string
}

type UpdateAccountInput struct {
	WorkspaceID uuid.UUID
	ID          uuid.UUID
	Name        string
	Kind        string
	BankName    *string
	Active      *bool
	Notes       *string
}

func (s *AccountService) Create(ctx context.Context, in CreateAccountInput) (*dom.Account, error) {
	now := time.Now().UTC()
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	acc := &dom.Account{
		ID:          uuid.New(),
		WorkspaceID: in.WorkspaceID,
		Name:        in.Name,
		Kind:        dom.AccountKind(in.Kind),
		BankName:    in.BankName,
		Active:      active,
		Notes:       in.Notes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := acc.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, acc); err != nil {
		return nil, err
	}
	return acc, nil
}

func (s *AccountService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Account, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

type ListAccountsResult struct {
	Items []dom.Account
	Total int64
}

func (s *AccountService) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) (*ListAccountsResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListAccountsResult{Items: items, Total: total}, nil
}

func (s *AccountService) Update(ctx context.Context, in UpdateAccountInput) (*dom.Account, error) {
	acc, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	acc.Name = in.Name
	acc.Kind = dom.AccountKind(in.Kind)
	acc.BankName = in.BankName
	if in.Active != nil {
		acc.Active = *in.Active
	}
	acc.Notes = in.Notes
	acc.UpdatedAt = time.Now().UTC()
	if err := acc.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, acc); err != nil {
		return nil, err
	}
	return acc, nil
}

func (s *AccountService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, workspaceID, id)
}
