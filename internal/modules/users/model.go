package users

import (
	"time"

	"gorm.io/gorm"
)

// UserRole is stored as a constrained string; prefer PostgreSQL ENUM or CHECK(role IN (...)) in migrations.
type UserRole string

const (
	UserRoleAdmin           UserRole = "admin"
	UserRoleLandlord        UserRole = "landlord"
	UserRoleManager         UserRole = "manager"
	UserRolePropertyManager UserRole = "property_manager"
	UserRoleAccountant      UserRole = "accountant"
	UserRoleStaff           UserRole = "staff"
	UserRoleTenant          UserRole = "tenant"
)

// UserStatus controls login and lifecycle for org users.
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
	UserStatusInvited  UserStatus = "invited"
)

// User is an account within a single organization.
//
// SQL migrations (critical):
//   - UNIQUE (organization_id, email) — enforced here via composite uniqueIndex; verify in migration.
//   - FK organization_id -> organizations(id), created_by/updated_by -> users(id) (self-ref).
//   - Optional: CHECK (role IN ('admin','landlord','manager','staff')).
//   - Optional: CHECK (status IN ('active','disabled','invited')).
type User struct {
	ID             string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string         `gorm:"type:uuid;not null;uniqueIndex:uq_user_org_email,priority:1" json:"organization_id"`
	Email          string         `gorm:"size:320;not null;uniqueIndex:uq_user_org_email,priority:2" json:"email"`
	PasswordHash   string         `gorm:"size:255;not null" json:"-"`
	Role           UserRole       `gorm:"size:32;not null" json:"role"`
	Status         UserStatus     `gorm:"size:16;not null;default:active" json:"status"`
	FirstName      string         `gorm:"size:120;not null" json:"first_name"`
	LastName       string         `gorm:"size:120" json:"last_name,omitempty"`
	Phone          string         `gorm:"size:64" json:"phone,omitempty"`
	TenantID       *string        `gorm:"type:uuid;index" json:"tenant_id,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy      *string        `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy      *string        `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (User) TableName() string { return "users" }
