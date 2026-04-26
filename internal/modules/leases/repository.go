package leases

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"gorm.io/gorm"

	"rms-be/internal/modules/audit"
)

const (
	unitTable          = "units"
	unitStatusOccupied = "occupied"
	unitStatusVacant   = "vacant"
)

// ListFilters holds optional list query parameters.
type ListFilters struct {
	Status             *LeaseStatus
	BuildingID         string
	UnitID             string
	TenantID           string
	AllowedBuildingIDs []string
	ActiveOn           *time.Time // calendar day (UTC) — lease active for any instant on this day
}

// Repository persists leases and related side effects (unit status, audit).
type Repository struct {
	db *gorm.DB
}

// NewRepository constructs a Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// TenantBelongsToOrg reports whether the tenant exists in the organization.
func (r *Repository) TenantBelongsToOrg(ctx context.Context, organizationID, tenantID string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("tenants").
		Where("id = ? AND organization_id = ?", strings.TrimSpace(tenantID), strings.TrimSpace(organizationID)).
		Count(&n).Error
	return n > 0, err
}

// UnitBelongsToOrg reports whether the unit exists in the organization.
func (r *Repository) UnitBelongsToOrg(ctx context.Context, organizationID, unitID string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Table(unitTable).
		Where("id = ? AND organization_id = ?", strings.TrimSpace(unitID), strings.TrimSpace(organizationID)).
		Count(&n).Error
	return n > 0, err
}

// List returns paginated leases scoped to an organization with optional filters.
func (r *Repository) List(ctx context.Context, organizationID string, f ListFilters, offset, limit int) ([]Lease, int64, error) {
	orgID := strings.TrimSpace(organizationID)
	q := r.db.WithContext(ctx).Model(&Lease{}).Where("organization_id = ?", orgID)
	if f.Status != nil && *f.Status != "" {
		q = q.Where("status = ?", string(*f.Status))
	}
	if strings.TrimSpace(f.UnitID) != "" {
		q = q.Where("unit_id = ?", strings.TrimSpace(f.UnitID))
	}
	if strings.TrimSpace(f.TenantID) != "" {
		q = q.Where("tenant_id = ?", strings.TrimSpace(f.TenantID))
	}
	if bid := strings.TrimSpace(f.BuildingID); bid != "" {
		q = q.Where("unit_id IN (SELECT id FROM units WHERE organization_id = ? AND building_id = ?)", orgID, bid)
	} else if len(f.AllowedBuildingIDs) > 0 {
		q = q.Where("unit_id IN (SELECT id FROM units WHERE organization_id = ? AND building_id IN ?)", orgID, f.AllowedBuildingIDs)
	}
	if f.ActiveOn != nil {
		day := time.Date(f.ActiveOn.Year(), f.ActiveOn.Month(), f.ActiveOn.Day(), 0, 0, 0, 0, time.UTC)
		dayEnd := day.Add(24 * time.Hour)
		q = q.Where("status = ?", LeaseStatusActive).
			Where("start_date < ? AND (end_date IS NULL OR end_date > ?)", dayEnd, day)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	tx := r.db.WithContext(ctx).Model(&Lease{}).Where("organization_id = ?", orgID)
	if f.Status != nil && *f.Status != "" {
		tx = tx.Where("status = ?", string(*f.Status))
	}
	if strings.TrimSpace(f.UnitID) != "" {
		tx = tx.Where("unit_id = ?", strings.TrimSpace(f.UnitID))
	}
	if strings.TrimSpace(f.TenantID) != "" {
		tx = tx.Where("tenant_id = ?", strings.TrimSpace(f.TenantID))
	}
	if bid := strings.TrimSpace(f.BuildingID); bid != "" {
		tx = tx.Where("unit_id IN (SELECT id FROM units WHERE organization_id = ? AND building_id = ?)", orgID, bid)
	} else if len(f.AllowedBuildingIDs) > 0 {
		tx = tx.Where("unit_id IN (SELECT id FROM units WHERE organization_id = ? AND building_id IN ?)", orgID, f.AllowedBuildingIDs)
	}
	if f.ActiveOn != nil {
		day := time.Date(f.ActiveOn.Year(), f.ActiveOn.Month(), f.ActiveOn.Day(), 0, 0, 0, 0, time.UTC)
		dayEnd := day.Add(24 * time.Hour)
		tx = tx.Where("status = ?", LeaseStatusActive).
			Where("start_date < ? AND (end_date IS NULL OR end_date > ?)", dayEnd, day)
	}
	var rows []Lease
	if err := tx.Order("start_date DESC, id ASC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// Create inserts a lease.
func (r *Repository) Create(ctx context.Context, l *Lease) error {
	return r.db.WithContext(ctx).Create(l).Error
}

// GetByIDInOrg loads a lease by id in an organization.
func (r *Repository) GetByIDInOrg(ctx context.Context, organizationID, leaseID string) (*Lease, error) {
	var l Lease
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(leaseID)).
		First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// Update applies updates for a lease in an organization.
func (r *Repository) Update(ctx context.Context, organizationID, leaseID string, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&Lease{}).
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(leaseID)).
		Updates(updates).Error
}

// ActivePrimaryLeasesForUnit returns active primary leases on a unit, optionally excluding one id (for overlap checks).
func (r *Repository) ActivePrimaryLeasesForUnit(ctx context.Context, organizationID, unitID, excludeLeaseID string) ([]Lease, error) {
	var rows []Lease
	q := r.db.WithContext(ctx).
		Where("organization_id = ? AND unit_id = ? AND status = ? AND is_primary = ?",
			strings.TrimSpace(organizationID), strings.TrimSpace(unitID), LeaseStatusActive, true)
	if strings.TrimSpace(excludeLeaseID) != "" {
		q = q.Where("id <> ?", strings.TrimSpace(excludeLeaseID))
	}
	if err := q.Order("start_date ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CountActiveLeasesOnUnit counts active leases for a unit (any primary flag).
func (r *Repository) CountActiveLeasesOnUnit(ctx context.Context, organizationID, unitID string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Lease{}).
		Where("organization_id = ? AND unit_id = ? AND status = ?", strings.TrimSpace(organizationID), strings.TrimSpace(unitID), LeaseStatusActive).
		Count(&n).Error
	return n, err
}

// SetUnitStatus sets the unit.status column when scoped to the organization (avoids importing the units package).
func (r *Repository) SetUnitStatus(ctx context.Context, organizationID, unitID, status string) error {
	return r.db.WithContext(ctx).Table(unitTable).
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(unitID)).
		Update("status", status).Error
}

// WriteAuditLog appends an audit row.
func (r *Repository) WriteAuditLog(ctx context.Context, orgID string, actorUserID *string, action, resourceID string, metadata any, ip, userAgent string) error {
	var meta string
	if metadata != nil {
		b, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		meta = string(b)
	}
	log := &audit.AuditLog{
		OrganizationID: strings.TrimSpace(orgID),
		ActorUserID:    actorUserID,
		Action:         action,
		ResourceType:   "lease",
		ResourceID:     strings.TrimSpace(resourceID),
		MetadataJSON:   meta,
		IPAddress:      strings.TrimSpace(ip),
		UserAgent:      strings.TrimSpace(userAgent),
		CreatedAt:      time.Now().UTC(),
	}
	return r.db.WithContext(ctx).Create(log).Error
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

// UnitBelongsToBuildings reports whether a unit is in one of the building IDs for org.
func (r *Repository) UnitBelongsToBuildings(ctx context.Context, organizationID, unitID string, buildingIDs []string) (bool, error) {
	if len(buildingIDs) == 0 {
		return false, nil
	}
	var n int64
	err := r.db.WithContext(ctx).Table("units").
		Where("organization_id = ? AND id = ? AND building_id IN ?", strings.TrimSpace(organizationID), strings.TrimSpace(unitID), buildingIDs).
		Count(&n).Error
	return n > 0, err
}

// TenantIDByUser returns tenant ID linked to a user in an organization.
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

func (r *Repository) ListHistoryForTenantAndUnit(ctx context.Context, organizationID, tenantID, unitID string) ([]Lease, error) {
	var rows []Lease
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND tenant_id = ? AND unit_id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(tenantID), strings.TrimSpace(unitID)).
		Order("start_date DESC, created_at DESC").
		Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateRenewalOffer(ctx context.Context, row *LeaseRenewalOffer) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) GetRenewalOffer(ctx context.Context, organizationID, leaseID, offerID string) (*LeaseRenewalOffer, error) {
	var row LeaseRenewalOffer
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND lease_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(leaseID), strings.TrimSpace(offerID)).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) UpdateRenewalOffer(ctx context.Context, organizationID, leaseID, offerID string, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&LeaseRenewalOffer{}).
		Where("organization_id = ? AND lease_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(leaseID), strings.TrimSpace(offerID)).
		Updates(updates).Error
}

func (r *Repository) CreateCloseout(ctx context.Context, row *LeaseCloseout) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) StatementRows(ctx context.Context, organizationID, tenantID string) ([]TenantStatementEntry, error) {
	type row struct {
		Kind        string    `gorm:"column:kind"`
		ReferenceID string    `gorm:"column:reference_id"`
		AmountMinor int64     `gorm:"column:amount_minor"`
		Currency    string    `gorm:"column:currency"`
		OccurredAt  time.Time `gorm:"column:occurred_at"`
		Description string    `gorm:"column:description"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
SELECT * FROM (
  SELECT 'charge' as kind,
         c.id as reference_id,
         c.amount_minor as amount_minor,
         COALESCE(NULLIF(TRIM(l.currency), ''), 'USD') as currency,
         c.due_at as occurred_at,
         COALESCE(NULLIF(TRIM(c.description), ''), 'rent charge') as description
  FROM rent_charges c
  JOIN leases l ON l.id = c.lease_id AND l.organization_id = c.organization_id
  WHERE c.organization_id = ? AND l.tenant_id = ?
  UNION ALL
  SELECT 'payment' as kind,
         p.id as reference_id,
         p.amount_minor as amount_minor,
         COALESCE(NULLIF(TRIM(p.currency), ''), 'USD') as currency,
         p.received_at as occurred_at,
         COALESCE(NULLIF(TRIM(p.external_ref), ''), 'payment') as description
  FROM payments p
  JOIN leases l ON l.id = p.lease_id AND l.organization_id = p.organization_id
  WHERE p.organization_id = ? AND l.tenant_id = ?
) x
ORDER BY occurred_at ASC
`, strings.TrimSpace(organizationID), strings.TrimSpace(tenantID), strings.TrimSpace(organizationID), strings.TrimSpace(tenantID)).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]TenantStatementEntry, 0, len(rows))
	for i := range rows {
		out = append(out, TenantStatementEntry(rows[i]))
	}
	return out, nil
}
