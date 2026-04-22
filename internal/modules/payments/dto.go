package payments

import "time"

// InitiatePaymentRequest creates a placeholder payment intent.
type InitiatePaymentRequest struct {
	LeaseID     string `json:"lease_id" binding:"required,uuid"`
	AmountMinor int64  `json:"amount_minor" binding:"required,min=1"`
	Currency    string `json:"currency" binding:"omitempty,len=3" example:"USD"`
	Method      string `json:"method" binding:"omitempty,oneof=ach card cash check wire other" example:"card"`
}

// PaymentResponse is the API projection for initiated payments.
type PaymentResponse struct {
	ID          string    `json:"id"`
	LeaseID     string    `json:"lease_id"`
	AmountMinor int64     `json:"amount_minor"`
	Currency    string    `json:"currency"`
	Method      string    `json:"method"`
	Status      string    `json:"status"`
	ExternalRef string    `json:"external_ref"`
	NextAction  string    `json:"next_action"`
	CreatedAt   time.Time `json:"created_at"`
}
