package app

import (
	"fmt"

	"gorm.io/gorm"

	"rms-be/internal/modules/audit"
	"rms-be/internal/modules/auth"
	"rms-be/internal/modules/buildings"
	"rms-be/internal/modules/charges"
	"rms-be/internal/modules/leases"
	"rms-be/internal/modules/notifications"
	"rms-be/internal/modules/organizations"
	"rms-be/internal/modules/payments"
	"rms-be/internal/modules/tenants"
	"rms-be/internal/modules/units"
	"rms-be/internal/modules/users"
)

// AutoMigrate applies all registered models to the database schema.
// It is intended for development and bootstrap environments. Production should prefer
// versioned SQL migrations (EXCLUDE constraints, CHECK triggers, etc. are not created here).
//
// PostgreSQL: ensure gen_random_uuid() is available (built-in from PG 13; otherwise enable pgcrypto).
func AutoMigrate(db *gorm.DB) error {
	// Order respects foreign keys created by GORM from struct relations where applicable.
	if err := db.AutoMigrate(
		&organizations.Organization{},
		&users.User{},
		&users.ManagerBuildingAssignment{},
		&auth.PasswordSetupToken{},
		&buildings.Building{},
		&units.Unit{},
		&tenants.Tenant{},
		&leases.Lease{},
		&charges.RentCharge{},
		&payments.Payment{},
		&payments.PaymentAllocation{},
		&notifications.NotificationTemplate{},
		&notifications.NotificationPreference{},
		&notifications.NotificationJob{},
		&notifications.NotificationLog{},
		&audit.AuditLog{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}
