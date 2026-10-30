//go:build integration

package mcpactions_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestMCPProjectMutationStorageFailureRollsBack(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, action := range []string{"create", "update", "delete"} {
			for _, table := range []string{"mutation_receipts", "work_item_effects"} {
				t.Run(action+"/"+table, func(t *testing.T) { projectMutationStorageFailure(t, f, call, action, table) })
			}
		}
	})
}

func projectMutationStorageFailure(t *testing.T, f milestoneFixture, call catalogueCall, action, table string) {
	t.Helper()
	input := map[string]any{"action": action, "request_id": uuid.NewString()}
	if action == "create" {
		input["name"], input["identifier"] = "Rollback "+strings.ReplaceAll(uuid.NewString(), "-", ""), strings.ToUpper(uuid.NewString()[:8])
	} else {
		created := milestoneObject(t, call("project", map[string]any{"action": "create", "name": "Rollback " + strings.ReplaceAll(uuid.NewString(), "-", ""), "identifier": strings.ToUpper(uuid.NewString()[:8])}))
		input["project_id"] = created["id"]
		_, err := f.pool.Exec(t.Context(), `INSERT INTO user_favorites(id,created_at,updated_at,is_folder,sequence,user_id,entity_identifier,entity_type,project_id,workspace_id) SELECT gen_random_uuid(),now(),now(),false,0,created_by_id,id,'project',id,workspace_id FROM projects WHERE id=$1`, created["id"])
		require.NoError(t, err)
		if action == "update" {
			input["name"] = "Changed " + strings.ReplaceAll(uuid.NewString(), "-", "")
		}
	}
	before := projectMutationStorageSnapshot(t, f)
	tableSQL := pgx.Identifier{table}.Sanitize()
	_, err := f.pool.Exec(t.Context(), "ALTER TABLE "+tableSQL+" ADD CONSTRAINT reject_project_mutation_storage CHECK(false) NOT VALID")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := f.pool.Exec(context.WithoutCancel(t.Context()), "ALTER TABLE "+tableSQL+" DROP CONSTRAINT IF EXISTS reject_project_mutation_storage")
		require.NoError(t, cleanupErr)
	})
	milestoneFailure(t, call("project", input), "internal_error")
	require.Equal(t, before, projectMutationStorageSnapshot(t, f), "storage failure must roll back project rows, initialization, revisions, receipts and durable effects")
	_, err = f.pool.Exec(t.Context(), "ALTER TABLE "+tableSQL+" DROP CONSTRAINT reject_project_mutation_storage")
	require.NoError(t, err)
	accepted := milestoneObject(t, call("project", input))
	require.Equal(t, false, accepted["replayed"], "failed write must leave its request ID reusable")
	after := projectMutationStorageSnapshot(t, f)
	replay := milestoneObject(t, call("project", input))
	require.Equal(t, true, replay["replayed"])
	require.Equal(t, accepted["revision"], replay["revision"])
	require.Equal(t, accepted["result"], replay["result"])
	require.Equal(t, after, projectMutationStorageSnapshot(t, f), "replay must not repeat effects or initialization")
}

func projectMutationStorageSnapshot(t *testing.T, f milestoneFixture) []byte {
	t.Helper()
	var snapshot []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT jsonb_build_object(
 'projects',(SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM projects p),
 'members',(SELECT jsonb_agg(to_jsonb(m) ORDER BY id) FROM project_members m),
 'properties',(SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM project_user_properties p),
 'favorites',(SELECT jsonb_agg(to_jsonb(f) ORDER BY id) FROM user_favorites f),
 'identifiers',(SELECT jsonb_agg(to_jsonb(i) ORDER BY id) FROM project_identifiers i),
 'states',(SELECT jsonb_agg(to_jsonb(s) ORDER BY id) FROM states s),
 'revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY project_id) FROM project_edit_revisions r),
 'receipts',(SELECT jsonb_agg(to_jsonb(r) ORDER BY workspace_id,project_id,action,request_id) FROM mutation_receipts r),
 'effects',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM work_item_effects e))`).Scan(&snapshot))
	return snapshot
}
