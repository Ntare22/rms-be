package users

import "time"

// CreateUserRequest is the body for POST /organizations/{id}/users.
// Password is required unless status is `invited` (a random credential is generated server-side for invited users).
type CreateUserRequest struct {
	Email     string `json:"email" binding:"required,email,max=320" example:"alex@acme.com"`
	Password  string `json:"password" binding:"omitempty,min=8,max=72" example:"SecurePass123"`
	FirstName string `json:"first_name" binding:"required,min=1,max=120" example:"Alex"`
	LastName  string `json:"last_name" binding:"omitempty,max=120" example:"Rivera"`
	Phone     string `json:"phone" binding:"omitempty,max=64" example:"+1-555-0199"`
	Role      string `json:"role" binding:"required,oneof=admin landlord manager staff" example:"staff"`
	Status    string `json:"status" binding:"omitempty,oneof=active disabled invited" example:"active"`
}

// PatchUserRequest updates a user. Omitted fields are unchanged.
type PatchUserRequest struct {
	Email     *string `json:"email" binding:"omitempty,email,max=320" example:"alex.new@acme.com"`
	Password  *string `json:"password" binding:"omitempty,min=8,max=72" example:"NewSecurePass456"`
	FirstName *string `json:"first_name" binding:"omitempty,min=1,max=120" example:"Alex"`
	LastName  *string `json:"last_name" binding:"omitempty,max=120" example:"Rivera"`
	Phone     *string `json:"phone" binding:"omitempty,max=64" example:"+1-555-0199"`
	Role      *string `json:"role" binding:"omitempty,oneof=admin landlord manager staff" example:"manager"`
	Status    *string `json:"status" binding:"omitempty,oneof=active disabled invited" example:"disabled"`
}

// UserResponse is a safe projection (no password hash).
type UserResponse struct {
	ID             string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440001"`
	OrganizationID string    `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email          string    `json:"email" example:"alex@acme.com"`
	Role           string    `json:"role" example:"staff"`
	Status         string    `json:"status" example:"active"`
	FirstName      string    `json:"first_name" example:"Alex"`
	LastName       string    `json:"last_name" example:"Rivera"`
	Phone          string    `json:"phone,omitempty" example:"+1-555-0199"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedBy      *string   `json:"created_by,omitempty"`
	UpdatedBy      *string   `json:"updated_by,omitempty"`
}

// UserListResponse is paginated list payload (wrapped in the standard envelope by handlers).
type UserListResponse struct {
	Items    []UserResponse `json:"items"`
	Page     int            `json:"page" example:"1"`
	PageSize int            `json:"page_size" example:"20"`
	Total    int64          `json:"total" example:"42"`
}
