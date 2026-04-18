package tenants

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts tenants under `/organizations/:id/tenants` (parent must apply JWT and RequireOrganizationParam("id")).
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.GET("", h.List)
	rg.POST("", h.Create)
	rg.GET("/:tenantId", h.Get)
	rg.PATCH("/:tenantId", h.Patch)
	rg.DELETE("/:tenantId", h.Delete)
}
