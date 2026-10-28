package parity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

const proofRowID = "11111111-1111-4111-8111-111111111111"

func TestPostgresTwoDatabaseTransition(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	migrationCalls := 0
	pair, err := postgres.Start(ctx, func(ctx context.Context, seedURL string) error {
		migrationCalls++
		return installParitySeed(ctx, seedURL)
	})
	if err != nil {
		t.Fatalf("postgres.Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := pair.Close(); err != nil {
			t.Errorf("close PostgreSQL testcontainer: %v", err)
		}
	})
	if pair.SourceURL == pair.TargetURL {
		t.Fatal("testcontainer returned identical source and target URLs")
	}

	for run := 1; run <= 2; run++ {
		if run > 1 {
			if err := pair.Reset(ctx); err != nil {
				t.Fatalf("Reset() run %d error = %v", run, err)
			}
		}
		assertProofRowCount(t, ctx, pair.SourceURL, 0)
		assertProofRowCount(t, ctx, pair.TargetURL, 0)
		runPostgresTwoDatabaseTransition(t, ctx, pair.SourceURL, pair.TargetURL)
	}
	if migrationCalls != 1 {
		t.Fatalf("seed migration calls = %d, want 1 across resets", migrationCalls)
	}
}

func runPostgresTwoDatabaseTransition(t *testing.T, ctx context.Context, sourceURL string, targetURL string) {
	t.Helper()
	diff, err := CaptureAndCompare(ctx, Config{
		SourcePostgresURL:     sourceURL,
		TargetPostgresURL:     targetURL,
		Schema:                "public",
		ExpectedChangedTables: []string{"instance_configurations"},
	}, func(ctx context.Context) (ActionResult, error) {
		const statement = `
INSERT INTO instance_configurations (
    created_at, updated_at, id, key, value, category, is_encrypted
) VALUES (
    '2026-08-13T00:00:00Z', '2026-08-13T00:00:00Z', $1, 'parity-proof', $2, 'GENERAL', false
)`
		for _, databaseURL := range []string{sourceURL, targetURL} {
			connection, err := pgx.Connect(ctx, databaseURL)
			if err != nil {
				return ActionResult{}, err
			}
			if _, err := connection.Exec(ctx, statement, proofRowID, json.RawMessage(`{"same":"value"}`)); err != nil {
				testsupport.CloseContext(t, connection, ctx)
				return ActionResult{}, err
			}
			if err := connection.Close(ctx); err != nil {
				return ActionResult{}, err
			}
		}
		response := testResponse(`{"created":true}`)
		return ActionResult{SourceResponse: response, TargetResponse: response}, nil
	})
	if err != nil {
		t.Fatalf("CaptureAndCompare() error = %v\ndiff:\n%s", err, diff)
	}
	if diff != "" {
		t.Fatalf("CaptureAndCompare() diff = %q, want match", diff)
	}
}

func TestPostgresIdentityRejectsSameDatabase(t *testing.T) {
	postgresURL := postgres.EmptyDatabase(t)
	config, err := pgx.ParseConfig(postgresURL)
	if err != nil {
		t.Fatalf("parse PostgreSQL URL: %v", err)
	}
	config.RuntimeParams["application_name"] = "parity-target-alias"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := requireSeparateDatabases(ctx, postgresURL, config.ConnString()); err == nil {
		t.Fatal("requireSeparateDatabases() error = nil, want same physical database rejection")
	} else if !strings.Contains(err.Error(), "must be separate") {
		t.Fatalf("requireSeparateDatabases() error = %v, want same physical database rejection", err)
	}
}

func installParitySeed(ctx context.Context, seedURL string) (err error) {
	tables, err := PlaneTables()
	if err != nil {
		return err
	}
	connection, err := pgx.Connect(ctx, seedURL)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, connection.Close(context.WithoutCancel(ctx))) }()
	for _, table := range tables {
		identifier := pgx.Identifier{table}.Sanitize()
		statement := "CREATE TABLE " + identifier + " (id uuid PRIMARY KEY)"
		if table == "instance_configurations" {
			statement = `CREATE TABLE instance_configurations (
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    id uuid PRIMARY KEY,
    key text NOT NULL,
    value text,
    category text NOT NULL,
    is_encrypted boolean NOT NULL
)`
		}
		if _, execErr := connection.Exec(ctx, statement); execErr != nil {
			return fmt.Errorf("apply migrated seed table %s: %w", table, execErr)
		}
	}
	_, err = connection.Exec(ctx, `INSERT INTO accounts (id) VALUES ('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa')`)
	return err
}

func assertProofRowCount(t *testing.T, ctx context.Context, databaseURL string, want int) {
	t.Helper()
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect for proof-row count: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := connection.Close(context.WithoutCancel(ctx)); closeErr != nil {
			t.Error(closeErr)
		}
	})
	var got int
	if err := connection.QueryRow(ctx, "SELECT count(*) FROM instance_configurations WHERE id = $1", proofRowID).Scan(&got); err != nil {
		t.Fatalf("query proof-row count: %v", err)
	}
	if got != want {
		t.Fatalf("proof-row count = %d, want %d", got, want)
	}
}
