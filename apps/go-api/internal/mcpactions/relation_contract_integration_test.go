//go:build integration

package mcpactions_test

import (
	"fmt"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPRelationDirectionsSymmetryAndDeletionKinds(t *testing.T) {
	runCatalogue(t, templateFixture(t), func(t *testing.T, f milestoneFixture, root catalogueCall) {
		first := f.item(t)
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": first})
		pairs := map[string]string{"blocking": "blocked_by", "blocked_by": "blocking", "start_before": "start_after", "start_after": "start_before", "finish_before": "finish_after", "finish_after": "finish_before"}
		var second string
		for kind, inverse := range pairs {
			second = f.item(t)
			input := map[string]any{"action": "create", "workitem_ids": []string{second}, "relation_type": kind}
			rows := catalogueArray(t, call("workitem_relation", input))
			require.Len(t, rows, 1)
			require.Equal(t, second, rows[0]["id"])
			require.Equal(t, kind, rows[0]["relation_type"])
			var endpointEvents int
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects WHERE kind='activity' AND payload->>'type'='workitem_relation.activity.created' AND ((payload->>'issue_id'=$1 AND payload->'requested_data'->>'related_id'=$2 AND payload->'requested_data'->>'label'=$3) OR (payload->>'issue_id'=$2 AND payload->'requested_data'->>'related_id'=$1 AND payload->'requested_data'->>'label'=$4)) AND payload->'requested_data'->>'identifier'<>''`, first, second, kind, inverse).Scan(&endpointEvents))
			require.Equal(t, 2, endpointEvents, "both endpoints commit immutable native activity inputs")
			before := coreSnapshot(t, f)
			require.Equal(t, rows, catalogueArray(t, call("workitem_relation", input)))
			require.Equal(t, before, coreSnapshot(t, f), "duplicate creation has no activity or revision effect")
			reverse := milestoneObject(t, root("workitem_relation", map[string]any{"action": "list", "project_id": testsupport.LabelProject, "workitem_id": second}))
			require.Contains(t, fmt.Sprint(reverse["dependencies"].(map[string]any)[inverse]), first)
		}
		definition := milestoneObject(t, root("workitem_relation", map[string]any{"action": "create_definition", "name": "Peers", "outward": "peer", "inward": "peer"}))["id"]
		custom := map[string]any{"action": "create", "workitem_ids": []string{second}, "relation_definition_id": definition, "relation_definition_label": "peer"}
		catalogueArray(t, call("workitem_relation", custom))
		before := coreSnapshot(t, f)
		catalogueArray(t, root("workitem_relation", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "workitem_id": second, "workitem_ids": []string{first}, "relation_definition_id": definition, "relation_definition_label": "peer"}))
		require.Equal(t, before, coreSnapshot(t, f))
		_, err := f.pool.Exec(t.Context(), `INSERT INTO issue_relations(id,workspace_id,project_id,issue_id,related_issue_id,relation_type,created_at,updated_at) VALUES($1,$2,$3,$4,$5,'relates_to',now(),now())`, uuid.New(), testsupport.LabelWorkspace, testsupport.LabelProject, first, second)
		require.NoError(t, err)
		catalogueNull(t, call("workitem_relation", map[string]any{"action": "delete", "related_workitem_id": second, "is_dependency": true}))
		listed := milestoneObject(t, call("workitem_relation", map[string]any{"action": "list"}))
		require.Contains(t, fmt.Sprint(listed["custom"]), second)
		milestoneObject(t, root("workitem_relation", map[string]any{"action": "update_definition", "definition_id": definition, "is_active": false}))
		require.Empty(t, milestoneObject(t, call("workitem_relation", map[string]any{"action": "list"}))["custom"])
		catalogueNull(t, call("workitem_relation", map[string]any{"action": "delete", "related_workitem_id": second}))
		var legacy, customLive int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM issue_relations WHERE relation_type='relates_to' AND deleted_at IS NULL),(SELECT count(*) FROM work_item_custom_relations WHERE deleted_at IS NULL)`).Scan(&legacy, &customLive))
		require.Equal(t, 1, legacy)
		require.Zero(t, customLive)
	})
}
func TestMCPRelationMixedTargetsFailAtomicallyAndHideRevokedEndpoints(t *testing.T) {
	f := templateFixture(t)
	_, foreignProject := addPrivateProjectInOtherWorkspace(t, f)
	var foreign string
	require.NoError(t, f.pool.QueryRow(t.Context(), `INSERT INTO issues(id,workspace_id,project_id,name,description_json,description_html,priority,sort_order,sequence_id,is_draft,created_at,updated_at) SELECT $1,workspace_id,id,'Foreign','{}','','none',0,1,false,now(),now() FROM projects WHERE id=$2 RETURNING id`, uuid.New(), foreignProject).Scan(&foreign))
	runCatalogueCredentials(t, f, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		first, second := f.item(t), f.item(t)
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": first})
		input := map[string]any{"action": "create", "relation_type": "blocking", "workitem_ids": []string{second, foreign}, "request_id": uuid.NewString()}
		before := coreSnapshot(t, f)
		milestoneFailure(t, call("workitem_relation", input), "not_found")
		require.Equal(t, before, coreSnapshot(t, f))
		input["workitem_ids"] = []string{second}
		require.Equal(t, false, milestoneObject(t, call("workitem_relation", input))["replayed"])
		_, err := f.pool.Exec(t.Context(), `UPDATE issues SET is_draft=true WHERE id=$1`, second)
		require.NoError(t, err)
		listed := milestoneObject(t, call("workitem_relation", map[string]any{"action": "list"}))
		require.Empty(t, listed["dependencies"].(map[string]any)["blocking"])
		before = coreSnapshot(t, f)
		milestoneFailure(t, call("workitem_relation", input), "not_found")
		require.Equal(t, before, coreSnapshot(t, f))
		_, err = f.pool.Exec(t.Context(), `UPDATE issues SET is_draft=false WHERE id=$1`, second)
		require.NoError(t, err)
		require.Equal(t, true, milestoneObject(t, call("workitem_relation", input))["replayed"])
	})
}
