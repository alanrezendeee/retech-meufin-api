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
		{RawName: "TGP"},           // alias exato -> matched
		{RawName: "glicoze"},       // erro de digitação -> ambiguous
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
