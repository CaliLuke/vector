//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPWorkLogAuthorAndCurrentAdministrator(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		item := f.item(t)
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item})
		setRole := func(role int) {
			_, err := f.pool.Exec(t.Context(), `UPDATE project_members SET role=$1 WHERE project_id=$2 AND member_id=$3`, role, testsupport.LabelProject, testsupport.LabelActor)
			require.NoError(t, err)
		}
		t.Cleanup(func() {
			_, err := f.pool.Exec(context.WithoutCancel(t.Context()), `UPDATE project_members SET role=20 WHERE project_id=$1 AND member_id=$2`, testsupport.LabelProject, testsupport.LabelActor)
			require.NoError(t, err)
		})
		setRole(15)
		created := milestoneObject(t, call("work_log", map[string]any{"action": "create", "duration": 10}))
		id := created["id"].(string)
		input := map[string]any{"action": "update", "work_log_id": id, "description": "Author edit", "request_id": uuid.NewString()}
		author := milestoneObject(t, call("work_log", input))
		require.Equal(t, false, author["replayed"])
		other := uuid.NewString()
		_, err := f.pool.Exec(t.Context(), `INSERT INTO users SELECT (jsonb_populate_record(NULL::users,to_jsonb(u)||jsonb_build_object('id',$1::text,'username',$1::text,'email',$1::text||'@example.test'))).* FROM users u WHERE id=$2`, other, testsupport.LabelActor)
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), `UPDATE work_logs SET logged_by_id=$2 WHERE id=$1`, id, other)
		require.NoError(t, err)
		before := workLogStorageSnapshot(t, f)
		require.Len(t, catalogueArray(t, call("work_log", map[string]any{"action": "list"})), 1)
		milestoneFailure(t, call("work_log", input), "forbidden")
		milestoneFailure(t, call("work_log", map[string]any{"action": "update", "work_log_id": id, "duration": 20}), "forbidden")
		deletion := map[string]any{"action": "delete", "work_log_id": id, "request_id": uuid.NewString()}
		milestoneFailure(t, call("work_log", deletion), "forbidden")
		require.Equal(t, before, workLogStorageSnapshot(t, f))
		setRole(20)
		require.Equal(t, true, milestoneObject(t, call("work_log", input))["replayed"])
		deleted := milestoneObject(t, call("work_log", deletion))
		require.Equal(t, false, deleted["replayed"])
		after := workLogStorageSnapshot(t, f)
		setRole(15)
		milestoneFailure(t, call("work_log", deletion), "forbidden")
		require.Equal(t, after, workLogStorageSnapshot(t, f))
		setRole(20)
		require.Equal(t, true, milestoneObject(t, call("work_log", deletion))["replayed"])
		require.Equal(t, after, workLogStorageSnapshot(t, f))
	})
}

func TestMCPWorkLogReadsAndReplayRequireCurrentMembership(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		item := f.item(t)
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item})
		input := map[string]any{"action": "create", "duration": 10, "request_id": uuid.NewString()}
		milestoneObject(t, call("work_log", input))
		for _, change := range []struct{ name, set, restore string }{
			{"inactive project member", "is_active=false", "is_active=true"},
			{"deleted project member", "deleted_at=now()", "deleted_at=NULL"},
			{"unknown project role", "role=7", "role=20"},
		} {
			t.Run(change.name, func(t *testing.T) {
				_, err := f.pool.Exec(t.Context(), "UPDATE project_members SET "+change.set+" WHERE project_id=$1 AND member_id=$2", testsupport.LabelProject, testsupport.LabelActor)
				require.NoError(t, err)
				t.Cleanup(func() {
					_, err := f.pool.Exec(context.WithoutCancel(t.Context()), "UPDATE project_members SET "+change.restore+" WHERE project_id=$1 AND member_id=$2", testsupport.LabelProject, testsupport.LabelActor)
					require.NoError(t, err)
				})
				before := workLogStorageSnapshot(t, f)
				milestoneFailure(t, call("work_log", map[string]any{"action": "list"}), "forbidden")
				milestoneFailure(t, call("work_log", input), "forbidden")
				milestoneFailure(t, root("project", map[string]any{"action": "worklog_summary", "project_id": testsupport.LabelProject}), "forbidden")
				require.Equal(t, before, workLogStorageSnapshot(t, f))
			})
		}
		require.Equal(t, true, milestoneObject(t, call("work_log", input))["replayed"])
	})
}
