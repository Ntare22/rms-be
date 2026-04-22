package billing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"rms-be/internal/api/logger"
	"rms-be/internal/modules/notifications"
)

type reminderCandidate struct {
	OrganizationID string    `gorm:"column:organization_id"`
	ChargeID       string    `gorm:"column:charge_id"`
	DueAt          time.Time `gorm:"column:due_at"`
	TenantID       string    `gorm:"column:tenant_id"`
	FullName       string    `gorm:"column:full_name"`
	Email          string    `gorm:"column:email"`
	Phone          string    `gorm:"column:phone"`
	EmailOptIn     bool      `gorm:"column:email_opt_in"`
	SmsOptIn       bool      `gorm:"column:sms_opt_in"`
	TenantChannel  string    `gorm:"column:tenant_channel"`
	LeaseChannel   string    `gorm:"column:lease_channel"`
	AmountMinor    int64     `gorm:"column:amount_minor"`
	Currency       string    `gorm:"column:currency"`
}

// RunDueSoonReminders sends reminders for charges due in 7 days.
func RunDueSoonReminders(ctx context.Context, db *gorm.DB, log logger.Logger, email notifications.EmailSender, sms notifications.SMSSender, fromName, fromEmail string, templateID int64) error {
	targetDay := time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")
	var rows []reminderCandidate
	err := db.WithContext(ctx).
		Table("rent_charges rc").
		Select(`rc.organization_id, rc.id as charge_id, rc.due_at, l.tenant_id, t.full_name, t.email, t.phone, t.email_opt_in, t.sms_opt_in,
COALESCE(NULLIF(TRIM(t.billing_channel), ''), '') as tenant_channel,
COALESCE(NULLIF(TRIM(l.billing_reminder_channel), ''), '') as lease_channel,
rc.amount_minor, l.currency`).
		Joins("JOIN leases l ON l.id = rc.lease_id AND l.organization_id = rc.organization_id").
		Joins("JOIN tenants t ON t.id = l.tenant_id AND t.organization_id = l.organization_id").
		Where("rc.status IN ?", []string{"scheduled", "posted"}).
		Where("DATE(rc.due_at) = DATE(?)", targetDay).
		Find(&rows).Error
	if err != nil {
		return err
	}

	for i := range rows {
		row := rows[i]
		channel := pickChannel(row)
		if alreadySent(ctx, db, row.OrganizationID, row.ChargeID, targetDay) {
			continue
		}
		switch channel {
		case "sms":
			if err := sms.Send(ctx, notifications.SMSMessage{
				ToPhone: row.Phone,
				Body:    fmt.Sprintf("Rent reminder: %d %s is due on %s.", row.AmountMinor, strings.ToUpper(strings.TrimSpace(row.Currency)), row.DueAt.Format("2006-01-02")),
			}); err != nil {
				log.Warn("billing reminder sms failed", "charge_id", row.ChargeID, "err", err)
				continue
			}
			writeLog(ctx, db, row, "sms", "delivered", targetDay)
		default:
			if strings.TrimSpace(row.Email) == "" {
				continue
			}
			dueDate := row.DueAt.Format("2006-01-02")
			amountDisplay := fmt.Sprintf("%d %s", row.AmountMinor, strings.ToUpper(strings.TrimSpace(row.Currency)))
			msg := notifications.EmailMessage{
				ToName:           strings.TrimSpace(row.FullName),
				ToEmail:          strings.TrimSpace(row.Email),
				Subject:          "Billing reminder",
				TextBody:         "Hello {{var:full_name}},\n\nYour rent payment of {{var:amount_display}} is due on {{var:due_date}}.",
				HTMLBody:         "<p>Hello {{var:full_name}},</p><p>Your rent payment of <strong>{{var:amount_display}}</strong> is due on <strong>{{var:due_date}}</strong>.</p>",
				FromName:         strings.TrimSpace(fromName),
				FromEmail:        strings.TrimSpace(fromEmail),
				TemplateID:       templateID,
				TemplateLanguage: true,
				Variables: map[string]any{
					"full_name":      strings.TrimSpace(row.FullName),
					"amount_display": amountDisplay,
					"due_date":       dueDate,
				},
			}
			if err := email.Send(ctx, msg); err != nil {
				log.Warn("billing reminder email failed", "charge_id", row.ChargeID, "err", err)
				continue
			}
			writeLog(ctx, db, row, "email", "delivered", targetDay)
		}
	}
	return nil
}

func pickChannel(r reminderCandidate) string {
	leaseCh := strings.ToLower(strings.TrimSpace(r.LeaseChannel))
	tenantCh := strings.ToLower(strings.TrimSpace(r.TenantChannel))
	if leaseCh == "sms" || leaseCh == "email" {
		return leaseCh
	}
	if tenantCh == "sms" || tenantCh == "email" {
		return tenantCh
	}
	if r.SmsOptIn && strings.TrimSpace(r.Phone) != "" {
		return "sms"
	}
	if r.EmailOptIn && strings.TrimSpace(r.Email) != "" {
		return "email"
	}
	return "email"
}

func alreadySent(ctx context.Context, db *gorm.DB, organizationID, chargeID, day string) bool {
	var n int64
	_ = db.WithContext(ctx).Table("notification_logs").
		Where("organization_id = ? AND template_key = ? AND payload_summary = ?", organizationID, "billing_due_7d", chargeID+":"+day).
		Count(&n).Error
	return n > 0
}

func writeLog(ctx context.Context, db *gorm.DB, row reminderCandidate, channel, result, day string) {
	_ = db.WithContext(ctx).Create(&notifications.NotificationLog{
		OrganizationID: row.OrganizationID,
		TenantID:       &row.TenantID,
		Channel:        channel,
		TemplateKey:    "billing_due_7d",
		Result:         result,
		PayloadSummary: row.ChargeID + ":" + day,
		SentAt:         time.Now().UTC(),
	}).Error
}
