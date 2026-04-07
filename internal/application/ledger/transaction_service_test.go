package ledger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/ledger"
)

type fakeAccRepo struct {
	acc *dom.Account
	err error
}

func (f *fakeAccRepo) Create(ctx context.Context, a *dom.Account) error { return nil }
func (f *fakeAccRepo) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Account, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.acc, nil
}
func (f *fakeAccRepo) Update(ctx context.Context, a *dom.Account) error   { return nil }
func (f *fakeAccRepo) Delete(ctx context.Context, workspaceID, id uuid.UUID) error { return nil }
func (f *fakeAccRepo) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]dom.Account, int64, error) {
	return nil, 0, nil
}

type fakeCatRepo struct {
	cat *dom.Category
	err error
}

func (f *fakeCatRepo) Create(ctx context.Context, c *dom.Category) error { return nil }
func (f *fakeCatRepo) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Category, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.cat, nil
}
func (f *fakeCatRepo) Update(ctx context.Context, c *dom.Category) error { return nil }
func (f *fakeCatRepo) Delete(ctx context.Context, workspaceID, id uuid.UUID) error { return nil }
func (f *fakeCatRepo) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]dom.Category, int64, error) {
	return nil, 0, nil
}

type fakeTxRepo struct {
	created *dom.Transaction
}

func (f *fakeTxRepo) Create(ctx context.Context, t *dom.Transaction) error {
	f.created = t
	return nil
}
func (f *fakeTxRepo) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Transaction, error) {
	return nil, dom.ErrNotFound
}
func (f *fakeTxRepo) Update(ctx context.Context, t *dom.Transaction) error { return nil }
func (f *fakeTxRepo) Delete(ctx context.Context, workspaceID, id uuid.UUID) error { return nil }
func (f *fakeTxRepo) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]dom.Transaction, int64, error) {
	return nil, 0, nil
}
func (f *fakeTxRepo) SumOutflowsByCategoryInMonth(ctx context.Context, workspaceID, categoryID uuid.UUID, year, month int) (int64, error) {
	return 0, nil
}

func TestTransactionService_Create_OK(t *testing.T) {
	ws := uuid.New()
	accID := uuid.New()
	catID := uuid.New()
	svc := NewTransactionService(
		&fakeTxRepo{},
		&fakeAccRepo{acc: &dom.Account{ID: accID, WorkspaceID: ws}},
		&fakeCatRepo{cat: &dom.Category{ID: catID, WorkspaceID: ws, Kind: dom.CategoryKindExpense}},
	)
	tx, err := svc.Create(context.Background(), CreateTransactionInput{
		WorkspaceID: ws,
		AccountID:   accID,
		CategoryID:  catID,
		AmountCents: 1500,
		Flow:        dom.FlowOut,
		OccurredAt:  time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if tx.AmountCents != 1500 || tx.Flow != dom.FlowOut {
		t.Fatalf("unexpected tx: %+v", tx)
	}
}

func TestTransactionService_Create_WrongCategoryKind(t *testing.T) {
	ws := uuid.New()
	accID := uuid.New()
	catID := uuid.New()
	svc := NewTransactionService(
		&fakeTxRepo{},
		&fakeAccRepo{acc: &dom.Account{ID: accID, WorkspaceID: ws}},
		&fakeCatRepo{cat: &dom.Category{ID: catID, WorkspaceID: ws, Kind: dom.CategoryKindIncome}},
	)
	_, err := svc.Create(context.Background(), CreateTransactionInput{
		WorkspaceID: ws,
		AccountID:   accID,
		CategoryID:  catID,
		AmountCents: 1500,
		Flow:        dom.FlowOut,
		OccurredAt:  time.Now().UTC(),
	})
	if !errors.Is(err, dom.ErrCategoryKindMismatch) {
		t.Fatalf("esperado ErrCategoryKindMismatch, obtido: %v", err)
	}
}
