package leases

import (
	"time"

	"gorm.io/gorm"
)

// LeaseStatus is a constrained string; prefer ENUM or CHECK in PostgreSQL.
type LeaseStatus string

const (
	LeaseStatusDraft           LeaseStatus = "draft"
	LeaseStatusPendingApproval LeaseStatus = "pending_approval"
	LeaseStatusActive          LeaseStatus = "active"
	LeaseStatusRejected        LeaseStatus = "rejected"
	LeaseStatusEnded           LeaseStatus = "ended"
	LeaseStatusCancelled       LeaseStatus = "cancelled"
)

// Lease ties a tenant to a unit for a date range within an organization.
//
// SQL migrations (critical):
//   - EXCLUDE USING gist on (unit_id, tstzrange(start_date, COALESCE(end_date, 'infinity'::timestamptz), '[)')) WHERE is_primary AND status = 'active' (partial index / constraint pattern).
//   - Ensure tenant and unit organization_id match lease.organization_id.
//   - CHECK (end_date IS NULL OR end_date > start_date).
type Lease struct {
	ID                         string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID             string         `gorm:"type:uuid;not null;index" json:"organization_id"`
	UnitID                     string         `gorm:"type:uuid;not null;index" json:"unit_id"`
	TenantID                   string         `gorm:"type:uuid;not null;index" json:"tenant_id"`
	StartDate                  time.Time      `gorm:"type:timestamptz;not null;index" json:"start_date"`
	EndDate                    *time.Time     `gorm:"type:timestamptz" json:"end_date,omitempty"`
	MonthlyRentAmountMinor     int64          `gorm:"type:bigint;not null" json:"monthly_rent_amount_minor"`
	DepositAmountMinor         *int64         `gorm:"type:bigint" json:"deposit_amount_minor,omitempty"`
	Currency                   string         `gorm:"size:3;not null;default:USD" json:"currency"`
	BillingDueDay              *int           `json:"billing_due_day,omitempty"`
	BillingAmountOverrideMinor *int64         `gorm:"type:bigint" json:"billing_amount_override_minor,omitempty"`
	BillingReminderChannel     string         `gorm:"size:16" json:"billing_reminder_channel,omitempty"`
	Status                     LeaseStatus    `gorm:"size:24;not null;default:draft;index" json:"status"`
	IsPrimary                  bool           `gorm:"not null;default:true" json:"is_primary"`
	CreatedAt                  time.Time      `json:"created_at"`
	UpdatedAt                  time.Time      `json:"updated_at"`
	DeletedAt                  gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy                  *string        `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy                  *string        `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (Lease) TableName() string { return "leases" }

// LeaseRenewalOffer stores renewal proposal lifecycle per lease.
type LeaseRenewalOffer struct {
	ID                     string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID         string         `gorm:"type:uuid;not null;index" json:"organization_id"`
	LeaseID                string         `gorm:"type:uuid;not null;index" json:"lease_id"`
	StartDate              time.Time      `gorm:"type:timestamptz;not null" json:"start_date"`
	EndDate                *time.Time     `gorm:"type:timestamptz" json:"end_date,omitempty"`
	MonthlyRentAmountMinor int64          `gorm:"type:bigint;not null" json:"monthly_rent_amount_minor"`
	Currency               string         `gorm:"size:3;not null;default:USD" json:"currency"`
	Status                 string         `gorm:"size:24;not null;default:offered;index" json:"status"`
	DecisionNote           string         `gorm:"size:2000" json:"decision_note,omitempty"`
	DecidedAt              *time.Time     `gorm:"type:timestamptz" json:"decided_at,omitempty"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
	DeletedAt              gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy              *string        `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy              *string        `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (LeaseRenewalOffer) TableName() string { return "lease_renewal_offers" }

// LeaseCloseout persists final settlement and lease exit details.
type LeaseCloseout struct {
	ID                   string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID       string         `gorm:"type:uuid;not null;index" json:"organization_id"`
	LeaseID              string         `gorm:"type:uuid;not null;uniqueIndex" json:"lease_id"`
	MoveOutDate          time.Time      `gorm:"type:timestamptz;not null" json:"move_out_date"`
	FinalSettlementMinor int64          `gorm:"type:bigint;not null;default:0" json:"final_settlement_minor"`
	Currency             string         `gorm:"size:3;not null;default:USD" json:"currency"`
	Notes                string         `gorm:"size:4000" json:"notes,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy            *string        `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy            *string        `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (LeaseCloseout) TableName() string { return "lease_closeouts" }
