package egosms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type HTTPClient struct {
	cfg    Config
	client *http.Client
}

func NewHTTPClient(cfg Config) *HTTPClient {
	return &HTTPClient{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

func (h *HTTPClient) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	payload := providerSendRequest{
		Method: "SendSms",
		UserData: providerUserData{
			Username: strings.TrimSpace(h.cfg.Username),
			Password: strings.TrimSpace(h.cfg.Password),
		},
		MsgData: []providerSMSRecord{
			{
				Number:   strings.TrimPrefix(strings.TrimSpace(req.To), "+"),
				Message:  strings.TrimSpace(req.Message),
				SenderID: strings.TrimSpace(req.SenderID),
				Priority: "0",
			},
		},
	}
	out, err := h.callWithRetry(ctx, payload)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.ReferenceID) == "" {
		out.ReferenceID = strings.TrimSpace(req.Reference)
	}
	return out, nil
}

func (h *HTTPClient) Health(ctx context.Context) (*HealthResult, error) {
	// TODO: replace with official EGO health/auth endpoint if provided.
	// Current probe validates endpoint reachability and request format.
	payload := providerSendRequest{
		Method: "SendSms",
		UserData: providerUserData{
			Username: strings.TrimSpace(h.cfg.Username),
			Password: strings.TrimSpace(h.cfg.Password),
		},
		MsgData: []providerSMSRecord{},
	}
	out, err := h.callWithRetry(ctx, payload)
	if err != nil {
		return nil, err
	}
	msg := strings.ToLower(strings.TrimSpace(out.ProviderMessage))
	healthy := out.Success || strings.Contains(msg, "wrong json passed") || strings.Contains(msg, "msgdata")
	return &HealthResult{
		Healthy:            healthy,
		ProviderStatusCode: out.ProviderStatusCode,
		ProviderMessage:    out.ProviderMessage,
		RawPayload:         out.RawPayload,
	}, nil
}

func (h *HTTPClient) callWithRetry(ctx context.Context, payload providerSendRequest) (*SendResult, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	h.debugf("egosms request payload=%s", string(maskPayload(body)))
	var lastErr error
	backoff := 200 * time.Millisecond
	for attempt := 0; attempt < h.cfg.MaxRetries; attempt++ {
		attemptStart := time.Now().UTC()
		out, retry, err := h.callOnce(ctx, body)
		h.debugf("egosms attempt=%d retryable=%t latency_ms=%d error=%v", attempt+1, retry, time.Since(attemptStart).Milliseconds(), err)
		if err == nil {
			return out, nil
		}
		lastErr = err
		if !retry || attempt == h.cfg.MaxRetries-1 {
			break
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
		backoff *= 2
	}
	return nil, lastErr
}

func (h *HTTPClient) callOnce(ctx context.Context, payload []byte) (*SendResult, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(h.cfg.BaseURL, "/")+"/", bytes.NewReader(payload))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, true, err
		}
		return nil, true, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	h.debugf("egosms response status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))

	retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, retryable, fmt.Errorf("egosms status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var pr providerResponse
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &pr)
	}
	status := strings.ToLower(strings.TrimSpace(pr.Status))
	success := status == "success" || status == "ok" || status == "sent"
	if !success && pr.Code == 200 {
		success = true
	}
	if pr.Message == "" && !success {
		pr.Message = "provider returned non-success response"
	}
	out := &SendResult{
		Success:            success,
		ProviderStatusCode: nonZero(pr.Code, resp.StatusCode),
		ProviderMessage:    strings.TrimSpace(pr.Message),
		MessageID:          strings.TrimSpace(pr.MessageID),
		ReferenceID:        strings.TrimSpace(pr.Reference),
		RawPayload:         raw,
	}
	return out, false, nil
}

func nonZero(v, fallback int) int {
	if v != 0 {
		return v
	}
	return fallback
}

func (h *HTTPClient) debugf(format string, args ...any) {
	if !h.cfg.Debug {
		return
	}
	log.Printf("[egosms-debug] "+format, args...)
}

func maskPayload(raw []byte) []byte {
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return raw
	}
	if ud, ok := body["userdata"].(map[string]any); ok {
		if _, found := ud["password"]; found {
			ud["password"] = "***"
		}
	}
	out, err := json.Marshal(body)
	if err != nil {
		return raw
	}
	return out
}
