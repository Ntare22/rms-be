package pesapal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HTTPClient calls PesaPal REST APIs.
type HTTPClient struct {
	cfg    Config
	client *http.Client
}

func NewHTTPClient(cfg Config) *HTTPClient {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &HTTPClient{
		cfg:    cfg,
		client: &http.Client{Timeout: timeout},
	}
}

func (h *HTTPClient) RequestToken(ctx context.Context) (string, error) {
	payload := map[string]string{
		"consumer_key":    strings.TrimSpace(h.cfg.ConsumerKey),
		"consumer_secret": strings.TrimSpace(h.cfg.ConsumerSecret),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(h.cfg.BaseURL, "/")+"/Auth/RequestToken", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("pesapal token request failed with status %d", resp.StatusCode)
	}
	var out RequestTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.Token), nil
}

func (h *HTTPClient) SubmitOrder(ctx context.Context, in SubmitOrderRequest) (*SubmitOrderResponse, error) {
	token, err := h.RequestToken(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.NotificationID) == "" {
		in.NotificationID = h.cfg.IPNID
	}
	b, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(h.cfg.BaseURL, "/")+"/Transactions/SubmitOrderRequest", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("pesapal submit order failed with status %d", resp.StatusCode)
	}
	var out SubmitOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
