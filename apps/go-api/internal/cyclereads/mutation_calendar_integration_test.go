//go:build integration

package cyclereads

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func cycleMutationCalendarFixture(t *testing.T) (*LifecycleRepository, *pgxpool.Pool) {
	t.Helper()
	schema := postgresfixture.NewTemplate(bootstrap.MigrateSchema)
	t.Cleanup(func() { require.NoError(t, schema.Close()) })
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	database, err := schema.CloneDatabase(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.WithoutCancel(t.Context()), 10*time.Second)
		defer stop()
		require.NoError(t, database.Close(cleanup))
	})
	pool, err := pgxpool.New(ctx, database.URL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	_, err = pool.Exec(ctx, testsupport.MilestoneWriteSeed)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE projects SET cycle_view=true`)
	require.NoError(t, err)
	repository, err := NewLifecycleRepository(pool, projectmutations.Effects{}, "https://vector.example")
	require.NoError(t, err)
	return repository, pool
}

func TestCycleCompletionAndArchiveUseCurrentActorDay(t *testing.T) {
	repository, pool := cycleMutationCalendarFixture(t)
	for _, test := range []struct{ name, zone, now, action, end, wantEnd string }{
		{"archive preserves completed end", "UTC", "2026-10-30T02:00:00Z", "archive", "2026-10-28T23:00:00Z", "2026-10-28T23:00:00Z"},
		{"archive UTC", "UTC", "2026-10-30T02:00:00Z", "archive", "", "2026-10-30T00:00:00Z"},
		{"archive west of UTC", "America/Los_Angeles", "2026-10-30T02:00:00Z", "archive", "", "2026-10-29T07:00:00Z"},
		{"complete west of UTC", "America/Los_Angeles", "2026-10-30T02:00:00Z", "complete", "", "2026-10-29T07:00:00Z"},
		{"archive later today", "UTC", "2026-10-30T02:00:00Z", "archive", "2026-10-30T23:00:00Z", "2026-10-30T00:00:00Z"},
		{"complete east next day", "Asia/Tokyo", "2026-10-30T23:00:00Z", "complete", "", "2026-10-30T15:00:00Z"},
		{"archive DST transition", "America/Los_Angeles", "2026-11-01T08:30:00Z", "archive", "", "2026-11-01T07:00:00Z"},
	} {
		t.Run(test.name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, test.now)
			require.NoError(t, err)
			repository.now = func() time.Time { return now }
			_, err = pool.Exec(t.Context(), `UPDATE users SET user_timezone=$1 WHERE id=$2`, test.zone, testsupport.LabelActor)
			require.NoError(t, err)
			fields := map[string]any{"name": uuid.NewString(), "owned_by": testsupport.LabelActor}
			if test.end != "" {
				fields["end_date"] = test.end
			}
			cmd := projectmutations.Command{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), ProjectID: uuid.MustParse(testsupport.LabelProject), WorkspaceSlug: "milestones", Action: "create", Fields: fields}
			created, err := repository.MutateResource(t.Context(), cmd)
			require.NoError(t, err)
			requestID := uuid.New()
			cmd.Action, cmd.ResourceID, cmd.Fields, cmd.RequestID = test.action, uuid.MustParse(created.ID), nil, &requestID
			result, err := repository.MutateResource(t.Context(), cmd)
			require.NoError(t, err)
			require.False(t, *result.Replayed)
			var end time.Time
			var archived bool
			require.NoError(t, pool.QueryRow(t.Context(), `SELECT end_date,archived_at IS NOT NULL FROM cycles WHERE id=$1`, created.ID).Scan(&end, &archived))
			expected, err := time.Parse(time.RFC3339, test.wantEnd)
			require.NoError(t, err)
			require.Equal(t, expected, end.UTC())
			require.True(t, end.Before(now), "completion must not synthesize a future end time")
			require.Equal(t, test.action == "archive", archived)

			repository.now = func() time.Time { return now.Add(24 * time.Hour) }
			_, err = pool.Exec(t.Context(), `UPDATE users SET user_timezone='Pacific/Auckland' WHERE id=$1`, testsupport.LabelActor)
			require.NoError(t, err)
			replay, err := repository.MutateResource(t.Context(), cmd)
			require.NoError(t, err)
			require.True(t, *replay.Replayed)
			require.Equal(t, result.Revision, replay.Revision)
			var replayedEnd time.Time
			require.NoError(t, pool.QueryRow(t.Context(), `SELECT end_date FROM cycles WHERE id=$1`, created.ID).Scan(&replayedEnd))
			require.Equal(t, end, replayedEnd, "a later retry must not recompute today")
		})
	}
}
