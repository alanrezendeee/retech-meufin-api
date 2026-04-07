package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const HeaderRequestID = "X-Request-ID"
const ctxKeyRequestID = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(HeaderRequestID)
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set(ctxKeyRequestID, rid)
		c.Writer.Header().Set(HeaderRequestID, rid)
		c.Next()
	}
}

func GetRequestID(c *gin.Context) string {
	v, ok := c.Get(ctxKeyRequestID)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}
