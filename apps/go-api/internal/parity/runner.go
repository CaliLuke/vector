package parity

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
)

// Config selects separate source and target PostgreSQL databases.
type Config struct {
	SourcePostgresURL     string
	TargetPostgresURL     string
	Schema                string
	Normalizations        Normalizations
	ExpectedChangedTables []string
}

// ActionResult contains the recorder-canonicalized responses from equivalent runtime actions.
type ActionResult struct {
	SourceResponse CapturedResponse
	TargetResponse CapturedResponse
}

type snapshotResult struct {
	role     string
	snapshot Snapshot
	err      error
}

type databaseIdentity struct {
	name                string
	oid                 int64
	systemIdentifier    string
	postmasterStartedAt time.Time
}

// CaptureAndCompare records complete before/after dumps around one equivalent external action.
func CaptureAndCompare(
	ctx context.Context,
	config Config,
	betweenSnapshots func(context.Context) (ActionResult, error),
) (string, error) {
	if config.SourcePostgresURL == "" || config.TargetPostgresURL == "" {
		return "", fmt.Errorf("source and target PostgreSQL URLs are required")
	}
	if err := requireSeparateDatabases(ctx, config.SourcePostgresURL, config.TargetPostgresURL); err != nil {
		return "", err
	}
	if config.Schema == "" {
		config.Schema = "public"
	}
	if err := config.Normalizations.Validate(); err != nil {
		return "", err
	}
	option := DeclaredNormalizationsOption(config.Normalizations)

	sourceBefore, targetBefore, err := snapshotPair(ctx, config)
	if err != nil {
		return "", err
	}
	if err := config.Normalizations.validateColumns(sourceBefore.Tables); err != nil {
		return "", fmt.Errorf("source normalizations: %w", err)
	}
	if err := config.Normalizations.validateColumns(targetBefore.Tables); err != nil {
		return "", fmt.Errorf("target normalizations: %w", err)
	}
	if betweenSnapshots == nil {
		return "", fmt.Errorf("between-snapshots action is required")
	}
	actionResult, err := betweenSnapshots(ctx)
	if err != nil {
		return "", fmt.Errorf("between-snapshots action failed: %w", err)
	}
	if err := actionResult.SourceResponse.validate("source"); err != nil {
		return "", err
	}
	if err := actionResult.TargetResponse.validate("target"); err != nil {
		return "", err
	}
	sourceAfter, targetAfter, err := snapshotPair(ctx, config)
	if err != nil {
		return "", err
	}

	sourceTransition, err := NewRuntimeTransition(
		sourceBefore,
		sourceAfter,
		actionResult.SourceResponse,
		config.ExpectedChangedTables,
	)
	if err != nil {
		return "", fmt.Errorf("source transition: %w", err)
	}
	targetTransition, err := NewRuntimeTransition(
		targetBefore,
		targetAfter,
		actionResult.TargetResponse,
		config.ExpectedChangedTables,
	)
	if err != nil {
		return "", fmt.Errorf("target transition: %w", err)
	}
	return CompareTransitions(sourceTransition, targetTransition, option)
}

func snapshotPair(ctx context.Context, config Config) (Snapshot, Snapshot, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan snapshotResult, 2)
	go snapshotInto(ctx, results, "source", config.SourcePostgresURL, config.Schema)
	go snapshotInto(ctx, results, "target", config.TargetPostgresURL, config.Schema)

	var source Snapshot
	var target Snapshot
	for range 2 {
		result := <-results
		if result.err != nil {
			cancel()
			return Snapshot{}, Snapshot{}, result.err
		}
		if result.role == "source" {
			source = result.snapshot
		} else {
			target = result.snapshot
		}
	}
	return source, target, nil
}

func snapshotInto(ctx context.Context, results chan<- snapshotResult, role string, postgresURL string, schema string) {
	snapshot, err := snapshotDatabase(ctx, role, postgresURL, schema)
	results <- snapshotResult{role: role, snapshot: snapshot, err: err}
}

func requireSeparateDatabases(ctx context.Context, sourceURL string, targetURL string) error {
	source, err := loadDatabaseIdentity(ctx, "source", sourceURL)
	if err != nil {
		return err
	}
	target, err := loadDatabaseIdentity(ctx, "target", targetURL)
	if err != nil {
		return err
	}
	if samePhysicalDatabase(source, target) {
		return fmt.Errorf("source and target must be separate PostgreSQL databases")
	}
	return nil
}

func loadDatabaseIdentity(ctx context.Context, role string, postgresURL string) (databaseIdentity, error) {
	connection, err := pgx.Connect(ctx, postgresURL)
	if err != nil {
		return databaseIdentity{}, safeDatabaseError(role, "connect for identity check", err)
	}
	defer func() { _ = connection.Close(context.WithoutCancel(ctx)) }()

	const query = `
SELECT database.datname,
       database.oid::bigint,
       control.system_identifier::text,
       pg_catalog.pg_postmaster_start_time()
FROM pg_catalog.pg_database AS database
 CROSS JOIN pg_catalog.pg_control_system() AS control
WHERE database.datname = current_database()`
	var identity databaseIdentity
	if err := connection.QueryRow(ctx, query).Scan(
		&identity.name,
		&identity.oid,
		&identity.systemIdentifier,
		&identity.postmasterStartedAt,
	); err != nil {
		return databaseIdentity{}, safeDatabaseError(role, "read physical identity", err)
	}
	return identity, nil
}

func samePhysicalDatabase(source databaseIdentity, target databaseIdentity) bool {
	return source.name == target.name &&
		source.oid == target.oid &&
		source.systemIdentifier == target.systemIdentifier &&
		source.postmasterStartedAt.Equal(target.postmasterStartedAt)
}

func sortedUnique(values []string) []string {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	compact := slices.Compact(sorted)
	if compact == nil {
		return []string{}
	}
	return compact
}
