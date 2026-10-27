//go:build integration

package milestones_test

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type writeFixture struct {
	ctx  context.Context
	db   *pgx.Conn
	repo *milestones.Repository
	opts milestones.WriteOptions
}

func newWriteFixture(t *testing.T) writeFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	pair, err := postgresfixture.StartUncloned(ctx, bootstrap.MigrateSchema)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pair.Close()) })
	db, err := pgx.Connect(ctx, pair.SeedURL())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close(context.WithoutCancel(ctx))) })
	config, err := pgxpool.ParseConfig(pair.SeedURL())
	require.NoError(t, err)
	config.MaxConns = 2
	config.ConnConfig.RuntimeParams["application_name"] = "milestone-writes"
	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	repo, err := milestones.NewRepository(pool)
	require.NoError(t, err)
	_, err = db.Exec(ctx, testsupport.MilestoneWriteSeed)
	require.NoError(t, err)
	return writeFixture{ctx, db, repo, milestones.WriteOptions{Scope: milestones.Scope{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), ProjectID: uuid.MustParse(testsupport.LabelProject), WorkspaceSlug: "milestones", Timezone: "UTC"}, Origin: "https://example.test"}}
}

func textField(value string) milestones.StringField {
	return milestones.StringField{Present: true, Value: &value}
}

func (f writeFixture) counts(t *testing.T, records, effects, receipts int) {
	t.Helper()
	var gotRecords, gotEffects, gotReceipts int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM milestones),(SELECT count(*) FROM work_item_effects),(SELECT count(*) FROM mutation_receipts)`).Scan(&gotRecords, &gotEffects, &gotReceipts))
	require.Equal(t, records, gotRecords)
	require.Equal(t, effects, gotEffects)
	require.Equal(t, receipts, gotReceipts)
}
