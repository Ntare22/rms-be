package notifications

import (
	"time"

	"gorm.io/gorm"
)

// NotificationChannel is a constrained string; prefer ENUM/CHECK in SQL.
type NotificationChannel string

const (
	ChannelEmail NotificationChannel = "email"
	ChannelSMS   NotificationChannel = "sms"
	ChannelPush  NotificationChannel = "push"
)

// NotificationTemplate defines reusable content for outbound notifications.
//
// SQL migrations:
//   - UNIQUE (organization_id, template_key) — see composite uniqueIndex.
//   - Optional: CHECK (channel IN (...)).
type NotificationTemplate struct {
	ID             string                `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string                `gorm:"type:uuid;not null;uniqueIndex:uq_notif_tpl_org_key,priority:1" json:"organization_id"`
	TemplateKey    string                `gorm:"size:128;not null;uniqueIndex:uq_notif_tpl_org_key,priority:2" json:"template_key"`
	Channel        NotificationChannel   `gorm:"size:16;not null" json:"channel"`
	Subject        string                `gorm:"size:512" json:"subject,omitempty"`
	Body           string                `gorm:"type:text;not null" json:"body"`
	Active         bool                  `gorm:"not null;default:true" json:"active"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
	DeletedAt      gorm.DeletedAt        `gorm:"index" json:"-"`
	CreatedBy      *string               `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy      *string               `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (NotificationTemplate) TableName() string { return "notification_templates" }

// NotificationPreference stores per-user or per-tenant channel opt-in/out within an org.
//
// SQL migrations:
//   - Partial unique indexes as needed, e.g. one row per (organization_id, user_id, channel) when user_id set.
type NotificationPreference struct {
	ID             string                `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string                `gorm:"type:uuid;not null;index" json:"organization_id"`
	UserID         *string               `gorm:"type:uuid;index" json:"user_id,omitempty"`
	TenantID       *string               `gorm:"type:uuid;index" json:"tenant_id,omitempty"`
	Channel        NotificationChannel   `gorm:"size:16;not null" json:"channel"`
	Enabled        bool                  `gorm:"not null;default:true" json:"enabled"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
	CreatedBy      *string               `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy      *string               `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (NotificationPreference) TableName() string { return "notification_preferences" }

// NotificationJobStatus is a constrained string for job lifecycle.
type NotificationJobStatus string

const (
	NotificationJobQueued  NotificationJobStatus = "queued"
	NotificationJobRunning NotificationJobStatus = "running"
	NotificationJobSent    NotificationJobStatus = "sent"
	NotificationJobFailed  NotificationJobStatus = "failed"
)

// NotificationJob is a durable outbox-style work item for sending a notification.
//
// SQL migrations (critical):
//   - UNIQUE (organization_id, idempotency_key) — composite uniqueIndex; required for safe retries.
//   - Optional: CHECK (status IN (...)).
type NotificationJob struct {
	ID               string                 `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID   string                 `gorm:"type:uuid;not null;uniqueIndex:uq_notif_job_org_idem,priority:1" json:"organization_id"`
	IdempotencyKey   string                 `gorm:"size:190;not null;uniqueIndex:uq_notif_job_org_idem,priority:2" json:"idempotency_key"`
	NotificationType string                 `gorm:"size:64;not null" json:"notification_type"`
	PayloadJSON      string                 `gorm:"type:text;not null" json:"payload_json"`
	Status           NotificationJobStatus  `gorm:"size:24;not null;default:queued" json:"status"`
	ScheduledAt      time.Time              `gorm:"not null" json:"scheduled_at"`
	ProcessedAt      *time.Time             `json:"processed_at,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	CreatedBy        *string                `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy        *string                `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (NotificationJob) TableName() string { return "notification_jobs" }

// NotificationLog is an append-only delivery record (no soft delete, no updated_at).
//
// SQL migrations:
//   - No ON UPDATE CASCADE mutating historical rows; treat as immutable.
type NotificationLog struct {
	ID                string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID    string    `gorm:"type:uuid;not null;index" json:"organization_id"`
	NotificationJobID *string  `gorm:"type:uuid;index" json:"notification_job_id,omitempty"`
	UserID            *string   `gorm:"type:uuid;index" json:"user_id,omitempty"`
	TenantID          *string   `gorm:"type:uuid;index" json:"tenant_id,omitempty"`
	Channel           string    `gorm:"size:16;not null" json:"channel"`
	TemplateKey       string    `gorm:"size:128" json:"template_key,omitempty"`
	Result            string    `gorm:"size:32;not null" json:"result"` // delivered | bounced | error
	PayloadSummary    string    `gorm:"size:512" json:"payload_summary,omitempty"`
	SentAt            time.Time `gorm:"not null" json:"sent_at"`
	CreatedAt         time.Time `json:"created_at"`
}

func (NotificationLog) TableName() string { return "notification_logs" }
