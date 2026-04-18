package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Logger is a small structured logging abstraction (slog-backed by default).
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// SlogLogger adapts slog to the Logger interface.
type SlogLogger struct {
	inner *slog.Logger
}

// NewSlog builds a slog-backed Logger from environment-style inputs.
func NewSlog(appEnv, level string) *SlogLogger {
	json := strings.EqualFold(appEnv, "production")
	opts := &slog.HandlerOptions{Level: parseLevel(level)}
	var h slog.Handler
	if json {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	return &SlogLogger{inner: slog.New(h)}
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (l *SlogLogger) Debug(msg string, args ...any) { l.inner.Debug(msg, args...) }
func (l *SlogLogger) Info(msg string, args ...any)  { l.inner.Info(msg, args...) }
func (l *SlogLogger) Warn(msg string, args ...any)  { l.inner.Warn(msg, args...) }
func (l *SlogLogger) Error(msg string, args ...any) { l.inner.Error(msg, args...) }
