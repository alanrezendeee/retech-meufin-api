package handlers

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

func TestIAMPathAllowed(t *testing.T) {
	ok := []string{"/v1/users", "/v1/users/abc", "/v1/users/abc/roles", "/v1/roles", "/v1/permissions"}
	nok := []string{"/v1/applications", "/v1/me", "/v1/authenticate", "/v1/usersx", "/v1/users/../applications", "/", ""}
	for _, p := range ok {
		if !iamPathAllowed(p) {
			t.Errorf("%q devia passar", p)
		}
	}
	for _, p := range nok {
		if iamPathAllowed(p) {
			t.Errorf("%q não devia passar", p)
		}
	}
}

func TestIAMProxyInjetaBearerERemoveCookie(t *testing.T) {
	var got *http.Request
	var gotBody string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Set-Cookie", "x=1")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"u1"}`))
	}))
	defer upstream.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewIAMProxyHandler(upstream.URL, slog.Default())
	r.Use(func(c *gin.Context) { c.Set(middleware.CtxAccessToken, "jwt-da-sessao") })
	r.Any("/api/v1/iam/*path", h.Proxy)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/iam/v1/users?page=2", strings.NewReader(`{"email":"a@b.c"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "meufin_session=segredo")
	req.Header.Set("Authorization", "Bearer do-browser-nao-vale")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated || w.Body.String() != `{"id":"u1"}` {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if got.URL.Path != "/v1/users" || got.URL.RawQuery != "page=2" {
		t.Fatalf("upstream path=%s query=%s", got.URL.Path, got.URL.RawQuery)
	}
	if got.Header.Get("Authorization") != "Bearer jwt-da-sessao" {
		t.Fatalf("Authorization=%q", got.Header.Get("Authorization"))
	}
	if got.Header.Get("Cookie") != "" {
		t.Fatal("cookie do browser vazou para o auth")
	}
	if gotBody != `{"email":"a@b.c"}` {
		t.Fatalf("body=%s", gotBody)
	}
	if w.Header().Get("Set-Cookie") != "" || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("headers do auth (Set-Cookie/CORS) não devem vazar ao browser")
	}
}

func TestIAMProxyRotaForaDaAllowlist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewIAMProxyHandler("http://127.0.0.1:1", slog.Default())
	r.Use(func(c *gin.Context) { c.Set(middleware.CtxAccessToken, "jwt") })
	r.Any("/api/v1/iam/*path", h.Proxy)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/iam/v1/applications", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d", w.Code)
	}
}

func TestIAMProxyDesabilitadoSemBaseURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewIAMProxyHandler("", slog.Default())
	r.Use(func(c *gin.Context) { c.Set(middleware.CtxAccessToken, "jwt") })
	r.Any("/api/v1/iam/*path", h.Proxy)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/iam/v1/users", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("code=%d", w.Code)
	}
}
