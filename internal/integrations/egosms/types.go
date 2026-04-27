package egosms

import "encoding/json"

type SendRequest struct {
	To        string
	Message   string
	SenderID  string
	Reference string
}

type SendResult struct {
	Success            bool            `json:"success"`
	ProviderStatusCode int             `json:"provider_status_code"`
	ProviderMessage    string          `json:"provider_message"`
	MessageID          string          `json:"message_id,omitempty"`
	ReferenceID        string          `json:"reference_id,omitempty"`
	RawPayload         json.RawMessage `json:"raw_payload,omitempty"`
}

type HealthResult struct {
	Healthy            bool            `json:"healthy"`
	ProviderStatusCode int             `json:"provider_status_code"`
	ProviderMessage    string          `json:"provider_message"`
	RawPayload         json.RawMessage `json:"raw_payload,omitempty"`
}

// NOTE: EGO's payload format can differ per account plan/version.
// Keep provider mapping isolated here to make updates safe.
type providerSendRequest struct {
	Method   string              `json:"method"`
	UserData providerUserData    `json:"userdata"`
	MsgData  []providerSMSRecord `json:"msgdata"`
}

type providerUserData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type providerSMSRecord struct {
	Number   string `json:"number"`
	Message  string `json:"message"`
	SenderID string `json:"senderid"`
	Priority string `json:"priority"`
}

type providerResponse struct {
	Status    string `json:"Status"`
	Message   string `json:"Message"`
	Code      int    `json:"Code"`
	MessageID string `json:"MessageID"`
	Reference string `json:"Reference"`
}
