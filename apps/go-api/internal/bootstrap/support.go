package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	support "github.com/CaliLuke/go-plane/migrations/support"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

// MigrateRuntimeSchemas retains the existing River and support migration
// journals. Each library owns versioning, locking, and migration execution.
func MigrateRuntimeSchemas(ctx context.Context, databaseURL string, pool *pgxpool.Pool) (resultErr error) {
	river, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("configure River migrations: %w", err)
	}
	if _, migrateErr := river.Migrate(ctx, rivermigrate.DirectionUp, nil); migrateErr != nil {
		return fmt.Errorf("apply River migrations: %w", migrateErr)
	}
	source, err := iofs.New(support.Files, ".")
	if err != nil {
		return fmt.Errorf("open support migrations: %w", err)
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return closeMigrationSource("support", source.Close, err)
	}
	parsed.Scheme = "pgx5"
	migration, err := migrate.NewWithSourceInstance("iofs", source, parsed.String())
	if err != nil {
		return closeMigrationSource("support", source.Close, fmt.Errorf("configure support migrations: %w", err))
	}
	defer func() { resultErr = closeMigration("support", migration.Close, resultErr) }()
	if err := migration.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply support migrations: %w", err)
	}
	return nil
}
