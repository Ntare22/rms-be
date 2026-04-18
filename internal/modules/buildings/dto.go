package buildings

import "time"

// CreateBuildingRequest is the body for POST /organizations/{id}/buildings.
type CreateBuildingRequest struct {
	Name         string `json:"name" binding:"required,min=1,max=255" example:"Riverside Tower"`
	Status       string `json:"status" binding:"omitempty,oneof=active inactive archived" example:"active"`
	AddressLine1 string `json:"address_line_1" binding:"omitempty,max=255" example:"100 Main St"`
	AddressLine2 string `json:"address_line_2" binding:"omitempty,max=255" example:"Building A"`
	City         string `json:"city" binding:"omitempty,max=120" example:"Austin"`
	State        string `json:"state" binding:"omitempty,max=120" example:"TX"`
	PostalCode   string `json:"postal_code" binding:"omitempty,max=32" example:"78701"`
	Country      string `json:"country" binding:"omitempty,len=2" example:"US"`
	Timezone     string `json:"timezone" binding:"omitempty,max=64" example:"America/Chicago"`
}

// PatchBuildingRequest updates a building; omitted fields are unchanged.
type PatchBuildingRequest struct {
	Name         *string `json:"name" binding:"omitempty,min=1,max=255" example:"Riverside Tower — East"`
	Status       *string `json:"status" binding:"omitempty,oneof=active inactive archived" example:"inactive"`
	AddressLine1 *string `json:"address_line_1" binding:"omitempty,max=255"`
	AddressLine2 *string `json:"address_line_2" binding:"omitempty,max=255"`
	City         *string `json:"city" binding:"omitempty,max=120"`
	State        *string `json:"state" binding:"omitempty,max=120"`
	PostalCode   *string `json:"postal_code" binding:"omitempty,max=32"`
	Country      *string `json:"country" binding:"omitempty,len=2"`
	Timezone     *string `json:"timezone" binding:"omitempty,max=64"`
}

// BuildingResponse is the API projection for a building.
type BuildingResponse struct {
	ID             string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440010"`
	OrganizationID string    `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name           string    `json:"name" example:"Riverside Tower"`
	Status         string    `json:"status" example:"active"`
	AddressLine1   string    `json:"address_line_1,omitempty"`
	AddressLine2   string    `json:"address_line_2,omitempty"`
	City           string    `json:"city,omitempty"`
	State          string    `json:"state,omitempty"`
	PostalCode     string    `json:"postal_code,omitempty"`
	Country        string    `json:"country,omitempty"`
	Timezone       string    `json:"timezone,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedBy      *string   `json:"created_by,omitempty"`
	UpdatedBy      *string   `json:"updated_by,omitempty"`
}

// BuildingListResponse is a paginated list of buildings for an organization.
type BuildingListResponse struct {
	Items    []BuildingResponse `json:"items"`
	Page     int                `json:"page" example:"1"`
	PageSize int                `json:"page_size" example:"20"`
	Total    int64              `json:"total" example:"5"`
}

// ListBuildingsQuery binds list query parameters (pagination + optional status filter).
type ListBuildingsQuery struct {
	Status string `form:"status" binding:"omitempty,oneof=active inactive archived" example:"active"`
}
