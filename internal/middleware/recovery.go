package middleware

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/logger"
	"rms-be/internal/api/response"
)

// Recovery recovers from panics and returns a consistent JSON error via internal/api/response.
func Recovery(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("panic recovered",
					"panic", rec,
					"stack", string(debug.Stack()),
					"request_id", RequestIDFromContext(c),
				)
				response.Error(c, apierrors.ErrInternal)
				c.Abort()
			}
		}()
		c.Next()
	}
}
