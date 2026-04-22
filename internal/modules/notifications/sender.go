package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// EmailMessage is a provider-neutral outbound email payload.
type EmailMessage struct {
	FromName         string
	FromEmail        string
	ToName           string
	ToEmail          string
	Subject          string
	HTMLBody         string
	TextBody         string
	TemplateID       int64
	TemplateLanguage bool
	Variables        map[string]any
}

// EmailSender abstracts outbound email providers.
type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) error
}

// SMSMessage is a provider-neutral SMS payload.
type SMSMessage struct {
	ToPhone string
	Body    string
}

// SMSSender abstracts outbound SMS providers.
type SMSSender interface {
	Send(ctx context.Context, msg SMSMessage) error
}

// NoopEmailSender intentionally drops outbound emails.
type NoopEmailSender struct{}

// Send implements EmailSender.
func (NoopEmailSender) Send(_ context.Context, _ EmailMessage) error { return nil }

// NoopSMSSender intentionally drops outbound SMS.
type NoopSMSSender struct{}

// Send implements SMSSender.
func (NoopSMSSender) Send(_ context.Context, _ SMSMessage) error { return nil }

// MailjetSender sends transactional email via Mailjet v3.1 API.
type MailjetSender struct {
	apiKey    string
	apiSecret string
	client    *http.Client
}

// NewMailjetSender builds a MailjetSender when credentials are provided.
func NewMailjetSender(apiKey, apiSecret string) *MailjetSender {
	return &MailjetSender{
		apiKey:    strings.TrimSpace(apiKey),
		apiSecret: strings.TrimSpace(apiSecret),
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Enabled reports whether sender credentials are configured.
func (m *MailjetSender) Enabled() bool {
	return strings.TrimSpace(m.apiKey) != "" && strings.TrimSpace(m.apiSecret) != ""
}

// Send implements EmailSender.
func (m *MailjetSender) Send(ctx context.Context, msg EmailMessage) error {
	if !m.Enabled() {
		return fmt.Errorf("mailjet credentials are not configured")
	}
	payload := map[string]any{
		"Messages": []map[string]any{
			{
				"From": map[string]string{
					"Email": strings.TrimSpace(msg.FromEmail),
					"Name":  strings.TrimSpace(msg.FromName),
				},
				"To": []map[string]string{
					{
						"Email": strings.TrimSpace(msg.ToEmail),
						"Name":  strings.TrimSpace(msg.ToName),
					},
				},
			},
		},
	}
	mail := payload["Messages"].([]map[string]any)[0]
	if msg.TemplateID > 0 {
		mail["TemplateID"] = msg.TemplateID
		mail["TemplateLanguage"] = true
		if len(msg.Variables) > 0 {
			mail["Variables"] = msg.Variables
		}
	} else {
		mail["Subject"] = strings.TrimSpace(msg.Subject)
		mail["HTMLPart"] = msg.HTMLBody
		mail["TextPart"] = msg.TextBody
		if msg.TemplateLanguage {
			mail["TemplateLanguage"] = true
		}
		if len(msg.Variables) > 0 {
			mail["Variables"] = msg.Variables
		}
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.mailjet.com/v3.1/send", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.SetBasicAuth(m.apiKey, m.apiSecret)
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mailjet send failed with status %d", resp.StatusCode)
	}
	return nil
}
