package finance

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

type CreditCardService struct {
	repo dom.CreditCardRepository
}

func NewCreditCardService(repo dom.CreditCardRepository) *CreditCardService {
	return &CreditCardService{repo: repo}
}

type CreateCreditCardInput struct {
	WorkspaceID uuid.UUID
	Name        string
	Brand       *string
	Bank        *string
	ClosingDay  *int
	DueDay      *int
	Active      *bool
	Notes       *string
}

type UpdateCreditCardInput struct {
	WorkspaceID uuid.UUID
	ID          uuid.UUID
	Name        string
	Brand       *string
	Bank        *string
	ClosingDay  *int
	DueDay      *int
	Active      *bool
	Notes       *string
}

func (s *CreditCardService) Create(ctx context.Context, in CreateCreditCardInput) (*dom.CreditCard, error) {
	now := time.Now().UTC()
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	card := &dom.CreditCard{
		ID:          uuid.New(),
		WorkspaceID: in.WorkspaceID,
		Name:        in.Name,
		Brand:       in.Brand,
		Bank:        in.Bank,
		ClosingDay:  in.ClosingDay,
		DueDay:      in.DueDay,
		Active:      active,
		Notes:       in.Notes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := card.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, card); err != nil {
		return nil, err
	}
	return card, nil
}

func (s *CreditCardService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.CreditCard, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

type ListCreditCardsResult struct {
	Items []dom.CreditCard
	Total int64
}

func (s *CreditCardService) List(ctx context.Context, workspaceID uuid.UUID, filter dom.CreditCardFilter, limit, offset int) (*ListCreditCardsResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListCreditCardsResult{Items: items, Total: total}, nil
}

func (s *CreditCardService) Update(ctx context.Context, in UpdateCreditCardInput) (*dom.CreditCard, error) {
	card, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	card.Name = in.Name
	card.Brand = in.Brand
	card.Bank = in.Bank
	card.ClosingDay = in.ClosingDay
	card.DueDay = in.DueDay
	if in.Active != nil {
		card.Active = *in.Active
	}
	card.Notes = in.Notes
	card.UpdatedAt = time.Now().UTC()
	if err := card.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, card); err != nil {
		return nil, err
	}
	return card, nil
}

func (s *CreditCardService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, workspaceID, id)
}
