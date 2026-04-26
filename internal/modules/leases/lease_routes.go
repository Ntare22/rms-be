package leases

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts lease routes under `/organizations/:id/leases`.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.GET("", h.List)
	rg.POST("", h.Create)
	rg.GET("/:leaseId", h.Get)
	rg.GET("/:leaseId/history", h.History)
	rg.PATCH("/:leaseId", h.Patch)
	rg.POST("/:leaseId/approve", h.Approve)
	rg.POST("/:leaseId/reject", h.Reject)
	rg.POST("/:leaseId/renewals", h.CreateRenewalOffer)
	rg.POST("/:leaseId/renewals/:offerId/accept", h.AcceptRenewalOffer)
	rg.POST("/:leaseId/renewals/:offerId/reject", h.RejectRenewalOffer)
	rg.POST("/:leaseId/closeout", h.Closeout)
	rg.POST("/:leaseId/end", h.End)
	rg.GET("/tenants/:tenantId/statement", h.TenantStatement)
	rg.GET("/tenants/:tenantId/statement/export", h.TenantStatementExport)
}
