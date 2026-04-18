package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/response"
	"rms-be/internal/api/security"
)

const ctxUserClaimsKey = "rms_user_claims"

// UserClaims is the JWT-derived identity attached to the request context.
type UserClaims struct {
	UserID         string `json:"sub"`
	OrganizationID string `json:"org_id"`
	Role           string `json:"role"`
}

// JWTAuth validates Bearer access tokens and loads UserClaims into the Gin context.
func JWTAuth(issuer security.TokenIssuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := strings.TrimSpace(c.GetHeader("Authorization"))
		if len(h) < 7 || strings.ToLower(h[:7]) != "bearer " {
			response.Error(c, apierrors.ErrUnauthorized)
			c.Abort()
			return
		}
		raw := strings.TrimSpace(h[7:])
		subject, rawClaims, err := issuer.ParseAccessToken(raw)
		if err != nil {
			response.Error(c, apierrors.ErrUnauthorized)
			c.Abort()
			return
		}
		orgID, _ := rawClaims["org_id"].(string)
		role, _ := rawClaims["role"].(string)
		claims := &UserClaims{
			UserID:         subject,
			OrganizationID: orgID,
			Role:           strings.TrimSpace(strings.ToLower(role)),
		}
		c.Set(ctxUserClaimsKey, claims)
		c.Next()
	}
}

// ClaimsFromContext returns claims set by JWTAuth.
func ClaimsFromContext(c *gin.Context) (*UserClaims, bool) {
	v, ok := c.Get(ctxUserClaimsKey)
	if !ok {
		return nil, false
	}
	cl, ok := v.(*UserClaims)
	return cl, ok
}
