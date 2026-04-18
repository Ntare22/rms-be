package tenants

import (
	"time"

	"gorm.io/gorm"
)

// TenantStatus is the lifecycle for a tenant record.
type TenantStatus string

const (
	TenantStatusActive   TenantStatus = "active"
	TenantStatusInactive TenantStatus = "inactive"
	TenantStatusArchived TenantStatus = "archived"
)

// Tenant is a person or company leasing from the organization.
//
// SQL migrations:
//   - FK organization_id -> organizations(id).
//   - Optional: unique (organization_id, email) where email is not null.
//   - Optional: CHECK constraints on status and booleans.
type Tenant struct {
	ID             string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string         `gorm:"type:uuid;not null;index" json:"organization_id"`
	FirstName      string         `gorm:"size:120;not null" json:"first_name"`
	LastName       string         `gorm:"size:120" json:"last_name,omitempty"`
	FullName       string         `gorm:"size:512;not null" json:"full_name"`
	Email          string         `gorm:"size:320;index" json:"email,omitempty"`
	Phone          string         `gorm:"size:64" json:"phone,omitempty"`
	EmailOptIn     bool           `gorm:"not null;default:false" json:"email_opt_in"`
	SmsOptIn       bool           `gorm:"not null;default:false" json:"sms_opt_in"`
	SmsVerified    bool           `gorm:"not null;default:false" json:"sms_verified"`
	Locale         string         `gorm:"size:32" json:"locale,omitempty"`
	Timezone       string         `gorm:"size:64" json:"timezone,omitempty"`
	Status         TenantStatus   `gorm:"size:24;not null;default:active;index" json:"status"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy      *string        `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy      *string        `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (Tenant) TableName() string { return "tenants" }
