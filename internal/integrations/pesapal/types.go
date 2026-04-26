package pesapal

// APIResponseMeta captures common API status metadata.
type APIResponseMeta struct {
	Error   any    `json:"error"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// RequestTokenResponse is returned by Auth/RequestToken endpoint.
type RequestTokenResponse struct {
	Token      string `json:"token"`
	ExpiryDate string `json:"expiryDate"`
	APIResponseMeta
}

// SubmitOrderRequest is provider-neutral order payload for checkout creation.
type SubmitOrderRequest struct {
	ID             string          `json:"id"`
	Currency       string          `json:"currency"`
	Amount         int64           `json:"amount"`
	Description    string          `json:"description"`
	CallbackURL    string          `json:"callback_url"`
	NotificationID string          `json:"notification_id"`
	BillingAddress *BillingAddress `json:"billing_address,omitempty"`
}

// BillingAddress is the billing contact/address block required by Pesapal SubmitOrderRequest.
type BillingAddress struct {
	EmailAddress string `json:"email_address,omitempty"`
	PhoneNumber  string `json:"phone_number,omitempty"`
	CountryCode  string `json:"country_code,omitempty"`
	FirstName    string `json:"first_name,omitempty"`
	MiddleName   string `json:"middle_name,omitempty"`
	LastName     string `json:"last_name,omitempty"`
	Line1        string `json:"line_1,omitempty"`
	Line2        string `json:"line_2,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
	PostalCode   string `json:"postal_code,omitempty"`
	ZipCode      string `json:"zip_code,omitempty"`
}

// SubmitOrderResponse returns redirect/reference details.
type SubmitOrderResponse struct {
	OrderTrackingID string `json:"order_tracking_id"`
	MerchantRef     string `json:"merchant_reference"`
	RedirectURL     string `json:"redirect_url"`
	APIResponseMeta
}

// RegisterIPNRequest creates a Pesapal notification endpoint.
type RegisterIPNRequest struct {
	URL                 string `json:"url"`
	IPNNotificationType string `json:"ipn_notification_type"`
}

// RegisterIPNResponse is returned when registering an IPN URL.
type RegisterIPNResponse struct {
	URL                            string `json:"url"`
	CreatedDate                    string `json:"created_date"`
	IPNID                          string `json:"ipn_id"`
	NotificationType               int    `json:"notification_type"`
	IPNNotificationTypeDescription string `json:"ipn_notification_type_description"`
	IPNStatus                      int    `json:"ipn_status"`
	IPNStatusDescription           string `json:"ipn_status_description"`
	APIResponseMeta
}

// IPNListItem is one row from GetIpnList.
type IPNListItem struct {
	URL         string `json:"url"`
	CreatedDate string `json:"created_date"`
	IPNID       string `json:"ipn_id"`
	Error       any    `json:"error"`
	Status      string `json:"status"`
}

// TransactionStatusResponse is returned by GetTransactionStatus.
type TransactionStatusResponse struct {
	PaymentMethod            string  `json:"payment_method"`
	Amount                   float64 `json:"amount"`
	CreatedDate              string  `json:"created_date"`
	ConfirmationCode         string  `json:"confirmation_code"`
	PaymentStatusDescription string  `json:"payment_status_description"`
	Description              string  `json:"description"`
	PaymentAccount           string  `json:"payment_account"`
	CallbackURL              string  `json:"call_back_url"`
	StatusCode               int     `json:"status_code"`
	MerchantReference        string  `json:"merchant_reference"`
	PaymentStatusCode        string  `json:"payment_status_code"`
	Currency                 string  `json:"currency"`
	APIResponseMeta
}
