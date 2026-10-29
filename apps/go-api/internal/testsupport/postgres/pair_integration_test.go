//go:build integration

package postgres

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5"
)

func TestSeedMigrationRunsOnceAfterSQLReadinessAndCleansUpOnFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	want := errors.New("seed migration rejected after a write")
	calls := 0
	var seedURL string
	pair, err := StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		calls++
		seedURL = databaseURL
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer func() { testsupport.CloseContext(t, connection, context.WithoutCancel(ctx)) }()
		if _, err := connection.Exec(ctx, `CREATE TABLE fixture_probe(value int)`); err != nil {
			return err
		}
		return want
	})
	if pair != nil || !errors.Is(err, want) || calls != 1 {
		t.Fatalf("pair=%v error=%v migration calls=%d; want original failure from one migration", pair, err, calls)
	}
	if connection, err := pgx.Connect(ctx, seedURL); err == nil {
		testsupport.CloseContext(t, connection, context.WithoutCancel(ctx))
		t.Fatal("failed migration left its disposable database running")
	}
}

func TestOpenSQLInstallsItsSchemaBeforeReturningAConnection(t *testing.T) {
	ctx, connection := OpenSQL(t, `CREATE TABLE fixture_probe(value text); INSERT INTO fixture_probe VALUES('ready')`)
	var value string
	if err := connection.QueryRow(ctx, `SELECT value FROM fixture_probe`).Scan(&value); err != nil {
		t.Fatal(err)
	}
	if value != "ready" || ctx.Err() != nil {
		t.Fatalf("fixture value=%q context error=%v", value, ctx.Err())
	}
}

func TestPairResetRestoresIndependentClonesFromTheSeed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pair, err := Start(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer func() { testsupport.CloseContext(t, connection, context.WithoutCancel(ctx)) }()
		_, err = connection.Exec(ctx, `CREATE TABLE fixture_probe(value int); INSERT INTO fixture_probe VALUES(10)`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	source, err := pgx.Connect(ctx, pair.SourceURL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.CloseContext(t, source, context.WithoutCancel(ctx)) }()
	target, err := pgx.Connect(ctx, pair.TargetURL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.CloseContext(t, target, context.WithoutCancel(ctx)) }()
	if _, err = source.Exec(ctx, `UPDATE fixture_probe SET value=20`); err != nil {
		t.Fatal(err)
	}
	if _, err = target.Exec(ctx, `DELETE FROM fixture_probe`); err != nil {
		t.Fatal(err)
	}
	if err = pair.Reset(ctx); err != nil {
		t.Fatal(err)
	}
	restoredSource, err := pgx.Connect(ctx, pair.SourceURL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.CloseContext(t, restoredSource, context.WithoutCancel(ctx)) }()
	restoredTarget, err := pgx.Connect(ctx, pair.TargetURL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.CloseContext(t, restoredTarget, context.WithoutCancel(ctx)) }()
	var sourceValue, targetValue int
	if err = restoredSource.QueryRow(ctx, `SELECT value FROM fixture_probe`).Scan(&sourceValue); err != nil {
		t.Fatal(err)
	}
	if err = restoredTarget.QueryRow(ctx, `SELECT value FROM fixture_probe`).Scan(&targetValue); err != nil {
		t.Fatal(err)
	}
	if sourceValue != 10 || targetValue != 10 {
		t.Fatalf("reset clones contain source=%d target=%d", sourceValue, targetValue)
	}
	if pair.SeedURL() == "" || pair.ContainerID() == "" {
		t.Fatal("fixture lost its seed or disposable container identity")
	}
	t.Run("per-test databases stay isolated and close independently", func(t *testing.T) {
		first, err := pair.CloneDatabase(ctx)
		if err != nil {
			t.Fatal(err)
		}
		second, err := pair.CloneDatabase(ctx)
		if err != nil {
			t.Fatal(err)
		}
		writer, err := pgx.Connect(ctx, first.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { testsupport.CloseContext(t, writer, context.WithoutCancel(ctx)) }()
		if _, writeErr := writer.Exec(ctx, `UPDATE fixture_probe SET value=99`); writeErr != nil {
			t.Fatal(writeErr)
		}
		// Closing a clone must reclaim even a connection left open by a test.
		if closeErr := first.Close(ctx); closeErr != nil {
			t.Fatal(closeErr)
		}
		reader, err := pgx.Connect(ctx, second.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { testsupport.CloseContext(t, reader, context.WithoutCancel(ctx)) }()
		var value int
		if err := reader.QueryRow(ctx, `SELECT value FROM fixture_probe`).Scan(&value); err != nil || value != 10 {
			t.Fatalf("independent clone value=%d error=%v", value, err)
		}
		if err := second.Close(ctx); err != nil {
			t.Fatal(err)
		}
		if err := first.Close(ctx); err != nil {
			t.Fatalf("repeated cleanup: %v", err)
		}
	})
	for _, read := range []struct {
		name    string
		address func() (string, error)
		want    string
	}{
		{"seed", pair.SeedSidecarURL, "/plane_parity_seed"},
		{"source", pair.SourceSidecarURL, "/plane_parity_source"},
		{"target", pair.TargetSidecarURL, "/plane_parity_target"},
	} {
		t.Run(read.name, func(t *testing.T) {
			address, err := read.address()
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := url.Parse(address)
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Host != "127.0.0.1:5432" || parsed.Path != read.want {
				t.Fatalf("sidecar host=%q database=%q", parsed.Host, parsed.Path)
			}
		})
	}
}
