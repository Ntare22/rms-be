package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"rms-be/internal/api/logger"
)

// StructuredLogger logs method, path, HTTP status, duration, and request ID.
func StructuredLogger(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		c.Next()

		duration := time.Since(start)
		args := []any{
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"duration_ms", duration.Milliseconds(),
			"client_ip", c.ClientIP(),
		}
		if rid := RequestIDFromContext(c); rid != "" {
			args = append(args, "request_id", rid)
		}
		if len(c.Errors) > 0 {
			args = append(args, "errors", c.Errors.String())
		}

		msg := "http_request"
		switch {
		case c.Writer.Status() >= 500:
			log.Error(msg, args...)
		case c.Writer.Status() >= 400:
			log.Warn(msg, args...)
		default:
			log.Info(msg, args...)
		}
	}
}
