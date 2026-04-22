package auth

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"rms-be/internal/modules/organizations"
	"rms-be/internal/modules/users"
)

// Repository handles persistence for authentication flows.
type Repository struct {
	db *gorm.DB
}

// NewRepository constructs a Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CreateOrganizationAndUser creates an organization and its first user in one transaction.
func (r *Repository) CreateOrganizationAndUser(ctx context.Context, org *organizations.Organization, u *users.User) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(org).Error; err != nil {
			return err
		}
		u.OrganizationID = org.ID
		if err := tx.Create(u).Error; err != nil {
			return err
		}
		return nil
	})
}

// GetOrganizationBySlug returns an organization by case-insensitive slug.
func (r *Repository) GetOrganizationBySlug(ctx context.Context, slug string) (*organizations.Organization, error) {
	var o organizations.Organization
	q := strings.TrimSpace(strings.ToLower(slug))
	if err := r.db.WithContext(ctx).Where("LOWER(slug) = ?", q).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// GetOrganizationByID returns an organization by primary key.
func (r *Repository) GetOrganizationByID(ctx context.Context, id string) (*organizations.Organization, error) {
	var o organizations.Organization
	if err := r.db.WithContext(ctx).First(&o, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// GetUserByEmailAndOrganization returns a user scoped to an organization.
func (r *Repository) GetUserByEmailAndOrganization(ctx context.Context, email, organizationID string) (*users.User, error) {
	var u users.User
	q := strings.TrimSpace(strings.ToLower(email))
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND LOWER(email) = ?", organizationID, q).
		First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByID returns a user by id (any organization).
func (r *Repository) GetUserByID(ctx context.Context, userID string) (*users.User, error) {
	var u users.User
	if err := r.db.WithContext(ctx).First(&u, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByEmailInOrg returns user by email scoped to organization.
func (r *Repository) GetUserByEmailInOrg(ctx context.Context, organizationID, email string) (*users.User, error) {
	var u users.User
	q := strings.TrimSpace(strings.ToLower(email))
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND LOWER(email) = ?", strings.TrimSpace(organizationID), q).
		First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateUser inserts an org user.
func (r *Repository) CreateUser(ctx context.Context, u *users.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

// UpdateUser updates user columns by id and organization.
func (r *Repository) UpdateUser(ctx context.Context, organizationID, userID string, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&users.User{}).
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(userID)).
		Updates(updates).Error
}

// CreatePasswordSetupToken inserts a one-time password setup token.
func (r *Repository) CreatePasswordSetupToken(ctx context.Context, t *PasswordSetupToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

// GetPasswordSetupTokenByHash returns a non-used setup token by hash.
func (r *Repository) GetPasswordSetupTokenByHash(ctx context.Context, tokenHash string) (*PasswordSetupToken, error) {
	var t PasswordSetupToken
	if err := r.db.WithContext(ctx).
		Where("token_hash = ? AND used_at IS NULL", strings.TrimSpace(tokenHash)).
		First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// MarkPasswordSetupTokenUsed marks token as consumed.
func (r *Repository) MarkPasswordSetupTokenUsed(ctx context.Context, tokenID string, usedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&PasswordSetupToken{}).
		Where("id = ? AND used_at IS NULL", strings.TrimSpace(tokenID)).
		Update("used_at", usedAt.UTC()).Error
}
