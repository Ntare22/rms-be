package egosms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPClientSendMapsSuccess(t *testing.T) {
	var captured providerSendRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Status":"Success","Message":"queued","Code":200,"MessageID":"m1","Reference":"r1"}`))
	}))
	defer srv.Close()

	c := NewClient(Config{
		BaseURL:   srv.URL,
		Username:  "u",
		Password:  "p",
		SenderID:  "RMS",
		Timeout:   2 * time.Second,
		MaxRetries: 1,
	})
	out, err := c.Send(context.Background(), SendRequest{
		To:       "+254700000001",
		Message:  "hello",
		SenderID: "RMS",
	})
	if err != nil {
		t.Fatalf("Send err: %v", err)
	}
	if !out.Success || out.ProviderStatusCode != 200 || out.MessageID != "m1" {
		t.Fatalf("unexpected result: %+v", out)
	}
	if captured.Method != "SendSms" {
		t.Fatalf("expected method SendSms, got %s", captured.Method)
	}
	if captured.UserData.Username != "u" || captured.UserData.Password != "p" {
		t.Fatalf("unexpected userdata: %+v", captured.UserData)
	}
	if len(captured.MsgData) != 1 || captured.MsgData[0].Number != "254700000001" || captured.MsgData[0].SenderID != "RMS" {
		t.Fatalf("unexpected msgdata: %+v", captured.MsgData)
	}
}

func TestHTTPClientSendRetriesAndFails(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"Status":"Failed","Message":"down"}`))
	}))
	defer srv.Close()

	c := NewClient(Config{
		BaseURL:   srv.URL,
		Username:  "u",
		Password:  "p",
		SenderID:  "RMS",
		Timeout:   2 * time.Second,
		MaxRetries: 3,
	})
	_, err := c.Send(context.Background(), SendRequest{
		To:       "+254700000001",
		Message:  "hello",
		SenderID: "RMS",
	})
	if err == nil {
		t.Fatalf("expected failure")
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}
