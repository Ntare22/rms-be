package leases

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts lease routes under `/organizations/:id/leases`.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.GET("", h.List)
	rg.POST("", h.Create)
	rg.GET("/:leaseId", h.Get)
	rg.PATCH("/:leaseId", h.Patch)
	rg.POST("/:leaseId/approve", h.Approve)
	rg.POST("/:leaseId/reject", h.Reject)
	rg.POST("/:leaseId/end", h.End)
}
