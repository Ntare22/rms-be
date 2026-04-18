package buildings

import (
	"time"

	"gorm.io/gorm"
)

// BuildingStatus is the operational lifecycle for a property.
type BuildingStatus string

const (
	BuildingStatusActive   BuildingStatus = "active"
	BuildingStatusInactive BuildingStatus = "inactive"
	BuildingStatusArchived BuildingStatus = "archived"
)

// Building is a physical property belonging to an organization.
//
// SQL migrations:
//   - FK organization_id -> organizations(id) ON DELETE RESTRICT.
//   - Optional: CHECK (status IN ('active','inactive','archived')).
type Building struct {
	ID             string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string         `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	Status         BuildingStatus `gorm:"size:24;not null;default:active;index" json:"status"`
	AddressLine1   string         `gorm:"size:255" json:"address_line1,omitempty"`
	AddressLine2   string         `gorm:"size:255" json:"address_line2,omitempty"`
	City           string         `gorm:"size:120" json:"city,omitempty"`
	State          string         `gorm:"size:120" json:"state,omitempty"`
	PostalCode     string         `gorm:"size:32" json:"postal_code,omitempty"`
	Country        string         `gorm:"size:2;default:US" json:"country"`
	Timezone       string         `gorm:"size:64" json:"timezone,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy      *string        `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy      *string        `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (Building) TableName() string { return "buildings" }
