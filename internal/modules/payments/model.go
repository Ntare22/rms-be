package payments

import (
	"time"

	"gorm.io/gorm"
)

// PaymentMethod is a constrained string for how money was received.
type PaymentMethod string

const (
	PaymentMethodACH   PaymentMethod = "ach"
	PaymentMethodCard  PaymentMethod = "card"
	PaymentMethodCash  PaymentMethod = "cash"
	PaymentMethodCheck PaymentMethod = "check"
	PaymentMethodWire  PaymentMethod = "wire"
	PaymentMethodOther PaymentMethod = "other"
)

// PaymentStatus is a constrained string; prefer CHECK in SQL.
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusVoid      PaymentStatus = "void"
)

// ReminderLevel is escalation stage for collection reminders.
type ReminderLevel string

const (
	ReminderLevel1     ReminderLevel = "reminder_1"
	ReminderLevel2     ReminderLevel = "reminder_2"
	ReminderLevelFinal ReminderLevel = "final_notice"
)

// Payment records money received against a lease (minor units).
//
// SQL migrations:
//   - FK lease_id -> leases(id); enforce lease.organization_id = payments.organization_id.
//   - Optional: CHECK (amount_minor > 0).
type Payment struct {
	ID             string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string         `gorm:"type:uuid;not null;index" json:"organization_id"`
	LeaseID        string         `gorm:"type:uuid;not null;index" json:"lease_id"`
	AmountMinor    int64          `gorm:"not null" json:"amount_minor"`
	Currency       string         `gorm:"size:3;not null;default:'USD'" json:"currency"`
	ReceivedAt     time.Time      `gorm:"not null" json:"received_at"`
	Method         PaymentMethod  `gorm:"size:24;not null" json:"method"`
	Status         PaymentStatus  `gorm:"size:24;not null;default:pending" json:"status"`
	ExternalRef    string         `gorm:"size:255" json:"external_ref,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy      *string        `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy      *string        `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (Payment) TableName() string { return "payments" }

// PaymentAllocation is an immutable application of part of a payment to a rent charge.
// No soft delete and no updated_at by design.
//
// SQL migrations (critical — not expressible by GORM alone):
//   - CHECK (amount_minor > 0).
//   - Ensure SUM(amount_minor) OVER allocations per payment_id <= payments.amount_minor:
//     use a DEFERRABLE CONSTRAINT TRIGGER or a materialized check after insert/update on payments.
//   - FK payment_id -> payments(id), rent_charge_id -> rent_charges(id).
//   - Enforce payment.organization_id = payment_allocations.organization_id and charge.organization_id match.
type PaymentAllocation struct {
	ID             string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string    `gorm:"type:uuid;not null;index" json:"organization_id"`
	PaymentID      string    `gorm:"type:uuid;not null;index" json:"payment_id"`
	RentChargeID   string    `gorm:"type:uuid;not null;index" json:"rent_charge_id"`
	AmountMinor    int64     `gorm:"not null" json:"amount_minor"`
	CreatedAt      time.Time `json:"created_at"`
}

func (PaymentAllocation) TableName() string { return "payment_allocations" }

// PaymentReminder tracks outbound reminder and collection attempts for unpaid charges.
type PaymentReminder struct {
	ID             string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string         `gorm:"type:uuid;not null;index" json:"organization_id"`
	ChargeID       string         `gorm:"type:uuid;not null;index" json:"charge_id"`
	LeaseID        string         `gorm:"type:uuid;not null;index" json:"lease_id"`
	TenantID       string         `gorm:"type:uuid;not null;index" json:"tenant_id"`
	Level          ReminderLevel  `gorm:"size:32;not null;index" json:"level"`
	Channel        string         `gorm:"size:16;not null" json:"channel"`
	Status         string         `gorm:"size:24;not null;default:sent" json:"status"`
	Note           string         `gorm:"size:512" json:"note,omitempty"`
	SentAt         time.Time      `gorm:"not null;index" json:"sent_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy      *string        `gorm:"type:uuid" json:"created_by,omitempty"`
}

func (PaymentReminder) TableName() string { return "payment_reminders" }
