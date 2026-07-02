package finance

import "testing"

func TestValidEntryType(t *testing.T) {
	s := func(v string) *string { return &v }
	cases := []struct {
		name string
		kind Kind
		typ  *string
		want bool
	}{
		{"nulo permitido", KindCredit, nil, true},
		{"vazio permitido", KindDebit, s(""), true},
		{"receita válida", KindCredit, s("pj_contrato"), true},
		{"receita com categoria de despesa", KindCredit, s("mercado"), false},
		{"despesa válida", KindDebit, s("equipamentos"), true},
		{"fatura pai", KindDebit, s("cartao"), true},
		{"despesa com tipo de receita", KindDebit, s("salario"), false},
		{"fora do catálogo", KindDebit, s("miscelanea"), false},
	}
	for _, tc := range cases {
		if got := validEntryType(tc.kind, tc.typ); got != tc.want {
			t.Errorf("%s: validEntryType(%s, %v) = %v, quer %v", tc.name, tc.kind, tc.typ, got, tc.want)
		}
	}
}

func TestNormalizeExpenseCategory(t *testing.T) {
	s := func(v string) *string { return &v }
	cases := []struct {
		in   *string
		want string
	}{
		{nil, "outros"},
		{s(""), "outros"},
		{s("mercado"), "mercado"},
		{s("  Mercado "), "mercado"},
		{s("restaurantes"), "outros"},   // fora do catálogo → outros
		{s("cartao"), "outros"},         // reservado ao sistema → não aceita de fora
	}
	for _, tc := range cases {
		if got := NormalizeExpenseCategory(tc.in); *got != tc.want {
			t.Errorf("NormalizeExpenseCategory(%v) = %q, quer %q", tc.in, *got, tc.want)
		}
	}
}
