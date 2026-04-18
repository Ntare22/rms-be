package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/response"
)

// HeaderOrganizationID is an optional header when no :org_id path param is present.
const HeaderOrganizationID = "X-Organization-ID"

// organizationIDFromRequest resolves an organization id from a path parameter or optional header.
func organizationIDFromRequest(c *gin.Context, param string) string {
	if v := strings.TrimSpace(c.Param(param)); v != "" {
		return v
	}
	return strings.TrimSpace(c.GetHeader(HeaderOrganizationID))
}

// RequireOrganizationParam ensures non-admin callers operate only within their own organization.
// Admins may access any organization id present in the path or X-Organization-ID header.
func RequireOrganizationParam(param string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c)
		if !ok || claims == nil {
			response.Error(c, apierrors.ErrUnauthorized)
			c.Abort()
			return
		}
		if claims.Role == RoleAdmin {
			c.Next()
			return
		}
		orgID := organizationIDFromRequest(c, param)
		if orgID == "" {
			response.Error(c, apierrors.ErrValidation)
			c.Abort()
			return
		}
		if !strings.EqualFold(orgID, claims.OrganizationID) {
			response.Error(c, apierrors.ErrForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}
