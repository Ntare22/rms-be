package organizations

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// Repository persists organizations.
type Repository struct {
	db *gorm.DB
}

// NewRepository constructs a Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new organization row.
func (r *Repository) Create(ctx context.Context, o *Organization) error {
	return r.db.WithContext(ctx).Create(o).Error
}

// GetByID loads an organization by primary key.
func (r *Repository) GetByID(ctx context.Context, id string) (*Organization, error) {
	var o Organization
	if err := r.db.WithContext(ctx).First(&o, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// GetBySlugCI returns an organization by case-insensitive slug.
func (r *Repository) GetBySlugCI(ctx context.Context, slug string) (*Organization, error) {
	var o Organization
	q := strings.TrimSpace(strings.ToLower(slug))
	if err := r.db.WithContext(ctx).Where("LOWER(slug) = ?", q).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Update applies column updates (map) for the given id.
func (r *Repository) Update(ctx context.Context, id string, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&Organization{}).Where("id = ?", strings.TrimSpace(id)).Updates(updates).Error
}
