// @title RMS API
// @version 1.0
// @description Rent Management System HTTP API (bootstrap; module routes are placeholders).
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"rms-be/internal/api/clock"
	applogger "rms-be/internal/api/logger"
	"rms-be/internal/app"
	"rms-be/internal/config"
	"rms-be/internal/database"
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
	defer func() {
		if err := db.Close(); err != nil {
			logr.Error("database close", "error", err)
		}
	}()

	deps, err := app.NewDependencies(cfg, logr, db, clk)
	if err != nil {
		log.Fatalf("dependencies: %v", err)
	}
	srv := app.NewServer(deps)

	done := make(chan struct{}, 1)
	go func() {
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		<-ctx.Done()
		logr.Info("shutdown signal received")
		deps.MarkShuttingDown()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logr.Error("http shutdown", "error", err)
		}
		close(done)
	}()

	logr.Info("http listening", "port", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("http: %v", err)
	}

	<-done
	logr.Info("shutdown complete")
}
