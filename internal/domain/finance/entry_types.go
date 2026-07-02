package finance

import "strings"

// A coluna `type` de financial_entries serve dois domínios conforme o kind:
// tipos de receita (credit) e categorias de despesa (debit, incluindo o
// especial "cartao" da fatura pai). Ambos são catálogos CURADOS — a semântica
// dos indicadores depende disso — validados aqui (erro amigável) e no banco
// (CHECK da migration 000008, defesa em profundidade).

// IncomeTypes é o catálogo de tipos de receita (espelho do front).
var IncomeTypes = map[string]struct{}{
	"salario": {}, "pj_contrato": {}, "pro_labore": {}, "dividendos": {},
	"rendimento": {}, "aluguel": {}, "freela": {}, "ferias_13": {},
	"beneficio": {}, "reembolso": {}, "outro": {},
}

// ExpenseCategories é o catálogo de categorias de despesa (espelho do front).
// "cartao" é reservado à fatura pai criada pelo sistema.
var ExpenseCategories = map[string]struct{}{
	"cartao": {}, "moradia": {}, "alimentacao": {}, "mercado": {}, "saude": {},
	"transporte": {}, "educacao": {}, "lazer": {}, "contas_fixas": {},
	"servicos": {}, "impostos": {}, "equipamentos": {}, "outros": {},
}

// validEntryType valida o type conforme o kind. Type vazio/nulo é permitido.
func validEntryType(kind Kind, t *string) bool {
	if t == nil || *t == "" {
		return true
	}
	switch kind {
	case KindCredit:
		_, ok := IncomeTypes[*t]
		return ok
	case KindDebit:
		_, ok := ExpenseCategories[*t]
		return ok
	}
	return false
}

// NormalizeExpenseCategory mapeia uma categoria vinda de fora do sistema
// (ex.: sugestão da LLM na extração de fatura) para o catálogo curado.
// Desconhecida ou vazia vira "outros" — nunca deixamos valor fora do catálogo
// entrar no banco.
func NormalizeExpenseCategory(s *string) *string {
	fallback := "outros"
	if s == nil {
		return &fallback
	}
	norm := strings.ToLower(strings.TrimSpace(*s))
	if norm == "" {
		return &fallback
	}
	if _, ok := ExpenseCategories[norm]; ok && norm != "cartao" {
		return &norm
	}
	return &fallback
}
