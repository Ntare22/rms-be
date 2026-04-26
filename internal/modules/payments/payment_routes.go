package payments

import "github.com/gin-gonic/gin"

// RegisterOrgRoutes mounts organization-scoped payment endpoints under /organizations/:id/payments.
func RegisterOrgRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.POST("/initiate", h.Initiate)
	rg.GET("/summary", h.Summary)
	rg.GET("/method-split", h.MethodSplit)
	rg.GET("/reminders/candidates", h.ReminderCandidates)
	rg.GET("/reminders/history", h.ReminderHistory)
	rg.POST("/reminders/send", h.SendReminders)
}

// RegisterIntegrationRoutes mounts global Pesapal integration endpoints under /payments.
func RegisterIntegrationRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.GET("/transaction-status", h.GetTransactionStatus)
	rg.POST("/pesapal/ipn/register", h.RegisterPesapalIPN)
	rg.GET("/pesapal/ipn/list", h.ListPesapalIPN)
}

// RegisterPublicIntegrationRoutes mounts public integration test endpoints under /payments.
func RegisterPublicIntegrationRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.POST("/pesapal/token", h.GeneratePesapalToken)
}
