package audit

import "time"

// AuditLog is an append-only security / domain audit entry (no soft delete, no updated_at).
//
// SQL migrations:
//   - BRIN or btree index on (organization_id, created_at) for time-range queries.
//   - Optional: partition by month for very high volume.
type AuditLog struct {
	ID             string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string    `gorm:"type:uuid;not null;index" json:"organization_id"`
	ActorUserID    *string   `gorm:"type:uuid;index" json:"actor_user_id,omitempty"`
	Action         string    `gorm:"size:128;not null" json:"action"`
	ResourceType   string    `gorm:"size:128;not null" json:"resource_type"`
	ResourceID     string    `gorm:"size:64" json:"resource_id,omitempty"`
	MetadataJSON   string    `gorm:"type:text" json:"metadata_json,omitempty"`
	IPAddress      string    `gorm:"size:64" json:"ip_address,omitempty"`
	UserAgent      string    `gorm:"size:512" json:"user_agent,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }
