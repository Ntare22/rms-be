package app

import "github.com/gin-gonic/gin"

// registerAPIModuleRouteGroups creates /api/v1 sub-groups for future function-specific route files.
// Auth public routes are registered separately in NewRouter (rate limiting).
// Organization-scoped routes (users, buildings, units, tenants, leases) live under /organizations/:id/... in NewRouter.
func registerAPIModuleRouteGroups(v1 *gin.RouterGroup) {
	_ = v1.Group("/charges")
	_ = v1.Group("/payments")
	_ = v1.Group("/reports")
	_ = v1.Group("/notifications")
	_ = v1.Group("/audit-logs")
}
