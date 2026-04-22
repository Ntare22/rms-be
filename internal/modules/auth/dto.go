package auth

import "time"

// RegisterRequest bootstraps an organization and its first user (intended for controlled / admin use in production).
type RegisterRequest struct {
	OrganizationName string `json:"organization_name" binding:"required,min=2,max=255" example:"Acme Property Management"`
	OrganizationSlug string `json:"organization_slug" binding:"required,min=2,max=128" example:"acme-prop"`
	Email            string `json:"email" binding:"required,email" example:"owner@acme.com"`
	Password         string `json:"password" binding:"required,min=8,max=72" example:"securePass123"`
	FirstName        string `json:"first_name" binding:"required,min=1,max=120" example:"Jordan"`
	LastName         string `json:"last_name" binding:"omitempty,max=120" example:"Lee"`
	Role             string `json:"role" binding:"omitempty,oneof=admin landlord manager" example:"admin"`
}

// LoginRequest authenticates a user within a single organization.
type LoginRequest struct {
	Email            string `json:"email" binding:"required,email" example:"owner@acme.com"`
	Password         string `json:"password" binding:"required" example:"securePass123"`
	OrganizationID   string `json:"organization_id" binding:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	OrganizationSlug string `json:"organization_slug" binding:"omitempty,max=128" example:"acme-prop"`
}

// RefreshRequest exchanges a refresh token for a new token pair.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"eyJhbGciOiJIUzI1NiIs..."`
}

// PasswordSetupRequest starts an invite password-setup flow for an existing user.
type PasswordSetupRequest struct {
	OrganizationID string `json:"organization_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email          string `json:"email" binding:"required,email" example:"tenant@acme.com"`
}

// PasswordSetupConfirmRequest finalizes password setup from invite token.
type PasswordSetupConfirmRequest struct {
	Token    string `json:"token" binding:"required,min=20" example:"ABCD..."`
	Password string `json:"password" binding:"required,min=8,max=72" example:"securePass123"`
}

// UserSummary is a non-sensitive projection for API responses.
type UserSummary struct {
	ID             string `json:"id" example:"550e8400-e29b-41d4-a716-446655440001"`
	OrganizationID string `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email          string `json:"email" example:"owner@acme.com"`
	FirstName      string `json:"first_name" example:"Jordan"`
	LastName       string `json:"last_name" example:"Lee"`
	Role           string `json:"role" example:"admin"`
}

// AuthSessionResponse is returned from register and login.
type AuthSessionResponse struct {
	User             UserSummary `json:"user"`
	AccessToken      string      `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	RefreshToken     string      `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	AccessExpiresAt  time.Time   `json:"access_expires_at"`
	RefreshExpiresAt time.Time   `json:"refresh_expires_at"`
}

// RefreshResponse is returned from refresh.
type RefreshResponse struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

// MeResponse returns the current user profile plus token claims context.
type MeResponse struct {
	User                 UserSummary `json:"user"`
	ClaimsOrganizationID string      `json:"claims_organization_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ClaimsRole           string      `json:"claims_role" example:"admin"`
}

// StatusResponse is a generic status payload for simple auth endpoints.
type StatusResponse struct {
	Status string `json:"status" example:"ok"`
}
