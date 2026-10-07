package middleware

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP devolve o IP do cliente atrás do reverse proxy (Railway edge e/ou
// nginx do admin). Usa o ÚLTIMO hop de X-Forwarded-For: é o que o proxy
// imediatamente à frente anexou, logo não é controlado pelo cliente — o
// primeiro hop é texto livre e permitiria burlar rate limit por IP. Sem XFF,
// cai em X-Real-IP e por fim em RemoteAddr.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if last := strings.TrimSpace(parts[len(parts)-1]); last != "" {
			return last
		}
	}
	if rip := strings.TrimSpace(r.Header.Get("X-Real-IP")); rip != "" {
		return rip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
