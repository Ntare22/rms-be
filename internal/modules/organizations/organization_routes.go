package organizations

import (
	"github.com/gin-gonic/gin"

	"rms-be/internal/middleware"
)

// RegisterRoutes mounts organization CRUD under a group that already uses JWTAuth.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.POST("", middleware.RequireRoles(middleware.RoleAdmin), h.Create)
	rg.GET("/:id", h.Get)
	rg.PATCH("/:id", h.Patch)
}
