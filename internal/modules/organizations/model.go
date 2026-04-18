package organizations

import "time"

// Organization is the top-level tenant of the RMS (property manager / landlord company).
//
// SQL migrations (not expressible or incomplete in GORM alone):
//   - Optional: CHECK on slug format, reserved slugs.
//   - RLS policies if row-level security is enabled per organization_id.
type Organization struct {
	ID        string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	Slug      string    `gorm:"size:128;not null;uniqueIndex:uq_org_slug" json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy *string   `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy *string   `gorm:"type:uuid" json:"updated_by,omitempty"`

	// Profile — landlord / property-management company (SaaS tenant)
	LegalName           string `gorm:"size:255" json:"legal_name,omitempty"`
	BillingEmail        string `gorm:"size:320" json:"billing_email,omitempty"`
	Phone               string `gorm:"size:64" json:"phone,omitempty"`
	Website             string `gorm:"size:512" json:"website,omitempty"`
	AddressLine1        string `gorm:"size:255" json:"address_line1,omitempty"`
	AddressLine2        string `gorm:"size:255" json:"address_line2,omitempty"`
	City                string `gorm:"size:120" json:"city,omitempty"`
	Region              string `gorm:"size:120" json:"region,omitempty"`
	PostalCode          string `gorm:"size:32" json:"postal_code,omitempty"`
	Country             string `gorm:"size:2" json:"country,omitempty"`          // ISO 3166-1 alpha-2
	DefaultCurrency     string `gorm:"size:3" json:"default_currency,omitempty"` // ISO 4217
	Timezone            string `gorm:"size:64" json:"timezone,omitempty"`        // IANA, e.g. America/New_York
	TaxID               string `gorm:"size:64" json:"tax_id,omitempty"`
	CompanyRegistration string `gorm:"size:128" json:"company_registration,omitempty"`
}

func (Organization) TableName() string { return "organizations" }
