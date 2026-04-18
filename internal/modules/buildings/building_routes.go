package buildings

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts buildings under `/organizations/:id/buildings` (parent must set JWT + RequireOrganizationParam("id")).
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.GET("", h.List)
	rg.POST("", h.Create)
	rg.GET("/:buildingId", h.Get)
	rg.PATCH("/:buildingId", h.Patch)
	rg.DELETE("/:buildingId", h.Delete)
}
