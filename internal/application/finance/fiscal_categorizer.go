package finance

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"github.com/retechfin/retechfin-api/internal/infrastructure/cache"
	"github.com/retechfin/retechfin-api/internal/infrastructure/extraction"
)

// MaxNewFiscalCategoriesPerReceipt limita quantas categorias NOVAS um único
// cupom pode propor/criar — backstop contra o LLM criar várias variações num só
// recibo (controle rígido anti-duplicação, camada 5).
const MaxNewFiscalCategoriesPerReceipt = 2

// ItemCategory é a categoria resolvida (e validada) de um item.
type ItemCategory struct {
	Slug  string // "" = não categorizado
	Name  string // nome de exibição (usado quando IsNew, para auto-cadastro)
	Group string // grupo global
	IsNew bool   // true = ainda não existe na tenant (sugestão a confirmar)
}

// FiscalCategorizer classifica descrições de itens em categorias da tenant,
// com reuse-first + validação server-side + teto de novas por cupom. É a
// camada que garante integridade: o LLM nunca grava direto.
type FiscalCategorizer struct {
	categories *ExpenseCategoryService // List() com seed/top-up das padrão
	llm        extraction.Categorizer
	cache      *cache.Cache // cache por (workspace + descrição normalizada)
}

func NewFiscalCategorizer(categories *ExpenseCategoryService, llm extraction.Categorizer, c *cache.Cache) *FiscalCategorizer {
	return &FiscalCategorizer{categories: categories, llm: llm, cache: c}
}

// Enabled indica se há classificador operacional.
func (fc *FiscalCategorizer) Enabled() bool {
	return fc != nil && fc.llm != nil && fc.llm.Enabled()
}

// Categorize devolve a categoria resolvida de cada descrição (alinhado por
// índice). Itens não classificados voltam com Slug vazio. Falha nunca bloqueia
// a ingestão — no pior caso os itens ficam sem categoria.
func (fc *FiscalCategorizer) Categorize(ctx context.Context, workspaceID uuid.UUID, descriptions []string) []ItemCategory {
	out := make([]ItemCategory, len(descriptions))
	if !fc.Enabled() || len(descriptions) == 0 {
		return out
	}

	cats, err := fc.categories.List(ctx, workspaceID)
	if err != nil {
		return out
	}
	slugSet := make(map[string]dom.ExpenseCategory, len(cats))
	nameToSlug := make(map[string]string, len(cats))
	catOpts := make([]extraction.CategoryOption, 0, len(cats))
	for _, c := range cats {
		if !c.Active {
			continue
		}
		slugSet[c.Slug] = c
		nameToSlug[dom.SlugifyCategory(c.Name)] = c.Slug
		catOpts = append(catOpts, extraction.CategoryOption{Slug: c.Slug, Name: c.Name, Group: c.GroupSlug})
	}

	// Cache por descrição: itens já classificados (existentes) não re-chamam o LLM.
	misses := make([]int, 0, len(descriptions))
	for i, d := range descriptions {
		if v, _ := fc.cache.Get(ctx, fc.cacheKey(workspaceID, d)); v != "" {
			slug, group := splitPipe(v)
			if c, ok := slugSet[slug]; ok {
				g := group
				if g == "" {
					g = c.GroupSlug
				}
				out[i] = ItemCategory{Slug: slug, Group: g}
				continue
			}
		}
		misses = append(misses, i)
	}
	if len(misses) == 0 {
		return out
	}

	descs := make([]string, len(misses))
	for j, idx := range misses {
		descs[j] = descriptions[idx]
	}
	groupOpts := make([]extraction.CategoryOption, 0, len(dom.ExpenseGroups))
	for slug, name := range dom.ExpenseGroups {
		groupOpts = append(groupOpts, extraction.CategoryOption{Slug: slug, Name: name})
	}

	res, err := fc.llm.Categorize(ctx, extraction.CategorizeInput{
		Descriptions: descs,
		Categories:   catOpts,
		Groups:       groupOpts,
	})
	if err != nil {
		return out // sem categoria; não bloqueia
	}

	byLocalIdx := make(map[int]extraction.CategorizedItem, len(res.Items))
	for _, it := range res.Items {
		if it.Index >= 0 && it.Index < len(misses) {
			byLocalIdx[it.Index] = it
		}
	}

	newCount := 0
	for j, origIdx := range misses {
		it, ok := byLocalIdx[j]
		if !ok {
			continue
		}
		resolved := resolveCategory(it, slugSet, nameToSlug, &newCount)
		out[origIdx] = resolved
		// Só cacheia categorias EXISTENTES resolvidas (novas ainda podem mudar).
		if resolved.Slug != "" && !resolved.IsNew {
			_ = fc.cache.Set(ctx, fc.cacheKey(workspaceID, descriptions[origIdx]), resolved.Slug+"|"+resolved.Group, 720*time.Hour)
		}
	}
	return out
}

// resolveCategory aplica as regras rígidas a UM resultado do LLM. Pura (sem I/O)
// para ser testável: existência por slug → por nome normalizado → nova (com
// grupo válido e dentro do teto) → fallback "outros".
func resolveCategory(
	it extraction.CategorizedItem,
	slugSet map[string]dom.ExpenseCategory,
	nameToSlug map[string]string,
	newCount *int,
) ItemCategory {
	fallback := ItemCategory{Slug: dom.FallbackCategorySlug, Group: "outros"}

	if !it.IsNew {
		if c, ok := slugSet[it.Slug]; ok {
			return ItemCategory{Slug: c.Slug, Group: c.GroupSlug}
		}
		if s, ok := nameToSlug[it.Slug]; ok {
			return ItemCategory{Slug: s, Group: slugSet[s].GroupSlug}
		}
		return fallback
	}

	name := it.NewName
	if name == "" {
		name = it.Slug
	}
	newSlug := dom.SlugifyCategory(name)
	if newSlug == "" {
		return fallback
	}
	// Dedup: já existe por slug ou por nome normalizado → reusa (não cria).
	if c, ok := slugSet[newSlug]; ok {
		return ItemCategory{Slug: c.Slug, Group: c.GroupSlug}
	}
	if s, ok := nameToSlug[newSlug]; ok {
		return ItemCategory{Slug: s, Group: slugSet[s].GroupSlug}
	}
	// Nova: grupo tem de ser um grupo global válido e dentro do teto do cupom.
	if _, ok := dom.ExpenseGroups[it.Group]; ok && *newCount < MaxNewFiscalCategoriesPerReceipt {
		*newCount++
		return ItemCategory{Slug: newSlug, Name: name, Group: it.Group, IsNew: true}
	}
	return fallback
}

func (fc *FiscalCategorizer) cacheKey(workspaceID uuid.UUID, description string) string {
	return "catv1:" + workspaceID.String() + ":" + dom.SlugifyCategory(description)
}

func splitPipe(s string) (string, string) {
	if i := strings.IndexByte(s, '|'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}
