package audit

import "time"

// AuditLogResponse is returned from audit endpoints.
type AuditLogResponse struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	ActorUserID    *string   `json:"actor_user_id,omitempty"`
	Action         string    `json:"action"`
	Resource       string    `json:"resource"`
	ResourceID     string    `json:"resource_id"`
	MetaJSON       string    `json:"meta_json"`
	CreatedAt      time.Time `json:"created_at"`
}
