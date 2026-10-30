//go:build integration

package mcpactions_test

import (
	"context"
	"testing"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPlanningCancelledLockWaitLeavesRequestReusable(t *testing.T) {
	for _, kind := range []string{"cycle", "module"} {
		t.Run(kind, func(t *testing.T) {
			f := planningAuthorizationWaitFixture(t)
			write := planningRepositoryWriter(t, f, kind)
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			key := uuid.New()
			command := projectmutations.Command{
				ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace),
				ProjectID: uuid.MustParse(testsupport.LabelProject), WorkspaceSlug: "workspace-one", Action: "create",
				RequestID: &key, Fields: map[string]any{"name": "Cancelled write"},
			}
			if kind == "cycle" {
				command.Fields["owned_by"] = testsupport.LabelActor
			}
			before := planningMutationSnapshot(t, f)
			blocker, err := f.pool.Begin(ctx)
			require.NoError(t, err)
			defer func() { require.NoError(t, dbutil.RollbackTransaction(ctx, blocker.Rollback, nil)) }()
			_, err = blocker.Exec(ctx, `SELECT id FROM projects WHERE id=$1 FOR UPDATE`, command.ProjectID)
			require.NoError(t, err)
			requestCtx, cancelRequest := context.WithCancel(ctx)
			defer cancelRequest()
			done := make(chan error, 1)
			go func() { _, requestErr := write(requestCtx, command); done <- requestErr }()
			require.Eventually(t, func() bool {
				var waiting bool
				queryErr := f.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND $1=ANY(pg_blocking_pids(pid)))`, blocker.Conn().PgConn().PID()).Scan(&waiting)
				return queryErr == nil && waiting
			}, 10*time.Second, 10*time.Millisecond, "request must reach the held project lock before cancellation")
			cancelRequest()
			select {
			case err = <-done:
				require.ErrorIs(t, err, context.Canceled)
			case <-ctx.Done():
				t.Fatal("cancelled request did not return while the project lock remained held")
			}
			require.Equal(t, before, planningMutationSnapshot(t, f))
			require.NoError(t, blocker.Commit(ctx))
			// Changing input with the same key proves the cancelled attempt did not
			// reserve a receipt or retain a request hash.
			command.Fields["name"] = "Accepted retry"
			accepted, err := write(ctx, command)
			require.NoError(t, err)
			require.NotNil(t, accepted.Replayed)
			require.False(t, *accepted.Replayed)
			require.Contains(t, string(accepted.Record), "Accepted retry")
			before = planningMutationSnapshot(t, f)
			replay, err := write(ctx, command)
			require.NoError(t, err)
			require.NotNil(t, replay.Replayed)
			require.True(t, *replay.Replayed)
			require.Equal(t, accepted.ID, replay.ID)
			require.Equal(t, accepted.Revision, replay.Revision)
			require.Equal(t, before, planningMutationSnapshot(t, f))
		})
	}
}
