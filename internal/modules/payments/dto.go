package payments

import "time"

// InitiatePaymentRequest creates a placeholder payment intent.
type InitiatePaymentRequest struct {
	LeaseID     string `json:"lease_id" binding:"required,uuid"`
	AmountMinor int64  `json:"amount_minor" binding:"required,min=1"`
	Currency    string `json:"currency" binding:"omitempty,len=3"`
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

// PaymentSummaryPoint represents a monthly aggregate.
type PaymentSummaryPoint struct {
	Month               string `json:"month" example:"2026-04"`
	CollectedMinor      int64  `json:"collected_minor"`
	PendingMinor        int64  `json:"pending_minor"`
	OutstandingDueMinor int64  `json:"outstanding_due_minor"`
}

// PaymentSummaryResponse provides monthly totals.
type PaymentSummaryResponse struct {
	Items []PaymentSummaryPoint `json:"items"`
}

// PaymentMethodSplitItem represents totals by payment method.
type PaymentMethodSplitItem struct {
	Method         string `json:"method" example:"card"`
	Count          int64  `json:"count"`
	CollectedMinor int64  `json:"collected_minor"`
}

// PaymentMethodSplitResponse is the method distribution.
type PaymentMethodSplitResponse struct {
	Items []PaymentMethodSplitItem `json:"items"`
}

// ReminderCandidate is an unpaid charge candidate for reminder send.
type ReminderCandidate struct {
	ChargeID       string    `json:"charge_id"`
	LeaseID        string    `json:"lease_id"`
	TenantID       string    `json:"tenant_id"`
	TenantName     string    `json:"tenant_name"`
	TenantEmail    string    `json:"tenant_email,omitempty"`
	TenantPhone    string    `json:"tenant_phone,omitempty"`
	AmountMinor    int64     `json:"amount_minor"`
	Currency       string    `json:"currency"`
	DueAt          time.Time `json:"due_at"`
	SuggestedLevel string    `json:"suggested_level"`
	Channel        string    `json:"channel"`
}

// ReminderCandidatesResponse returns candidates for reminders.
type ReminderCandidatesResponse struct {
	Items []ReminderCandidate `json:"items"`
}

// ReminderHistoryItem is a reminder log row.
type ReminderHistoryItem struct {
	ID       string    `json:"id"`
	ChargeID string    `json:"charge_id"`
	LeaseID  string    `json:"lease_id"`
	TenantID string    `json:"tenant_id"`
	Level    string    `json:"level"`
	Channel  string    `json:"channel"`
	Status   string    `json:"status"`
	Note     string    `json:"note,omitempty"`
	SentAt   time.Time `json:"sent_at"`
}

// ReminderHistoryResponse returns reminder history.
type ReminderHistoryResponse struct {
	Items []ReminderHistoryItem `json:"items"`
}

// SendRemindersRequest requests bulk reminder sends.
type SendRemindersRequest struct {
	ChargeIDs []string `json:"charge_ids" binding:"required,min=1,dive,uuid"`
	Level     string   `json:"level" binding:"omitempty,oneof=reminder_1 reminder_2 final_notice"`
	Channel   string   `json:"channel" binding:"omitempty,oneof=email sms"`
	Note      string   `json:"note" binding:"omitempty,max=512"`
}

// SendRemindersResponse contains send result.
type SendRemindersResponse struct {
	Sent      int `json:"sent"`
	Skipped   int `json:"skipped"`
	Requested int `json:"requested"`
}
