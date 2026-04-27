package sms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"rms-be/internal/integrations/egosms"
)

type stubClient struct {
	sendFn   func(context.Context, egosms.SendRequest) (*egosms.SendResult, error)
	healthFn func(context.Context) (*egosms.HealthResult, error)
}

func (s stubClient) Send(ctx context.Context, req egosms.SendRequest) (*egosms.SendResult, error) {
	return s.sendFn(ctx, req)
}
func (s stubClient) Health(ctx context.Context) (*egosms.HealthResult, error) { return s.healthFn(ctx) }

type testLogger struct{}

func (testLogger) Debug(string, ...any) {}
func (testLogger) Info(string, ...any)  {}
func (testLogger) Warn(string, ...any)  {}
func (testLogger) Error(string, ...any) {}

func TestHandlerSendValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := NewService(stubClient{
		sendFn: func(context.Context, egosms.SendRequest) (*egosms.SendResult, error) {
			return &egosms.SendResult{Success: true, ProviderStatusCode: 200}, nil
		},
		healthFn: func(context.Context) (*egosms.HealthResult, error) {
			return &egosms.HealthResult{Healthy: true, ProviderStatusCode: 200}, nil
		},
	}, testLogger{}, "RMS")
	h := NewHandler(svc)
	r := gin.New()
	r.POST("/sms/send", h.Send)

	req := httptest.NewRequest(http.MethodPost, "/sms/send", strings.NewReader(`{"to":"","message":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandlerSendSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := NewService(stubClient{
		sendFn: func(context.Context, egosms.SendRequest) (*egosms.SendResult, error) {
			return &egosms.SendResult{
				Success:            true,
				ProviderStatusCode: 200,
				ProviderMessage:    "queued",
				MessageID:          "m1",
			}, nil
		},
		healthFn: func(context.Context) (*egosms.HealthResult, error) {
			return &egosms.HealthResult{Healthy: true, ProviderStatusCode: 200}, nil
		},
	}, testLogger{}, "RMS")
	h := NewHandler(svc)
	r := gin.New()
	r.POST("/sms/send", h.Send)

	req := httptest.NewRequest(http.MethodPost, "/sms/send", strings.NewReader(`{"to":"+254700000001","message":"Hi"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}

func TestHandlerHealthTimeoutPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := NewService(stubClient{
		sendFn: func(context.Context, egosms.SendRequest) (*egosms.SendResult, error) {
			return nil, context.DeadlineExceeded
		},
		healthFn: func(context.Context) (*egosms.HealthResult, error) {
			return nil, context.DeadlineExceeded
		},
	}, testLogger{}, "RMS")
	h := NewHandler(svc)
	r := gin.New()
	r.GET("/sms/health", h.Health)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/sms/health", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d", w.Code)
	}
}
