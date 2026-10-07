package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/session"
)

type fakeAuth struct {
	authErr    error
	refreshErr error
	refreshed  int
	tokens     dom.Tokens
}

func (f *fakeAuth) Authenticate(_ context.Context, email, _ string) (dom.Authenticated, error) {
	if f.authErr != nil {
		return dom.Authenticated{}, f.authErr
	}
	return dom.Authenticated{
		User:   dom.UserInfo{ID: uuid.New(), Email: email, Name: "Alguém"},
		Tokens: f.tokens,
	}, nil
}

func (f *fakeAuth) Refresh(_ context.Context, _ string) (dom.Tokens, error) {
	f.refreshed++
	if f.refreshErr != nil {
		return dom.Tokens{}, f.refreshErr
	}
	return dom.Tokens{Access: "access-novo", Refresh: "refresh-novo", AccessExpiresAt: time.Now().Add(time.Hour)}, nil
}

type memStore struct {
	sessions map[string]dom.Session
	touched  int
}

func newMemStore() *memStore { return &memStore{sessions: map[string]dom.Session{}} }

func (m *memStore) Create(_ context.Context, s dom.Session) error { m.sessions[s.ID] = s; return nil }
func (m *memStore) Get(_ context.Context, id string) (dom.Session, error) {
	s, ok := m.sessions[id]
	if !ok {
		return dom.Session{}, dom.ErrNotFound
	}
	return s, nil
}
func (m *memStore) Touch(_ context.Context, id string, at time.Time) error {
	s := m.sessions[id]
	s.LastSeenAt = at
	m.sessions[id] = s
	m.touched++
	return nil
}
func (m *memStore) UpdateTokens(_ context.Context, id string, t dom.Tokens) error {
	s := m.sessions[id]
	s.Tokens = t
	m.sessions[id] = s
	return nil
}
func (m *memStore) Revoke(_ context.Context, id string, at time.Time) error {
	if s, ok := m.sessions[id]; ok {
		s.RevokedAt = &at
		m.sessions[id] = s
	}
	return nil
}
func (m *memStore) RevokeAllForUser(_ context.Context, userID uuid.UUID, at time.Time) (int64, error) {
	var n int64
	for id, s := range m.sessions {
		if s.User.ID == userID && s.RevokedAt == nil {
			s.RevokedAt = &at
			m.sessions[id] = s
			n++
		}
	}
	return n, nil
}
func (m *memStore) PurgeExpired(_ context.Context, before time.Time) (int64, error) {
	var n int64
	for id, s := range m.sessions {
		if s.ExpiresAt.Before(before) || (s.RevokedAt != nil && s.RevokedAt.Before(before)) {
			delete(m.sessions, id)
			n++
		}
	}
	return n, nil
}

func newSvc(t *testing.T, auth *fakeAuth, store *memStore) (*Service, *time.Time) {
	t.Helper()
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	svc := NewService(auth, store, 24*time.Hour, nil).WithClock(func() time.Time { return now })
	return svc, &now
}

func TestLoginCriaSessaoENaoPersisteTokenDoCookie(t *testing.T) {
	auth := &fakeAuth{tokens: dom.Tokens{Access: "a", Refresh: "r", AccessExpiresAt: time.Now().Add(time.Hour)}}
	store := newMemStore()
	svc, now := newSvc(t, auth, store)

	raw, sess, err := svc.Login(context.Background(), "x@y.z", "123456", ClientMeta{IP: "1.2.3.4"})
	if err != nil {
		t.Fatal(err)
	}
	if raw == "" || sess.ID != dom.HashToken(raw) {
		t.Fatalf("id da sessão deve ser o hash do token do cookie")
	}
	if _, ok := store.sessions[raw]; ok {
		t.Fatal("token em claro não pode ser a chave do store")
	}
	if !sess.ExpiresAt.Equal(now.Add(24 * time.Hour)) {
		t.Fatalf("expiração absoluta = %v", sess.ExpiresAt)
	}
}

func TestLoginPropagaCredenciaisInvalidas(t *testing.T) {
	svc, _ := newSvc(t, &fakeAuth{authErr: dom.ErrInvalidCredentials}, newMemStore())
	_, _, err := svc.Login(context.Background(), "x@y.z", "errada", ClientMeta{})
	if !errors.Is(err, dom.ErrInvalidCredentials) {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveSemRefreshQuandoAccessValido(t *testing.T) {
	auth := &fakeAuth{}
	store := newMemStore()
	svc, now := newSvc(t, auth, store)
	auth.tokens = dom.Tokens{Access: "a", Refresh: "r", AccessExpiresAt: now.Add(time.Hour)}
	raw, _, _ := svc.Login(context.Background(), "x@y.z", "123456", ClientMeta{})

	sess, err := svc.Resolve(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Tokens.Access != "a" || auth.refreshed != 0 {
		t.Fatalf("não devia renovar: access=%q refreshed=%d", sess.Tokens.Access, auth.refreshed)
	}
}

func TestResolveRenovaQuandoAccessPertoDeExpirar(t *testing.T) {
	auth := &fakeAuth{}
	store := newMemStore()
	svc, now := newSvc(t, auth, store)
	auth.tokens = dom.Tokens{Access: "a", Refresh: "r", AccessExpiresAt: now.Add(10 * time.Second)} // < skew
	raw, _, _ := svc.Login(context.Background(), "x@y.z", "123456", ClientMeta{})

	sess, err := svc.Resolve(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Tokens.Access != "access-novo" || auth.refreshed != 1 {
		t.Fatalf("devia renovar: access=%q refreshed=%d", sess.Tokens.Access, auth.refreshed)
	}
	if store.sessions[sess.ID].Tokens.Refresh != "refresh-novo" {
		t.Fatal("tokens novos não persistidos")
	}
}

func TestResolveRefreshRecusadoRevogaSessao(t *testing.T) {
	auth := &fakeAuth{refreshErr: dom.ErrRefreshRejected}
	store := newMemStore()
	svc, now := newSvc(t, auth, store)
	auth.tokens = dom.Tokens{Access: "a", Refresh: "r", AccessExpiresAt: now.Add(-time.Minute)}
	raw, sess, _ := svc.Login(context.Background(), "x@y.z", "123456", ClientMeta{})

	_, err := svc.Resolve(context.Background(), raw)
	if !errors.Is(err, dom.ErrRefreshRejected) {
		t.Fatalf("err = %v", err)
	}
	if store.sessions[sess.ID].RevokedAt == nil {
		t.Fatal("sessão devia estar revogada")
	}
	// Segunda tentativa: sessão revogada → ErrNotFound (cookie será limpo).
	if _, err := svc.Resolve(context.Background(), raw); !errors.Is(err, dom.ErrNotFound) {
		t.Fatalf("segunda tentativa err = %v", err)
	}
}

func TestResolveAuthForaDoArNaoRevoga(t *testing.T) {
	auth := &fakeAuth{refreshErr: dom.ErrAuthUnavailable}
	store := newMemStore()
	svc, now := newSvc(t, auth, store)
	auth.tokens = dom.Tokens{Access: "a", Refresh: "r", AccessExpiresAt: now.Add(-time.Minute)}
	raw, sess, _ := svc.Login(context.Background(), "x@y.z", "123456", ClientMeta{})

	_, err := svc.Resolve(context.Background(), raw)
	if !errors.Is(err, dom.ErrAuthUnavailable) {
		t.Fatalf("err = %v", err)
	}
	if store.sessions[sess.ID].RevokedAt != nil {
		t.Fatal("auth indisponível não pode derrubar a sessão")
	}
}

func TestResolveSessaoExpiradaOuRevogadaOuInexistente(t *testing.T) {
	auth := &fakeAuth{}
	store := newMemStore()
	svc, now := newSvc(t, auth, store)
	auth.tokens = dom.Tokens{Access: "a", Refresh: "r", AccessExpiresAt: now.Add(time.Hour)}
	raw, sess, _ := svc.Login(context.Background(), "x@y.z", "123456", ClientMeta{})

	if _, err := svc.Resolve(context.Background(), "nao-existe"); !errors.Is(err, dom.ErrNotFound) {
		t.Fatalf("inexistente err = %v", err)
	}
	if _, err := svc.Resolve(context.Background(), ""); !errors.Is(err, dom.ErrNotFound) {
		t.Fatalf("vazio err = %v", err)
	}
	if err := svc.Logout(context.Background(), raw); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Resolve(context.Background(), raw); !errors.Is(err, dom.ErrNotFound) {
		t.Fatalf("revogada err = %v", err)
	}

	// Expirada (relógio avança além do TTL).
	s := store.sessions[sess.ID]
	s.RevokedAt = nil
	store.sessions[sess.ID] = s
	*now = now.Add(25 * time.Hour)
	if _, err := svc.Resolve(context.Background(), raw); !errors.Is(err, dom.ErrNotFound) {
		t.Fatalf("expirada err = %v", err)
	}
}

func TestResolveTouchLimitadoPorIntervalo(t *testing.T) {
	auth := &fakeAuth{}
	store := newMemStore()
	svc, now := newSvc(t, auth, store)
	auth.tokens = dom.Tokens{Access: "a", Refresh: "r", AccessExpiresAt: now.Add(time.Hour)}
	raw, _, _ := svc.Login(context.Background(), "x@y.z", "123456", ClientMeta{})

	_, _ = svc.Resolve(context.Background(), raw)
	_, _ = svc.Resolve(context.Background(), raw)
	if store.touched != 0 {
		t.Fatalf("touch dentro do intervalo: %d", store.touched)
	}
	*now = now.Add(2 * time.Minute)
	_, _ = svc.Resolve(context.Background(), raw)
	if store.touched != 1 {
		t.Fatalf("touch após intervalo: %d", store.touched)
	}
}

func TestLogoutIdempotente(t *testing.T) {
	svc, _ := newSvc(t, &fakeAuth{}, newMemStore())
	if err := svc.Logout(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.Logout(context.Background(), "qualquer"); err != nil {
		t.Fatal(err)
	}
}

func TestRevokeAllForUserEPurge(t *testing.T) {
	auth := &fakeAuth{}
	store := newMemStore()
	svc, now := newSvc(t, auth, store)
	auth.tokens = dom.Tokens{Access: "a", Refresh: "r", AccessExpiresAt: now.Add(time.Hour)}
	_, s1, _ := svc.Login(context.Background(), "x@y.z", "123456", ClientMeta{})
	// Mesmo usuário em outro browser: força o mesmo ID de usuário.
	s2 := s1
	s2.ID = "outra"
	store.sessions["outra"] = s2

	n, err := svc.RevokeAllForUser(context.Background(), s1.User.ID)
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	// Purge com carência de 7 dias: nada ainda.
	if n, _ := svc.PurgeExpired(context.Background(), 7*24*time.Hour); n != 0 {
		t.Fatalf("purge precoce removeu %d", n)
	}
	*now = now.Add(8 * 24 * time.Hour)
	if n, _ := svc.PurgeExpired(context.Background(), 7*24*time.Hour); n != 2 {
		t.Fatalf("purge devia remover 2, removeu %d", n)
	}
}
