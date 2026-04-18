package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	applogger "rms-be/internal/api/logger"
	"rms-be/internal/config"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	logr := applogger.NewSlog(cfg.AppEnv, cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logr.Info("rms worker started")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logr.Info("rms worker stopped")
			return
		case <-ticker.C:
			logr.Debug("worker tick", "ts", time.Now().UTC().Format(time.RFC3339))
		}
	}
}
