package pesapal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	if !isSuccessStatus(out.Status) {
		return "", fmt.Errorf("pesapal token response rejected: %s", strings.TrimSpace(out.Status))
	}
	if out.Error != nil {
		return "", fmt.Errorf("pesapal token response error: %v", out.Error)
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
	req.Header.Set("Accept", "application/json")
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
	if !isSuccessStatus(out.Status) {
		return nil, fmt.Errorf("pesapal submit order rejected: %s", strings.TrimSpace(out.Status))
	}
	if out.Error != nil {
		return nil, fmt.Errorf("pesapal submit order response error: %v", out.Error)
	}
	return &out, nil
}

func (h *HTTPClient) RegisterIPN(ctx context.Context, url, notificationType string) (*RegisterIPNResponse, error) {
	token, err := h.RequestToken(ctx)
	if err != nil {
		return nil, err
	}
	t := strings.TrimSpace(strings.ToUpper(notificationType))
	if t == "" {
		t = "GET"
	}
	reqBody := RegisterIPNRequest{
		URL:                 strings.TrimSpace(url),
		IPNNotificationType: t,
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(h.cfg.BaseURL, "/")+"/URLSetup/RegisterIPN", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pesapal register ipn failed status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out RegisterIPNResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if !isSuccessStatus(out.Status) {
		return nil, fmt.Errorf("pesapal register ipn rejected: %s", strings.TrimSpace(out.Status))
	}
	if out.Error != nil {
		return nil, fmt.Errorf("pesapal register ipn response error: %v", out.Error)
	}
	return &out, nil
}

func (h *HTTPClient) GetIPNList(ctx context.Context) ([]IPNListItem, error) {
	token, err := h.RequestToken(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(h.cfg.BaseURL, "/")+"/URLSetup/GetIpnList", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pesapal get ipn list failed status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out []IPNListItem
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (h *HTTPClient) GetTransactionStatus(ctx context.Context, orderTrackingID string) (*TransactionStatusResponse, error) {
	token, err := h.RequestToken(ctx)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(orderTrackingID)
	if id == "" {
		return nil, fmt.Errorf("orderTrackingId is required")
	}
	u := strings.TrimRight(h.cfg.BaseURL, "/") + "/Transactions/GetTransactionStatus?orderTrackingId=" + url.QueryEscape(id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pesapal get transaction status failed status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out TransactionStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if !isSuccessStatus(out.Status) {
		return nil, fmt.Errorf("pesapal get transaction status rejected: %s", strings.TrimSpace(out.Status))
	}
	return &out, nil
}

func isSuccessStatus(s string) bool {
	switch strings.TrimSpace(s) {
	case "", "200", "201":
		return true
	default:
		return false
	}
}
