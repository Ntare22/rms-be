package payments

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts payment endpoints under /organizations/:id/payments.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.POST("/initiate", h.Initiate)
}
