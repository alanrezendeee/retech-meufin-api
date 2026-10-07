package handlers

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// IAMProxyHandler repassa as telas de administração (usuários, roles,
// permissions) ao retech-auth-api injetando o access token da sessão. Com o
// token fora do browser, o admin não consegue mais falar com o auth
// diretamente — este proxy é o único caminho.
//
// Só os prefixos listados passam; o resto é 404. Cookies nunca seguem ao auth.
type IAMProxyHandler struct {
	proxy   *httputil.ReverseProxy
	enabled bool
	log     *slog.Logger
}

// Prefixos (sob /v1 do auth) que o admin usa.
var iamAllowedPrefixes = []string{"/v1/users", "/v1/roles", "/v1/permissions"}

func NewIAMProxyHandler(authBaseURL string, log *slog.Logger) *IAMProxyHandler {
	h := &IAMProxyHandler{log: log}
	base := strings.TrimRight(strings.TrimSpace(authBaseURL), "/")
	if base == "" {
		return h
	}
	target, err := url.Parse(base)
	if err != nil || target.Host == "" {
		log.Warn("⚠️ AUTH_API_BASE_URL inválida; proxy IAM desabilitado", slog.String("url", authBaseURL))
		return h
	}
	h.proxy = &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.Out.Host = target.Host
			// Nada do browser além do método/corpo/query: sem cookies, sem
			// headers de autenticação próprios; o Bearer vem da sessão.
			pr.Out.Header.Del("Cookie")
			pr.Out.Header.Set("Authorization", "Bearer "+pr.In.Context().Value(iamTokenKey{}).(string))
			pr.Out.Header.Set("Accept", "application/json")
			if reqID := pr.In.Header.Get("X-Request-ID"); reqID != "" {
				pr.Out.Header.Set("X-Request-ID", reqID)
			}
		},
		ModifyResponse: func(resp *http.Response) error {
			// O browser fala só com esta API; headers de CORS/cookies do auth não fazem sentido aqui.
			resp.Header.Del("Set-Cookie")
			for k := range resp.Header {
				if strings.HasPrefix(strings.ToLower(k), "access-control-") {
					resp.Header.Del(k)
				}
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Warn("⚠️ proxy IAM: auth indisponível", slog.String("error", err.Error()))
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"serviço de autenticação indisponível; tente novamente em instantes"}}`))
		},
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			ResponseHeaderTimeout: 20 * time.Second,
			MaxIdleConns:          20,
			IdleConnTimeout:       90 * time.Second,
		},
	}
	h.enabled = true
	return h
}

type iamTokenKey struct{}

// Proxy atende ANY /api/v1/iam/*path → AUTH/*path. Registrar sob RequireAuth.
func (h *IAMProxyHandler) Proxy(c *gin.Context) {
	if !h.enabled {
		errrespond.Message(c, http.StatusServiceUnavailable, errrespond.CodeInternal, "integração com o auth não configurada (AUTH_API_BASE_URL)")
		return
	}
	token := c.GetString(middleware.CtxAccessToken)
	if token == "" {
		errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "token de autenticação ausente")
		return
	}
	upstreamPath := c.Param("path") // começa com "/" (ex.: /v1/users/123)
	if !iamPathAllowed(upstreamPath) {
		errrespond.Message(c, http.StatusNotFound, errrespond.CodeNotFound, "rota não disponível via proxy IAM")
		return
	}

	req := c.Request.Clone(c.Request.Context())
	req = req.WithContext(contextWithIAMToken(req, token))
	req.URL.Path = upstreamPath
	req.URL.RawPath = ""
	h.proxy.ServeHTTP(proxyWriter{c.Writer}, req)
}

// proxyWriter esconde CloseNotify do gin.ResponseWriter: a implementação do
// gin faz type assertion no writer subjacente e entra em pânico quando ele
// não é CloseNotifier (httptest.ResponseRecorder). ReverseProxy usa
// http.ResponseController para Flush, que chega ao writer real via Unwrap.
type proxyWriter struct{ http.ResponseWriter }

func (w proxyWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w proxyWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func iamPathAllowed(p string) bool {
	if strings.Contains(p, "..") {
		return false
	}
	for _, prefix := range iamAllowedPrefixes {
		if p == prefix || strings.HasPrefix(p, prefix+"/") {
			return true
		}
	}
	return false
}
