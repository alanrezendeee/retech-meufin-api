package finance

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ExpenseCategory é uma categoria de despesa do workspace (cadastro gerenciado:
// criar/renomear/arquivar; as padrão nascem de seed no primeiro uso). O slug é
// o valor gravado em financial_entries.type nas despesas.
type ExpenseCategory struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Slug        string
	Name        string
	// GroupSlug é o grupo canônico (curado): a dimensão estável dos
	// indicadores. Toda categoria pertence a exatamente um grupo.
	GroupSlug string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ExpenseGroups é o catálogo CURADO de grupos — nunca editável pelo usuário.
// É por ele que os indicadores agregam ("pra onde foi o dinheiro" estável).
var ExpenseGroups = map[string]string{
	"moradia":                "Moradia",
	"alimentacao":            "Alimentação",
	"transporte":             "Transporte",
	"saude":                  "Saúde",
	"educacao":               "Educação",
	"lazer":                  "Lazer & Cultura",
	"viagens":                "Viagens",
	"vestuario":              "Vestuário",
	"cuidados_pessoais":      "Cuidados Pessoais",
	"pets":                   "Pets",
	"presentes_doacoes":      "Presentes & Doações",
	"contas_servicos":        "Contas & Assinaturas",
	"seguros_protecao":       "Seguros & Proteção",
	"impostos_taxas":         "Impostos & Taxas",
	"dividas_financiamentos": "Dívidas & Financiamentos",
	"equipamentos_bens":      "Equipamentos & Bens",
	"trabalho_negocio":       "Trabalho & Negócio",
	"familia_dependentes":    "Família & Dependentes",
	"servicos_profissionais": "Serviços Profissionais",
	"investimentos":          "Investimentos & Aportes",
	"outros":                 "Outros",
}

// CartaoCategorySlug é reservado à fatura pai criada pelo sistema — não pode
// ser criado nem usado diretamente pelo usuário.
const CartaoCategorySlug = "cartao"

// FallbackCategorySlug recebe tudo que vem de fora do catálogo (ex.: LLM).
const FallbackCategorySlug = "outros"

var slugInvalidChars = regexp.MustCompile(`[^a-z0-9_]+`)

// SlugifyCategory deriva um slug estável a partir do nome exibido.
func SlugifyCategory(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a",
		"é", "e", "ê", "e", "í", "i",
		"ó", "o", "õ", "o", "ô", "o",
		"ú", "u", "ç", "c", " ", "_", "-", "_",
	)
	s = replacer.Replace(s)
	s = slugInvalidChars.ReplaceAllString(s, "")
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}

// Validate normaliza e valida a categoria.
func (c *ExpenseCategory) Validate() error {
	if c.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return &ValidationError{Msg: "nome da categoria é obrigatório"}
	}
	if len(c.Name) > 80 {
		return &ValidationError{Msg: "nome da categoria excede o tamanho máximo"}
	}
	if c.Slug == "" {
		c.Slug = SlugifyCategory(c.Name)
	}
	if c.Slug == "" {
		return &ValidationError{Msg: "não foi possível derivar o identificador da categoria"}
	}
	if c.Slug == CartaoCategorySlug {
		return &ValidationError{Msg: "'cartao' é reservado às faturas do sistema"}
	}
	if _, ok := ExpenseGroups[c.GroupSlug]; !ok {
		return &ValidationError{Msg: "grupo da categoria fora do catálogo"}
	}
	return nil
}

// DefaultExpenseCategories é o seed criado no primeiro uso de cada workspace.
func DefaultExpenseCategories() []ExpenseCategory {
	defaults := []struct{ slug, name, group string }{
		{"moradia", "Moradia", "moradia"},
		{"agua", "Água", "moradia"},
		{"luz", "Luz / Energia", "moradia"},
		{"gas", "Gás", "moradia"},
		{"condominio", "Condomínio", "moradia"},
		{"internet", "Internet", "contas_servicos"},
		{"celular", "Telefone / Celular", "contas_servicos"},
		{"streaming", "Streaming", "contas_servicos"},
		{"alimentacao", "Alimentação", "alimentacao"},
		{"mercado", "Mercado", "alimentacao"},
		{"saude", "Saúde", "saude"},
		{"transporte", "Transporte", "transporte"},
		{"educacao", "Educação", "educacao"},
		{"lazer", "Lazer", "lazer"},
		{"contas_fixas", "Contas fixas", "contas_servicos"},
		{"servicos", "Serviços", "contas_servicos"},
		{"impostos", "Impostos", "impostos_taxas"},
		{"equipamentos", "Equipamentos", "equipamentos_bens"},
		{"vestuario", "Vestuário", "vestuario"},
		{"viagens", "Viagens", "viagens"},
		{"pets", "Pets", "pets"},
		{"seguros", "Seguros", "seguros_protecao"},
		{"assinaturas", "Assinaturas", "contas_servicos"},
		{"financiamentos", "Financiamentos", "dividas_financiamentos"},
		{"presentes", "Presentes", "presentes_doacoes"},
		{"pensao", "Pensão alimentícia", "familia_dependentes"},
		{"servicos_profissionais", "Serviços profissionais", "servicos_profissionais"},
		{"outros", "Outros", "outros"},
	}
	out := make([]ExpenseCategory, len(defaults))
	for i, d := range defaults {
		out[i] = ExpenseCategory{Slug: d.slug, Name: d.name, GroupSlug: d.group, Active: true}
	}
	return out
}

// ExpenseCategoryRepository persiste categorias (workspace-scoped, soft-delete).
type ExpenseCategoryRepository interface {
	Create(ctx context.Context, c *ExpenseCategory) error
	CreateBatch(ctx context.Context, cs []*ExpenseCategory) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*ExpenseCategory, error)
	ExistsBySlug(ctx context.Context, workspaceID uuid.UUID, slug string) (bool, error)
	Update(ctx context.Context, c *ExpenseCategory) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID) ([]ExpenseCategory, error)
}
