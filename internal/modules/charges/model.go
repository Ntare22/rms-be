package charges

import (
	"time"

	"gorm.io/gorm"
)

// RentChargeType categorizes ledger-style charges.
type RentChargeType string

const (
	RentChargeTypeRent       RentChargeType = "rent"
	RentChargeTypeFee        RentChargeType = "fee"
	RentChargeTypeAdjustment RentChargeType = "adjustment"
	RentChargeTypeDeposit    RentChargeType = "deposit"
)

// RentChargeStatus is a constrained string; prefer CHECK in SQL.
type RentChargeStatus string

const (
	RentChargeStatusScheduled RentChargeStatus = "scheduled"
	RentChargeStatusPosted    RentChargeStatus = "posted"
	RentChargeStatusVoid      RentChargeStatus = "void"
)

// RentCharge is an amount owed on a lease (minor units / integer cents).
//
// SQL migrations:
//   - FK lease_id -> leases(id); enforce lease.organization_id = rent_charges.organization_id via trigger or app.
//   - Optional: CHECK (amount_minor >= 0) for charges that cannot be negative; adjust for credit notes.
//   - Optional: CHECK (status IN (...)).
type RentCharge struct {
	ID               string           `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID   string           `gorm:"type:uuid;not null;index" json:"organization_id"`
	LeaseID          string           `gorm:"type:uuid;not null;index" json:"lease_id"`
	ChargeType       RentChargeType   `gorm:"size:24;not null" json:"charge_type"`
	Description      string           `gorm:"size:512" json:"description,omitempty"`
	AmountMinor      int64            `gorm:"not null" json:"amount_minor"`
	DueAt            time.Time        `gorm:"not null" json:"due_at"`
	Status           RentChargeStatus `gorm:"size:24;not null;default:scheduled" json:"status"`
	PeriodStart      *time.Time       `json:"period_start,omitempty"`
	PeriodEnd        *time.Time       `json:"period_end,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	DeletedAt        gorm.DeletedAt   `gorm:"index" json:"-"`
	CreatedBy        *string          `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy        *string          `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (RentCharge) TableName() string { return "rent_charges" }
