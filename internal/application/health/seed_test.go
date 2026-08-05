package health

import (
	"testing"

	dom "github.com/retechfin/retechfin-api/internal/domain/health"
)

// TestSeedKeysUnique garante que nenhuma chave normalizada (nome canônico ou
// alias) se repete no catálogo base. Uma colisão faria o MatchExact do Resolve
// devolver o marcador errado — silenciosamente, e sempre o mesmo.
func TestSeedKeysUnique(t *testing.T) {
	owner := map[string]string{}
	for _, sd := range systemMarkerSeeds() {
		keys := append([]string{sd.name}, sd.aliases...)
		for _, k := range keys {
			norm := dom.Normalize(k)
			if norm == "" {
				t.Errorf("%s: chave %q normaliza para vazio", sd.name, k)
				continue
			}
			if prev, dup := owner[norm]; dup {
				t.Errorf("colisão em %q (normalizado %q): %s e %s", k, norm, prev, sd.name)
				continue
			}
			owner[norm] = sd.name
		}
	}
}

// TestSeedValidates garante que todo marcador do seed passa no Validate do
// domínio (categoria obrigatória, comparability válida etc.).
func TestSeedValidates(t *testing.T) {
	for _, sd := range systemMarkerSeeds() {
		unit := sd.unit
		m := &dom.Marker{
			Scope:         dom.ScopeSystem,
			CanonicalName: sd.name,
			Category:      sd.category,
			Comparability: sd.comparability,
			CanonicalUnit: &unit,
			Active:        true,
		}
		for _, a := range sd.aliases {
			m.Aliases = append(m.Aliases, dom.MarkerAlias{Scope: dom.ScopeSystem, Alias: a})
		}
		if err := m.Validate(); err != nil {
			t.Errorf("%s: %v", sd.name, err)
		}
	}
}
