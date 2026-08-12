package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
)

type fakeShareLinkRepo struct {
	links []dom.ShareLink
	views int
}

func (r *fakeShareLinkRepo) Create(_ context.Context, l *dom.ShareLink) error {
	r.links = append(r.links, *l)
	return nil
}
func (r *fakeShareLinkRepo) List(_ context.Context, ws uuid.UUID) ([]dom.ShareLink, error) {
	var out []dom.ShareLink
	for _, l := range r.links {
		if l.WorkspaceID == ws {
			out = append(out, l)
		}
	}
	return out, nil
}
func (r *fakeShareLinkRepo) GetByID(_ context.Context, ws, id uuid.UUID) (*dom.ShareLink, error) {
	for i := range r.links {
		if r.links[i].ID == id && r.links[i].WorkspaceID == ws {
			l := r.links[i]
			return &l, nil
		}
	}
	return nil, dom.ErrNotFound
}
func (r *fakeShareLinkRepo) GetByToken(_ context.Context, token string) (*dom.ShareLink, error) {
	for i := range r.links {
		if r.links[i].Token == token {
			l := r.links[i]
			return &l, nil
		}
	}
	return nil, dom.ErrNotFound
}
func (r *fakeShareLinkRepo) Update(_ context.Context, l *dom.ShareLink) error {
	for i := range r.links {
		if r.links[i].ID == l.ID {
			r.links[i] = *l
			return nil
		}
	}
	return dom.ErrNotFound
}
func (r *fakeShareLinkRepo) CountActive(_ context.Context, ws uuid.UUID, now time.Time) (int64, error) {
	var n int64
	for _, l := range r.links {
		if l.WorkspaceID == ws && l.Active(now) {
			n++
		}
	}
	return n, nil
}
func (r *fakeShareLinkRepo) RegisterView(_ context.Context, id uuid.UUID, when time.Time) error {
	r.views++
	return nil
}

type fakeMemberRepo struct{ members []dom.FamilyMember }

func (r *fakeMemberRepo) Create(_ context.Context, f *dom.FamilyMember) error { return nil }
func (r *fakeMemberRepo) GetByID(_ context.Context, ws, id uuid.UUID) (*dom.FamilyMember, error) {
	for i := range r.members {
		if r.members[i].ID == id && r.members[i].WorkspaceID == ws {
			m := r.members[i]
			return &m, nil
		}
	}
	return nil, dom.ErrNotFound
}
func (r *fakeMemberRepo) Update(_ context.Context, f *dom.FamilyMember) error       { return nil }
func (r *fakeMemberRepo) SoftDelete(_ context.Context, ws, id uuid.UUID) error      { return nil }
func (r *fakeMemberRepo) List(_ context.Context, ws uuid.UUID, _ dom.FamilyMemberFilter, _, _ int) ([]dom.FamilyMember, int64, error) {
	return nil, 0, nil
}
func (r *fakeMemberRepo) ListWithBirthDate(_ context.Context, ws uuid.UUID) ([]dom.FamilyMember, error) {
	return nil, nil
}
func (r *fakeMemberRepo) UpdateAvatar(_ context.Context, ws, id uuid.UUID, key *string) error {
	return nil
}

func newShareTestService(t *testing.T) (*ShareLinkService, *fakeShareLinkRepo, uuid.UUID, uuid.UUID) {
	t.Helper()
	t.Setenv("ADMIN_BASE_URL", "https://app.meufin.test")
	ws, member := uuid.New(), uuid.New()
	repo := &fakeShareLinkRepo{}
	members := &fakeMemberRepo{members: []dom.FamilyMember{{ID: member, WorkspaceID: ws, FullName: "Alan Leite"}}}
	return NewShareLinkService(repo, members), repo, ws, member
}

func TestShareLink_CreateAndResolve(t *testing.T) {
	svc, repo, ws, member := newShareTestService(t)

	created, err := svc.Create(context.Background(), CreateShareLinkInput{
		WorkspaceID: ws, FamilyMemberID: member, ExpiresInHours: 48,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(created.Link.Token) != 64 {
		t.Errorf("token deveria ter 64 chars, tem %d", len(created.Link.Token))
	}
	if created.URL != "https://app.meufin.test/compartilhado/"+created.Link.Token {
		t.Errorf("url errada: %s", created.URL)
	}

	link, m, err := svc.Resolve(context.Background(), created.Link.Token)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if link.FamilyMemberID != member || m.FullName != "Alan Leite" {
		t.Error("resolve deveria devolver o membro do link")
	}
	if repo.views != 1 {
		t.Errorf("resolve deveria registrar 1 view, registrou %d", repo.views)
	}
}

func TestShareLink_GoneCases(t *testing.T) {
	svc, repo, ws, member := newShareTestService(t)

	created, _ := svc.Create(context.Background(), CreateShareLinkInput{
		WorkspaceID: ws, FamilyMemberID: member, ExpiresInHours: 1,
	})

	// Revogado -> gone
	if err := svc.Revoke(context.Background(), ws, created.Link.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	var gone *dom.ShareLinkGoneError
	if _, _, err := svc.Resolve(context.Background(), created.Link.Token); !errors.As(err, &gone) {
		t.Errorf("revogado deveria dar gone, veio %v", err)
	}

	// Expirado -> gone
	repo.links[0].RevokedAt = nil
	repo.links[0].ExpiresAt = time.Now().UTC().Add(-time.Minute)
	if _, _, err := svc.Resolve(context.Background(), created.Link.Token); !errors.As(err, &gone) {
		t.Errorf("expirado deveria dar gone, veio %v", err)
	}

	// Inexistente -> gone (mesmo erro, anti-enumeração)
	if _, _, err := svc.Resolve(context.Background(), "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"); !errors.As(err, &gone) {
		t.Errorf("inexistente deveria dar gone, veio %v", err)
	}
}

func TestShareLink_ValidationRules(t *testing.T) {
	svc, repo, ws, member := newShareTestService(t)

	// TTL fora da lista
	if _, err := svc.Create(context.Background(), CreateShareLinkInput{
		WorkspaceID: ws, FamilyMemberID: member, ExpiresInHours: 3,
	}); err == nil {
		t.Error("ttl=3 deveria ser rejeitado")
	}

	// Limite de ativos
	now := time.Now().UTC()
	for i := 0; i < dom.MaxActiveShareLinks; i++ {
		repo.links = append(repo.links, dom.ShareLink{
			ID: uuid.New(), WorkspaceID: ws, FamilyMemberID: member,
			Token: uuid.NewString(), ExpiresAt: now.Add(time.Hour),
		})
	}
	if _, err := svc.Create(context.Background(), CreateShareLinkInput{
		WorkspaceID: ws, FamilyMemberID: member, ExpiresInHours: 24,
	}); err == nil {
		t.Error("acima do limite de ativos deveria ser rejeitado")
	}
}
