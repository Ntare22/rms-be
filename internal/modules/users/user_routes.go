package users

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts user routes on `/organizations/:orgId/users` (parent must include JWT and org scoping middleware).
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.GET("", h.List)
	rg.POST("", h.Create)
	rg.PATCH("/:userId", h.Patch)
}
