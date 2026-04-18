package database

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"rms-be/internal/api/clock"
)

func mustPostgresURL(t *testing.T) (func(context.Context, ...testcontainers.TerminateOption) error, string) {
	t.Helper()

	const (
		dbName = "rms_test"
		dbPwd  = "rms_test"
		dbUser = "rms_test"
	)

	dbContainer, err := postgres.Run(
		context.Background(),
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPwd),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("postgres container: %v", err)
	}

	host, err := dbContainer.Host(context.Background())
	if err != nil {
		_ = dbContainer.Terminate(context.Background())
		t.Fatalf("host: %v", err)
	}

	mapped, err := dbContainer.MappedPort(context.Background(), "5432/tcp")
	if err != nil {
		_ = dbContainer.Terminate(context.Background())
		t.Fatalf("port: %v", err)
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPwd, host, mapped.Port(), dbName)
	return dbContainer.Terminate, dsn
}

func TestOpenPostgresPingHealth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	teardown, dsn := mustPostgresURL(t)
	t.Cleanup(func() {
		if teardown == nil {
			return
		}
		if err := teardown(context.Background()); err != nil {
			log.Printf("teardown: %v", err)
		}
	})

	db, err := OpenPostgres(dsn, clock.RealClock{})
	if err != nil {
		t.Fatalf("OpenPostgres: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	stats := db.Health(ctx)
	if stats["status"] != "up" {
		t.Fatalf("expected up, got %#v", stats)
	}
}
