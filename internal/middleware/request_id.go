package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HeaderRequestID is the canonical request ID header name.
const HeaderRequestID = "X-Request-ID"

type stdRequestIDKey struct{}

const ctxGinRequestIDKey = "request_id"

// RequestID attaches a request ID to the Gin context, response headers, and request-scoped context.Context.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(HeaderRequestID)
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Writer.Header().Set(HeaderRequestID, rid)
		c.Set(ctxGinRequestIDKey, rid)
		ctx := context.WithValue(c.Request.Context(), stdRequestIDKey{}, rid)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// RequestIDFromContext returns the request ID from the Gin context.
func RequestIDFromContext(c *gin.Context) string {
	v, ok := c.Get(ctxGinRequestIDKey)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// RequestIDFromRequest returns the request ID from the standard context (if RequestID ran).
func RequestIDFromRequest(ctx context.Context) string {
	v := ctx.Value(stdRequestIDKey{})
	s, _ := v.(string)
	return s
}
