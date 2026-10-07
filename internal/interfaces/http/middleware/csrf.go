package middleware

import (
	"net/http"
	"net/url"
	"strings"
)

// Métodos que não mudam estado — isentos da verificação CSRF.
func safeMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

// crossSiteRequest decide se uma requisição autenticada POR COOKIE veio de
// outro site. Defesa em profundidade sobre o SameSite=Lax do cookie:
//
//   - Sec-Fetch-Site: cross-site → rejeita (browsers modernos sempre enviam).
//   - Origin presente e fora da lista permitida (e diferente do próprio
//     host da API) → rejeita.
//   - Sem nenhum dos dois (cliente não-browser com cookie) → aceita; Lax já
//     impede o browser de chegar aqui em POST cross-site.
func crossSiteRequest(r *http.Request, allowedOrigins map[string]struct{}) bool {
	if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
		return true
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" || origin == "null" {
		return origin == "null"
	}
	if _, ok := allowedOrigins[origin]; ok {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil {
		return true
	}
	// Mesma origem da própria API (same-origin via proxy do admin).
	return !strings.EqualFold(u.Host, r.Host)
}

func originSet(origins []string) map[string]struct{} {
	out := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		if o = strings.TrimSpace(o); o != "" {
			out[o] = struct{}{}
		}
	}
	return out
}
