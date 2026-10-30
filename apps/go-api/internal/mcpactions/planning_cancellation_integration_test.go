//go:build integration

package mcpactions_test

import (
	"context"
	"testing"
	"time"

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
			cancelProjectWriteWhileLocked(t, f, command.ProjectID, func(ctx context.Context) error {
				_, err := write(ctx, command)
				return err
			})
			require.Equal(t, before, planningMutationSnapshot(t, f))
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
