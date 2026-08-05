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
	}
	for _, c := range cases {
		if got := normalizeExamDate(c.in); got != c.want {
			t.Errorf("normalizeExamDate(%q) = %q, want %q", c.in, got, c.want)
		}
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
