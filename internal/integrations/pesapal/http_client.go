package pesapal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	h.debugf("request_token start base_url=%s consumer_key=%s consumer_secret=%s", strings.TrimSpace(h.cfg.BaseURL), maskSecret(strings.TrimSpace(h.cfg.ConsumerKey)), maskSecret(strings.TrimSpace(h.cfg.ConsumerSecret)))
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
	h.debugf("request_token success expiry=%s status=%s token=%s", strings.TrimSpace(out.ExpiryDate), strings.TrimSpace(out.Status), maskSecret(strings.TrimSpace(out.Token)))
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
	h.debugf("submit_order start id=%s amount=%d currency=%s callback_url=%s notification_id=%s", strings.TrimSpace(in.ID), in.Amount, strings.TrimSpace(in.Currency), strings.TrimSpace(in.CallbackURL), strings.TrimSpace(in.NotificationID))
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
	h.debugf("submit_order success order_tracking_id=%s merchant_ref=%s redirect_url=%s", strings.TrimSpace(out.OrderTrackingID), strings.TrimSpace(out.MerchantRef), strings.TrimSpace(out.RedirectURL))
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
	h.debugf("register_ipn start url=%s notification_type=%s", strings.TrimSpace(reqBody.URL), strings.TrimSpace(reqBody.IPNNotificationType))
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
	h.debugf("register_ipn success ipn_id=%s status=%s ipn_status=%s", strings.TrimSpace(out.IPNID), strings.TrimSpace(out.Status), strings.TrimSpace(out.IPNStatusDescription))
	return &out, nil
}

func (h *HTTPClient) GetIPNList(ctx context.Context) ([]IPNListItem, error) {
	h.debugf("get_ipn_list start")
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
	h.debugf("get_ipn_list success count=%d", len(out))
	return out, nil
}

func (h *HTTPClient) GetTransactionStatus(ctx context.Context, orderTrackingID string) (*TransactionStatusResponse, error) {
	h.debugf("get_transaction_status start order_tracking_id=%s", strings.TrimSpace(orderTrackingID))
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
	h.debugf("get_transaction_status success status=%s status_code=%d merchant_reference=%s", strings.TrimSpace(out.PaymentStatusDescription), out.StatusCode, strings.TrimSpace(out.MerchantReference))
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

func (h *HTTPClient) debugf(format string, args ...any) {
	if !h.cfg.Debug {
		return
	}
	log.Printf("[pesapal-debug] "+format, args...)
}

func maskSecret(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "..." + s[len(s)-4:]
}
