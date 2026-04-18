package database

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"rms-be/internal/api/clock"
)

// DB abstracts the primary PostgreSQL connection pool used by the app.
type DB interface {
	Ping(ctx context.Context) error
	Health(ctx context.Context) map[string]string
	Gorm() *gorm.DB
	Close() error
}

type gormDB struct {
	gorm *gorm.DB
}

// OpenPostgres opens a PostgreSQL pool using a libpq-style connection URL (DATABASE_URL).
func OpenPostgres(databaseURL string, clk clock.Clock) (DB, error) {
	gcfg := &gorm.Config{
		NowFunc: func() time.Time { return clk.Now().UTC() },
		Logger:  logger.Default.LogMode(logger.Warn),
	}

	gdb, err := gorm.Open(postgres.Open(databaseURL), gcfg)
	if err != nil {
		return nil, err
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	return &gormDB{gorm: gdb}, nil
}

func (g *gormDB) Gorm() *gorm.DB { return g.gorm }

func (g *gormDB) Ping(ctx context.Context) error {
	sqlDB, err := g.gorm.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (g *gormDB) Health(ctx context.Context) map[string]string {
	out := make(map[string]string)
	sqlDB, err := g.gorm.DB()
	if err != nil {
		out["status"] = "down"
		out["error"] = fmt.Sprintf("db: %v", err)
		return out
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		out["status"] = "down"
		out["error"] = fmt.Sprintf("db: %v", err)
		return out
	}
	out["status"] = "up"
	st := sqlDB.Stats()
	out["open_connections"] = strconv.Itoa(st.OpenConnections)
	out["in_use"] = strconv.Itoa(st.InUse)
	out["idle"] = strconv.Itoa(st.Idle)
	return out
}

func (g *gormDB) Close() error {
	sqlDB, err := g.gorm.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
