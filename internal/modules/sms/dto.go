package sms

import "time"

type SendSMSRequest struct {
	To        string `json:"to" binding:"required,min=8,max=32"`
	Message   string `json:"message" binding:"required,min=1,max=612"`
	SenderID  string `json:"senderId" binding:"omitempty,min=3,max=32"`
	Reference string `json:"reference" binding:"omitempty,max=128"`
}

type BulkSendSMSRequest struct {
	Recipients []string `json:"recipients" binding:"required,min=1,max=500,dive,required,min=8,max=32"`
	Message    string   `json:"message" binding:"required,min=1,max=612"`
	SenderID   string   `json:"senderId" binding:"omitempty,min=3,max=32"`
	Reference  string   `json:"reference" binding:"omitempty,max=128"`
}

type SendTemplateSMSRequest struct {
	To        string            `json:"to" binding:"required,min=8,max=32"`
	Template  string            `json:"template" binding:"required,min=1,max=612"`
	Variables map[string]string `json:"variables" binding:"required"`
	SenderID  string            `json:"senderId" binding:"omitempty,min=3,max=32"`
}

type SMSSendItemResult struct {
	To                 string `json:"to"`
	Success            bool   `json:"success"`
	ProviderStatusCode int    `json:"providerStatusCode"`
	ProviderMessage    string `json:"providerMessage"`
	MessageID          string `json:"messageId,omitempty"`
	ReferenceID        string `json:"referenceId,omitempty"`
}

type SendSMSResponse struct {
	Item SMSSendItemResult `json:"item"`
}

type BulkSendSMSResponse struct {
	Total   int                 `json:"total"`
	Sent    int                 `json:"sent"`
	Failed  int                 `json:"failed"`
	Results []SMSSendItemResult `json:"results"`
}

type SMSHealthResponse struct {
	Healthy            bool      `json:"healthy"`
	ProviderStatusCode int       `json:"providerStatusCode"`
	ProviderMessage    string    `json:"providerMessage"`
	CheckedAt          time.Time `json:"checkedAt"`
}
