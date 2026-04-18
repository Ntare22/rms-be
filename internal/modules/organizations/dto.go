package organizations

import "time"

// CreateOrganizationRequest is the body for POST /organizations (admin only).
type CreateOrganizationRequest struct {
	Name                string `json:"name" binding:"required,min=2,max=255" example:"Acme Property Management"`
	Slug                string `json:"slug" binding:"required,min=2,max=128" example:"acme-prop"`
	LegalName           string `json:"legal_name" binding:"omitempty,max=255" example:"Acme Property Management LLC"`
	BillingEmail        string `json:"billing_email" binding:"omitempty,email,max=320" example:"billing@acme.com"`
	Phone               string `json:"phone" binding:"omitempty,max=64" example:"+1-555-0100"`
	Website             string `json:"website" binding:"omitempty,url,max=512" example:"https://acme.com"`
	AddressLine1        string `json:"address_line1" binding:"omitempty,max=255" example:"1 Main St"`
	AddressLine2        string `json:"address_line2" binding:"omitempty,max=255" example:"Suite 200"`
	City                string `json:"city" binding:"omitempty,max=120" example:"Austin"`
	Region              string `json:"region" binding:"omitempty,max=120" example:"TX"`
	PostalCode          string `json:"postal_code" binding:"omitempty,max=32" example:"78701"`
	Country             string `json:"country" binding:"omitempty,len=2" example:"US"`
	DefaultCurrency     string `json:"default_currency" binding:"omitempty,len=3" example:"USD"`
	Timezone            string `json:"timezone" binding:"omitempty,max=64" example:"America/Chicago"`
	TaxID               string `json:"tax_id" binding:"omitempty,max=64" example:"12-3456789"`
	CompanyRegistration string `json:"company_registration" binding:"omitempty,max=128" example:"DE-12345"`
}

// PatchOrganizationRequest updates an organization. Slug is immutable after create.
// Admin may update any organization; landlord and manager may update only their own (staff cannot PATCH).
type PatchOrganizationRequest struct {
	Name                *string `json:"name" binding:"omitempty,min=2,max=255"`
	LegalName           *string `json:"legal_name" binding:"omitempty,max=255"`
	BillingEmail        *string `json:"billing_email" binding:"omitempty,email,max=320"`
	Phone               *string `json:"phone" binding:"omitempty,max=64"`
	Website             *string `json:"website" binding:"omitempty,max=512"`
	AddressLine1        *string `json:"address_line1" binding:"omitempty,max=255"`
	AddressLine2        *string `json:"address_line2" binding:"omitempty,max=255"`
	City                *string `json:"city" binding:"omitempty,max=120"`
	Region              *string `json:"region" binding:"omitempty,max=120"`
	PostalCode          *string `json:"postal_code" binding:"omitempty,max=32"`
	Country             *string `json:"country" binding:"omitempty,len=2"`
	DefaultCurrency     *string `json:"default_currency" binding:"omitempty,len=3"`
	Timezone            *string `json:"timezone" binding:"omitempty,max=64"`
	TaxID               *string `json:"tax_id" binding:"omitempty,max=64"`
	CompanyRegistration *string `json:"company_registration" binding:"omitempty,max=128"`
}

// OrganizationResponse is the API projection for an organization (no raw GORM-only fields).
type OrganizationResponse struct {
	ID                  string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name                string    `json:"name" example:"Acme Property Management"`
	Slug                string    `json:"slug" example:"acme-prop"`
	LegalName           string    `json:"legal_name,omitempty"`
	BillingEmail        string    `json:"billing_email,omitempty"`
	Phone               string    `json:"phone,omitempty"`
	Website             string    `json:"website,omitempty"`
	AddressLine1        string    `json:"address_line1,omitempty"`
	AddressLine2        string    `json:"address_line2,omitempty"`
	City                string    `json:"city,omitempty"`
	Region              string    `json:"region,omitempty"`
	PostalCode          string    `json:"postal_code,omitempty"`
	Country             string    `json:"country,omitempty"`
	DefaultCurrency     string    `json:"default_currency,omitempty"`
	Timezone            string    `json:"timezone,omitempty"`
	TaxID               string    `json:"tax_id,omitempty"`
	CompanyRegistration string    `json:"company_registration,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	CreatedBy           *string   `json:"created_by,omitempty"`
	UpdatedBy           *string   `json:"updated_by,omitempty"`
}
