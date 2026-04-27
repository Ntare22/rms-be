//go:build integration
// +build integration

package payments

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func mustTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	const (
		dbName = "rms_test"
		dbUser = "rms_test"
		dbPwd  = "rms_test"
	)

	container, err := tcpostgres.Run(
		context.Background(),
		"postgres:16-alpine",
		tcpostgres.WithDatabase(dbName),
		tcpostgres.WithUsername(dbUser),
		tcpostgres.WithPassword(dbPwd),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("postgres container: %v", err)
	}

	host, err := container.Host(context.Background())
	if err != nil {
		_ = container.Terminate(context.Background())
		t.Fatalf("container host: %v", err)
	}
	port, err := container.MappedPort(context.Background(), "5432/tcp")
	if err != nil {
		_ = container.Terminate(context.Background())
		t.Fatalf("container port: %v", err)
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPwd, host, port.Port(), dbName)
	db, err := gorm.Open(pgdriver.Open(dsn), &gorm.Config{})
	if err != nil {
		_ = container.Terminate(context.Background())
		t.Fatalf("gorm open: %v", err)
	}

	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		if err := container.Terminate(context.Background()); err != nil {
			log.Printf("container terminate: %v", err)
		}
	}
	return db, cleanup
}

func TestReminderCandidatesIncludesPendingStatuses(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if !dockerAvailable(t) {
		t.Skip("skipping integration test: docker is not available")
	}

	db, cleanup := mustTestDB(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	orgID := "org-1"
	leaseID := "lease-1"
	tenantID := "tenant-1"

	schema := []string{
		`CREATE TABLE tenants (
			id text primary key,
			organization_id text not null,
			full_name text,
			email text,
			phone text,
			billing_channel text
		);`,
		`CREATE TABLE leases (
			id text primary key,
			organization_id text not null,
			tenant_id text not null,
			status text not null,
			billing_due_day integer,
			billing_amount_override_minor bigint,
			monthly_rent_amount_minor bigint,
			currency text
		);`,
		`CREATE TABLE rent_charges (
			id text primary key,
			organization_id text not null,
			lease_id text not null,
			amount_minor bigint not null,
			due_at timestamptz not null,
			status text not null
		);`,
	}
	for _, q := range schema {
		if err := db.WithContext(ctx).Exec(q).Error; err != nil {
			t.Fatalf("create schema: %v", err)
		}
	}

	now := time.Now().UTC()
	dueInRange := now.AddDate(0, 0, -3)
	nextMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 2)

	if err := db.WithContext(ctx).Exec(
		`INSERT INTO tenants (id, organization_id, full_name, email, phone, billing_channel)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		tenantID, orgID, "Jane Tenant", "jane@example.com", "256700111222", "sms",
	).Error; err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	if err := db.WithContext(ctx).Exec(
		`INSERT INTO leases (id, organization_id, tenant_id, status, billing_due_day, monthly_rent_amount_minor, currency)
		 VALUES (?, ?, ?, 'active', 1, 100000, 'USD')`,
		leaseID, orgID, tenantID,
	).Error; err != nil {
		t.Fatalf("seed lease: %v", err)
	}

	seedCharges := []struct {
		ID     string
		Status string
		DueAt  time.Time
	}{
		{ID: "c-pending", Status: "pending", DueAt: dueInRange},
		{ID: "c-unpaid", Status: "unpaid", DueAt: dueInRange},
		{ID: "c-scheduled", Status: "scheduled", DueAt: dueInRange},
		{ID: "c-void", Status: "void", DueAt: dueInRange},
		{ID: "c-next-month", Status: "pending", DueAt: nextMonth},
	}
	for _, c := range seedCharges {
		if err := db.WithContext(ctx).Exec(
			`INSERT INTO rent_charges (id, organization_id, lease_id, amount_minor, due_at, status)
			 VALUES (?, ?, ?, 10000, ?, ?)`,
			c.ID, orgID, leaseID, c.DueAt, c.Status,
		).Error; err != nil {
			t.Fatalf("seed charge %s: %v", c.ID, err)
		}
	}

	repo := NewRepository(db)
	rows, err := repo.ReminderCandidates(ctx, orgID)
	if err != nil {
		t.Fatalf("ReminderCandidates: %v", err)
	}

	got := map[string]bool{}
	for i := range rows {
		got[rows[i].ChargeID] = true
	}

	if !got["c-pending"] {
		t.Fatalf("expected pending charge candidate to be included")
	}
	if !got["c-unpaid"] {
		t.Fatalf("expected unpaid charge candidate to be included")
	}
	if !got["c-scheduled"] {
		t.Fatalf("expected scheduled charge candidate to be included")
	}
	if got["c-void"] {
		t.Fatalf("expected void charge candidate to be excluded")
	}
	if got["c-next-month"] {
		t.Fatalf("expected next month charge candidate to be excluded by date window")
	}
}

func dockerAvailable(t *testing.T) bool {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}
