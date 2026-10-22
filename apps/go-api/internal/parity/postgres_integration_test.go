package parity

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

const proofRowID = "11111111-1111-4111-8111-111111111111"

func TestPostgresTwoDatabaseTransition(t *testing.T) {
	sourceURL := os.Getenv("PARITY_SOURCE_POSTGRES_URL")
	targetURL := os.Getenv("PARITY_TARGET_POSTGRES_URL")
	if sourceURL == "" || targetURL == "" {
		t.Skip("set PARITY_SOURCE_POSTGRES_URL and PARITY_TARGET_POSTGRES_URL for the PostgreSQL proof")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cleanupProofRows(t, ctx, sourceURL, targetURL)
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		cleanupProofRows(t, cleanupContext, sourceURL, targetURL)
	})
	runPostgresTwoDatabaseTransition(t, ctx, sourceURL, targetURL)
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
				_ = connection.Close(ctx)
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
	postgresURL := os.Getenv("PARITY_SOURCE_POSTGRES_URL")
	if postgresURL == "" {
		t.Skip("set PARITY_SOURCE_POSTGRES_URL for the PostgreSQL identity proof")
	}
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

func cleanupProofRows(t *testing.T, ctx context.Context, databaseURLs ...string) {
	t.Helper()
	for _, databaseURL := range databaseURLs {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			t.Errorf("connect for proof cleanup: %v", err)
			continue
		}
		if _, err := connection.Exec(ctx, "DELETE FROM instance_configurations WHERE id = '11111111-1111-4111-8111-111111111111'"); err != nil {
			t.Errorf("delete proof row: %v", err)
		}
		if err := connection.Close(ctx); err != nil {
			t.Errorf("close proof cleanup connection: %v", err)
		}
	}
}
