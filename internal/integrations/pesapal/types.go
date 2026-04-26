package pesapal

// RequestTokenResponse is returned by Auth/RequestToken endpoint.
type RequestTokenResponse struct {
	Token string `json:"token"`
}

// SubmitOrderRequest is provider-neutral order payload for checkout creation.
type SubmitOrderRequest struct {
	ID               string `json:"id"`
	Currency         string `json:"currency"`
	Amount           int64  `json:"amount"`
	Description      string `json:"description"`
	CallbackURL      string `json:"callback_url"`
	NotificationID   string `json:"notification_id"`
	BillingEmail     string `json:"billing_email,omitempty"`
	BillingPhone     string `json:"billing_phone,omitempty"`
	BillingFirstName string `json:"billing_first_name,omitempty"`
	BillingLastName  string `json:"billing_last_name,omitempty"`
}

// SubmitOrderResponse returns redirect/reference details.
type SubmitOrderResponse struct {
	OrderTrackingID string `json:"order_tracking_id"`
	MerchantRef     string `json:"merchant_reference"`
	RedirectURL     string `json:"redirect_url"`
}
