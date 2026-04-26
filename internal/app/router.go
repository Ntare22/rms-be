package app

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "rms-be/docs"
	"rms-be/internal/api/response"
	"rms-be/internal/middleware"
	"rms-be/internal/modules/auth"
	"rms-be/internal/modules/buildings"
	"rms-be/internal/modules/leases"
	"rms-be/internal/modules/organizations"
	"rms-be/internal/modules/payments"
	"rms-be/internal/modules/tenants"
	"rms-be/internal/modules/units"
	"rms-be/internal/modules/users"
)

// NewRouter configures Gin routes and middleware.
func NewRouter(deps *Dependencies) *gin.Engine {
	if deps.Config.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.Recovery(deps.Log))
	r.Use(middleware.RequestID())
	r.Use(middleware.StructuredLogger(deps.Log))
	r.Use(cors.New(cors.Config{
		AllowOrigins:     deps.Config.CORSAllowedOrigins,
		AllowMethods:     deps.Config.CORSAllowedMethods,
		AllowHeaders:     deps.Config.CORSAllowedHeaders,
		ExposeHeaders:    deps.Config.CORSExposeHeaders,
		AllowCredentials: deps.Config.CORSAllowCredentials,
		MaxAge:           deps.Config.CORSMaxAge,
	}))

	//	@Summary		Liveness
	//	@Tags			health
	//	@Produce		json
	//	@Success		200	{object}	map[string]string
	//	@Failure		503	{object}	map[string]string
	//	@Router			/health/live [get]
	r.GET("/health/live", func(c *gin.Context) {
		if deps.IsShuttingDown() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "shutting_down"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "live"})
	})

	//	@Summary		Readiness
	//	@Tags			health
	//	@Produce		json
	//	@Success		200	{object}	map[string]string
	//	@Failure		503	{object}	map[string]string
	//	@Router			/health/ready [get]
	r.GET("/health/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := deps.DB.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	v1.GET("/payments/ipn/pesapal", deps.Payments.PesapalIPNCallback)
	v1.POST("/payments/ipn/pesapal", deps.Payments.PesapalIPNCallback)
	payments.RegisterPublicIntegrationRoutes(v1.Group("/payments"), deps.Payments)

	// Public auth endpoints (rate-limited).
	authPublic := v1.Group("/auth")
	authPublic.Use(middleware.AuthLoginRateLimiter(deps.Config.AuthLoginRPM))
	auth.RegisterPublicRoutes(authPublic, deps.Auth)

	authProtected := v1.Group("/auth")
	authProtected.Use(middleware.JWTAuth(deps.TokenIssuer))
	auth.RegisterProtectedRoutes(authProtected, deps.Auth)

	registerAPIModuleRouteGroups(v1)

	// Authenticated examples (compose middleware: JWT -> role / org scope).
	protected := v1.Group("")
	protected.Use(middleware.JWTAuth(deps.TokenIssuer))

	orgs := protected.Group("/organizations")
	organizations.RegisterRoutes(orgs, deps.Organizations)

	// Gin: single `/organizations/:id` group (wildcard name must match org detail routes).
	orgTree := protected.Group("/organizations/:id")
	orgTree.Use(middleware.RequireOrganizationParam("id"))
	users.RegisterRoutes(orgTree.Group("/users"), deps.Users)
	buildings.RegisterRoutes(orgTree.Group("/buildings"), deps.Buildings)
	units.RegisterRoutes(orgTree.Group("/buildings/:buildingId/units"), deps.Units)
	tenants.RegisterRoutes(orgTree.Group("/tenants"), deps.Tenants)
	leases.RegisterRoutes(orgTree.Group("/leases"), deps.Leases)
	payments.RegisterOrgRoutes(orgTree.Group("/payments"), deps.Payments)
	payments.RegisterIntegrationRoutes(protected.Group("/payments"), deps.Payments)

	admin := protected.Group("/admin")
	admin.Use(middleware.RequireRoles(middleware.RoleAdmin))
	admin.GET("/ping", func(c *gin.Context) {
		response.OK(c, gin.H{"role": "admin"})
	})

	orgScoped := protected.Group("/orgs/:org_id")
	orgScoped.Use(middleware.RequireOrganizationParam("org_id"))
	orgScoped.GET("/summary", func(c *gin.Context) {
		response.OK(c, gin.H{"organization_id": c.Param("org_id")})
	})

	if deps.Config.InternalJobSecret != "" {
		internal := r.Group("/internal")
		internal.Use(middleware.InternalSecretAuth(deps.Config.InternalJobSecret))
		internal.GET("/status", func(c *gin.Context) {
			response.OK(c, gin.H{"status": "ok", "component": "internal"})
		})
	}

	return r
}
