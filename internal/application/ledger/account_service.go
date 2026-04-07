package ledger

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/ledger"
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
	Currency    string
}

type UpdateAccountInput struct {
	WorkspaceID uuid.UUID
	ID          uuid.UUID
	Name        string
	Currency    string
}

func (s *AccountService) Create(ctx context.Context, in CreateAccountInput) (*dom.Account, error) {
	a := &dom.Account{
		ID:          uuid.New(),
		WorkspaceID: in.WorkspaceID,
		Name:        in.Name,
		Currency:    in.Currency,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := a.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *AccountService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Account, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

func (s *AccountService) Update(ctx context.Context, in UpdateAccountInput) (*dom.Account, error) {
	a, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	a.Name = in.Name
	a.Currency = in.Currency
	a.UpdatedAt = time.Now().UTC()
	if err := a.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *AccountService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.Delete(ctx, workspaceID, id)
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

func IsNotFound(err error) bool {
	return errors.Is(err, dom.ErrNotFound)
}
