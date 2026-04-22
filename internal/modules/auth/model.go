package auth

import "time"

// PasswordSetupToken persists one-time tokens for invited users.
type PasswordSetupToken struct {
	ID             string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	OrganizationID string     `gorm:"type:uuid;not null;index"`
	UserID         string     `gorm:"type:uuid;not null;index"`
	TokenHash      string     `gorm:"size:64;not null;uniqueIndex"`
	ExpiresAt      time.Time  `gorm:"not null;index"`
	UsedAt         *time.Time `gorm:"index"`
	CreatedAt      time.Time
}

func (PasswordSetupToken) TableName() string { return "auth_password_setup_tokens" }
