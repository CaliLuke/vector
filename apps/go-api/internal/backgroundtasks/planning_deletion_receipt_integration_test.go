//go:build integration

package backgroundtasks

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/go-plane/internal/modulereads"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type planningDeletionWriter interface {
	MutateResource(context.Context, projectmutations.Command) (projectmutations.Result, error)
}

func TestPlanningPurgePreservesDeleteAuthorizationUntilReceiptExpires(t *testing.T) {
	_, _, pool := openBackgroundTaskDatabase(t)
	t.Cleanup(pool.Close)
	_, err := pool.Exec(t.Context(), testsupport.MilestoneWriteSeed)
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), `UPDATE projects SET cycle_view=true,module_view=true`)
	require.NoError(t, err)
	cycles, err := cyclereads.NewLifecycleRepository(pool, projectmutations.Effects{}, "https://vector.example")
	require.NoError(t, err)
	modules, err := modulereads.NewLifecycleRepository(pool, projectmutations.Effects{}, "https://vector.example")
	require.NoError(t, err)
	store, err := NewPostgresDeletionStore(pool)
	require.NoError(t, err)
	for _, domain := range []struct {
		name   string
		writer planningDeletionWriter
	}{{"cycle", cycles}, {"module", modules}} {
		t.Run(domain.name, func(t *testing.T) {
			fields := map[string]any{"name": uuid.NewString()}
			if domain.name == "cycle" {
				fields["owned_by"] = testsupport.LabelActor
			}
			cmd := projectmutations.Command{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), ProjectID: uuid.MustParse(testsupport.LabelProject), WorkspaceSlug: "milestones", Action: "create", Fields: fields}
			created, createErr := domain.writer.MutateResource(t.Context(), cmd)
			require.NoError(t, createErr)
			requestID := uuid.New()
			cmd.Action, cmd.ResourceID, cmd.Fields, cmd.RequestID = "delete", uuid.MustParse(created.ID), nil, &requestID
			deleted, deleteErr := domain.writer.MutateResource(t.Context(), cmd)
			require.NoError(t, deleteErr)
			require.False(t, *deleted.Replayed)
			table := pgx.Identifier{domain.name + "s"}.Sanitize()
			_, ageErr := pool.Exec(t.Context(), "UPDATE "+table+" SET deleted_at=now()-interval '3 days' WHERE id=$1", created.ID)
			require.NoError(t, ageErr)
			model := -1
			for index, m := range store.Policy().Models {
				if m.App == "db" && m.Name == domain.name {
					model = index
				}
			}
			require.NotEqual(t, -1, model)
			input := hardDeletionInput{Cutoff: time.Now().Add(-24 * time.Hour), Model: model}
			purged, purgeErr := store.HardDelete(t.Context(), input)
			require.NoError(t, purgeErr)
			require.Zero(t, purged, "a short purge policy must not remove current authorization during the retry window")
			replay, replayErr := domain.writer.MutateResource(t.Context(), cmd)
			require.NoError(t, replayErr)
			require.True(t, *replay.Replayed)
			require.Equal(t, deleted.Revision, replay.Revision)
			var records int
			require.NoError(t, pool.QueryRow(t.Context(), "SELECT count(*) FROM "+table+" WHERE id=$1", created.ID).Scan(&records))
			require.Equal(t, 1, records)

			_, expireErr := pool.Exec(t.Context(), `UPDATE mutation_receipts SET created_at=now()-interval '3 days',replay_expires_at=now()-interval '1 day' WHERE request_id=$1`, requestID)
			require.NoError(t, expireErr)
			purged, purgeErr = store.HardDelete(t.Context(), input)
			require.NoError(t, purgeErr)
			require.Equal(t, 1, purged, "expired receipts must not retain deleted resources indefinitely")
			require.NoError(t, pool.QueryRow(t.Context(), "SELECT count(*) FROM "+table+" WHERE id=$1", created.ID).Scan(&records))
			require.Zero(t, records)
		})
	}
}
