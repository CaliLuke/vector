package postgres

import (
	"context"
	"errors"
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
	pair, err := StartUncloned(ctx, func(seedContext context.Context, url string) (resultErr error) {
		connection, openErr := pgx.Connect(seedContext, url)
		if openErr != nil {
			return openErr
		}
		defer func() { resultErr = errors.Join(resultErr, closeConnection(seedContext, connection)) }()
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
	return ctx, openSQLConnection(t, ctx, pair.SeedURL())
}

// OpenSQL gives a test an independent clone, applies its additional schema or
// seed, and closes the connection before dropping the clone. Close the template
// in TestMain after all tests have finished.
func (template *Template) OpenSQL(t testing.TB, schema string) (context.Context, *pgx.Conn) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	address := template.SeedDatabase(t, func(context.Context, string) error { return nil })
	connection := openSQLConnection(t, ctx, address)
	if _, err := connection.Exec(ctx, schema); err != nil {
		t.Fatal(err)
	}
	return ctx, connection
}

func openSQLConnection(t testing.TB, ctx context.Context, address string) *pgx.Conn {
	t.Helper()
	connection, err := pgx.Connect(ctx, address)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := closeConnection(ctx, connection); closeErr != nil {
			t.Errorf("close fixture connection: %v", closeErr)
		}
	})
	return connection
}

// SeedDatabase owns an independent clone and runs the caller's seed only in that
// clone. Register template.Close in TestMain; individual tests own their connections.
func (template *Template) SeedDatabase(t testing.TB, migrate SeedMigration) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	database, err := template.CloneDatabase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if closeErr := database.Close(cleanup); closeErr != nil {
			t.Errorf("close template clone: %v", closeErr)
		}
	})

	if err := migrate(ctx, database.URL); err != nil {
		t.Fatal(err)
	}
	return database.URL
}
