package sms

import (
	"github.com/gin-gonic/gin"

	"rms-be/internal/middleware"
)

func RegisterPublicRoutes(rg *gin.RouterGroup, h *Handler, sendRPM, bulkRPM int) {
	rg.GET("/health", h.Health)
	rg.POST("/send", middleware.IPRateLimiter(sendRPM, 5), h.Send)
	rg.POST("/bulk-send", middleware.IPRateLimiter(bulkRPM, 2), h.BulkSend)
	rg.POST("/send-template", middleware.IPRateLimiter(sendRPM, 5), h.SendTemplate)
}
