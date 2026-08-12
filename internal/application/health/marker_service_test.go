package health

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
)

// fakeRepo é um MarkerRepository em memória para testar dedup/resolução sem banco.
type fakeRepo struct {
	markers []dom.Marker
}

func (r *fakeRepo) visible(m dom.Marker, ws uuid.UUID) bool {
	if m.Scope == dom.ScopeSystem {
		return true
	}
	return m.WorkspaceID != nil && *m.WorkspaceID == ws
}

func (r *fakeRepo) Create(_ context.Context, m *dom.Marker) error {
	r.markers = append(r.markers, *m)
	return nil
}
func (r *fakeRepo) Update(_ context.Context, m *dom.Marker) error {
	for i := range r.markers {
		if r.markers[i].ID == m.ID {
			r.markers[i] = *m
			return nil
		}
	}
	return dom.ErrNotFound
}
func (r *fakeRepo) SoftDelete(_ context.Context, ws, id uuid.UUID) error {
	for i := range r.markers {
		if r.markers[i].ID == id {
			r.markers = append(r.markers[:i], r.markers[i+1:]...)
			return nil
		}
	}
	return dom.ErrNotFound
}
func (r *fakeRepo) GetByID(_ context.Context, ws, id uuid.UUID) (*dom.Marker, error) {
	for i := range r.markers {
		if r.markers[i].ID == id && r.visible(r.markers[i], ws) {
			m := r.markers[i]
			return &m, nil
		}
	}
	return nil, dom.ErrNotFound
}
func (r *fakeRepo) Search(_ context.Context, ws uuid.UUID, _, _ string, _, _ int) ([]dom.Marker, int64, error) {
	var out []dom.Marker
	for _, m := range r.markers {
		if r.visible(m, ws) {
			out = append(out, m)
		}
	}
	return out, int64(len(out)), nil
}
func (r *fakeRepo) MatchExact(_ context.Context, ws uuid.UUID, normalized string) (*dom.Marker, error) {
	for i := range r.markers {
		m := r.markers[i]
		if !r.visible(m, ws) {
			continue
		}
		if m.NormalizedKey == normalized {
			return &m, nil
		}
		for _, a := range m.Aliases {
			if a.NormalizedAlias == normalized {
				return &m, nil
			}
		}
	}
	return nil, dom.ErrNotFound
}
func (r *fakeRepo) Candidates(_ context.Context, ws uuid.UUID, _ int) ([]dom.Marker, error) {
	var out []dom.Marker
	for _, m := range r.markers {
		if r.visible(m, ws) {
			out = append(out, m)
		}
	}
	return out, nil
}
func (r *fakeRepo) UpsertSystem(_ context.Context, m *dom.Marker) (bool, error) {
	for _, ex := range r.markers {
		if ex.Scope == dom.ScopeSystem && ex.NormalizedKey == m.NormalizedKey {
			return false, nil
		}
	}
	r.markers = append(r.markers, *m)
	return true, nil
}

func seededService(t *testing.T) (*MarkerService, uuid.UUID) {
	t.Helper()
	repo := &fakeRepo{}
	svc := NewMarkerService(repo)
	if _, err := svc.SeedSystem(context.Background()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return svc, uuid.New()
}

func TestCreate_DuplicateByCanonicalName(t *testing.T) {
	svc, ws := seededService(t)
	// "Glicose" já existe no seed (system)
	_, err := svc.Create(context.Background(), CreateMarkerInput{
		WorkspaceID: ws, CanonicalName: "glicose", Category: "bioquimica",
	})
	var dup *dom.DuplicateError
	if !errors.As(err, &dup) {
		t.Fatalf("esperava DuplicateError, veio %v", err)
	}
	if dup.Existing.CanonicalName != "Glicose" {
		t.Errorf("sugestão errada: %q", dup.Existing.CanonicalName)
	}
}

func TestCreate_DuplicateByAliasCollision(t *testing.T) {
	svc, ws := seededService(t)
	// "TGO" é alias de AST (TGO) no seed
	_, err := svc.Create(context.Background(), CreateMarkerInput{
		WorkspaceID: ws, CanonicalName: "Transaminase X", Category: "hepatico",
		Aliases: []string{"TGO"},
	})
	var dup *dom.DuplicateError
	if !errors.As(err, &dup) {
		t.Fatalf("esperava DuplicateError por alias, veio %v", err)
	}
}

func TestCreate_NewTenantMarkerOK(t *testing.T) {
	svc, ws := seededService(t)
	m, err := svc.Create(context.Background(), CreateMarkerInput{
		WorkspaceID: ws, CanonicalName: "Exame Custom do Tenant", Category: "outros",
		Aliases: []string{"ECT"},
	})
	if err != nil {
		t.Fatalf("criar custom: %v", err)
	}
	if m.Scope != dom.ScopeSystem && m.Scope != dom.ScopeTenant {
		t.Error("scope inválido")
	}
	if m.Scope != dom.ScopeTenant {
		t.Errorf("esperava tenant, veio %s", m.Scope)
	}
}

func TestResolve_MatchedAmbiguousUnresolved(t *testing.T) {
	svc, ws := seededService(t)
	res, err := svc.Resolve(context.Background(), ws, []ResolveItemInput{
		{RawName: "TGP"},             // alias exato -> matched
		{RawName: "glicoze"},         // erro de digitação -> ambiguous
		{RawName: "xyzabcnaoexiste"}, // nada -> unresolved
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if res[0].Status != ResolveMatched || res[0].Matched == nil {
		t.Errorf("TGP deveria dar matched, veio %s", res[0].Status)
	}
	if res[1].Status != ResolveAmbiguous || len(res[1].Candidates) == 0 {
		t.Errorf("glicoze deveria dar ambiguous com candidatos, veio %s", res[1].Status)
	}
	if res[2].Status != ResolveUnresolved {
		t.Errorf("inexistente deveria dar unresolved, veio %s", res[2].Status)
	}
}

func TestResolve_VariantsAndTokenOverlap(t *testing.T) {
	svc, ws := seededService(t)
	res, err := svc.Resolve(context.Background(), ws, []ResolveItemInput{
		{RawName: "Transaminase Oxalacética (TGO)"},  // alias no parêntese -> matched
		{RawName: "TGO - AST"},                       // segmento com alias -> matched
		{RawName: "TGO/TGP"},                         // dois marcadores distintos -> ambiguous
		{RawName: "Dosagem de Transaminase Pirúvica"}, // tokens contidos -> candidato fuzzy
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if res[0].Status != ResolveMatched || res[0].Matched == nil || res[0].Matched.CanonicalName != "AST (TGO)" {
		t.Errorf("TGO por extenso deveria dar matched AST (TGO), veio %s", res[0].Status)
	}
	if res[1].Status != ResolveMatched || res[1].Matched == nil || res[1].Matched.CanonicalName != "AST (TGO)" {
		t.Errorf("TGO - AST deveria dar matched AST (TGO), veio %s", res[1].Status)
	}
	if res[2].Status != ResolveAmbiguous || len(res[2].Candidates) != 2 {
		t.Errorf("TGO/TGP deveria dar ambiguous com 2 candidatos, veio %s (%d)", res[2].Status, len(res[2].Candidates))
	}
	if res[3].Status != ResolveAmbiguous || len(res[3].Candidates) == 0 {
		t.Errorf("nome com tokens do alias deveria dar ambiguous, veio %s", res[3].Status)
	} else if res[3].Candidates[0].Marker.CanonicalName != "ALT (TGP)" {
		t.Errorf("melhor candidato deveria ser ALT (TGP), veio %q", res[3].Candidates[0].Marker.CanonicalName)
	}
}

func TestResolve_PercentualVsAbsoluto(t *testing.T) {
	svc, ws := seededService(t)
	res, err := svc.Resolve(context.Background(), ws, []ResolveItemInput{
		{RawName: percentualName("Segmentados %")}, // escala percentual
		{RawName: "Segmentados"},                   // escala absoluta
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if res[0].Status != ResolveMatched || res[0].Matched == nil || res[0].Matched.CanonicalName != "Neutrófilos (percentual)" {
		t.Errorf("Segmentados %% deveria casar com Neutrófilos (percentual), veio %s", res[0].Status)
	}
	if res[1].Status != ResolveMatched || res[1].Matched == nil || res[1].Matched.CanonicalName != "Neutrófilos" {
		t.Errorf("Segmentados deveria casar com Neutrófilos, veio %s", res[1].Status)
	}
	if res[0].Matched != nil && res[1].Matched != nil && res[0].Matched.ID == res[1].Matched.ID {
		t.Error("percentual e absoluto não podem resolver para o mesmo marcador")
	}
}

func TestApplyItemDerived_CatalogDefaultRefFallback(t *testing.T) {
	repo := &fakeRepo{}
	msvc := NewMarkerService(repo)
	if _, err := msvc.SeedSystem(context.Background()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	ws := uuid.New()
	res, err := msvc.Resolve(context.Background(), ws, []ResolveItemInput{{RawName: "VLDL"}})
	if err != nil || res[0].Status != ResolveMatched {
		t.Fatalf("resolver VLDL: %v (%v)", err, res[0].Status)
	}
	vldlID := res[0].Matched.ID

	svc := NewExamResultService(nil, repo, nil)
	item := &dom.ExamResultItem{WorkspaceID: ws, ResultValue: "14 mg/dL", MarkerID: &vldlID}
	svc.applyItemDerived(context.Background(), ws, "", item)

	if item.InterpretationComputed == nil || *item.InterpretationComputed != "normal" {
		t.Errorf("VLDL 14 deveria interpretar 'normal' pela curadoria (<30), veio %v", item.InterpretationComputed)
	}
	if item.ReferenceMin != nil || item.ReferenceMax != nil {
		t.Error("faixa do item deve permanecer nula (fidelidade ao laudo); só a interpretação usa a curadoria")
	}
}

func TestApplyItemDerived_RiskTierInterpretation(t *testing.T) {
	repo := &fakeRepo{}
	msvc := NewMarkerService(repo)
	if _, err := msvc.SeedSystem(context.Background()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	ws := uuid.New()
	res, err := msvc.Resolve(context.Background(), ws, []ResolveItemInput{{RawName: "LDL"}})
	if err != nil || res[0].Status != ResolveMatched {
		t.Fatalf("resolver LDL: %v (%v)", err, res[0].Status)
	}
	ldlID := res[0].Matched.ID
	svc := NewExamResultService(nil, repo, nil)

	cases := []struct {
		risk string
		want *string
	}{
		{"baixo", ptr("normal")}, // 78 < 130
		{"alto", ptr("high")},    // 78 > 70
		{"", nil},                // sem risco cadastrado: tiers não interpretam
	}
	for _, c := range cases {
		item := &dom.ExamResultItem{WorkspaceID: ws, ResultValue: "78 mg/dL", MarkerID: &ldlID}
		svc.applyItemDerived(context.Background(), ws, c.risk, item)
		switch {
		case c.want == nil && item.InterpretationComputed != nil:
			t.Errorf("risco %q: esperava sem interpretação, veio %q", c.risk, *item.InterpretationComputed)
		case c.want != nil && (item.InterpretationComputed == nil || *item.InterpretationComputed != *c.want):
			t.Errorf("risco %q: esperava %q, veio %v", c.risk, *c.want, item.InterpretationComputed)
		}
	}
}

func ptr(s string) *string { return &s }

func TestUpdate_SystemMarkerImmutable(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewMarkerService(repo)
	_, _ = svc.SeedSystem(context.Background())
	ws := uuid.New()
	// pega um marcador system qualquer
	sys := repo.markers[0]
	_, err := svc.Update(context.Background(), UpdateMarkerInput{
		WorkspaceID: ws, ID: sys.ID, CanonicalName: "Alterado", Category: "outros",
	})
	if !errors.Is(err, dom.ErrImmutable) {
		t.Fatalf("esperava ErrImmutable ao editar system, veio %v", err)
	}
}
