package ledger

import (
	"context"

	"github.com/google/uuid"
)

type AccountRepository interface {
	Create(ctx context.Context, a *Account) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*Account, error)
	Update(ctx context.Context, a *Account) error
	Delete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]Account, int64, error)
}

type CategoryRepository interface {
	Create(ctx context.Context, c *Category) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*Category, error)
	Update(ctx context.Context, c *Category) error
	Delete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]Category, int64, error)
}

type TransactionRepository interface {
	Create(ctx context.Context, t *Transaction) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*Transaction, error)
	Update(ctx context.Context, t *Transaction) error
	Delete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]Transaction, int64, error)
	// SumOutflowsByCategoryInMonth soma lançamentos de saída (flow=out) no mês civil informado.
	SumOutflowsByCategoryInMonth(ctx context.Context, workspaceID, categoryID uuid.UUID, year, month int) (int64, error)
}
