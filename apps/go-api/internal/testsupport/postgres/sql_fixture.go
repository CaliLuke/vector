package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// EmptyDatabase owns a disposable PostgreSQL container without installing a
// schema. Tests can exercise installation or define only their required tables.
func EmptyDatabase(t testing.TB) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	pair, err := StartUncloned(ctx, func(context.Context, string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Errorf("close empty PostgreSQL fixture: %v", closeErr)
		}
	})
	return pair.SeedURL()
}

// OpenSQL starts an isolated integration database with the supplied schema.
func OpenSQL(t testing.TB, schema string) (context.Context, *pgx.Conn) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	pair, err := StartUncloned(ctx, func(seedContext context.Context, url string) error {
		connection, openErr := pgx.Connect(seedContext, url)
		if openErr != nil {
			return openErr
		}
		defer func() { _ = connection.Close(context.WithoutCancel(seedContext)) }()
		_, seedErr := connection.Exec(seedContext, schema)
		return seedErr
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Errorf("close fixture: %v", closeErr)
		}
	})
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(ctx)) })
	return ctx, connection
}
