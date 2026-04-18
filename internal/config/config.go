package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds application configuration loaded from the environment.
type Config struct {
	AppEnv string
	Port   int

	DatabaseURL string

	JWTSecret         string
	JWTRefreshSecret  string
	JWTAccessTTL      time.Duration
	JWTRefreshTTL     time.Duration

	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	SMTPFromName string

	SMSProvider string
	SMSAPIKey   string

	InternalJobSecret string

	// AuthLoginRPM is max auth/login (and register) requests per client IP per minute.
	AuthLoginRPM int

	LogLevel string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	port, err := strconv.Atoi(getEnv("PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	jwtRefreshSecret := strings.TrimSpace(os.Getenv("JWT_REFRESH_SECRET"))
	if jwtRefreshSecret == "" {
		return nil, fmt.Errorf("JWT_REFRESH_SECRET is required")
	}

	accessMin, err := strconv.Atoi(getEnv("JWT_ACCESS_TTL_MINUTES", "15"))
	if err != nil || accessMin <= 0 {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL_MINUTES")
	}
	refreshHrs, err := strconv.Atoi(getEnv("JWT_REFRESH_TTL_HOURS", "168"))
	if err != nil || refreshHrs <= 0 {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL_HOURS")
	}

	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT: %w", err)
	}

	authRPM, err := strconv.Atoi(getEnv("RATE_LIMIT_AUTH_RPM", "30"))
	if err != nil || authRPM < 0 {
		return nil, fmt.Errorf("invalid RATE_LIMIT_AUTH_RPM")
	}

	cfg := &Config{
		AppEnv:        getEnv("APP_ENV", "development"),
		Port:          port,
		DatabaseURL:   databaseURL,
		JWTSecret:     jwtSecret,
		JWTRefreshSecret: jwtRefreshSecret,
		JWTAccessTTL:     time.Duration(accessMin) * time.Minute,
		JWTRefreshTTL:    time.Duration(refreshHrs) * time.Hour,
		SMTPHost:         getEnv("SMTP_HOST", ""),
		SMTPPort:         smtpPort,
		SMTPUser:         getEnv("SMTP_USER", ""),
		SMTPPassword:     getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:         getEnv("SMTP_FROM", ""),
		SMTPFromName:     getEnv("SMTP_FROM_NAME", ""),
		SMSProvider:      getEnv("SMS_PROVIDER", ""),
		SMSAPIKey:        getEnv("SMS_API_KEY", ""),
		InternalJobSecret: strings.TrimSpace(os.Getenv("INTERNAL_JOB_SECRET")),
		AuthLoginRPM:      authRPM,
		LogLevel:          getEnv("LOG_LEVEL", "info"),
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
