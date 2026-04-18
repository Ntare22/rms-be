package users

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// Repository persists users scoped by organization.
type Repository struct {
	db *gorm.DB
}

// NewRepository constructs a Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ListByOrganization returns a page of users and total count for the organization.
func (r *Repository) ListByOrganization(ctx context.Context, organizationID string, offset, limit int) ([]User, int64, error) {
	orgID := strings.TrimSpace(organizationID)
	var total int64
	q := r.db.WithContext(ctx).Model(&User{}).Where("organization_id = ?", orgID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []User
	if err := r.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// Create inserts a user row.
func (r *Repository) Create(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

// GetByIDInOrg loads a user by id constrained to organization_id.
func (r *Repository) GetByIDInOrg(ctx context.Context, organizationID, userID string) (*User, error) {
	var u User
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(userID)).
		First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByEmailInOrg returns a user by normalized email within an organization.
func (r *Repository) GetByEmailInOrg(ctx context.Context, organizationID, email string) (*User, error) {
	var u User
	q := strings.TrimSpace(strings.ToLower(email))
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND LOWER(email) = ?", strings.TrimSpace(organizationID), q).
		First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// EmailTakenByOtherUser reports whether email is used by another user in the org (excluding excludeUserID when set).
func (r *Repository) EmailTakenByOtherUser(ctx context.Context, organizationID, email, excludeUserID string) (bool, error) {
	q := strings.TrimSpace(strings.ToLower(email))
	tx := r.db.WithContext(ctx).Model(&User{}).
		Where("organization_id = ? AND LOWER(email) = ?", strings.TrimSpace(organizationID), q)
	if strings.TrimSpace(excludeUserID) != "" {
		tx = tx.Where("id <> ?", strings.TrimSpace(excludeUserID))
	}
	var n int64
	if err := tx.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// Update applies column updates for a user in an organization.
func (r *Repository) Update(ctx context.Context, organizationID, userID string, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&User{}).
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(userID)).
		Updates(updates).Error
}
