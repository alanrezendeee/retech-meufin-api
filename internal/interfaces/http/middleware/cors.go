package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS libera apenas as origins informadas (lista do admin). Responde preflight.
// `Authorization` só entra em Allow-Headers quando Bearer está habilitado: com
// o admin por cookie, o browser não tem motivo para mandar esse header.
func CORS(allowedOrigins []string, allowAuthorizationHeader bool) gin.HandlerFunc {
	allowHeaders := "Content-Type, X-Request-ID"
	if allowAuthorizationHeader {
		allowHeaders = "Authorization, " + allowHeaders
	}
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		if o != "" {
			allowed[o] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", allowHeaders)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Max-Age", "86400")
			}
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
