package budget

import (
	"context"
	"time"

	"github.com/google/uuid"
	domb "github.com/retechfin/retechfin-api/internal/domain/budget"
	doml "github.com/retechfin/retechfin-api/internal/domain/ledger"
)

type Service struct {
	budgetRepo  domb.Repository
	categoryRepo doml.CategoryRepository
	txRepo      doml.TransactionRepository
}

func NewService(
	budgetRepo domb.Repository,
	categoryRepo doml.CategoryRepository,
	txRepo doml.TransactionRepository,
) *Service {
	return &Service{
		budgetRepo:   budgetRepo,
		categoryRepo: categoryRepo,
		txRepo:       txRepo,
	}
}

type CreateBudgetInput struct {
	WorkspaceID uuid.UUID
	CategoryID  uuid.UUID
	Year        int
	Month       int
	LimitCents  int64
}

func (s *Service) Create(ctx context.Context, in CreateBudgetInput) (*domb.Budget, error) {
	cat, err := s.categoryRepo.GetByID(ctx, in.WorkspaceID, in.CategoryID)
	if err != nil {
		return nil, err
	}
	if cat.Kind != doml.CategoryKindExpense {
		return nil, &domb.ValidationError{Msg: "orçamento só pode ser definido para categoria de despesa (expense)"}
	}

	b := &domb.Budget{
		ID:          uuid.New(),
		WorkspaceID: in.WorkspaceID,
		CategoryID:  in.CategoryID,
		Year:        in.Year,
		Month:       in.Month,
		LimitCents:  in.LimitCents,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	if err := s.budgetRepo.Create(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

type UpdateBudgetInput struct {
	WorkspaceID uuid.UUID
	ID          uuid.UUID
	LimitCents  int64
}

func (s *Service) Update(ctx context.Context, in UpdateBudgetInput) (*domb.Budget, error) {
	b, err := s.budgetRepo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	b.LimitCents = in.LimitCents
	b.UpdatedAt = time.Now().UTC()
	if err := b.Validate(); err != nil {
		return nil, err
	}
	if err := s.budgetRepo.Update(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) Get(ctx context.Context, workspaceID, id uuid.UUID) (*domb.Budget, error) {
	return s.budgetRepo.GetByID(ctx, workspaceID, id)
}

func (s *Service) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.budgetRepo.Delete(ctx, workspaceID, id)
}

type ListBudgetsResult struct {
	Items []domb.Budget
	Total int64
}

func (s *Service) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) (*ListBudgetsResult, error) {
	items, total, err := s.budgetRepo.List(ctx, workspaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListBudgetsResult{Items: items, Total: total}, nil
}

// BudgetLine é o resultado da verificação de estouro para uma categoria orçada.
type BudgetLine struct {
	CategoryID  uuid.UUID `json:"category_id"`
	Year        int       `json:"year"`
	Month       int       `json:"month"`
	LimitCents  int64     `json:"limit_cents"`
	SpentCents  int64     `json:"spent_cents"`
	OverBudget  bool      `json:"over_budget"`
	RemainingCents int64  `json:"remaining_cents"`
}

type ValidateBudgetInput struct {
	WorkspaceID uuid.UUID
	Year        int
	Month       int
}

// ValidateBudget compara gastos reais (saídas) no mês com os limites definidos.
func (s *Service) ValidateBudget(ctx context.Context, in ValidateBudgetInput) ([]BudgetLine, error) {
	if in.Month < 1 || in.Month > 12 || in.Year < 2000 || in.Year > 2100 {
		return nil, &domb.ValidationError{Msg: "ano ou mês inválido"}
	}
	budgets, err := s.budgetRepo.ListByWorkspaceMonth(ctx, in.WorkspaceID, in.Year, in.Month)
	if err != nil {
		return nil, err
	}
	out := make([]BudgetLine, 0, len(budgets))
	for _, b := range budgets {
		spent, err := s.txRepo.SumOutflowsByCategoryInMonth(ctx, in.WorkspaceID, b.CategoryID, in.Year, in.Month)
		if err != nil {
			return nil, err
		}
		remaining := b.LimitCents - spent
		line := BudgetLine{
			CategoryID:     b.CategoryID,
			Year:           in.Year,
			Month:          in.Month,
			LimitCents:     b.LimitCents,
			SpentCents:     spent,
			OverBudget:     spent > b.LimitCents,
			RemainingCents: remaining,
		}
		out = append(out, line)
	}
	return out, nil
}
