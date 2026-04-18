package tenants

import "time"

// CreateTenantRequest is the body for POST /organizations/{id}/tenants.
type CreateTenantRequest struct {
	FirstName   string `json:"first_name" binding:"required,min=1,max=120" example:"Jordan"`
	LastName    string `json:"last_name" binding:"omitempty,max=120" example:"Lee"`
	FullName    string `json:"full_name" binding:"omitempty,max=512" example:"Jordan Lee"`
	Email       string `json:"email" binding:"omitempty,email,max=320" example:"jordan.lee@example.com"`
	Phone       string `json:"phone" binding:"omitempty,max=32" example:"+1-555-0100"`
	EmailOptIn  bool   `json:"email_opt_in" example:"true"`
	SmsOptIn    bool   `json:"sms_opt_in" example:"false"`
	SmsVerified bool   `json:"sms_verified" example:"false"`
	Locale      string `json:"locale" binding:"omitempty,max=32" example:"en-US"`
	Timezone    string `json:"timezone" binding:"omitempty,max=64" example:"America/Chicago"`
	Status      string `json:"status" binding:"omitempty,oneof=active inactive archived" example:"active"`
}

// PatchTenantRequest updates a tenant; omitted JSON fields are left unchanged (use pointers where partial update is needed — booleans use wrapper in JSON or send full object for toggles).
type PatchTenantRequest struct {
	FirstName   *string `json:"first_name" binding:"omitempty,min=1,max=120"`
	LastName    *string `json:"last_name" binding:"omitempty,max=120"`
	FullName    *string `json:"full_name" binding:"omitempty,max=512"`
	Email       *string `json:"email" binding:"omitempty,email,max=320"`
	Phone       *string `json:"phone" binding:"omitempty,max=32"`
	EmailOptIn  *bool   `json:"email_opt_in"`
	SmsOptIn    *bool   `json:"sms_opt_in"`
	SmsVerified *bool   `json:"sms_verified"`
	Locale      *string `json:"locale" binding:"omitempty,max=32"`
	Timezone    *string `json:"timezone" binding:"omitempty,max=64"`
	Status      *string `json:"status" binding:"omitempty,oneof=active inactive archived"`
}

// CurrentUnitSummary is derived from an active lease (optional on responses).
type CurrentUnitSummary struct {
	UnitID    string `json:"unit_id" example:"550e8400-e29b-41d4-a716-446655440011"`
	UnitLabel string `json:"unit_label" example:"12B"`
}

// TenantResponse is the API projection for a tenant.
type TenantResponse struct {
	ID               string              `json:"id" example:"550e8400-e29b-41d4-a716-446655440040"`
	OrganizationID   string              `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FirstName        string              `json:"first_name" example:"Jordan"`
	LastName         string              `json:"last_name,omitempty" example:"Lee"`
	FullName         string              `json:"full_name" example:"Jordan Lee"`
	Email            string              `json:"email,omitempty" example:"jordan.lee@example.com"`
	Phone            string              `json:"phone,omitempty" example:"+15550100"`
	EmailOptIn       bool                `json:"email_opt_in"`
	SmsOptIn         bool                `json:"sms_opt_in"`
	SmsVerified      bool                `json:"sms_verified"`
	Locale           string              `json:"locale,omitempty" example:"en-US"`
	Timezone         string              `json:"timezone,omitempty" example:"America/Chicago"`
	Status           string              `json:"status" example:"active"`
	ActiveLeaseCount *int                `json:"active_lease_count,omitempty" example:"1"`
	CurrentUnit      *CurrentUnitSummary `json:"current_unit,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
	CreatedBy        *string             `json:"created_by,omitempty"`
	UpdatedBy        *string             `json:"updated_by,omitempty"`
}

// TenantListResponse is a paginated list of tenants.
type TenantListResponse struct {
	Items    []TenantResponse `json:"items"`
	Page     int              `json:"page" example:"1"`
	PageSize int              `json:"page_size" example:"20"`
	Total    int64            `json:"total" example:"100"`
}

// ListTenantsQuery binds list filters (search + pagination params documented on handler).
type ListTenantsQuery struct {
	Name   string `form:"name" example:"Jordan"`
	Email  string `form:"email" example:"lee@"`
	Phone  string `form:"phone" example:"555"`
	Status string `form:"status" binding:"omitempty,oneof=active inactive archived" example:"active"`
}
