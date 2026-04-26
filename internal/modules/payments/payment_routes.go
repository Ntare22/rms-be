package payments

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts payment endpoints under /organizations/:id/payments.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.POST("/initiate", h.Initiate)
	rg.GET("/summary", h.Summary)
	rg.GET("/method-split", h.MethodSplit)
	rg.GET("/transaction-status", h.GetTransactionStatus)
	rg.GET("/reminders/candidates", h.ReminderCandidates)
	rg.GET("/reminders/history", h.ReminderHistory)
	rg.POST("/reminders/send", h.SendReminders)
	rg.POST("/pesapal/ipn/register", h.RegisterPesapalIPN)
	rg.GET("/pesapal/ipn/list", h.ListPesapalIPN)
}
