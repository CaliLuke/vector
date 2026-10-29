//go:build integration

package backgroundtasks

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var backgroundTaskSchema = postgresfixture.NewTemplate(bootstrap.MigrateSchema)

func TestMain(m *testing.M) {
	status := m.Run()
	if err := backgroundTaskSchema.Close(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "close background-task test template:", err)
		status = 1
	}
	os.Exit(status)
}

func openBackgroundTaskDatabase(t *testing.T) (string, string, *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	databaseURL, pool := openBackgroundTaskClone(t, ctx)
	_, err := pool.Exec(ctx, `CREATE SCHEMA plane_execution`)
	require.NoError(t, err)
	executionURL, err := withExecutionSearchPath(databaseURL)
	require.NoError(t, err)
	return databaseURL, executionURL, pool
}

// openBackgroundTaskClone supplies only the migrated application schema. Tests
// that exercise workflow setup still create their own execution schema.
func openBackgroundTaskClone(t *testing.T, ctx context.Context) (string, *pgxpool.Pool) {
	t.Helper()
	database, err := backgroundTaskSchema.CloneDatabase(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		require.NoError(t, database.Close(cleanup))
	})
	pool, err := pgxpool.New(ctx, database.URL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return database.URL, pool
}
