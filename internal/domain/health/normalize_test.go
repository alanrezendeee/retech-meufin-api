package health

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"Glicose":             "glicose",
		"  GLICOSE  ":         "glicose",
		"Hemoglobina glicada": "hemoglobina glicada",
		"Ácido úrico":         "acido urico",
		"Vitamina D (25-OH)":  "vitamina d 25 oh",
		"TGO/AST":             "tgo ast",
		"Colesterol   HDL":    "colesterol hdl",
		"Hematócrito":         "hematocrito",
		"":                    "",
		"!!!":                 "",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, quer %q", in, got, want)
		}
	}
}

func TestSimilarity(t *testing.T) {
	if Similarity("ferritina", "ferritina") != 1 {
		t.Error("iguais devem dar 1")
	}
	if Similarity("ferritina", "feritina") < 0.8 {
		t.Error("erro de digitação deve ter alta similaridade")
	}
	if Similarity("glicose", "colesterol") > 0.5 {
		t.Error("marcadores distintos devem ter baixa similaridade")
	}
}
