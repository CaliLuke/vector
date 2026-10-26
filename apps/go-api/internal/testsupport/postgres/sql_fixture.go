//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

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
