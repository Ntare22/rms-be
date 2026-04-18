package app

import (
	"fmt"
	"sync/atomic"

	"rms-be/internal/api/clock"
	"rms-be/internal/api/logger"
	"rms-be/internal/api/security"
	"rms-be/internal/config"
	"rms-be/internal/database"
	"rms-be/internal/modules/auth"
	"rms-be/internal/modules/buildings"
	"rms-be/internal/modules/leases"
	"rms-be/internal/modules/organizations"
	"rms-be/internal/modules/tenants"
	"rms-be/internal/modules/units"
	"rms-be/internal/modules/users"
)

// Dependencies holds cross-cutting runtime dependencies for HTTP handlers and jobs.
type Dependencies struct {
	Config *config.Config
	Log    logger.Logger
	DB     database.DB
	Clock  clock.Clock

	PasswordHasher security.PasswordHasher
	TokenIssuer    security.TokenIssuer
	Auth           *auth.Handler
	Organizations  *organizations.Handler
	Users          *users.Handler
	Buildings      *buildings.Handler
	Units          *units.Handler
	Tenants        *tenants.Handler
	Leases         *leases.Handler

	shuttingDown atomic.Bool
}

// NewDependencies wires bootstrap dependencies and runs GORM AutoMigrate on the primary DB.
func NewDependencies(cfg *config.Config, log logger.Logger, db database.DB, clk clock.Clock) (*Dependencies, error) {
	if err := AutoMigrate(db.Gorm()); err != nil {
		return nil, fmt.Errorf("dependencies: %w", err)
	}

	gdb := db.Gorm()
	issuer := security.NewHS256TokenIssuer(cfg)
	ph := security.BcryptHasher{}
	authRepo := auth.NewRepository(gdb)
	authSvc := auth.NewService(authRepo, ph, issuer)
	authHandler := auth.NewHandler(authSvc)

	orgRepo := organizations.NewRepository(gdb)
	orgSvc := organizations.NewService(orgRepo)
	orgHandler := organizations.NewHandler(orgSvc)

	userRepo := users.NewRepository(gdb)
	userSvc := users.NewService(userRepo, ph)
	userHandler := users.NewHandler(userSvc)

	bldRepo := buildings.NewRepository(gdb)
	bldSvc := buildings.NewService(bldRepo)
	bldHandler := buildings.NewHandler(bldSvc)

	unitRepo := units.NewRepository(gdb)
	unitSvc := units.NewService(unitRepo)
	unitHandler := units.NewHandler(unitSvc)

	tntRepo := tenants.NewRepository(gdb)
	tntSvc := tenants.NewService(tntRepo)
	tntHandler := tenants.NewHandler(tntSvc)

	leaseRepo := leases.NewRepository(gdb)
	leaseSvc := leases.NewService(leaseRepo)
	leaseHandler := leases.NewHandler(leaseSvc)

	return &Dependencies{
		Config:         cfg,
		Log:            log,
		DB:             db,
		Clock:          clk,
		PasswordHasher: ph,
		TokenIssuer:    issuer,
		Auth:           authHandler,
		Organizations:  orgHandler,
		Users:          userHandler,
		Buildings:      bldHandler,
		Units:          unitHandler,
		Tenants:        tntHandler,
		Leases:         leaseHandler,
	}, nil
}

// MarkShuttingDown marks the process as draining (liveness should report not-ready).
func (d *Dependencies) MarkShuttingDown() {
	d.shuttingDown.Store(true)
}

// IsShuttingDown reports whether graceful shutdown has started.
func (d *Dependencies) IsShuttingDown() bool {
	return d.shuttingDown.Load()
}
