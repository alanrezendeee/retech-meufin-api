package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimitPerIP limita requisições por IP em janela fixa (in-memory).
// Suficiente para proteger endpoints públicos (ex.: share-links) de abuso;
// para múltiplas réplicas o limite vale por instância — aceitável no MVP.
func RateLimitPerIP(max int, window time.Duration) gin.HandlerFunc {
	type bucket struct {
		count int
		reset time.Time
	}
	var mu sync.Mutex
	buckets := map[string]*bucket{}

	return func(c *gin.Context) {
		now := time.Now()
		ip := c.ClientIP()

		mu.Lock()
		b, ok := buckets[ip]
		if !ok || now.After(b.reset) {
			// Janela nova; aproveita para varrer entradas velhas de vez em
			// quando (mapa não cresce sem limite).
			if len(buckets) > 10_000 {
				for k, v := range buckets {
					if now.After(v.reset) {
						delete(buckets, k)
					}
				}
			}
			b = &bucket{reset: now.Add(window)}
			buckets[ip] = b
		}
		b.count++
		over := b.count > max
		mu.Unlock()

		if over {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{"code": "RATE_LIMITED", "message": "muitas requisições — tente novamente em instantes"},
			})
			return
		}
		c.Next()
	}
}
