package budget

import (
	"context"
	"testing"

	"github.com/google/uuid"
	domb "github.com/retechfin/retechfin-api/internal/domain/budget"
	doml "github.com/retechfin/retechfin-api/internal/domain/ledger"
)

type memBudgetRepo struct {
	created *domb.Budget
}

func (m *memBudgetRepo) Create(ctx context.Context, b *domb.Budget) error {
	m.created = b
	return nil
}
func (m *memBudgetRepo) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*domb.Budget, error) {
	return nil, domb.ErrNotFound
}
func (m *memBudgetRepo) Update(ctx context.Context, b *domb.Budget) error { return nil }
func (m *memBudgetRepo) Delete(ctx context.Context, workspaceID, id uuid.UUID) error { return nil }
func (m *memBudgetRepo) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]domb.Budget, int64, error) {
	return nil, 0, nil
}
func (m *memBudgetRepo) GetByCategoryPeriod(ctx context.Context, workspaceID, categoryID uuid.UUID, year, month int) (*domb.Budget, error) {
	return nil, domb.ErrNotFound
}
func (m *memBudgetRepo) ListByWorkspaceMonth(ctx context.Context, workspaceID uuid.UUID, year, month int) ([]domb.Budget, error) {
	return nil, nil
}

type catRepoStub struct {
	kind doml.CategoryKind
}

func (c *catRepoStub) Create(ctx context.Context, x *doml.Category) error { return nil }
func (c *catRepoStub) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*doml.Category, error) {
	return &doml.Category{Kind: c.kind, WorkspaceID: workspaceID, ID: id}, nil
}
func (c *catRepoStub) Update(ctx context.Context, x *doml.Category) error { return nil }
func (c *catRepoStub) Delete(ctx context.Context, workspaceID, id uuid.UUID) error { return nil }
func (c *catRepoStub) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]doml.Category, int64, error) {
	return nil, 0, nil
}

type txRepoStub struct{}

func (txRepoStub) Create(ctx context.Context, t *doml.Transaction) error { return nil }
func (txRepoStub) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*doml.Transaction, error) {
	return nil, nil
}
func (txRepoStub) Update(ctx context.Context, t *doml.Transaction) error { return nil }
func (txRepoStub) Delete(ctx context.Context, workspaceID, id uuid.UUID) error { return nil }
func (txRepoStub) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]doml.Transaction, int64, error) {
	return nil, 0, nil
}
func (txRepoStub) SumOutflowsByCategoryInMonth(ctx context.Context, workspaceID, categoryID uuid.UUID, year, month int) (int64, error) {
	return 0, nil
}

func TestService_Create_ExpenseOnly(t *testing.T) {
	ws := uuid.New()
	cat := uuid.New()
	repo := &memBudgetRepo{}
	svc := NewService(repo, &catRepoStub{kind: doml.CategoryKindExpense}, txRepoStub{})
	b, err := svc.Create(context.Background(), CreateBudgetInput{
		WorkspaceID: ws,
		CategoryID:  cat,
		Year:        2026,
		Month:       4,
		LimitCents:  100000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.LimitCents != 100000 || repo.created == nil {
		t.Fatal("budget não persistido")
	}
}

func TestService_Create_RejectsIncomeCategory(t *testing.T) {
	ws := uuid.New()
	cat := uuid.New()
	svc := NewService(&memBudgetRepo{}, &catRepoStub{kind: doml.CategoryKindIncome}, txRepoStub{})
	_, err := svc.Create(context.Background(), CreateBudgetInput{
		WorkspaceID: ws,
		CategoryID:  cat,
		Year:        2026,
		Month:       4,
		LimitCents:  100000,
	})
	if err == nil {
		t.Fatal("esperado erro de validação")
	}
}
