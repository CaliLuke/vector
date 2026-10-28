//go:build integration

package parity

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestTestcontainersTwoDatabaseTransitionIsRepeatable(t *testing.T) {
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

func installParitySeed(ctx context.Context, seedURL string) error {
	tables, err := PlaneTables()
	if err != nil {
		return err
	}
	connection, err := pgx.Connect(ctx, seedURL)
	if err != nil {
		return err
	}
	defer connection.Close(context.WithoutCancel(ctx))
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
	defer connection.Close(context.WithoutCancel(ctx))
	var got int
	if err := connection.QueryRow(ctx, "SELECT count(*) FROM instance_configurations WHERE id = $1", proofRowID).Scan(&got); err != nil {
		t.Fatalf("query proof-row count: %v", err)
	}
	if got != want {
		t.Fatalf("proof-row count = %d, want %d", got, want)
	}
}
