package middleware

import (
	"crypto/subtle"

	"github.com/gin-gonic/gin"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/response"
)

// HeaderInternalKey is the HTTP header carrying INTERNAL_JOB_SECRET for internal routes.
const HeaderInternalKey = "X-Internal-Key"

// InternalSecretAuth protects internal job routes using INTERNAL_JOB_SECRET (constant-time compare).
func InternalSecretAuth(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expected == "" {
			response.Error(c, apierrors.ErrForbidden)
			c.Abort()
			return
		}
		got := c.GetHeader(HeaderInternalKey)
		if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
			response.Error(c, apierrors.ErrForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}
