package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders aplica cabeçalhos defensivos a toda resposta da API (mesmo
// conjunto do CashFlowfy): sem sniffing de tipo, sem Referer e sem cache de
// respostas autenticadas (tokens/dados pessoais nunca ficam em cache de
// browser ou proxy).
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		c.Next()
	}
}
