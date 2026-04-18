package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/response"
)

// Well-known application roles for RBAC middleware.
const (
	RoleAdmin    = "admin"
	RoleLandlord = "landlord"
	RoleManager  = "manager"
)

// RequireRoles enforces that the authenticated user has one of the allowed roles.
func RequireRoles(allowed ...string) gin.HandlerFunc {
	norm := make([]string, 0, len(allowed))
	for _, r := range allowed {
		norm = append(norm, strings.TrimSpace(strings.ToLower(r)))
	}
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c)
		if !ok || claims == nil {
			response.Error(c, apierrors.ErrUnauthorized)
			c.Abort()
			return
		}
		role := strings.TrimSpace(strings.ToLower(claims.Role))
		for _, a := range norm {
			if role == a {
				c.Next()
				return
			}
		}
		response.Error(c, apierrors.ErrForbidden)
		c.Abort()
	}
}
