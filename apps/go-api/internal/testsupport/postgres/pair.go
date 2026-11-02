package postgres

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // Register pgx for the SQL readiness probe.
	"github.com/moby/moby/api/types/network"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	postgresImage    = "postgres:18.6-alpine"
	seedDatabase     = "plane_parity_seed"
	sourceDatabase   = "plane_parity_source"
	targetDatabase   = "plane_parity_target"
	postgresUser     = "plane"
	postgresPassword = "plane-test-only"
)

// SeedMigration installs the migrated schema and deterministic seed rows once.
type SeedMigration func(context.Context, string) error

// Pair owns one disposable PostgreSQL container and two databases cloned from one seed.
type Pair struct {
	container *postgrescontainer.PostgresContainer
	adminPool *pgxpool.Pool
	adminURL  string
	seedURL   string

	SourceURL string
	TargetURL string
}

// Database is an independent clone of a fixture's migrated seed.
type Database struct {
	URL      string
	adminURL string
	name     string
}

// CloneDatabase preserves the seed and gives each test its own database.
func (pair *Pair) CloneDatabase(ctx context.Context) (result *Database, resultErr error) {
	name := "plane_test_" + uuid.NewString()
	cloneURL, err := databaseURL(pair.seedURL, name)
	if err != nil {
		return nil, err
	}
	database := &Database{URL: cloneURL, adminURL: pair.adminURL, name: name}
	connection, err := pair.adminPool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect clone admin database: %w", err)
	}
	defer func() {
		connection.Release()
		if resultErr != nil {
			// Cancellation can obscure whether CREATE DATABASE finished. Reclaim
			// the owned name before returning an error.
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
			defer cancel()
			resultErr = errors.Join(resultErr, database.Close(cleanup))
			result = nil
		}
	}()
	if _, err := connection.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()+" TEMPLATE "+pgx.Identifier{seedDatabase}.Sanitize()); err != nil {
		return nil, fmt.Errorf("create fixture database %s: %w", name, err)
	}
	return database, nil
}

// Close drops only this test's clone, including any connections it left open.
func (database *Database) Close(ctx context.Context) (resultErr error) {
	connection, err := pgx.Connect(ctx, database.adminURL)
	if err != nil {
		return fmt.Errorf("connect drop admin database: %w", err)
	}
	defer func() { resultErr = errors.Join(resultErr, closeConnection(ctx, connection)) }()
	_, err = connection.Exec(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{database.name}.Sanitize()+" WITH (FORCE)")
	if err != nil {
		return fmt.Errorf("drop fixture database %s: %w", database.name, err)
	}
	return nil
}

func closeConnection(ctx context.Context, connection *pgx.Conn) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := connection.Close(cleanup); err != nil {
		return fmt.Errorf("close fixture connection: %w", err)
	}
	return nil
}

// Start creates a ready PostgreSQL container, migrates its seed database, and clones two targets.
func Start(ctx context.Context, migrate SeedMigration) (_ *Pair, err error) {
	pair, err := StartUncloned(ctx, migrate)
	if err != nil {
		return nil, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			err = errors.Join(err, pair.Close())
		}
	}()
	if err := pair.Reset(ctx); err != nil {
		return nil, err
	}
	cleanup = false
	return pair, nil
}

// StartUncloned creates a ready seed database and runs its migration without target clones.
func StartUncloned(ctx context.Context, migrate SeedMigration) (_ *Pair, err error) {
	if migrate == nil {
		return nil, fmt.Errorf("seed migration is required")
	}
	container, err := postgrescontainer.Run(
		ctx,
		postgresImage,
		postgrescontainer.WithDatabase(seedDatabase),
		postgrescontainer.WithUsername(postgresUser),
		postgrescontainer.WithPassword(postgresPassword),
		postgrescontainer.BasicWaitStrategies(),
		// A forwarded TCP port can accept connections before PostgreSQL's
		// protocol is usable. Retry only this read, never the seed migration.
		testcontainers.WithAdditionalWaitStrategy(wait.ForSQL("5432/tcp", "pgx", func(host string, port network.Port) string {
			return (&url.URL{
				Scheme: "postgres", User: url.UserPassword(postgresUser, postgresPassword),
				Host: net.JoinHostPort(host, port.Port()), Path: seedDatabase,
				RawQuery: "sslmode=disable&connect_timeout=2",
			}).String()
		}).WithStartupTimeout(time.Minute)),
	)
	cleanup := true
	defer func() {
		if cleanup {
			err = errors.Join(err, testcontainers.TerminateContainer(container))
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("start PostgreSQL testcontainer: %w", err)
	}
	seedURL, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("get seed connection string: %w", err)
	}
	adminURL, err := databaseURL(seedURL, "postgres")
	if err != nil {
		return nil, err
	}
	if migrationErr := migrate(ctx, seedURL); migrationErr != nil {
		return nil, fmt.Errorf("migrate parity seed: %w", migrationErr)
	}
	adminPool, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		return nil, fmt.Errorf("configure fixture admin connections: %w", err)
	}
	cleanup = false
	return &Pair{container: container, adminPool: adminPool, adminURL: adminURL, seedURL: seedURL}, nil
}

// Reset discards both targets and restores them independently from the unchanged seed database.
func (pair *Pair) Reset(ctx context.Context) (resultErr error) {
	connection, err := pgx.Connect(ctx, pair.adminURL)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL testcontainer admin database: %w", err)
	}
	defer func() { resultErr = errors.Join(resultErr, closeConnection(ctx, connection)) }()

	for _, database := range []string{sourceDatabase, targetDatabase} {
		if _, execErr := connection.Exec(
			ctx,
			"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()",
			database,
		); execErr != nil {
			return fmt.Errorf("terminate connections to %s: %w", database, execErr)
		}
		identifier := pgx.Identifier{database}.Sanitize()
		if _, execErr := connection.Exec(ctx, "DROP DATABASE IF EXISTS "+identifier); execErr != nil {
			return fmt.Errorf("drop disposable database %s: %w", database, execErr)
		}
		if _, execErr := connection.Exec(
			ctx,
			"CREATE DATABASE "+identifier+" TEMPLATE "+pgx.Identifier{seedDatabase}.Sanitize(),
		); execErr != nil {
			return fmt.Errorf("restore disposable database %s: %w", database, execErr)
		}
	}
	pair.SourceURL, err = databaseURL(pair.seedURL, sourceDatabase)
	if err != nil {
		return err
	}
	pair.TargetURL, err = databaseURL(pair.seedURL, targetDatabase)
	if err != nil {
		return err
	}
	return nil
}

// Close terminates the disposable container and all databases it owns.
func (pair *Pair) Close() error {
	if pair == nil {
		return nil
	}
	if pair.adminPool != nil {
		pair.adminPool.Close()
	}
	if pair.container == nil {
		return nil
	}
	return testcontainers.TerminateContainer(pair.container)
}

// SeedURL returns the connection URL for the seed database before target clones are made.
func (pair *Pair) SeedURL() string {
	if pair == nil {
		return ""
	}
	return pair.seedURL
}

// ContainerID returns the PostgreSQL container identifier for sidecar migration tools.
func (pair *Pair) ContainerID() string {
	if pair == nil || pair.container == nil {
		return ""
	}
	return pair.container.GetContainerID()
}

// SeedSidecarURL returns the seed URL for a container that shares PostgreSQL's network namespace.
func (pair *Pair) SeedSidecarURL() (string, error) {
	if pair == nil {
		return "", fmt.Errorf("PostgreSQL pair is required")
	}
	return sidecarURL(pair.seedURL)
}

// SourceSidecarURL returns the source URL for a container in PostgreSQL's network namespace.
func (pair *Pair) SourceSidecarURL() (string, error) {
	if pair == nil {
		return "", fmt.Errorf("PostgreSQL pair is required")
	}
	return sidecarURL(pair.SourceURL)
}

// TargetSidecarURL returns the target URL for a container in PostgreSQL's network namespace.
func (pair *Pair) TargetSidecarURL() (string, error) {
	if pair == nil {
		return "", fmt.Errorf("PostgreSQL pair is required")
	}
	return sidecarURL(pair.TargetURL)
}

func sidecarURL(postgresURL string) (string, error) {
	parsed, err := url.Parse(postgresURL)
	if err != nil {
		return "", fmt.Errorf("parse PostgreSQL URL for sidecar: %w", err)
	}
	parsed.Host = "127.0.0.1:5432"
	return parsed.String(), nil
}

func databaseURL(postgresURL string, database string) (string, error) {
	parsed, err := url.Parse(postgresURL)
	if err != nil {
		return "", fmt.Errorf("parse PostgreSQL testcontainer URL: %w", err)
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return "", fmt.Errorf("PostgreSQL testcontainer URL must use postgres or postgresql scheme")
	}
	parsed.Path = "/" + database
	return parsed.String(), nil
}
