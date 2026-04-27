package egosms

import (
	"context"
	"strings"
	"time"
)

// Client defines the provider operations used by the SMS module.
type Client interface {
	Send(ctx context.Context, req SendRequest) (*SendResult, error)
	Health(ctx context.Context) (*HealthResult, error)
}

type Config struct {
	BaseURL   string
	Username  string
	Password  string
	SenderID  string
	Timeout   time.Duration
	MaxRetries int
	Debug     bool
}

func (c Config) Enabled() bool {
	return strings.TrimSpace(c.BaseURL) != "" &&
		strings.TrimSpace(c.Username) != "" &&
		strings.TrimSpace(c.Password) != "" &&
		strings.TrimSpace(c.SenderID) != ""
}

func NewClient(cfg Config) Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	return NewHTTPClient(cfg)
}
