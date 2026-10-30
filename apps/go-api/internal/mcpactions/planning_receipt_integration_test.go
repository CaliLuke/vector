//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func planningMutationSnapshot(t *testing.T, f milestoneFixture) []byte {
	t.Helper()
	var snapshot []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT jsonb_build_object(
 'cycles',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM cycles r),'[]'::jsonb),
 'modules',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM modules r),'[]'::jsonb),
 'cycle_revisions',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY cycle_id) FROM cycle_edit_revisions r),'[]'::jsonb),
 'module_revisions',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY module_id) FROM module_edit_revisions r),'[]'::jsonb),
 'cycle_issues',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM cycle_issues r),'[]'::jsonb),
 'module_issues',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM module_issues r),'[]'::jsonb),
 'module_members',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM module_members r),'[]'::jsonb),
 'issues',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM issues r),'[]'::jsonb),
 'issue_revisions',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY issue_id) FROM issue_edit_revisions r),'[]'::jsonb),
 'favorites',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM user_favorites r),'[]'::jsonb),
 'effects',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM work_item_effects r),'[]'::jsonb),
 'receipts',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY workspace_id,tool,action,project_id,request_id) FROM mutation_receipts r WHERE tool IN ('cycle','module')),'[]'::jsonb))`).Scan(&snapshot))
	return snapshot
}

func createPlanningReceipt(t *testing.T, call catalogueCall, kind string) (map[string]any, map[string]any) {
	t.Helper()
	input := map[string]any{"action": "create", "name": uuid.NewString(), "request_id": uuid.NewString()}
	if kind == "cycle" {
		input["owned_by"] = testsupport.LabelActor
	}
	created := milestoneObject(t, call(kind, input))
	require.Equal(t, false, created["replayed"])
	require.Equal(t, testsupport.LabelProject, created["result"].(map[string]any)["project"])
	return input, created
}

func runPlanningMatrix(t *testing.T, exercise func(*testing.T, milestoneFixture, catalogueCall, string)) {
	t.Helper()
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		milestoneObject(t, call("project", map[string]any{"action": "update_features", "cycles": true, "modules": true}))
		for _, kind := range []string{"cycle", "module"} {
			t.Run(kind, func(t *testing.T) { exercise(t, f, call, kind) })
		}
	})
}

func TestMCPPlanningReceiptsPreserveOtherWriterEdits(t *testing.T) {
	runPlanningMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall, kind string) {
		input, created := createPlanningReceipt(t, call, kind)
		id := created[kind+"_id"].(string)
		// The native table is also written outside MCP. Its trigger must invalidate
		// old tokens without changing the historical response held by a receipt.
		_, err := f.pool.Exec(t.Context(), "UPDATE "+pgx.Identifier{kind + "s"}.Sanitize()+" SET description=$2 WHERE id=$1", id, "written outside MCP")
		require.NoError(t, err)
		detail := milestoneObject(t, call(kind, map[string]any{"action": "retrieve", kind + "_id": id, "include_revision": true}))
		require.Equal(t, "written outside MCP", detail["description"])
		require.NotEqual(t, created["revision"], detail["revision"])
		before := planningMutationSnapshot(t, f)
		replay := milestoneObject(t, call(kind, input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, created["result"], replay["result"])
		require.Equal(t, created["revision"], replay["revision"])
		require.Equal(t, before, planningMutationSnapshot(t, f))
		input["name"] = uuid.NewString()
		milestoneFailure(t, call(kind, input), "request_conflict")
		milestoneFailure(t, call(kind, map[string]any{"action": "update", kind + "_id": id, "name": "stale edit", "expected_revision": created["revision"], "request_id": uuid.NewString()}), "revision_conflict")
		require.Equal(t, before, planningMutationSnapshot(t, f))
	})
}

func TestMCPPlanningMixedAssociationFailureRollsBack(t *testing.T) {
	runPlanningMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall, kind string) {
		_, created := createPlanningReceipt(t, call, kind)
		id, item, missing := created[kind+"_id"].(string), f.item(t), uuid.NewString()
		input := map[string]any{"action": "manage_workitems", kind + "_id": id, "add_ids": item, "remove_ids": missing, "request_id": uuid.NewString(), "expected_revision": created["revision"]}
		before := planningMutationSnapshot(t, f)
		milestoneFailure(t, call(kind, input), "not_found")
		require.Equal(t, before, planningMutationSnapshot(t, f), "failed removal must roll back the earlier addition, its effects and both revisions")
		// The failed transaction did not reserve its request key or consume the
		// revision. Correcting the input can commit with that same key and token.
		delete(input, "remove_ids")
		accepted := milestoneObject(t, call(kind, input))
		require.Equal(t, false, accepted["replayed"])
		require.Nil(t, accepted["result"])
		require.NotEqual(t, created["revision"], accepted["revision"])
		var linked int
		require.NoError(t, f.pool.QueryRow(t.Context(), "SELECT count(*) FROM "+pgx.Identifier{kind + "_issues"}.Sanitize()+" WHERE "+pgx.Identifier{kind + "_id"}.Sanitize()+"=$1 AND issue_id=$2 AND deleted_at IS NULL", id, item).Scan(&linked))
		require.Equal(t, 1, linked)
		before = planningMutationSnapshot(t, f)
		input["add_ids"] = "[\"" + item + "\",\"" + item + "\"]"
		replay := milestoneObject(t, call(kind, input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, accepted["revision"], replay["revision"])
		require.Equal(t, before, planningMutationSnapshot(t, f))
	})
}

func TestMCPPlanningEffectFailureRollsBackAssociation(t *testing.T) {
	runPlanningMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall, kind string) {
		_, created := createPlanningReceipt(t, call, kind)
		id, item := created[kind+"_id"].(string), f.item(t)
		input := map[string]any{"action": "manage_workitems", kind + "_id": id, "add_ids": item, "request_id": uuid.NewString(), "expected_revision": created["revision"]}
		before := planningMutationSnapshot(t, f)
		_, err := f.pool.Exec(t.Context(), `CREATE FUNCTION reject_mcp_planning_effect() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected planning effect failure'; END $$;
CREATE TRIGGER reject_mcp_planning_effect BEFORE INSERT ON work_item_effects FOR EACH STATEMENT EXECUTE FUNCTION reject_mcp_planning_effect()`)
		require.NoError(t, err)
		milestoneFailure(t, call(kind, input), "internal_error")
		require.Equal(t, before, planningMutationSnapshot(t, f))
		_, err = f.pool.Exec(t.Context(), `DROP TRIGGER reject_mcp_planning_effect ON work_item_effects; DROP FUNCTION reject_mcp_planning_effect()`)
		require.NoError(t, err)
		accepted := milestoneObject(t, call(kind, input))
		require.Equal(t, false, accepted["replayed"])
		require.Nil(t, accepted["result"])
		require.NotEqual(t, created["revision"], accepted["revision"])
		var linked, effects int
		require.NoError(t, f.pool.QueryRow(t.Context(), "SELECT count(*) FROM "+pgx.Identifier{kind + "_issues"}.Sanitize()+" WHERE "+pgx.Identifier{kind + "_id"}.Sanitize()+"=$1 AND issue_id=$2 AND deleted_at IS NULL", id, item).Scan(&linked))
		require.Equal(t, 1, linked)
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects
 WHERE kind='activity' AND payload->>'type'=$1 AND payload->>'project_id'=$2 AND payload->>'actor_id'=$3
 AND payload::text LIKE '%' || $4::text || '%'`, kind+".activity.created", testsupport.LabelProject, testsupport.LabelActor, item).Scan(&effects))
		require.Equal(t, 1, effects, "accepted association must retain one scoped durable activity")
		before = planningMutationSnapshot(t, f)
		require.Equal(t, true, milestoneObject(t, call(kind, input))["replayed"])
		require.Equal(t, before, planningMutationSnapshot(t, f))
	})
}
