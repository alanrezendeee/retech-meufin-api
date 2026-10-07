package handlers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	appsess "github.com/retechfin/retechfin-api/internal/application/session"
	domsess "github.com/retechfin/retechfin-api/internal/domain/session"
	"github.com/retechfin/retechfin-api/internal/infrastructure/authclient"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// Teste de integração do gateway: auth central falso (httptest) + serviço real
// + store em memória + middleware real. Cobre login → cookie → rota protegida
// → refresh automático → /me → logout.

// Credenciais FICTÍCIAS aceitas pelo auth falso deste teste. Não são segredo:
// existem só para o httptest distinguir login certo de errado.
const (
	fixtureEmail    = "ana@meufin.app"
	fixtureUserID   = "11111111-1111-1111-1111-111111111111"
	fixtureSecretOK = "fixture-ok-" + "nao-e-senha-real"
	fixtureSecretKO = "fixture-ko-" + "nao-e-senha-real"
)

// loginBody monta o JSON de login a partir das fixtures.
func loginBody(email, secret string) string {
	b, _ := json.Marshal(map[string]string{"email": email, "password": secret})
	return string(b)
}

type fakeAuthServer struct {
	key      *rsa.PrivateKey
	jwks     *keyfunc.JWKS
	tenant   string
	mu       sync.Mutex
	refreshs int
	accessTT time.Duration // validade do access token emitido
}

func newFakeAuthServer(t *testing.T) (*fakeAuthServer, *httptest.Server) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pub := key.PublicKey
	raw, _ := json.Marshal(map[string]any{"keys": []map[string]any{{
		"kty": "RSA", "kid": "k1", "alg": "RS256", "use": "sig",
		"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}}})
	jwks, err := keyfunc.NewJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeAuthServer{key: key, jwks: jwks, tenant: uuid.NewString(), accessTT: time.Hour}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/authenticate", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["email"] != fixtureEmail || body["password"] != fixtureSecretOK || body["application_code"] != "meufin" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"Credenciais inválidas"}`))
			return
		}
		f.writeTokens(w)
	})
	mux.HandleFunc("POST /v1/refresh", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !strings.HasPrefix(body["refresh_token"], "refresh-") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		f.mu.Lock()
		f.refreshs++
		f.mu.Unlock()
		f.writeTokens(w)
	})
	mux.HandleFunc("GET /v1/me", func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if _, err := jwt.Parse(tok, f.jwks.Keyfunc, jwt.WithValidMethods([]string{"RS256"})); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"user":{"id":"u1","email":"ana@meufin.app","name":"Ana"},"abilities":[{"action":"manage","subject":"all"}]}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return f, srv
}

func (f *fakeAuthServer) writeTokens(w http.ResponseWriter) {
	f.mu.Lock()
	ttl := f.accessTT
	f.mu.Unlock()
	claims := middleware.AuthClaims{
		UserID: fixtureUserID, Email: fixtureEmail, Name: "Ana",
		ApplicationID: "app-meufin", TenantID: &f.tenant, Roles: []string{"admin"}, Perms: []string{"all:manage"},
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl))},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = "k1"
	signed, _ := tok.SignedString(f.key)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": signed, "refresh_token": "refresh-" + uuid.NewString(), "token_type": "Bearer",
		"expires_in": int(ttl.Seconds()),
		"user":       map[string]string{"id": fixtureUserID, "email": fixtureEmail, "name": "Ana"},
	})
}

// store em memória (o repositório GORM é trivial e fica fora deste teste).
type memStore struct {
	mu sync.Mutex
	m  map[string]domsess.Session
}

func (s *memStore) Create(_ context.Context, sess domsess.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[sess.ID] = sess
	return nil
}
func (s *memStore) Get(_ context.Context, id string) (domsess.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.m[id]
	if !ok {
		return domsess.Session{}, domsess.ErrNotFound
	}
	return sess, nil
}
func (s *memStore) Touch(_ context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.m[id]
	sess.LastSeenAt = at
	s.m[id] = sess
	return nil
}
func (s *memStore) UpdateTokens(_ context.Context, id string, t domsess.Tokens) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.m[id]
	sess.Tokens = t
	s.m[id] = sess
	return nil
}
func (s *memStore) Revoke(_ context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.m[id]; ok {
		sess.RevokedAt = &at
		s.m[id] = sess
	}
	return nil
}
func (s *memStore) RevokeAllForUser(context.Context, uuid.UUID, time.Time) (int64, error) {
	return 0, nil
}
func (s *memStore) PurgeExpired(context.Context, time.Time) (int64, error) { return 0, nil }

func newGatewayApp(t *testing.T) (*gin.Engine, *fakeAuthServer, *memStore) {
	t.Helper()
	fake, srv := newFakeAuthServer(t)
	t.Setenv("AUTH_API_BASE_URL", srv.URL)
	t.Setenv("AUTH_BOOTSTRAP_SECRET", "x")
	client := authclient.New(authclient.ConfigFromEnv())
	auth := authclient.NewPublicAuthenticator(client, "meufin")
	store := &memStore{m: map[string]domsess.Session{}}
	svc := appsess.NewService(auth, store, 24*time.Hour, nil)
	cookie := middleware.SessionCookie{Name: "meufin_session", Secure: true, MaxAge: 86400}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	requireAuth := middleware.RequireAuth(middleware.AuthOptions{
		JWKS: fake.jwks, ApplicationID: "app-meufin", Sessions: svc, Cookie: cookie,
		AllowedOrigins: []string{"https://admin.meufin.app"},
	})
	h := NewSessionHandler(svc, auth, cookie)
	a := r.Group("/api/v1/auth")
	a.POST("/login", h.Login)
	a.POST("/logout", h.Logout)
	a.GET("/me", requireAuth, h.Me)
	v1 := r.Group("/api/v1", requireAuth)
	v1.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"ws": c.MustGet(middleware.CtxWorkspaceID).(uuid.UUID).String()})
	})
	v1.POST("/ping", func(c *gin.Context) { c.Status(204) })
	return r, fake, store
}

func cookieFrom(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func call(r *gin.Engine, method, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	var rd *strings.Reader
	if body != "" {
		rd = strings.NewReader(body)
	} else {
		rd = strings.NewReader("")
	}
	req := httptest.NewRequest(method, "https://admin.meufin.app"+path, rd)
	req.Host = "admin.meufin.app"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://admin.meufin.app")
	for _, c := range cookies {
		if c != nil {
			req.AddCookie(c)
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGatewayFluxoCompleto(t *testing.T) {
	r, fake, store := newGatewayApp(t)

	// Login errado: 401, sem cookie.
	w := call(r, http.MethodPost, "/api/v1/auth/login", loginBody(fixtureEmail, fixtureSecretKO))
	if w.Code != http.StatusUnauthorized || cookieFrom(w, "meufin_session") != nil {
		t.Fatalf("login errado: code=%d cookies=%v", w.Code, w.Result().Cookies())
	}

	// Login certo: 204 + cookie HttpOnly/Secure/Lax, sem JWT no corpo.
	w = call(r, http.MethodPost, "/api/v1/auth/login", loginBody(fixtureEmail, fixtureSecretOK))
	if w.Code != http.StatusNoContent {
		t.Fatalf("login: code=%d body=%s", w.Code, w.Body.String())
	}
	ck := cookieFrom(w, "meufin_session")
	if ck == nil || !ck.HttpOnly || !ck.Secure || ck.SameSite != http.SameSiteLaxMode || ck.Value == "" {
		t.Fatalf("cookie inválido: %+v", ck)
	}
	if strings.Contains(ck.Value, ".") || len(ck.Value) > 64 {
		t.Fatalf("cookie parece um JWT, devia ser opaco: %s", ck.Value)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("login não deve devolver tokens no corpo: %s", w.Body.String())
	}
	// Banco guarda só o hash e tokens (em memória aqui; cifrados no GORM).
	if _, ok := store.m[ck.Value]; ok {
		t.Fatal("store indexado pelo token em claro")
	}
	if _, ok := store.m[domsess.HashToken(ck.Value)]; !ok {
		t.Fatal("sessão não encontrada pelo hash")
	}

	// Rota protegida via cookie resolve workspace do tenant_id.
	w = call(r, http.MethodGet, "/api/v1/ping", "", ck)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), fake.tenant) {
		t.Fatalf("ping: code=%d body=%s", w.Code, w.Body.String())
	}

	// /me repassa o perfil do auth.
	w = call(r, http.MethodGet, "/api/v1/auth/me", "", ck)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"abilities"`) {
		t.Fatalf("me: code=%d body=%s", w.Code, w.Body.String())
	}

	// CSRF: POST cross-site com cookie → 403; mesma origem → ok.
	req := httptest.NewRequest(http.MethodPost, "https://admin.meufin.app/api/v1/ping", nil)
	req.Host = "admin.meufin.app"
	req.Header.Set("Origin", "https://evil.example")
	req.AddCookie(ck)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("csrf: code=%d", w.Code)
	}
	if w = call(r, http.MethodPost, "/api/v1/ping", "", ck); w.Code != http.StatusNoContent {
		t.Fatalf("post same-origin: code=%d", w.Code)
	}

	// Access token perto de expirar → refresh transparente no auth.
	id := domsess.HashToken(ck.Value)
	sess := store.m[id]
	sess.Tokens.AccessExpiresAt = time.Now().Add(5 * time.Second)
	store.m[id] = sess
	w = call(r, http.MethodGet, "/api/v1/ping", "", ck)
	if w.Code != http.StatusOK || fake.refreshs != 1 {
		t.Fatalf("refresh: code=%d refreshs=%d", w.Code, fake.refreshs)
	}
	if !store.m[id].Tokens.AccessExpiresAt.After(time.Now().Add(30 * time.Minute)) {
		t.Fatal("tokens renovados não persistidos")
	}

	// Logout: 204, cookie limpo, sessão revogada; cookie antigo vira 401 + limpeza.
	w = call(r, http.MethodPost, "/api/v1/auth/logout", "", ck)
	if w.Code != http.StatusNoContent {
		t.Fatalf("logout: code=%d", w.Code)
	}
	if c := cookieFrom(w, "meufin_session"); c == nil || c.MaxAge >= 0 && c.Value != "" {
		t.Fatalf("logout não limpou o cookie: %+v", c)
	}
	w = call(r, http.MethodGet, "/api/v1/ping", "", ck)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("após logout: code=%d", w.Code)
	}
	if c := cookieFrom(w, "meufin_session"); c == nil || c.Value != "" {
		t.Fatalf("401 por sessão revogada deve limpar cookie: %+v", c)
	}

	// Logout sem cookie: idempotente.
	if w = call(r, http.MethodPost, "/api/v1/auth/logout", ""); w.Code != http.StatusNoContent {
		t.Fatalf("logout idempotente: code=%d", w.Code)
	}
}

// Sem Bearer: um JWT válido do auth no header é recusado — a única porta é o cookie.
func TestGatewayRecusaBearerMesmoValido(t *testing.T) {
	r, fake, _ := newGatewayApp(t)
	w := httptest.NewRecorder()
	fake.writeTokens(w)
	var tr struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &tr)

	req := httptest.NewRequest(http.MethodGet, "https://api.meufin.app/api/v1/ping", nil)
	req.Header.Set("Authorization", "Bearer "+tr.AccessToken)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "não aceita Authorization") {
		t.Fatalf("bearer: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestGatewayLoginValidacaoERateLimitShape(t *testing.T) {
	r, _, _ := newGatewayApp(t)
	for _, body := range []string{`{}`, loginBody("x", "123456"), loginBody("a@b.c", "123"), `nao-json`} {
		if w := call(r, http.MethodPost, "/api/v1/auth/login", body); w.Code != http.StatusBadRequest {
			t.Errorf("body %s: code=%d", body, w.Code)
		}
	}
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
