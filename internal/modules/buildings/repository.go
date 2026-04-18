package buildings

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"rms-be/internal/modules/units"
)

// Repository persists buildings scoped by organization_id.
type Repository struct {
	db *gorm.DB
}

// NewRepository constructs a Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ListByOrganization returns buildings for exactly one organization with optional status filter.
func (r *Repository) ListByOrganization(ctx context.Context, organizationID string, status *BuildingStatus, offset, limit int) ([]Building, int64, error) {
	orgID := strings.TrimSpace(organizationID)
	q := r.db.WithContext(ctx).Model(&Building{}).Where("organization_id = ?", orgID)
	if status != nil && *status != "" {
		q = q.Where("status = ?", string(*status))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Building
	tx := r.db.WithContext(ctx).Where("organization_id = ?", orgID)
	if status != nil && *status != "" {
		tx = tx.Where("status = ?", string(*status))
	}
	if err := tx.Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// Create inserts a building row (organization_id must match caller intent).
func (r *Repository) Create(ctx context.Context, b *Building) error {
	return r.db.WithContext(ctx).Create(b).Error
}

// GetByIDInOrg loads a building by id constrained to organization_id.
func (r *Repository) GetByIDInOrg(ctx context.Context, organizationID, buildingID string) (*Building, error) {
	var b Building
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(buildingID)).
		First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// Update applies column updates for a building in an organization.
func (r *Repository) Update(ctx context.Context, organizationID, buildingID string, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&Building{}).
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(buildingID)).
		Updates(updates).Error
}

// SoftDelete removes a building row (soft delete) when scoped to the organization.
func (r *Repository) SoftDelete(ctx context.Context, organizationID, buildingID string) error {
	return r.db.WithContext(ctx).
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(buildingID)).
		Delete(&Building{}).Error
}

// CountUnitsForBuilding counts non-deleted units for a building within the same organization (defense in depth).
func (r *Repository) CountUnitsForBuilding(ctx context.Context, organizationID, buildingID string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&units.Unit{}).
		Where("organization_id = ? AND building_id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(buildingID)).
		Count(&n).Error
	return n, err
}
