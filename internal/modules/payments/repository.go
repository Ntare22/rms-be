package payments

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

type leaseScope struct {
	ID       string `gorm:"column:id"`
	TenantID string `gorm:"column:tenant_id"`
	UnitID   string `gorm:"column:unit_id"`
}

type tenantBillingProfile struct {
	FirstName string `gorm:"column:first_name"`
	LastName  string `gorm:"column:last_name"`
	FullName  string `gorm:"column:full_name"`
	Email     string `gorm:"column:email"`
	Phone     string `gorm:"column:phone"`
}

type summaryRow struct {
	Month               string `gorm:"column:month"`
	CollectedMinor      int64  `gorm:"column:collected_minor"`
	PendingMinor        int64  `gorm:"column:pending_minor"`
	OutstandingDueMinor int64  `gorm:"column:outstanding_due_minor"`
}

type methodSplitRow struct {
	Method         string `gorm:"column:method"`
	Count          int64  `gorm:"column:count"`
	CollectedMinor int64  `gorm:"column:collected_minor"`
}

type reminderCandidateRow struct {
	ChargeID       string    `gorm:"column:charge_id"`
	LeaseID        string    `gorm:"column:lease_id"`
	TenantID       string    `gorm:"column:tenant_id"`
	TenantName     string    `gorm:"column:tenant_name"`
	TenantEmail    string    `gorm:"column:tenant_email"`
	TenantPhone    string    `gorm:"column:tenant_phone"`
	AmountMinor    int64     `gorm:"column:amount_minor"`
	Currency       string    `gorm:"column:currency"`
	DueAt          time.Time `gorm:"column:due_at"`
	SuggestedLevel string    `gorm:"column:suggested_level"`
	Channel        string    `gorm:"column:channel"`
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

func (r *Repository) UpdateExternalRef(ctx context.Context, organizationID, paymentID, externalRef string) error {
	return r.db.WithContext(ctx).Model(&Payment{}).
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(paymentID)).
		Update("external_ref", strings.TrimSpace(externalRef)).Error
}

func (r *Repository) UpdateGatewayState(ctx context.Context, organizationID, paymentID, provider, externalRef, orderTrackingID, providerStatus string) error {
	updates := map[string]any{
		"provider":          strings.TrimSpace(provider),
		"external_ref":      strings.TrimSpace(externalRef),
		"order_tracking_id": strings.TrimSpace(orderTrackingID),
		"provider_status":   strings.TrimSpace(providerStatus),
	}
	return r.db.WithContext(ctx).Model(&Payment{}).
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(paymentID)).
		Updates(updates).Error
}

func (r *Repository) FindByMerchantRefOrTrackingID(ctx context.Context, merchantRef, orderTrackingID string) (*Payment, error) {
	var out Payment
	q := r.db.WithContext(ctx).Model(&Payment{})
	if strings.TrimSpace(merchantRef) != "" && strings.TrimSpace(orderTrackingID) != "" {
		q = q.Where("external_ref = ? OR order_tracking_id = ?", strings.TrimSpace(merchantRef), strings.TrimSpace(orderTrackingID))
	} else if strings.TrimSpace(merchantRef) != "" {
		q = q.Where("external_ref = ?", strings.TrimSpace(merchantRef))
	} else if strings.TrimSpace(orderTrackingID) != "" {
		q = q.Where("order_tracking_id = ?", strings.TrimSpace(orderTrackingID))
	} else {
		return nil, gorm.ErrRecordNotFound
	}
	if err := q.First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *Repository) UpdateFromIPN(ctx context.Context, paymentID string, status PaymentStatus, providerStatus, callbackRaw string) error {
	return r.db.WithContext(ctx).Model(&Payment{}).
		Where("id = ?", strings.TrimSpace(paymentID)).
		Updates(map[string]any{
			"status":          status,
			"provider_status": strings.TrimSpace(providerStatus),
			"callback_raw":    strings.TrimSpace(callbackRaw),
		}).Error
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

func (r *Repository) GetTenantBillingProfile(ctx context.Context, organizationID, tenantID string) (*tenantBillingProfile, error) {
	var row tenantBillingProfile
	if err := r.db.WithContext(ctx).Table("tenants").
		Select("first_name, last_name, full_name, email, phone").
		Where("organization_id = ? AND id = ?", strings.TrimSpace(organizationID), strings.TrimSpace(tenantID)).
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

func (r *Repository) MonthlySummary(ctx context.Context, organizationID string, months int) ([]summaryRow, error) {
	if months <= 0 {
		months = 6
	}
	start := time.Now().UTC().AddDate(0, -(months - 1), 0)
	start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	var rows []summaryRow
	err := r.db.WithContext(ctx).Raw(`
SELECT to_char(month_bucket, 'YYYY-MM') as month,
       COALESCE(collected_minor,0) as collected_minor,
       COALESCE(pending_minor,0) as pending_minor,
       COALESCE(outstanding_due_minor,0) as outstanding_due_minor
FROM (
  SELECT date_trunc('month', p.received_at) as month_bucket,
         SUM(CASE WHEN p.status = 'completed' THEN p.amount_minor ELSE 0 END) as collected_minor,
         SUM(CASE WHEN p.status = 'pending' THEN p.amount_minor ELSE 0 END) as pending_minor,
         0::bigint as outstanding_due_minor
  FROM payments p
  WHERE p.organization_id = ? AND p.received_at >= ?
  GROUP BY 1
  UNION ALL
  SELECT date_trunc('month', c.due_at) as month_bucket,
         0::bigint as collected_minor,
         0::bigint as pending_minor,
         SUM(CASE WHEN c.due_at <= now() AND c.status IN ('scheduled','posted') THEN c.amount_minor ELSE 0 END) as outstanding_due_minor
  FROM rent_charges c
  WHERE c.organization_id = ? AND c.due_at >= ?
  GROUP BY 1
) t
GROUP BY month_bucket, collected_minor, pending_minor, outstanding_due_minor
ORDER BY month_bucket ASC
`, strings.TrimSpace(organizationID), start, strings.TrimSpace(organizationID), start).Scan(&rows).Error
	return rows, err
}

func (r *Repository) MethodSplit(ctx context.Context, organizationID string) ([]methodSplitRow, error) {
	var rows []methodSplitRow
	err := r.db.WithContext(ctx).Raw(`
SELECT method, COUNT(*) as count, COALESCE(SUM(amount_minor),0) as collected_minor
FROM payments
WHERE organization_id = ? AND status = 'completed'
GROUP BY method
ORDER BY collected_minor DESC
`, strings.TrimSpace(organizationID)).Scan(&rows).Error
	return rows, err
}

func (r *Repository) ReminderCandidates(ctx context.Context, organizationID string) ([]reminderCandidateRow, error) {
	var rows []reminderCandidateRow
	err := r.db.WithContext(ctx).Raw(`
WITH charge_candidates AS (
  SELECT c.id as charge_id,
         c.lease_id,
         l.tenant_id,
         COALESCE(NULLIF(TRIM(t.full_name), ''), 'Tenant') as tenant_name,
         COALESCE(NULLIF(TRIM(t.email), ''), '') as tenant_email,
         COALESCE(NULLIF(TRIM(t.phone), ''), '') as tenant_phone,
         c.amount_minor,
         COALESCE(NULLIF(TRIM(l.currency), ''), 'USD') as currency,
         c.due_at,
         CASE
           WHEN c.due_at < now() - interval '14 day' THEN 'final_notice'
           WHEN c.due_at < now() - interval '7 day' THEN 'reminder_2'
           ELSE 'reminder_1'
         END as suggested_level,
         CASE
           WHEN LOWER(COALESCE(NULLIF(TRIM(t.billing_channel), ''), 'email')) = 'sms' THEN 'sms'
           ELSE 'email'
         END as channel
  FROM rent_charges c
  JOIN leases l ON l.id = c.lease_id AND l.organization_id = c.organization_id
  JOIN tenants t ON t.id = l.tenant_id AND t.organization_id = c.organization_id
  WHERE c.organization_id = ?
    AND LOWER(COALESCE(NULLIF(TRIM(c.status::text), ''), 'scheduled')) IN ('scheduled','posted','pending','unpaid')
    AND c.due_at < date_trunc('month', now()) + interval '1 month'
),
lease_due_candidates AS (
  SELECT l.id as charge_id,
         l.id as lease_id,
         l.tenant_id,
         COALESCE(NULLIF(TRIM(t.full_name), ''), 'Tenant') as tenant_name,
         COALESCE(NULLIF(TRIM(t.email), ''), '') as tenant_email,
         COALESCE(NULLIF(TRIM(t.phone), ''), '') as tenant_phone,
         COALESCE(l.billing_amount_override_minor, l.monthly_rent_amount_minor) as amount_minor,
         COALESCE(NULLIF(TRIM(l.currency), ''), 'USD') as currency,
         due.due_at,
         'reminder_1' as suggested_level,
         CASE
           WHEN LOWER(COALESCE(NULLIF(TRIM(t.billing_channel), ''), 'email')) = 'sms' THEN 'sms'
           ELSE 'email'
         END as channel
  FROM leases l
  JOIN tenants t ON t.id = l.tenant_id AND t.organization_id = l.organization_id
  JOIN LATERAL (
    SELECT make_timestamptz(
             EXTRACT(YEAR FROM now())::int,
             EXTRACT(MONTH FROM now())::int,
             LEAST(
               GREATEST(COALESCE(l.billing_due_day, 1), 1),
               EXTRACT(DAY FROM (date_trunc('month', now()) + interval '1 month - 1 day'))::int
             )::int,
             0, 0, 0
           ) as due_at
  ) due ON true
  WHERE l.organization_id = ?
    AND l.status IN ('active', 'pending_approval')
    AND due.due_at >= date_trunc('month', now())
    AND due.due_at < date_trunc('month', now()) + interval '1 month'
    AND NOT EXISTS (
      SELECT 1
      FROM rent_charges rc
      WHERE rc.organization_id = l.organization_id
        AND rc.lease_id = l.id
        AND LOWER(COALESCE(NULLIF(TRIM(rc.status::text), ''), 'scheduled')) IN ('scheduled','posted','pending','unpaid')
        AND rc.due_at >= date_trunc('month', due.due_at)
        AND rc.due_at < date_trunc('month', due.due_at) + interval '1 month'
    )
)
SELECT *
FROM (
  SELECT * FROM charge_candidates
  UNION ALL
  SELECT * FROM lease_due_candidates
) q
ORDER BY q.due_at ASC
`, strings.TrimSpace(organizationID), strings.TrimSpace(organizationID)).Scan(&rows).Error
	return rows, err
}

func (r *Repository) ReminderHistory(ctx context.Context, organizationID string, limit int) ([]PaymentReminder, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var out []PaymentReminder
	err := r.db.WithContext(ctx).
		Where("organization_id = ?", strings.TrimSpace(organizationID)).
		Order("sent_at DESC").
		Limit(limit).
		Find(&out).Error
	return out, err
}

func (r *Repository) CreateReminder(ctx context.Context, row *PaymentReminder) error {
	return r.db.WithContext(ctx).Create(row).Error
}
