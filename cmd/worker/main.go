package main

import (
	"context"
	"log"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"rms-be/internal/api/clock"
	applogger "rms-be/internal/api/logger"
	"rms-be/internal/config"
	"rms-be/internal/database"
	"rms-be/internal/modules/billing"
	"rms-be/internal/modules/notifications"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	logr := applogger.NewSlog(cfg.AppEnv, cfg.LogLevel)
	clk := clock.RealClock{}
	db, err := database.OpenPostgres(cfg.DatabaseURL, clk)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	mailjet := notifications.NewMailjetSender(cfg.MailjetAPIKey, cfg.MailjetAPISecret)
	var emailSender notifications.EmailSender = notifications.NoopEmailSender{}
	if mailjet.Enabled() {
		emailSender = mailjet
	}
	var smsSender notifications.SMSSender = notifications.NoopSMSSender{}

	fromEmail := strings.TrimSpace(cfg.MailjetFromEmail)
	if fromEmail == "" {
		fromEmail = strings.TrimSpace(cfg.MailjetFrom)
	}
	fromName := strings.TrimSpace(cfg.MailjetFromName)
	if fromName == "" {
		fromName = "RMS"
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logr.Info("rms worker started")
	if err := billing.RunDueSoonReminders(ctx, db.Gorm(), logr, emailSender, smsSender, fromName, fromEmail, cfg.MailjetTemplateBillingReminderID); err != nil {
		logr.Error("billing reminder run failed", "err", err)
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logr.Info("rms worker stopped")
			return
		case <-ticker.C:
			if err := billing.RunDueSoonReminders(ctx, db.Gorm(), logr, emailSender, smsSender, fromName, fromEmail, cfg.MailjetTemplateBillingReminderID); err != nil {
				logr.Error("billing reminder run failed", "err", err)
				continue
			}
			logr.Info("billing reminder run completed", "ts", time.Now().UTC().Format(time.RFC3339))
		}
	}
}
