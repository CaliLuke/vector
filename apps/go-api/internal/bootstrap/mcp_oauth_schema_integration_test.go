//go:build integration

package bootstrap_test

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestBootstrapInstallsMCPAccountLinksAndReceiptsAndCanRepeat(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	pair, err := postgres.StartUncloned(ctx, bootstrap.MigrateSchema)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pair.Close()) })
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, connection.Close(context.WithoutCancel(ctx))) })
	for _, table := range []string{"mcp_oauth_links", "mcp_oauth_link_flows", "mutation_receipts"} {
		var exists bool
		require.NoError(t, connection.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+table).Scan(&exists))
		require.True(t, exists, table)
	}
	require.NoError(t, bootstrap.MigrateSchema(ctx, pair.SeedURL()))
}
