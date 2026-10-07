package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SessionCookie descreve o cookie opaco de sessão do gateway.
type SessionCookie struct {
	Name   string
	Secure bool
	Domain string
	// MaxAge em segundos quando emitido no login.
	MaxAge int
}

// Set grava o cookie de sessão (HttpOnly, SameSite=Lax, Path=/).
// Lax bloqueia envio em POST cross-site (CSRF) e ainda permite navegação
// top-level por link; same-origin via proxy do admin não precisa de Domain.
func (sc SessionCookie) Set(c *gin.Context, raw string) {
	http.SetCookie(c.Writer, sc.build(raw, sc.MaxAge))
}

// Clear expira o cookie no browser.
func (sc SessionCookie) Clear(c *gin.Context) {
	http.SetCookie(c.Writer, sc.build("", -1))
}

// Read devolve o valor do cookie ("" se ausente).
func (sc SessionCookie) Read(c *gin.Context) string {
	if sc.Name == "" {
		return ""
	}
	ck, err := c.Request.Cookie(sc.Name)
	if err != nil {
		return ""
	}
	return ck.Value
}

func (sc SessionCookie) build(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     sc.Name,
		Value:    value,
		Path:     "/",
		Domain:   sc.Domain,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   sc.Secure,
		SameSite: http.SameSiteLaxMode,
	}
}
