package ledger

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/ledger"
)

type TransactionService struct {
	txRepo dom.TransactionRepository
	accRepo dom.AccountRepository
	catRepo dom.CategoryRepository
}

func NewTransactionService(
	txRepo dom.TransactionRepository,
	accRepo dom.AccountRepository,
	catRepo dom.CategoryRepository,
) *TransactionService {
	return &TransactionService{txRepo: txRepo, accRepo: accRepo, catRepo: catRepo}
}

type CreateTransactionInput struct {
	WorkspaceID uuid.UUID
	AccountID   uuid.UUID
	CategoryID  uuid.UUID
	AmountCents int64
	Flow        dom.Flow
	Description string
	OccurredAt  time.Time
}

type UpdateTransactionInput struct {
	WorkspaceID uuid.UUID
	ID          uuid.UUID
	AccountID   uuid.UUID
	CategoryID  uuid.UUID
	AmountCents int64
	Flow        dom.Flow
	Description string
	OccurredAt  time.Time
}

func (s *TransactionService) Create(ctx context.Context, in CreateTransactionInput) (*dom.Transaction, error) {
	if _, err := s.accRepo.GetByID(ctx, in.WorkspaceID, in.AccountID); err != nil {
		return nil, err
	}
	cat, err := s.catRepo.GetByID(ctx, in.WorkspaceID, in.CategoryID)
	if err != nil {
		return nil, err
	}

	tx := &dom.Transaction{
		ID:          uuid.New(),
		WorkspaceID: in.WorkspaceID,
		AccountID:   in.AccountID,
		CategoryID:  in.CategoryID,
		AmountCents: in.AmountCents,
		Flow:        in.Flow,
		Description: in.Description,
		OccurredAt:  in.OccurredAt.UTC(),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := tx.Validate(); err != nil {
		return nil, err
	}
	if !tx.MatchesCategoryKind(cat.Kind) {
		return nil, dom.ErrCategoryKindMismatch
	}

	if err := s.txRepo.Create(ctx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (s *TransactionService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Transaction, error) {
	return s.txRepo.GetByID(ctx, workspaceID, id)
}

func (s *TransactionService) Update(ctx context.Context, in UpdateTransactionInput) (*dom.Transaction, error) {
	tx, err := s.txRepo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	_, err = s.accRepo.GetByID(ctx, in.WorkspaceID, in.AccountID)
	if err != nil {
		return nil, err
	}
	cat, err := s.catRepo.GetByID(ctx, in.WorkspaceID, in.CategoryID)
	if err != nil {
		return nil, err
	}

	tx.AccountID = in.AccountID
	tx.CategoryID = in.CategoryID
	tx.AmountCents = in.AmountCents
	tx.Flow = in.Flow
	tx.Description = in.Description
	tx.OccurredAt = in.OccurredAt.UTC()
	tx.UpdatedAt = time.Now().UTC()

	if err := tx.Validate(); err != nil {
		return nil, err
	}
	if !tx.MatchesCategoryKind(cat.Kind) {
		return nil, dom.ErrCategoryKindMismatch
	}
	if err := s.txRepo.Update(ctx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (s *TransactionService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.txRepo.Delete(ctx, workspaceID, id)
}

type ListTransactionsResult struct {
	Items []dom.Transaction
	Total int64
}

func (s *TransactionService) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) (*ListTransactionsResult, error) {
	items, total, err := s.txRepo.List(ctx, workspaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListTransactionsResult{Items: items, Total: total}, nil
}
