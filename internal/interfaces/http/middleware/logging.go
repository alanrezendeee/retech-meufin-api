package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func AccessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()
		rid := GetRequestID(c)
		log.Info("http",
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("request_id", rid),
		)
	}
}
