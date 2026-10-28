//go:build integration

package bootstrap_test

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestFreshBootstrapIncludesWorkItemPropertySchema(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pair, err := postgresfixture.StartUncloned(ctx, bootstrap.MigrateSchema)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.Close(t, pair) })
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.CloseContext(t, connection, context.WithoutCancel(ctx)) })
	for _, table := range []string{"work_item_type_settings", "work_item_properties", "work_item_type_properties", "work_item_property_versions", "work_item_releases"} {
		var exists bool
		if err := connection.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+table).Scan(&exists); err != nil || !exists {
			t.Fatalf("fresh bootstrap did not install public.%s: exists=%t error=%v", table, exists, err)
		}
	}
	if err := bootstrap.MigrateSchema(ctx, pair.SeedURL()); err != nil {
		t.Fatalf("repeating bootstrap after all migrations: %v", err)
	}
}
