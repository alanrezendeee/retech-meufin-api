package finance

// A coluna `type` de financial_entries serve dois domínios conforme o kind:
//   - credit: tipos de receita — catálogo CURADO abaixo (semântica fiscal),
//     validado aqui e no CHECK do banco;
//   - debit: categorias de despesa — cadastro GERENCIADO por workspace
//     (finance_expense_categories, com grupo canônico curado); a validação é
//     dinâmica e acontece no serviço (consulta ao repositório). "cartao" é
//     reservado à fatura pai criada pelo sistema.

// IncomeTypes é o catálogo de tipos de receita (espelho do front).
var IncomeTypes = map[string]struct{}{
	"salario": {}, "pj_contrato": {}, "pro_labore": {}, "dividendos": {},
	"rendimento": {}, "aluguel": {}, "freela": {}, "ferias_13": {},
	"beneficio": {}, "reembolso": {}, "outro": {},
}

// validEntryType valida o que é estático: tipos de receita. Categorias de
// despesa passam aqui (validação dinâmica no serviço).
func validEntryType(kind Kind, t *string) bool {
	if t == nil || *t == "" {
		return true
	}
	if kind == KindCredit {
		_, ok := IncomeTypes[*t]
		return ok
	}
	return true
}
