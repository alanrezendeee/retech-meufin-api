package finance

import (
	"testing"

	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"github.com/retechfin/retechfin-api/internal/infrastructure/extraction"
)

func testCatSets() (map[string]dom.ExpenseCategory, map[string]string) {
	slugSet := map[string]dom.ExpenseCategory{
		"mercado": {Slug: "mercado", Name: "Mercado", GroupSlug: "alimentacao"},
		"outros":  {Slug: "outros", Name: "Outros", GroupSlug: "outros"},
	}
	nameToSlug := map[string]string{
		"mercado": "mercado",
		"outros":  "outros",
	}
	return slugSet, nameToSlug
}

func TestResolveCategory_ExistingSlug(t *testing.T) {
	slugSet, nameToSlug := testCatSets()
	n := 0
	got := resolveCategory(extraction.CategorizedItem{Slug: "mercado", Group: "alimentacao"}, slugSet, nameToSlug, &n)
	if got.Slug != "mercado" || got.IsNew || got.Group != "alimentacao" {
		t.Fatalf("existente deveria reusar mercado: %+v", got)
	}
}

func TestResolveCategory_NotNewButUnknown_FallsBackToOutros(t *testing.T) {
	slugSet, nameToSlug := testCatSets()
	n := 0
	got := resolveCategory(extraction.CategorizedItem{Slug: "inexistente", Group: "alimentacao"}, slugSet, nameToSlug, &n)
	if got.Slug != dom.FallbackCategorySlug || got.IsNew {
		t.Fatalf("slug inexistente (não-novo) deveria cair em outros: %+v", got)
	}
}

func TestResolveCategory_NewDedupsToExisting(t *testing.T) {
	slugSet, nameToSlug := testCatSets()
	n := 0
	// Propõe "Mercado" como nova, mas já existe pelo slug → reusa, não cria.
	got := resolveCategory(extraction.CategorizedItem{IsNew: true, NewName: "Mercado", Group: "alimentacao"}, slugSet, nameToSlug, &n)
	if got.IsNew || got.Slug != "mercado" {
		t.Fatalf("nova que colide com existente deveria reusar: %+v", got)
	}
	if n != 0 {
		t.Fatalf("dedup não deveria contar como nova; newCount=%d", n)
	}
}

func TestResolveCategory_NewWithValidGroup_Created(t *testing.T) {
	slugSet, nameToSlug := testCatSets()
	n := 0
	got := resolveCategory(extraction.CategorizedItem{IsNew: true, NewName: "Padaria", Group: "alimentacao"}, slugSet, nameToSlug, &n)
	if !got.IsNew || got.Slug != "padaria" || got.Group != "alimentacao" || got.Name != "Padaria" {
		t.Fatalf("nova válida deveria ser proposta: %+v", got)
	}
	if n != 1 {
		t.Fatalf("newCount deveria incrementar; got %d", n)
	}
}

func TestResolveCategory_NewInvalidGroup_FallsBack(t *testing.T) {
	slugSet, nameToSlug := testCatSets()
	n := 0
	got := resolveCategory(extraction.CategorizedItem{IsNew: true, NewName: "Padaria", Group: "grupo_que_nao_existe"}, slugSet, nameToSlug, &n)
	if got.IsNew || got.Slug != dom.FallbackCategorySlug {
		t.Fatalf("grupo inválido deveria cair em outros: %+v", got)
	}
}

func TestResolveCategory_CapOnNewCategories(t *testing.T) {
	slugSet, nameToSlug := testCatSets()
	n := MaxNewFiscalCategoriesPerReceipt // já no teto
	got := resolveCategory(extraction.CategorizedItem{IsNew: true, NewName: "Farmácia", Group: "saude"}, slugSet, nameToSlug, &n)
	if got.IsNew || got.Slug != dom.FallbackCategorySlug {
		t.Fatalf("acima do teto deveria cair em outros: %+v", got)
	}
}
