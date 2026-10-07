package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func preflight(t *testing.T, origin string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS([]string{"https://admin.meufin.app"}))
	r.PUT("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodOptions, "/x", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// O preflight precisa anunciar todos os métodos que o router usa. Um método
// roteado mas ausente daqui falha só no navegador, com erro de CORS que não
// aponta para a causa — foi o que aconteceu quando o primeiro PATCH do app
// foi registrado.
func TestCORSAnunciaMetodosRoteados(t *testing.T) {
	w := preflight(t, "https://admin.meufin.app")
	got := w.Header().Get("Access-Control-Allow-Methods")
	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"} {
		if !strings.Contains(got, m) {
			t.Errorf("método %s ausente em Access-Control-Allow-Methods: %q", m, got)
		}
	}
	if w.Code != http.StatusNoContent {
		t.Errorf("preflight = %d, quer 204", w.Code)
	}
}

func TestCORSRecusaOriginDesconhecida(t *testing.T) {
	w := preflight(t, "https://evil.example.com")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("origin não permitida recebeu liberação: %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "" {
		t.Errorf("origin não permitida recebeu métodos: %q", got)
	}
}

func TestCORSSemOriginNaoEmiteCabecalhos(t *testing.T) {
	w := preflight(t, "")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("requisição sem Origin não deveria receber cabeçalho: %q", got)
	}
}
