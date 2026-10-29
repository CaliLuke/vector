//go:build integration

package milestones_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var milestoneSchema = postgresfixture.NewTemplate(bootstrap.MigrateSchema)

// Read fixtures intentionally admit malformed imported links, so they retain
// their minimal schema instead of cloning the constrained write schema.
var milestoneReadSchema = postgresfixture.NewTemplate(func(ctx context.Context, address string) error {
	pool, err := pgxpool.New(ctx, address)
	if err != nil {
		return err
	}
	defer pool.Close()
	_, err = pool.Exec(ctx, testsupport.LabelReadSchema+testsupport.LabelReadSeed+milestoneReadSeed)
	return err
})

func TestMain(m *testing.M) {
	status := m.Run()
	if err := errors.Join(milestoneSchema.Close(), milestoneReadSchema.Close()); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "close milestone test templates:", err)
		status = 1
	}
	os.Exit(status)
}

func cloneMilestoneDatabase(t *testing.T, ctx context.Context, schema *postgresfixture.Template) string {
	t.Helper()
	database, err := schema.CloneDatabase(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		require.NoError(t, database.Close(cleanup))
	})
	return database.URL
}
