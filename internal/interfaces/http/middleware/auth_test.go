package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	domsess "github.com/retechfin/retechfin-api/internal/domain/session"
)

// ---- infra de teste: chave RSA + JWKS em memória ----

type signer struct {
	key  *rsa.PrivateKey
	jwks *keyfunc.JWKS
}

func newSigner(t *testing.T) *signer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pub := key.PublicKey
	jwk := map[string]any{
		"keys": []map[string]any{{
			"kty": "RSA", "kid": "test", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}},
	}
	raw, _ := json.Marshal(jwk)
	jwks, err := keyfunc.NewJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	return &signer{key: key, jwks: jwks}
}

func (s *signer) token(t *testing.T, tenant string, exp time.Duration) string {
	t.Helper()
	claims := AuthClaims{
		UserID: uuid.NewString(), Email: "x@y.z", ApplicationID: "app-1", Roles: []string{"admin"},
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp))},
	}
	if tenant != "" {
		claims.TenantID = &tenant
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = "test"
	signed, err := tok.SignedString(s.key)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

type fakeResolver struct {
	byToken map[string]domsess.Session
	err     error
}

func (f *fakeResolver) Resolve(_ context.Context, raw string) (domsess.Session, error) {
	if f.err != nil {
		return domsess.Session{}, f.err
	}
	s, ok := f.byToken[raw]
	if !ok {
		return domsess.Session{}, domsess.ErrNotFound
	}
	return s, nil
}

const cookieName = "meufin_session"

func newApp(t *testing.T, s *signer, resolver SessionResolver) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireAuth(AuthOptions{
		JWKS: s.jwks, ApplicationID: "app-1", Sessions: resolver,
		Cookie:         SessionCookie{Name: cookieName, Secure: true},
		AllowedOrigins: []string{"https://admin.meufin.app"},
	}))
	handler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"via": c.GetString(CtxAuthVia), "ws": c.MustGet(CtxWorkspaceID).(uuid.UUID).String()})
	}
	r.GET("/x", handler)
	r.POST("/x", handler)
	return r
}

func do(r *gin.Engine, method string, mutate func(*http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "https://api.meufin.app/x", nil)
	req.Host = "api.meufin.app"
	if mutate != nil {
		mutate(req)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---- Bearer (comportamento legado preservado) ----

func TestBearerValido(t *testing.T) {
	s := newSigner(t)
	tenant := uuid.NewString()
	w := do(newApp(t, s, nil), http.MethodGet, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+s.token(t, tenant, time.Hour))
	})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"via":"bearer"`) || !strings.Contains(w.Body.String(), tenant) {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestBearerExpiradoOuSemTenant(t *testing.T) {
	s := newSigner(t)
	app := newApp(t, s, nil)
	if w := do(app, http.MethodGet, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+s.token(t, uuid.NewString(), -time.Minute))
	}); w.Code != http.StatusUnauthorized {
		t.Fatalf("expirado code=%d", w.Code)
	}
	if w := do(app, http.MethodGet, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+s.token(t, "", time.Hour))
	}); w.Code != http.StatusForbidden {
		t.Fatalf("sem tenant code=%d", w.Code)
	}
	if w := do(app, http.MethodGet, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("sem credencial code=%d", w.Code)
	}
}

// ---- Cookie de sessão ----

func TestCookieSessaoValida(t *testing.T) {
	s := newSigner(t)
	tenant := uuid.NewString()
	res := &fakeResolver{byToken: map[string]domsess.Session{
		"tok": {ID: "sid", Tokens: domsess.Tokens{Access: s.token(t, tenant, time.Hour)}},
	}}
	w := do(newApp(t, s, res), http.MethodGet, func(r *http.Request) {
		r.AddCookie(&http.Cookie{Name: cookieName, Value: "tok"})
	})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"via":"cookie"`) {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestCookieSessaoDesconhecidaLimpaCookie(t *testing.T) {
	s := newSigner(t)
	w := do(newApp(t, s, &fakeResolver{byToken: map[string]domsess.Session{}}), http.MethodGet, func(r *http.Request) {
		r.AddCookie(&http.Cookie{Name: cookieName, Value: "nao-existe"})
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", w.Code)
	}
	sc := w.Header().Get("Set-Cookie")
	if !strings.Contains(sc, cookieName+"=;") || !strings.Contains(sc, "Max-Age=0") || !strings.Contains(sc, "HttpOnly") {
		t.Fatalf("cookie não foi limpo: %q", sc)
	}
}

func TestCookieAuthIndisponivelDa503SemLimparCookie(t *testing.T) {
	s := newSigner(t)
	w := do(newApp(t, s, &fakeResolver{err: domsess.ErrAuthUnavailable}), http.MethodGet, func(r *http.Request) {
		r.AddCookie(&http.Cookie{Name: cookieName, Value: "tok"})
	})
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("code=%d", w.Code)
	}
	if w.Header().Get("Set-Cookie") != "" {
		t.Fatal("não deve limpar cookie quando o auth está fora do ar")
	}
}

func TestCookieIgnoradoQuandoGatewayDesabilitado(t *testing.T) {
	s := newSigner(t)
	w := do(newApp(t, s, nil), http.MethodGet, func(r *http.Request) {
		r.AddCookie(&http.Cookie{Name: cookieName, Value: "tok"})
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", w.Code)
	}
}

func TestBearerTemPrecedenciaSobreCookie(t *testing.T) {
	s := newSigner(t)
	res := &fakeResolver{err: errors.New("não devia ser chamado")}
	w := do(newApp(t, s, res), http.MethodGet, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+s.token(t, uuid.NewString(), time.Hour))
		r.AddCookie(&http.Cookie{Name: cookieName, Value: "tok"})
	})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"via":"bearer"`) {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

// ---- CSRF (só para requisições autenticadas por cookie) ----

func TestCSRFPostCrossSiteRecusado(t *testing.T) {
	s := newSigner(t)
	res := &fakeResolver{byToken: map[string]domsess.Session{
		"tok": {ID: "sid", Tokens: domsess.Tokens{Access: s.token(t, uuid.NewString(), time.Hour)}},
	}}
	app := newApp(t, s, res)
	withCookie := func(r *http.Request) { r.AddCookie(&http.Cookie{Name: cookieName, Value: "tok"}) }

	cases := []struct {
		name   string
		mutate func(*http.Request)
		want   int
	}{
		{"Sec-Fetch-Site cross-site", func(r *http.Request) { withCookie(r); r.Header.Set("Sec-Fetch-Site", "cross-site") }, http.StatusForbidden},
		{"Origin estranha", func(r *http.Request) { withCookie(r); r.Header.Set("Origin", "https://evil.example") }, http.StatusForbidden},
		{"Origin null", func(r *http.Request) { withCookie(r); r.Header.Set("Origin", "null") }, http.StatusForbidden},
		{"Origin do admin (CORS)", func(r *http.Request) { withCookie(r); r.Header.Set("Origin", "https://admin.meufin.app") }, http.StatusOK},
		{"Origin da própria API (same-origin via proxy)", func(r *http.Request) { withCookie(r); r.Header.Set("Origin", "https://api.meufin.app") }, http.StatusOK},
		{"Sec-Fetch-Site same-origin", func(r *http.Request) { withCookie(r); r.Header.Set("Sec-Fetch-Site", "same-origin") }, http.StatusOK},
		{"sem headers (cliente não-browser)", withCookie, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if w := do(app, http.MethodPost, tc.mutate); w.Code != tc.want {
				t.Fatalf("code=%d want=%d body=%s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}

func TestCSRFGetCrossSiteNaoBloqueia(t *testing.T) {
	s := newSigner(t)
	res := &fakeResolver{byToken: map[string]domsess.Session{
		"tok": {ID: "sid", Tokens: domsess.Tokens{Access: s.token(t, uuid.NewString(), time.Hour)}},
	}}
	w := do(newApp(t, s, res), http.MethodGet, func(r *http.Request) {
		r.AddCookie(&http.Cookie{Name: cookieName, Value: "tok"})
		r.Header.Set("Sec-Fetch-Site", "cross-site")
	})
	if w.Code != http.StatusOK {
		t.Fatalf("GET não muda estado; code=%d", w.Code)
	}
}

func TestCSRFNaoSeAplicaAoBearer(t *testing.T) {
	s := newSigner(t)
	w := do(newApp(t, s, nil), http.MethodPost, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+s.token(t, uuid.NewString(), time.Hour))
		r.Header.Set("Origin", "https://evil.example")
	})
	if w.Code != http.StatusOK {
		t.Fatalf("Bearer não é vulnerável a CSRF; code=%d", w.Code)
	}
}

func TestSessionCookieAtributos(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	SessionCookie{Name: "s", Secure: true, MaxAge: 3600}.Set(c, "abc")
	sc := w.Header().Get("Set-Cookie")
	for _, want := range []string{"s=abc", "Path=/", "Max-Age=3600", "HttpOnly", "Secure", "SameSite=Lax"} {
		if !strings.Contains(sc, want) {
			t.Errorf("Set-Cookie sem %q: %s", want, sc)
		}
	}
}

func TestBearerDesabilitadoRecusaMesmoTokenValido(t *testing.T) {
	s := newSigner(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	res := &fakeResolver{byToken: map[string]domsess.Session{
		"tok": {ID: "sid", Tokens: domsess.Tokens{Access: s.token(t, uuid.NewString(), time.Hour)}},
	}}
	r.Use(RequireAuth(AuthOptions{
		JWKS: s.jwks, ApplicationID: "app-1", Sessions: res, BearerDisabled: true,
		Cookie: SessionCookie{Name: cookieName, Secure: true},
	}))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := do(r, http.MethodGet, func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+s.token(t, uuid.NewString(), time.Hour))
	})
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "Bearer desabilitada") {
		t.Fatalf("bearer devia ser recusado: code=%d body=%s", w.Code, w.Body.String())
	}
	// Cookie continua funcionando.
	if w := do(r, http.MethodGet, func(req *http.Request) {
		req.AddCookie(&http.Cookie{Name: cookieName, Value: "tok"})
	}); w.Code != http.StatusOK {
		t.Fatalf("cookie: code=%d", w.Code)
	}
}
