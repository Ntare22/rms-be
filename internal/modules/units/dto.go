package units

import "time"

// CreateUnitRequest is the body for POST .../buildings/{buildingId}/units.
type CreateUnitRequest struct {
	UnitLabel              string `json:"unit_label" binding:"required,min=1,max=64" example:"101-A"`
	Bedrooms               *int   `json:"bedrooms" binding:"omitempty,min=0,max=32" example:"2"`
	DefaultRentAmountMinor *int64 `json:"default_rent_amount_minor" binding:"omitempty,min=0" example:"185000"`
	Currency               string `json:"currency" binding:"omitempty,len=3" example:"USD"`
	Status                 string `json:"status" binding:"omitempty,oneof=vacant occupied offline maintenance" example:"vacant"`
}

// PatchUnitRequest updates a unit; omitted fields are unchanged.
type PatchUnitRequest struct {
	UnitLabel              *string `json:"unit_label" binding:"omitempty,min=1,max=64" example:"101-B"`
	Bedrooms               *int    `json:"bedrooms" binding:"omitempty,min=0,max=32"`
	DefaultRentAmountMinor *int64  `json:"default_rent_amount_minor" binding:"omitempty,min=0"`
	Currency               *string `json:"currency" binding:"omitempty,len=3"`
	Status                 *string `json:"status" binding:"omitempty,oneof=vacant occupied offline maintenance"`
}

// OccupancySummary is derived from an active lease for the unit (optional on responses).
type OccupancySummary struct {
	HasActiveLease bool    `json:"has_active_lease" example:"true"`
	ActiveLeaseID  *string `json:"active_lease_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440020"`
	TenantID       *string `json:"tenant_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440030"`
}

// UnitResponse is the API projection for a unit.
type UnitResponse struct {
	ID                     string            `json:"id" example:"550e8400-e29b-41d4-a716-446655440011"`
	OrganizationID         string            `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	BuildingID             string            `json:"building_id" example:"550e8400-e29b-41d4-a716-446655440010"`
	UnitLabel              string            `json:"unit_label" example:"101-A"`
	Bedrooms               *int              `json:"bedrooms,omitempty" example:"2"`
	DefaultRentAmountMinor *int64            `json:"default_rent_amount_minor,omitempty" example:"185000"`
	Currency               string            `json:"currency" example:"USD"`
	Status                 string            `json:"status" example:"vacant"`
	Occupancy              *OccupancySummary `json:"occupancy,omitempty"`
	CreatedAt              time.Time         `json:"created_at"`
	UpdatedAt              time.Time         `json:"updated_at"`
	CreatedBy              *string           `json:"created_by,omitempty"`
	UpdatedBy              *string           `json:"updated_by,omitempty"`
}

// UnitListResponse is a paginated list of units for a building.
type UnitListResponse struct {
	Items    []UnitResponse `json:"items"`
	Page     int            `json:"page" example:"1"`
	PageSize int            `json:"page_size" example:"20"`
	Total    int64          `json:"total" example:"12"`
}

// ListUnitsQuery binds list query parameters (see `include_occupancy` query param in handler / Swagger).
type ListUnitsQuery struct {
	Status string `form:"status" binding:"omitempty,oneof=vacant occupied offline maintenance" example:"vacant"`
}
