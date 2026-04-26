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

	DatabaseURL          string
	AppBaseURL           string
	CORSAllowedOrigins   []string
	CORSAllowedMethods   []string
	CORSAllowedHeaders   []string
	CORSExposeHeaders    []string
	CORSAllowCredentials bool
	CORSMaxAge           time.Duration

	JWTSecret        string
	JWTRefreshSecret string
	JWTAccessTTL     time.Duration
	JWTRefreshTTL    time.Duration

	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	SMTPFromName string

	MailjetAPIKey                    string
	MailjetAPISecret                 string
	MailjetFrom                      string
	MailjetFromName                  string
	MailjetFromEmail                 string
	MailjetTemplateInviteID          int64
	MailjetTemplateBillingReminderID int64

	PesaPalBaseURL        string
	PesaPalConsumerKey    string
	PesaPalConsumerSecret string
	PesaPalIPNID          string
	PesaPalTimeout        time.Duration

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
	corsAllowCredentials, err := strconv.ParseBool(getEnv("CORS_ALLOW_CREDENTIALS", "true"))
	if err != nil {
		return nil, fmt.Errorf("invalid CORS_ALLOW_CREDENTIALS")
	}
	corsMaxAgeSec, err := strconv.Atoi(getEnv("CORS_MAX_AGE_SECONDS", "43200"))
	if err != nil || corsMaxAgeSec < 0 {
		return nil, fmt.Errorf("invalid CORS_MAX_AGE_SECONDS")
	}
	pesapalTimeoutSec, err := strconv.Atoi(getEnv("PESAPAL_TIMEOUT_SECONDS", "10"))
	if err != nil || pesapalTimeoutSec <= 0 {
		return nil, fmt.Errorf("invalid PESAPAL_TIMEOUT_SECONDS")
	}
	inviteTemplateID, err := parseInt64Env("MAILJET_TEMPLATE_INVITE_ID", 0)
	if err != nil {
		return nil, fmt.Errorf("invalid MAILJET_TEMPLATE_INVITE_ID: %w", err)
	}
	billingTemplateID, err := parseInt64Env("MAILJET_TEMPLATE_BILLING_REMINDER_ID", 0)
	if err != nil {
		return nil, fmt.Errorf("invalid MAILJET_TEMPLATE_BILLING_REMINDER_ID: %w", err)
	}

	cfg := &Config{
		AppEnv:                           getEnv("APP_ENV", "development"),
		Port:                             port,
		DatabaseURL:                      databaseURL,
		AppBaseURL:                       getEnv("APP_BASE_URL", "http://localhost:5173"),
		CORSAllowedOrigins:               csvOrDefault(os.Getenv("CORS_ALLOWED_ORIGINS"), []string{"http://localhost:5173", "http://127.0.0.1:5173"}),
		CORSAllowedMethods:               csvOrDefault(os.Getenv("CORS_ALLOWED_METHODS"), []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
		CORSAllowedHeaders:               csvOrDefault(os.Getenv("CORS_ALLOWED_HEADERS"), []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID", "X-Organization-ID", "X-Internal-Key"}),
		CORSExposeHeaders:                csvOrDefault(os.Getenv("CORS_EXPOSE_HEADERS"), []string{"X-Request-ID"}),
		CORSAllowCredentials:             corsAllowCredentials,
		CORSMaxAge:                       time.Duration(corsMaxAgeSec) * time.Second,
		JWTSecret:                        jwtSecret,
		JWTRefreshSecret:                 jwtRefreshSecret,
		JWTAccessTTL:                     time.Duration(accessMin) * time.Minute,
		JWTRefreshTTL:                    time.Duration(refreshHrs) * time.Hour,
		SMTPHost:                         getEnv("SMTP_HOST", ""),
		SMTPPort:                         smtpPort,
		SMTPUser:                         getEnv("SMTP_USER", ""),
		SMTPPassword:                     getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:                         getEnv("SMTP_FROM", ""),
		SMTPFromName:                     getEnv("SMTP_FROM_NAME", ""),
		MailjetAPIKey:                    getEnv("MAILJET_API_KEY", ""),
		MailjetAPISecret:                 getEnv("MAILJET_API_SECRET", ""),
		MailjetFrom:                      getEnv("MAILJET_FROM", ""),
		MailjetFromName:                  getEnv("MAILJET_FROM_NAME", ""),
		MailjetFromEmail:                 getEnv("MAILJET_FROM_EMAIL", ""),
		MailjetTemplateInviteID:          inviteTemplateID,
		MailjetTemplateBillingReminderID: billingTemplateID,
		PesaPalBaseURL:                   getEnv("PESAPAL_BASE_URL", ""),
		PesaPalConsumerKey:               getEnv("PESAPAL_CONSUMER_KEY", ""),
		PesaPalConsumerSecret:            getEnv("PESAPAL_CONSUMER_SECRET", ""),
		PesaPalIPNID:                     getEnv("PESAPAL_IPN_ID", ""),
		PesaPalTimeout:                   time.Duration(pesapalTimeoutSec) * time.Second,
		SMSProvider:                      getEnv("SMS_PROVIDER", ""),
		SMSAPIKey:                        getEnv("SMS_API_KEY", ""),
		InternalJobSecret:                strings.TrimSpace(os.Getenv("INTERNAL_JOB_SECRET")),
		AuthLoginRPM:                     authRPM,
		LogLevel:                         getEnv("LOG_LEVEL", "info"),
	}

	return cfg, nil
}

func parseInt64Env(key string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func csvOrDefault(raw string, fallback []string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return append([]string(nil), fallback...)
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return append([]string(nil), fallback...)
	}
	return out
}
