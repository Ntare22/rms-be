package units

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts units under `/organizations/:id/buildings/:buildingId/units` (parent must apply JWT and RequireOrganizationParam("id")).
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.GET("", h.List)
	rg.POST("", h.Create)
	rg.GET("/:unitId", h.Get)
	rg.PATCH("/:unitId", h.Patch)
	rg.DELETE("/:unitId", h.Delete)
}
