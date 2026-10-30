//go:build integration

package mcpactions_test

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func setPlanningProjectAccess(t *testing.T, f milestoneFixture, change string) {
	t.Helper()
	_, err := f.pool.Exec(t.Context(), change)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 10*time.Second)
		defer cancel()
		_, restoreErr := f.pool.Exec(ctx, `UPDATE workspace_members SET role=20;
UPDATE project_members SET role=20,is_active=true,deleted_at=NULL;
UPDATE projects SET archived_at=NULL,deleted_at=NULL`)
		require.NoError(t, restoreErr)
	})
}

func TestMCPPlanningDeniedScopePreservesMutationsAndReceipts(t *testing.T) {
	runPlanningMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall, kind string) {
		for _, access := range []struct{ name, change string }{
			{"project guest with workspace administration", `UPDATE project_members SET role=5`},
			{"inactive project membership", `UPDATE project_members SET is_active=false`},
			{"deleted project membership", `UPDATE project_members SET deleted_at=now()`},
			{"unrecognized project role", `UPDATE project_members SET role=7`},
			{"archived project", `UPDATE projects SET archived_at=now()`},
			{"deleted project", `UPDATE projects SET deleted_at=now()`},
		} {
			t.Run(access.name, func(t *testing.T) {
				create, created := createPlanningReceipt(t, call, kind)
				_, other := createPlanningReceipt(t, call, kind)
				item := f.item(t)
				setPlanningProjectAccess(t, f, access.change)
				before := planningMutationSnapshot(t, f)
				milestoneFailure(t, call(kind, create), "forbidden")
				fresh := map[string]any{"action": "create", "name": uuid.NewString(), "request_id": uuid.NewString()}
				if kind == "cycle" {
					fresh["owned_by"] = testsupport.LabelActor
				}
				milestoneFailure(t, call(kind, fresh), "forbidden")
				inputs := []map[string]any{
					{"action": "update", "name": "not authorized"},
					{"action": "delete"},
					{"action": "archive"},
					{"action": "unarchive"},
					{"action": "manage_workitems", "add_ids": item},
				}
				if kind == "cycle" {
					inputs = append(inputs, map[string]any{"action": "complete"}, map[string]any{"action": "transfer_workitems", "new_cycle_id": other["cycle_id"]})
				}
				for _, input := range inputs {
					input[kind+"_id"], input["request_id"] = created[kind+"_id"], uuid.NewString()
					milestoneFailure(t, call(kind, input), "forbidden")
				}
				require.Equal(t, before, planningMutationSnapshot(t, f))
			})
		}
	})
}

func TestMCPPlanningMembersCanWriteAndReplayOwnedDeletes(t *testing.T) {
	runPlanningMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall, kind string) {
		for _, access := range []struct{ name, change string }{
			{"workspace guest and project member", `UPDATE workspace_members SET role=5; UPDATE project_members SET role=15`},
			{"workspace guest and project administrator", `UPDATE workspace_members SET role=5; UPDATE project_members SET role=20`},
			{"member in both scopes", `UPDATE workspace_members SET role=15; UPDATE project_members SET role=15`},
		} {
			t.Run(access.name, func(t *testing.T) {
				create, created := createPlanningReceipt(t, call, kind)
				setPlanningProjectAccess(t, f, access.change)
				before := planningMutationSnapshot(t, f)
				milestoneFailure(t, call(kind, map[string]any{"action": "delete", kind + "_id": uuid.NewString(), "request_id": uuid.NewString()}), "not_found")
				require.Equal(t, before, planningMutationSnapshot(t, f))
				require.Equal(t, true, milestoneObject(t, call(kind, create))["replayed"])
				require.Equal(t, before, planningMutationSnapshot(t, f))
				_, fresh := createPlanningReceipt(t, call, kind)
				require.NotEqual(t, created[kind+"_id"], fresh[kind+"_id"])
				name := uuid.NewString()
				updated := milestoneObject(t, call(kind, map[string]any{"action": "update", kind + "_id": created[kind+"_id"], "name": name, "expected_revision": created["revision"], "request_id": uuid.NewString()}))
				require.Equal(t, name, updated["result"].(map[string]any)["name"])
				deletion := map[string]any{"action": "delete", kind + "_id": created[kind+"_id"], "expected_revision": updated["revision"], "request_id": uuid.NewString()}
				deleted := milestoneObject(t, call(kind, deletion))
				require.Equal(t, false, deleted["replayed"])
				require.Nil(t, deleted["result"])
				var savedName string
				var softDeleted bool
				require.NoError(t, f.pool.QueryRow(t.Context(), "SELECT name,deleted_at IS NOT NULL FROM "+pgx.Identifier{kind + "s"}.Sanitize()+" WHERE id=$1", created[kind+"_id"]).Scan(&savedName, &softDeleted))
				require.Equal(t, name, savedName)
				require.True(t, softDeleted)
				before = planningMutationSnapshot(t, f)
				require.Equal(t, true, milestoneObject(t, call(kind, deletion))["replayed"])
				require.Equal(t, before, planningMutationSnapshot(t, f))
			})
		}
	})
}
