package payments

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

type leaseScope struct {
	ID       string `gorm:"column:id"`
	TenantID string `gorm:"column:tenant_id"`
	UnitID   string `gorm:"column:unit_id"`
}

// Repository persists payment records.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, p *Payment) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *Repository) GetLeaseScope(ctx context.Context, organizationID, leaseID string) (*leaseScope, error) {
	var row leaseScope
	if err := r.db.WithContext(ctx).Table("leases").
		Select("id, tenant_id, unit_id").
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(leaseID)).
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) TenantIDByUser(ctx context.Context, organizationID, userID string) (string, error) {
	type row struct {
		ID string `gorm:"column:id"`
	}
	var out row
	if err := r.db.WithContext(ctx).Table("tenants").
		Select("id").
		Where("organization_id = ? AND user_id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(userID)).
		Limit(1).
		Scan(&out).Error; err != nil {
		return "", err
	}
	return strings.TrimSpace(out.ID), nil
}

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

func (r *Repository) UnitInBuildings(ctx context.Context, organizationID, unitID string, buildingIDs []string) (bool, error) {
	if len(buildingIDs) == 0 {
		return false, nil
	}
	var n int64
	err := r.db.WithContext(ctx).Table("units").
		Where("organization_id = ? AND id = ? AND building_id IN ?", strings.TrimSpace(organizationID), strings.TrimSpace(unitID), buildingIDs).
		Count(&n).Error
	return n > 0, err
}
