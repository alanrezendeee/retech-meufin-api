package budget

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, b *Budget) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*Budget, error)
	Update(ctx context.Context, b *Budget) error
	Delete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]Budget, int64, error)
	GetByCategoryPeriod(ctx context.Context, workspaceID, categoryID uuid.UUID, year, month int) (*Budget, error)
	ListByWorkspaceMonth(ctx context.Context, workspaceID uuid.UUID, year, month int) ([]Budget, error)
}
