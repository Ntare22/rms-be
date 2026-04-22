package units

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"rms-be/internal/modules/leases"
)

// Repository persists units scoped by organization and building.
type Repository struct {
	db *gorm.DB
}

// NewRepository constructs a Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// BuildingExistsInOrg reports whether a building id belongs to the organization (uses buildings table by name to avoid import cycles).
func (r *Repository) BuildingExistsInOrg(ctx context.Context, organizationID, buildingID string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("buildings").
		Where("id = ? AND organization_id = ?", strings.TrimSpace(buildingID), strings.TrimSpace(organizationID)).
		Count(&n).Error
	return n > 0, err
}

// ListByBuilding returns units for a building within an organization with optional status filter.
func (r *Repository) ListByBuilding(ctx context.Context, organizationID, buildingID string, status *UnitStatus, offset, limit int) ([]Unit, int64, error) {
	orgID := strings.TrimSpace(organizationID)
	bid := strings.TrimSpace(buildingID)
	q := r.db.WithContext(ctx).Model(&Unit{}).
		Where("organization_id = ? AND building_id = ?", orgID, bid)
	if status != nil && *status != "" {
		q = q.Where("status = ?", string(*status))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	tx := r.db.WithContext(ctx).Where("organization_id = ? AND building_id = ?", orgID, bid)
	if status != nil && *status != "" {
		tx = tx.Where("status = ?", string(*status))
	}
	var rows []Unit
	if err := tx.Order("unit_label ASC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// Create inserts a unit row.
func (r *Repository) Create(ctx context.Context, u *Unit) error {
	return r.db.WithContext(ctx).Create(u).Error
}

// GetByIDInBuilding loads a unit scoped to organization and building.
func (r *Repository) GetByIDInBuilding(ctx context.Context, organizationID, buildingID, unitID string) (*Unit, error) {
	var u Unit
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND building_id = ? AND id = ?",
			strings.TrimSpace(organizationID), strings.TrimSpace(buildingID), strings.TrimSpace(unitID)).
		First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// Update applies column updates for a unit scoped to org and building.
func (r *Repository) Update(ctx context.Context, organizationID, buildingID, unitID string, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&Unit{}).
		Where("organization_id = ? AND building_id = ? AND id = ?",
			strings.TrimSpace(organizationID), strings.TrimSpace(buildingID), strings.TrimSpace(unitID)).
		Updates(updates).Error
}

// SoftDelete removes a unit row (soft delete) when scoped to org and building.
func (r *Repository) SoftDelete(ctx context.Context, organizationID, buildingID, unitID string) error {
	return r.db.WithContext(ctx).
		Where("organization_id = ? AND building_id = ? AND id = ?",
			strings.TrimSpace(organizationID), strings.TrimSpace(buildingID), strings.TrimSpace(unitID)).
		Delete(&Unit{}).Error
}

// CountActiveLeasesForUnit counts non-deleted leases in active status for the unit within the organization.
func (r *Repository) CountActiveLeasesForUnit(ctx context.Context, organizationID, unitID string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&leases.Lease{}).
		Where("organization_id = ? AND unit_id = ? AND status = ?",
			strings.TrimSpace(organizationID), strings.TrimSpace(unitID), leases.LeaseStatusActive).
		Count(&n).Error
	return n, err
}

// ActiveLeasesByUnitIDs returns at most one active lease per unit id (latest by updated_at if duplicates exist).
func (r *Repository) ActiveLeasesByUnitIDs(ctx context.Context, organizationID string, unitIDs []string) (map[string]leases.Lease, error) {
	out := make(map[string]leases.Lease)
	if len(unitIDs) == 0 {
		return out, nil
	}
	var ls []leases.Lease
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND status = ? AND unit_id IN ?",
			strings.TrimSpace(organizationID), leases.LeaseStatusActive, unitIDs).
		Order("updated_at DESC").
		Find(&ls).Error; err != nil {
		return nil, err
	}
	for i := range ls {
		l := ls[i]
		if _, ok := out[l.UnitID]; !ok {
			out[l.UnitID] = l
		}
	}
	return out, nil
}

// ListManagerBuildingIDs returns manager-assigned building IDs.
func (r *Repository) ListManagerBuildingIDs(ctx context.Context, organizationID, userID string) ([]string, error) {
	type row struct {
		BuildingID string `gorm:"column:building_id"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).Table("manager_building_assignments").
		Select("building_id").
		Where("organization_id = ? AND user_id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(userID)).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].BuildingID)
	}
	return out, nil
}
