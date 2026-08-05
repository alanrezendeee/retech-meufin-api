package health

import "testing"

func TestNormalizeExamDate(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"2026-07-15", "2026-07-15"},
		{"15/07/2026", "2026-07-15"},
		{"15-07-2026", "2026-07-15"},
		{"15.07.2026", "2026-07-15"},
		{"15/07/26", "2026-07-15"},
		{"  2026-07-15  ", "2026-07-15"},
		{"", ""},
		{"ilegível", ""},
		{"07/2026", ""},
		// Laudos reais colam hora e fuso na data.
		{"28/01/2026 12:19 BRT", "2026-01-28"},
		{"17/01/26 11:39", "2026-01-17"},
		{"2026-01-14 08:34", "2026-01-14"},
	}
	for _, c := range cases {
		if got := normalizeExamDate(c.in); got != c.want {
			t.Errorf("normalizeExamDate(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "2026-01-05", "2026-01-06"); got != "2026-01-05" {
		t.Errorf("esperava a coleta como fallback, veio %q", got)
	}
	if got := firstNonEmpty("", "", ""); got != "" {
		t.Errorf("tudo vazio deve devolver vazio, veio %q", got)
	}
}

func TestOptStr(t *testing.T) {
	if optStr("  ") != nil {
		t.Error("optStr de branco deve ser nil")
	}
	if v := optStr(" mg/dL "); v == nil || *v != "mg/dL" {
		t.Errorf("optStr deve trimar: %v", v)
	}
}
