package finance

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// ExpenseCategoryService gerencia as categorias de despesa do workspace.
// As padrão nascem por seed no primeiro List (lazy) — todo workspace começa
// com o conjunto base sem precisar de passo manual.
type ExpenseCategoryService struct {
	repo dom.ExpenseCategoryRepository
}

func NewExpenseCategoryService(repo dom.ExpenseCategoryRepository) *ExpenseCategoryService {
	return &ExpenseCategoryService{repo: repo}
}

// List retorna as categorias do workspace, completando as padrão que
// faltarem (top-up): workspace novo ganha o seed inteiro; workspace antigo
// ganha as padrão adicionadas depois. Excluídas pelo usuário NÃO voltam —
// o slug soft-deletado ainda ocupa a unique e o insert é ignorado.
func (s *ExpenseCategoryService) List(ctx context.Context, workspaceID uuid.UUID) ([]dom.ExpenseCategory, error) {
	cats, err := s.repo.List(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	existing := make(map[string]struct{}, len(cats))
	for i := range cats {
		existing[cats[i].Slug] = struct{}{}
	}

	now := time.Now().UTC()
	var missing []*dom.ExpenseCategory
	for _, c := range dom.DefaultExpenseCategories() {
		if _, ok := existing[c.Slug]; ok {
			continue
		}
		cc := c
		cc.ID = uuid.New()
		cc.WorkspaceID = workspaceID
		cc.CreatedAt = now
		cc.UpdatedAt = now
		missing = append(missing, &cc)
	}
	if len(missing) == 0 {
		return cats, nil
	}
	if err := s.repo.CreateBatch(ctx, missing); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, workspaceID)
}

type CreateExpenseCategoryInput struct {
	WorkspaceID uuid.UUID
	Name        string
	GroupSlug   string
}

func (s *ExpenseCategoryService) Create(ctx context.Context, in CreateExpenseCategoryInput) (*dom.ExpenseCategory, error) {
	now := time.Now().UTC()
	cat := &dom.ExpenseCategory{
		ID:          uuid.New(),
		WorkspaceID: in.WorkspaceID,
		Name:        in.Name,
		GroupSlug:   strings.TrimSpace(in.GroupSlug),
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := cat.Validate(); err != nil {
		return nil, err
	}
	exists, err := s.repo.ExistsBySlug(ctx, in.WorkspaceID, cat.Slug)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, &dom.ValidationError{Msg: "já existe uma categoria com esse nome"}
	}
	if err := s.repo.Create(ctx, cat); err != nil {
		return nil, err
	}
	return cat, nil
}

type UpdateExpenseCategoryInput struct {
	WorkspaceID uuid.UUID
	ID          uuid.UUID
	Name        string
	GroupSlug   string
	Active      *bool
}

// Update renomeia/move de grupo/ativa-arquiva. O slug NÃO muda (histórico de
// lançamentos referencia por slug) — renomear só troca o rótulo exibido.
func (s *ExpenseCategoryService) Update(ctx context.Context, in UpdateExpenseCategoryInput) (*dom.ExpenseCategory, error) {
	cat, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	cat.Name = in.Name
	cat.GroupSlug = strings.TrimSpace(in.GroupSlug)
	if in.Active != nil {
		cat.Active = *in.Active
	}
	cat.UpdatedAt = time.Now().UTC()
	if err := cat.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func (s *ExpenseCategoryService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	cat, err := s.repo.GetByID(ctx, workspaceID, id)
	if err != nil {
		return err
	}
	if cat.Slug == dom.FallbackCategorySlug {
		return &dom.ValidationError{Msg: "a categoria 'Outros' não pode ser removida (é o destino padrão)"}
	}
	return s.repo.SoftDelete(ctx, workspaceID, id)
}
