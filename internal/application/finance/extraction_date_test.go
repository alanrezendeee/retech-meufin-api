package finance

import "testing"

func TestNormalizePurchaseDate(t *testing.T) {
	due := "2026-01-10" // fatura vencendo em janeiro/2026

	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"ISO passa direto", "2025-12-07", "2025-12-07"},
		{"DD/MM/YYYY", "07/12/2025", "2025-12-07"},
		{"DD-MM-YYYY", "07-12-2025", "2025-12-07"},
		{"DD.MM.YYYY", "07.12.2025", "2025-12-07"},
		{"DD/MM/YY", "07/12/25", "2025-12-07"},
		{"DD/MM mesmo mês do vencimento", "05/01", "2026-01-05"},
		{"DD/MM mês posterior ao vencimento = ano anterior", "07/12", "2025-12-07"},
		{"DD/MM com um dígito", "7/6", "2025-06-07"},
		{"vazio", "", ""},
		{"ilegível", "07 JUN", ""},
		{"mês inválido", "07/13", ""},
		{"dia inexistente no mês", "31/02", ""},
	}
	for _, tc := range cases {
		if got := normalizePurchaseDate(tc.raw, due); got != tc.want {
			t.Errorf("%s: normalizePurchaseDate(%q) = %q, quer %q", tc.name, tc.raw, got, tc.want)
		}
	}

	// vencimento da fatura ausente/inválido: DD/MM sem ano não é inferível
	if got := normalizePurchaseDate("07/12", ""); got != "" {
		t.Errorf("sem due date, DD/MM deve retornar vazio, veio %q", got)
	}
	// mas formatos com ano continuam funcionando
	if got := normalizePurchaseDate("07/12/2025", ""); got != "2025-12-07" {
		t.Errorf("com ano explícito não depende do due date, veio %q", got)
	}
}
