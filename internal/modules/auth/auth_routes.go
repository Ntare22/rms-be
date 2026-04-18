package auth

import "github.com/gin-gonic/gin"

// RegisterPublicRoutes registers unauthenticated auth endpoints (apply rate limiting outside).
func RegisterPublicRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.POST("/register", h.Register)
	rg.POST("/login", h.Login)
	rg.POST("/refresh", h.Refresh)
}

// RegisterProtectedRoutes registers auth routes that require a valid Bearer access token.
func RegisterProtectedRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.GET("/me", h.Me)
}
