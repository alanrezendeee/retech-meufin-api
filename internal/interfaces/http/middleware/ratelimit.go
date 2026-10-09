package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/theretechlabs/retech-authkit/clientip"
	"github.com/theretechlabs/retech-authkit/ratelimit"
)

// RateLimitPerIP limita requisições por IP (último hop do X-Forwarded-For) em
// janela fixa, em memória (retech-authkit/ratelimit). Com réplicas o limite
// vale por instância.
func RateLimitPerIP(max int, window time.Duration) gin.HandlerFunc {
	l := ratelimit.New(max, window)
	return func(c *gin.Context) {
		ok, retry := l.Allow(clientip.FromRequest(c.Request))
		if !ok {
			c.Header("Retry-After", ratelimit.RetryAfterSeconds(retry))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{"code": "RATE_LIMITED", "message": "muitas requisições — tente novamente em instantes"},
			})
			return
		}
		c.Next()
	}
}

// ClientIP devolve o IP do cliente pelo último hop do X-Forwarded-For.
func ClientIP(r *http.Request) string { return clientip.FromRequest(r) }
