package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/theretechlabs/retech-authkit/cookie"
	"github.com/theretechlabs/retech-authkit/jwtverify"
	"github.com/theretechlabs/retech-authkit/session"

	domsess "github.com/retechfin/retechfin-api/internal/domain/session"
)

// signer emite tokens como o retech-auth-api e expõe o verificador correspondente.
type signer struct {
	key      *rsa.PrivateKey
	verifier *jwtverify.Verifier
}

func newSigner(t *testing.T) *signer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	v := jwtverify.NewWithKeyfunc(func(*jwt.Token) (any, error) { return &key.PublicKey, nil },
		jwtverify.Options{Issuer: "retech-auth-api", Audience: "meufin", RequireTyp: jwtverify.TypAccess, RequireTenant: true})
	return &signer{key: key, verifier: v}
}

func (s *signer) token(t *testing.T, tenant string, exp time.Duration) string {
	t.Helper()
	claims := AuthClaims{
		UserID: uuid.New(), Email: "x@y.z", ApplicationID: uuid.New(), Roles: []string{"admin"}, Typ: jwtverify.TypAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "retech-auth-api", Audience: jwt.ClaimStrings{"meufin"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)), IssuedAt: jwt.NewNumericDate(time.Now()),
		},
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

// fakeResolver simula o serviço de sessão: resolve o cookie e verifica o JWT
// como o kit faria.
type fakeResolver struct {
	byToken  map[string]domsess.Session
	verifier *jwtverify.Verifier
	err      error
}

func (f *fakeResolver) Resolve(_ context.Context, raw string) (session.Resolved, error) {
	if f.err != nil {
		return session.Resolved{}, f.err
	}
	s, ok := f.byToken[raw]
	if !ok {
		return session.Resolved{}, domsess.ErrNotFound
	}
	claims, err := f.verifier.Verify(s.Tokens.Access)
	if err != nil {
		return session.Resolved{}, domsess.ErrTokenInvalid
	}
	return session.Resolved{Session: s, Claims: claims}, nil
}

const cookieName = "meufin_session"

func newApp(t *testing.T, resolver SessionResolver) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireAuth(AuthOptions{
		Sessions:       resolver,
		Cookie:         cookie.Config{Name: cookieName, Secure: true, TTL: time.Hour},
		AllowedOrigins: []string{"https://admin.meufin.app"},
	}))
	handler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ws": c.MustGet(CtxWorkspaceID).(uuid.UUID).String(), "sid": c.GetString(CtxSessionID)})
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

func withSession(t *testing.T, s *signer, tenant string, exp time.Duration) *fakeResolver {
	t.Helper()
	return &fakeResolver{verifier: s.verifier, byToken: map[string]domsess.Session{
		"tok": {ID: "sid", Tokens: domsess.Tokens{Access: s.token(t, tenant, exp)}},
	}}
}

func withCookie(r *http.Request) { r.AddCookie(&http.Cookie{Name: cookieName, Value: "tok"}) }

// ---- Cookie de sessão: única autenticação ----

func TestCookieSessaoValida(t *testing.T) {
	s := newSigner(t)
	tenant := uuid.NewString()
	w := do(newApp(t, withSession(t, s, tenant, time.Hour)), http.MethodGet, withCookie)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), tenant) || !strings.Contains(w.Body.String(), `"sid":"sid"`) {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestSemCookie401(t *testing.T) {
	s := newSigner(t)
	if w := do(newApp(t, withSession(t, s, uuid.NewString(), time.Hour)), http.MethodGet, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", w.Code)
	}
}

func TestBearerSempreRecusado(t *testing.T) {
	s := newSigner(t)
	app := newApp(t, withSession(t, s, uuid.NewString(), time.Hour))
	w := do(app, http.MethodGet, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+s.token(t, uuid.NewString(), time.Hour))
	})
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "não aceita Authorization") {
		t.Fatalf("jwt válido no header devia ser recusado: code=%d body=%s", w.Code, w.Body.String())
	}
	// Header presente derruba mesmo com cookie válido junto: não há fallback.
	w = do(app, http.MethodGet, func(r *http.Request) {
		withCookie(r)
		r.Header.Set("Authorization", "Bearer x")
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("header+cookie: code=%d", w.Code)
	}
}

func TestJWTDaSessaoExpiradoOuSemTenant(t *testing.T) {
	s := newSigner(t)
	w := do(newApp(t, withSession(t, s, uuid.NewString(), -time.Minute)), http.MethodGet, withCookie)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("expirado: code=%d set-cookie=%q", w.Code, w.Header().Get("Set-Cookie"))
	}
	// Sem tenant o kit recusa o token (RequireTenant): sessão inválida → 401 + cookie limpo.
	if w := do(newApp(t, withSession(t, s, "", time.Hour)), http.MethodGet, withCookie); w.Code != http.StatusUnauthorized {
		t.Fatalf("sem tenant: code=%d", w.Code)
	}
}

func TestCookieSessaoDesconhecidaLimpaCookie(t *testing.T) {
	s := newSigner(t)
	w := do(newApp(t, &fakeResolver{verifier: s.verifier, byToken: map[string]domsess.Session{}}), http.MethodGet, withCookie)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", w.Code)
	}
	sc := w.Header().Get("Set-Cookie")
	if !strings.Contains(sc, cookieName+"=;") || !strings.Contains(sc, "Max-Age=0") || !strings.Contains(sc, "HttpOnly") {
		t.Fatalf("cookie não foi limpo: %q", sc)
	}
}

func TestCookieAuthIndisponivelDa503SemLimparCookie(t *testing.T) {
	w := do(newApp(t, &fakeResolver{err: domsess.ErrAuthUnavailable}), http.MethodGet, withCookie)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("code=%d", w.Code)
	}
	if w.Header().Get("Set-Cookie") != "" {
		t.Fatal("não deve limpar cookie quando o auth está fora do ar")
	}
}

// ---- CSRF ----

func TestCSRFPostCrossSiteRecusado(t *testing.T) {
	s := newSigner(t)
	app := newApp(t, withSession(t, s, uuid.NewString(), time.Hour))
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
	w := do(newApp(t, withSession(t, s, uuid.NewString(), time.Hour)), http.MethodGet, func(r *http.Request) {
		withCookie(r)
		r.Header.Set("Sec-Fetch-Site", "cross-site")
	})
	if w.Code != http.StatusOK {
		t.Fatalf("GET não muda estado; code=%d", w.Code)
	}
}

func TestCSRFMiddlewareIsolado(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/logout", CSRF([]string{"https://admin.meufin.app"}), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodPost, "https://api.meufin.app/logout", nil)
	req.Host = "api.meufin.app"
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("logout cross-site: code=%d", w.Code)
	}
}
