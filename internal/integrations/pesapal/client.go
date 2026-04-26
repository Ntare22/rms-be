package pesapal

import (
	"context"
	"strings"
	"time"
)

// Client defines PesaPal integration behaviors used by payment orchestration.
type Client interface {
	RequestToken(ctx context.Context) (string, error)
	SubmitOrder(ctx context.Context, req SubmitOrderRequest) (*SubmitOrderResponse, error)
	RegisterIPN(ctx context.Context, url, notificationType string) (*RegisterIPNResponse, error)
	GetIPNList(ctx context.Context) ([]IPNListItem, error)
	GetTransactionStatus(ctx context.Context, orderTrackingID string) (*TransactionStatusResponse, error)
}

// Config controls PesaPal HTTP client wiring.
type Config struct {
	BaseURL        string
	ConsumerKey    string
	ConsumerSecret string
	IPNID          string
	Timeout        time.Duration
}

// Enabled reports whether integration has enough credentials to run.
func (c Config) Enabled() bool {
	return strings.TrimSpace(c.BaseURL) != "" &&
		strings.TrimSpace(c.ConsumerKey) != "" &&
		strings.TrimSpace(c.ConsumerSecret) != ""
}

// NewClient returns a noop client when config is incomplete.
func NewClient(cfg Config) Client {
	if !cfg.Enabled() {
		return NoopClient{}
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return NewHTTPClient(cfg)
}
