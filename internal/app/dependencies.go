package app

import (
	"fmt"
	"strings"
	"sync/atomic"

	"rms-be/internal/api/clock"
	"rms-be/internal/api/logger"
	"rms-be/internal/api/security"
	"rms-be/internal/config"
	"rms-be/internal/database"
	"rms-be/internal/integrations/pesapal"
	"rms-be/internal/modules/auth"
	"rms-be/internal/modules/buildings"
	"rms-be/internal/modules/leases"
	"rms-be/internal/modules/notifications"
	"rms-be/internal/modules/organizations"
	"rms-be/internal/modules/payments"
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
	Payments       *payments.Handler

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
	mailjet := notifications.NewMailjetSender(cfg.MailjetAPIKey, cfg.MailjetAPISecret)
	var mailer notifications.EmailSender = notifications.NoopEmailSender{}
	if mailjet.Enabled() {
		mailer = mailjet
	}
	fromEmail := strings.TrimSpace(cfg.MailjetFromEmail)
	if fromEmail == "" {
		fromEmail = strings.TrimSpace(cfg.MailjetFrom)
	}
	fromName := strings.TrimSpace(cfg.MailjetFromName)
	if fromName == "" {
		fromName = strings.TrimSpace(cfg.SMTPFromName)
	}
	if fromName == "" {
		fromName = "RMS"
	}
	authRepo := auth.NewRepository(gdb)
	authSvc := auth.NewService(authRepo, ph, issuer, mailer, auth.PasswordSetupConfig{
		BaseURL:          cfg.AppBaseURL,
		FromName:         fromName,
		From:             fromEmail,
		InviteTemplateID: cfg.MailjetTemplateInviteID,
	})
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
	tntSvc := tenants.NewService(tntRepo, authSvc)
	tntHandler := tenants.NewHandler(tntSvc)

	leaseRepo := leases.NewRepository(gdb)
	leaseSvc := leases.NewService(leaseRepo)
	leaseHandler := leases.NewHandler(leaseSvc)

	paymentRepo := payments.NewRepository(gdb)
	pesaClient := pesapal.NewClient(pesapal.Config{
		BaseURL:        cfg.PesaPalBaseURL,
		ConsumerKey:    cfg.PesaPalConsumerKey,
		ConsumerSecret: cfg.PesaPalConsumerSecret,
		IPNID:          cfg.PesaPalIPNID,
		Timeout:        cfg.PesaPalTimeout,
	})
	paymentSvc := payments.NewService(paymentRepo, pesaClient, cfg.AppBaseURL, cfg.PesaPalIPNID, cfg.PesaPalIPNNotificationType)
	paymentHandler := payments.NewHandler(paymentSvc)

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
		Payments:       paymentHandler,
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
