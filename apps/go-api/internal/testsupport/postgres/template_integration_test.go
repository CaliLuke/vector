//go:build integration

package postgres

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5"
)

func TestTemplateMigratesOnceForConcurrentIndependentDatabases(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	var migrations atomic.Int32
	template := NewTemplate(func(ctx context.Context, address string) error {
		migrations.Add(1)
		connection, err := pgx.Connect(ctx, address)
		if err != nil {
			return err
		}
		defer func() { testsupport.CloseContext(t, connection, context.WithoutCancel(ctx)) }()
		_, err = connection.Exec(ctx, `CREATE TABLE marker(value int); INSERT INTO marker VALUES(42)`)
		return err
	})
	t.Cleanup(func() {
		if err := template.Close(); err != nil {
			t.Error(err)
		}
	})
	var clones [2]*Database
	var failures [2]error
	var callers sync.WaitGroup
	for index := range clones {
		callers.Go(func() { clones[index], failures[index] = template.CloneDatabase(ctx) })
	}
	callers.Wait()
	if migrations.Load() != 1 {
		t.Fatalf("migrated %d times", migrations.Load())
	}
	for index, database := range clones {
		if failures[index] != nil {
			t.Fatal(failures[index])
		}
		connection, err := pgx.Connect(ctx, database.URL)
		if err != nil {
			t.Fatal(err)
		}
		var value int
		if err := connection.QueryRow(ctx, `DELETE FROM marker RETURNING value`).Scan(&value); err != nil || value != 42 {
			t.Fatalf("clone %d value=%d error=%v", index, value, err)
		}
		if err := connection.Close(ctx); err != nil {
			t.Fatal(err)
		}
		if err := database.Close(ctx); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTemplateSQLFixturesIsolateWritesAndCleanUpEachClone(t *testing.T) {
	template := NewTemplate(func(context.Context, string) error { return nil })
	t.Cleanup(func() { testsupport.Close(t, template) })
	parent := t
	for _, name := range []string{"first", "second"} {
		t.Run(name, func(t *testing.T) {
			ctx, db := template.OpenSQL(t, `CREATE TABLE marker(value int); INSERT INTO marker VALUES(42)`)
			var value int
			if err := db.QueryRow(ctx, `DELETE FROM marker RETURNING value`).Scan(&value); err != nil || value != 42 {
				t.Fatalf("independent fixture value=%d error=%v", value, err)
			}
			// Leave another connection open: cleanup must still reclaim the database.
			leaked, err := pgx.Connect(ctx, db.Config().ConnString())
			if err != nil {
				t.Fatal(err)
			}
			// Close the client after the child cleanup has dropped its database.
			parent.Cleanup(func() { testsupport.CloseContext(parent, leaked, context.Background()) })
		})
		var clones int
		if err := template.pair.adminPool.QueryRow(t.Context(), `SELECT count(*) FROM pg_database WHERE datname LIKE 'plane_test_%'`).Scan(&clones); err != nil || clones != 0 {
			t.Fatalf("fixture cleanup left %d databases: %v", clones, err)
		}
	}
}
