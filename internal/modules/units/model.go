package units

import (
	"time"

	"gorm.io/gorm"
)

// UnitStatus is a constrained lifecycle string for a unit.
type UnitStatus string

const (
	UnitStatusVacant      UnitStatus = "vacant"
	UnitStatusOccupied    UnitStatus = "occupied"
	UnitStatusOffline     UnitStatus = "offline"
	UnitStatusMaintenance UnitStatus = "maintenance"
)

// Unit is a leasable space; organization_id is denormalized for org-scoped queries (must match building.organization_id).
//
// SQL migrations (critical):
//   - UNIQUE (building_id, unit_label) — composite uniqueIndex below; enforce same in SQL.
//   - CHECK (organization_id = (SELECT b.organization_id FROM buildings b WHERE b.id = building_id)) — trigger or app validation.
//   - FK building_id -> buildings(id), organization_id -> organizations(id).
//   - Optional: CHECK (status IN (...)).
type Unit struct {
	ID                     string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID         string         `gorm:"type:uuid;not null;index" json:"organization_id"`
	BuildingID             string         `gorm:"type:uuid;not null;uniqueIndex:uq_unit_building_label,priority:1" json:"building_id"`
	UnitLabel              string         `gorm:"size:64;not null;uniqueIndex:uq_unit_building_label,priority:2" json:"unit_label"`
	Bedrooms               *int           `json:"bedrooms,omitempty"`
	DefaultRentAmountMinor *int64         `gorm:"type:bigint" json:"default_rent_amount_minor,omitempty"`
	Currency               string         `gorm:"size:3;not null;default:USD" json:"currency"`
	Status                 UnitStatus     `gorm:"size:24;not null;default:vacant;index" json:"status"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
	DeletedAt              gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy              *string        `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy              *string        `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (Unit) TableName() string { return "units" }
