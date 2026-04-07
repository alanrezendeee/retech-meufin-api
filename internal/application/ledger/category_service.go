package ledger

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/ledger"
)

type CategoryService struct {
	repo dom.CategoryRepository
}

func NewCategoryService(repo dom.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

type CreateCategoryInput struct {
	WorkspaceID uuid.UUID
	Name        string
	Kind        dom.CategoryKind
	ParentID    *uuid.UUID
}

type UpdateCategoryInput struct {
	WorkspaceID uuid.UUID
	ID          uuid.UUID
	Name        string
	Kind        dom.CategoryKind
	ParentID    *uuid.UUID
}

func (s *CategoryService) Create(ctx context.Context, in CreateCategoryInput) (*dom.Category, error) {
	c := &dom.Category{
		ID:          uuid.New(),
		WorkspaceID: in.WorkspaceID,
		Name:        in.Name,
		Kind:        in.Kind,
		ParentID:    in.ParentID,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CategoryService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Category, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

func (s *CategoryService) Update(ctx context.Context, in UpdateCategoryInput) (*dom.Category, error) {
	c, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	c.Name = in.Name
	c.Kind = in.Kind
	c.ParentID = in.ParentID
	c.UpdatedAt = time.Now().UTC()
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CategoryService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.Delete(ctx, workspaceID, id)
}

type ListCategoriesResult struct {
	Items []dom.Category
	Total int64
}

func (s *CategoryService) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) (*ListCategoriesResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListCategoriesResult{Items: items, Total: total}, nil
}
