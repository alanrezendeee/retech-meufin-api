package finance

import "testing"

func TestNormalizePaymentMethod(t *testing.T) {
	cases := map[string]string{
		"":                  "",
		"credito":           "credito",
		"CRÉDITO":           "credito",
		"Cartão de Crédito": "credito",
		"CARTAO CREDITO":    "credito",
		"cartão de débito":  "debito",
		"DEBITO":            "debito",
		"PIX":               "pix",
		"pagamento via pix": "pix",
		"Dinheiro":          "dinheiro",
		"espécie":           "dinheiro",
		"vale alimentação":  "outros",
		"boleto":            "outros",
	}
	for in, want := range cases {
		if got := normalizePaymentMethod(in); got != want {
			t.Errorf("normalizePaymentMethod(%q) = %q, want %q", in, got, want)
		}
	}
}
