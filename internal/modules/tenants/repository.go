package tenants

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"rms-be/internal/modules/leases"
	"rms-be/internal/modules/units"
)

// Repository persists tenants scoped by organization_id.
type Repository struct {
	db *gorm.DB
}

// NewRepository constructs a Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// List searches and paginates tenants for an organization.
func (r *Repository) List(ctx context.Context, organizationID string, nameQ, emailQ, phoneQ string, status *TenantStatus, offset, limit int) ([]Tenant, int64, error) {
	orgID := strings.TrimSpace(organizationID)
	q := r.db.WithContext(ctx).Model(&Tenant{}).Where("organization_id = ?", orgID)
	if status != nil && *status != "" {
		q = q.Where("status = ?", string(*status))
	}
	if p := likePattern(nameQ); p != "" {
		q = q.Where("(full_name ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?)", p, p, p)
	}
	if p := likePattern(emailQ); p != "" {
		q = q.Where("email ILIKE ?", p)
	}
	if p := likePattern(phoneQ); p != "" {
		q = q.Where("phone ILIKE ?", p)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Tenant
	tx := r.db.WithContext(ctx).Where("organization_id = ?", orgID)
	if status != nil && *status != "" {
		tx = tx.Where("status = ?", string(*status))
	}
	if p := likePattern(nameQ); p != "" {
		tx = tx.Where("(full_name ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?)", p, p, p)
	}
	if p := likePattern(emailQ); p != "" {
		tx = tx.Where("email ILIKE ?", p)
	}
	if p := likePattern(phoneQ); p != "" {
		tx = tx.Where("phone ILIKE ?", p)
	}
	if err := tx.Order("full_name ASC, id ASC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// ListByBuildings scopes tenant list to tenants with leases on provided building IDs.
func (r *Repository) ListByBuildings(ctx context.Context, organizationID string, buildingIDs []string, nameQ, emailQ, phoneQ string, status *TenantStatus, offset, limit int) ([]Tenant, int64, error) {
	if len(buildingIDs) == 0 {
		return []Tenant{}, 0, nil
	}
	orgID := strings.TrimSpace(organizationID)
	q := r.db.WithContext(ctx).Model(&Tenant{}).
		Where("organization_id = ?", orgID).
		Where("id IN (SELECT l.tenant_id FROM leases l JOIN units u ON u.id = l.unit_id WHERE l.organization_id = ? AND u.building_id IN ?)", orgID, buildingIDs)
	if status != nil && *status != "" {
		q = q.Where("status = ?", string(*status))
	}
	if p := likePattern(nameQ); p != "" {
		q = q.Where("(full_name ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?)", p, p, p)
	}
	if p := likePattern(emailQ); p != "" {
		q = q.Where("email ILIKE ?", p)
	}
	if p := likePattern(phoneQ); p != "" {
		q = q.Where("phone ILIKE ?", p)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Tenant
	if err := q.Order("full_name ASC, id ASC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// Create inserts a tenant.
func (r *Repository) Create(ctx context.Context, t *Tenant) error {
	return r.db.WithContext(ctx).Create(t).Error
}

// GetByIDInOrg loads a tenant by id within an organization.
func (r *Repository) GetByIDInOrg(ctx context.Context, organizationID, tenantID string) (*Tenant, error) {
	var t Tenant
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(tenantID)).
		First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// GetByIDInOrgAndBuildings loads a tenant scoped to assigned buildings.
func (r *Repository) GetByIDInOrgAndBuildings(ctx context.Context, organizationID, tenantID string, buildingIDs []string) (*Tenant, error) {
	var t Tenant
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(tenantID)).
		Where("id IN (SELECT l.tenant_id FROM leases l JOIN units u ON u.id = l.unit_id WHERE l.organization_id = ? AND u.building_id IN ?)", strings.TrimSpace(organizationID), buildingIDs).
		First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// Update applies column updates for a tenant in an organization.
func (r *Repository) Update(ctx context.Context, organizationID, tenantID string, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&Tenant{}).
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(tenantID)).
		Updates(updates).Error
}

// SoftDelete soft-deletes a tenant when scoped to the organization.
func (r *Repository) SoftDelete(ctx context.Context, organizationID, tenantID string) error {
	return r.db.WithContext(ctx).
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(tenantID)).
		Delete(&Tenant{}).Error
}

// CountLeasesForTenant counts non-deleted leases for the tenant in the organization (any status).
func (r *Repository) CountLeasesForTenant(ctx context.Context, organizationID, tenantID string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&leases.Lease{}).
		Where("organization_id = ? AND tenant_id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(tenantID)).
		Count(&n).Error
	return n, err
}

// ActiveLeaseStats returns active-lease counts and a preferred “current” unit per tenant (latest lease by updated_at).
func (r *Repository) ActiveLeaseStats(ctx context.Context, organizationID string, tenantIDs []string) (counts map[string]int, unitByTenant map[string]CurrentUnitSummary, err error) {
	counts = make(map[string]int)
	unitByTenant = make(map[string]CurrentUnitSummary)
	if len(tenantIDs) == 0 {
		return counts, unitByTenant, nil
	}
	orgID := strings.TrimSpace(organizationID)

	type cntRow struct {
		TenantID string `gorm:"column:tenant_id"`
		Cnt      int64  `gorm:"column:cnt"`
	}
	var crows []cntRow
	if err = r.db.WithContext(ctx).Model(&leases.Lease{}).
		Select("tenant_id, count(*) as cnt").
		Where("organization_id = ? AND status = ? AND tenant_id IN ?", orgID, leases.LeaseStatusActive, tenantIDs).
		Group("tenant_id").
		Scan(&crows).Error; err != nil {
		return nil, nil, err
	}
	for i := range crows {
		counts[crows[i].TenantID] = int(crows[i].Cnt)
	}

	var ls []leases.Lease
	if err = r.db.WithContext(ctx).
		Where("organization_id = ? AND status = ? AND tenant_id IN ?", orgID, leases.LeaseStatusActive, tenantIDs).
		Order("updated_at DESC").
		Find(&ls).Error; err != nil {
		return nil, nil, err
	}
	seen := make(map[string]bool)
	unitIDs := make([]string, 0, len(ls))
	for i := range ls {
		if seen[ls[i].TenantID] {
			continue
		}
		seen[ls[i].TenantID] = true
		unitIDs = append(unitIDs, ls[i].UnitID)
	}
	if len(unitIDs) == 0 {
		return counts, unitByTenant, nil
	}
	var us []units.Unit
	if err = r.db.WithContext(ctx).Where("id IN ? AND organization_id = ?", unitIDs, orgID).Find(&us).Error; err != nil {
		return nil, nil, err
	}
	unitLabel := make(map[string]string, len(us))
	for i := range us {
		unitLabel[us[i].ID] = us[i].UnitLabel
	}
	seen = make(map[string]bool)
	for i := range ls {
		l := ls[i]
		if seen[l.TenantID] {
			continue
		}
		seen[l.TenantID] = true
		unitByTenant[l.TenantID] = CurrentUnitSummary{UnitID: l.UnitID, UnitLabel: unitLabel[l.UnitID]}
	}
	return counts, unitByTenant, nil
}

// TenantIDByUser resolves linked tenant for a user.
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

func likePattern(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "%", "")
	s = strings.ReplaceAll(s, "_", "")
	return "%" + s + "%"
}
