package middleware

import "testing"

func TestHasModuleAccess(t *testing.T) {
	cases := []struct {
		name   string
		codes  []string
		prefix string
		want   bool
	}{
		{"master acessa tudo", []string{"all:manage"}, "finance.", true},
		{"perm do módulo", []string{"finance.income:view"}, "finance.", true},
		{"manage do módulo", []string{"finance.payables:manage"}, "finance.", true},
		{"módulo errado", []string{"health.dashboard:view"}, "finance.", false},
		{"prefixo não confunde submódulo", []string{"financex.hack:view"}, "finance.", false},
		{"sem perms", nil, "finance.", false},
		{"vazio", []string{}, "health.", false},
	}
	for _, tc := range cases {
		if got := hasModuleAccess(tc.codes, tc.prefix); got != tc.want {
			t.Errorf("%s: hasModuleAccess(%v, %q) = %v, quer %v", tc.name, tc.codes, tc.prefix, got, tc.want)
		}
	}
}

func TestEnforcementModeFromEnv(t *testing.T) {
	cases := map[string]EnforcementMode{
		"off":    EnforcementOff,
		"STRICT": EnforcementStrict,
		"warn":   EnforcementWarn,
		"":       EnforcementWarn,
		"banana": EnforcementWarn,
	}
	for val, want := range cases {
		t.Setenv("PERMS_ENFORCEMENT", val)
		if got := EnforcementModeFromEnv(); got != want {
			t.Errorf("PERMS_ENFORCEMENT=%q → %s, quer %s", val, got, want)
		}
	}
}
