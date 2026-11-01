//go:build integration

package milestones_test

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/go-plane/internal/testsupport"
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
	started := time.Now()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	address := cloneMilestoneDatabase(t, ctx, milestoneSchema)
	config, err := pgxpool.ParseConfig(address)
	require.NoError(t, err)
	config.ConnConfig.Tracer = milestoneFixtureTrace(t)
	db, err := pgx.ConnectConfig(ctx, config.ConnConfig.Copy())
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer stop()
		testsupport.CloseContext(t, db, cleanup)
	})
	config.MaxConns = 2
	config.ConnConfig.RuntimeParams["application_name"] = "milestone-writes"
	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	repo, err := milestones.NewRepository(pool)
	require.NoError(t, err)
	_, err = db.Exec(ctx, testsupport.MilestoneWriteSeed)
	require.NoError(t, err)
	setupElapsed := time.Since(started)
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("milestone fixture setup=%s total=%s", setupElapsed, time.Since(started))
			diagnostics, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer stop()
			waits, queryErr := milestoneDatabaseWaits(diagnostics, db)
			if queryErr != nil {
				t.Logf("milestone database diagnostics unavailable: error_type=%T", queryErr)
				return
			}
			t.Logf("milestone database waits: %+v", waits)
		}
	})
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
