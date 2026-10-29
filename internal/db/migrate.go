package db

import (
	"context"

	"github.com/TED-H-com/Freedom-VPN/db/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// withMigrations avoids goose global configuration. Its session advisory lock
// serializes both migration initialization and execution across processes.
func withMigrations(pool *pgxpool.Pool, fn func(*goose.Provider) error) error {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()
	locker, err := lock.NewPostgresSessionLocker(lock.WithLockID(0x4656504e4d4947))
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS, goose.WithSessionLocker(locker))
	if err != nil {
		return err
	}
	return fn(provider)
}

// Migrate applies forward migrations only (§15.5). Use a DDL-capable identity;
// normal control/worker identities should not own tables or execute DDL.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	return withMigrations(pool, func(p *goose.Provider) error { _, err := p.Up(ctx); return err })
}

// MigrationStatus reports the embedded migrations against the database.
func MigrationStatus(ctx context.Context, pool *pgxpool.Pool) (status []*goose.MigrationStatus, err error) {
	err = withMigrations(pool, func(p *goose.Provider) error { var e error; status, e = p.Status(ctx); return e })
	return
}
