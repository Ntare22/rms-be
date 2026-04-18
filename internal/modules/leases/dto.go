package leases

import "time"

// CreateLeaseRequest is the body for POST /organizations/{id}/leases.
type CreateLeaseRequest struct {
	TenantID               string     `json:"tenant_id" binding:"required,uuid"`
	UnitID                 string     `json:"unit_id" binding:"required,uuid"`
	StartDate              time.Time  `json:"start_date" binding:"required"`
	EndDate                *time.Time `json:"end_date"`
	MonthlyRentAmountMinor int64      `json:"monthly_rent_amount_minor" binding:"required,min=0"`
	DepositAmountMinor     *int64     `json:"deposit_amount_minor" binding:"omitempty,min=0"`
	Currency               string     `json:"currency" binding:"omitempty,len=3" example:"USD"`
	Status                 string     `json:"status" binding:"omitempty,oneof=draft active ended cancelled" example:"draft"`
	IsPrimary              *bool      `json:"is_primary"`
}

// PatchLeaseRequest updates a lease (partial).
type PatchLeaseRequest struct {
	StartDate              *time.Time `json:"start_date"`
	EndDate                *time.Time `json:"end_date"`
	MonthlyRentAmountMinor *int64     `json:"monthly_rent_amount_minor" binding:"omitempty,min=0"`
	DepositAmountMinor     *int64     `json:"deposit_amount_minor" binding:"omitempty,min=0"`
	Currency               *string    `json:"currency" binding:"omitempty,len=3"`
	Status                 *string    `json:"status" binding:"omitempty,oneof=draft active ended cancelled"`
	IsPrimary              *bool      `json:"is_primary"`
}

// EndLeaseRequest is the body for POST .../leases/{leaseId}/end. If `end_date` is omitted, the server uses the current time (must be after `start_date`).
type EndLeaseRequest struct {
	EndDate *time.Time `json:"end_date"`
	Note    string     `json:"note" binding:"omitempty,max=2000"`
}

// LeaseResponse is the API projection for a lease.
type LeaseResponse struct {
	ID                     string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440050"`
	OrganizationID         string     `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID               string     `json:"tenant_id"`
	UnitID                 string     `json:"unit_id"`
	StartDate              time.Time  `json:"start_date"`
	EndDate                *time.Time `json:"end_date,omitempty"`
	MonthlyRentAmountMinor int64      `json:"monthly_rent_amount_minor" example:"150000"`
	DepositAmountMinor     *int64     `json:"deposit_amount_minor,omitempty"`
	Currency               string     `json:"currency" example:"USD"`
	Status                 string     `json:"status" example:"active"`
	IsPrimary              bool       `json:"is_primary" example:"true"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	CreatedBy              *string    `json:"created_by,omitempty"`
	UpdatedBy              *string    `json:"updated_by,omitempty"`
}

// LeaseListResponse is paginated.
type LeaseListResponse struct {
	Items    []LeaseResponse `json:"items"`
	Page     int             `json:"page" example:"1"`
	PageSize int             `json:"page_size" example:"20"`
	Total    int64           `json:"total" example:"50"`
}

// ListLeasesQuery binds list filters.
type ListLeasesQuery struct {
	Status     string `form:"status" binding:"omitempty,oneof=draft active ended cancelled"`
	BuildingID string `form:"building_id" binding:"omitempty,uuid"`
	UnitID     string `form:"unit_id" binding:"omitempty,uuid"`
	TenantID   string `form:"tenant_id" binding:"omitempty,uuid"`
	ActiveOn   string `form:"active_on" binding:"omitempty" example:"2026-04-18"`
}
