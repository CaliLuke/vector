//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestMCPPlanningAssociationsPreserveForeignWorkitems(t *testing.T) {
	f := planningAuthorizationWaitFixture(t)
	otherWorkspace, otherProject := addPrivateProjectInOtherWorkspace(t, f)
	_, err := f.pool.Exec(t.Context(), `UPDATE projects SET cycle_view=true,module_view=true`)
	require.NoError(t, err)
	runSelectedWorkspaceMatrix(t, f, func(t *testing.T, call catalogueCall) {
		project := milestoneObject(t, call("project", map[string]any{"action": "create", "workspace_slug": "workspace-one", "name": "Associations " + strings.ReplaceAll(uuid.NewString(), "-", ""), "identifier": strings.ToUpper(uuid.NewString()[:8]), "cycles": true, "modules": true}))
		for _, kind := range []string{"cycle", "module"} {
			t.Run(kind, func(t *testing.T) {
				primary := catalogueScoped(call, map[string]any{"workspace_slug": "workspace-one", "project_id": testsupport.LabelProject})
				_, created := createPlanningReceipt(t, primary, kind)
				localItem := f.item(t)
				var foreignItems, foreignResources []string
				for _, scope := range []struct{ slug, workspace, project string }{
					{"workspace-one", testsupport.LabelWorkspace, project["id"].(string)},
					{"other", otherWorkspace, otherProject},
				} {
					item := f.itemInScope(t, scope.workspace, scope.project)
					scoped := catalogueScoped(call, map[string]any{"workspace_slug": scope.slug, "project_id": scope.project})
					input := map[string]any{"action": "create", "name": uuid.NewString(), "request_id": uuid.NewString()}
					if kind == "cycle" {
						input["owned_by"] = testsupport.LabelActor
					}
					container := milestoneObject(t, scoped(kind, input))
					id := container[kind+"_id"].(string)
					milestoneObject(t, scoped(kind, map[string]any{"action": "manage_workitems", kind + "_id": id, "add_ids": item, "request_id": uuid.NewString()}))
					foreignItems, foreignResources = append(foreignItems, item), append(foreignResources, id)
				}
				foreignBefore := planningAssociationSnapshot(t, f, kind, foreignItems, foreignResources)
				ids, err := json.Marshal(append([]string{localItem}, foreignItems...))
				require.NoError(t, err)
				input := map[string]any{"action": "manage_workitems", kind + "_id": created[kind+"_id"], "add_ids": string(ids), "request_id": uuid.NewString()}
				accepted := milestoneObject(t, primary(kind, input))
				require.Equal(t, false, accepted["replayed"])
				var linked []string
				require.NoError(t, f.pool.QueryRow(t.Context(), "SELECT COALESCE(array_agg(issue_id::text ORDER BY issue_id),'{}') FROM "+pgx.Identifier{kind + "_issues"}.Sanitize()+" WHERE "+pgx.Identifier{kind + "_id"}.Sanitize()+"=$1 AND deleted_at IS NULL", created[kind+"_id"]).Scan(&linked))
				require.Equal(t, []string{localItem}, linked)
				require.Equal(t, foreignBefore, planningAssociationSnapshot(t, f, kind, foreignItems, foreignResources))
				before := planningMutationSnapshot(t, f)
				for _, foreign := range foreignItems {
					milestoneFailure(t, primary(kind, map[string]any{"action": "manage_workitems", kind + "_id": created[kind+"_id"], "remove_ids": foreign, "request_id": uuid.NewString()}), "not_found")
				}
				require.Equal(t, before, planningMutationSnapshot(t, f))
				replay := milestoneObject(t, primary(kind, input))
				require.Equal(t, true, replay["replayed"])
				require.Equal(t, accepted["revision"], replay["revision"])
				require.Equal(t, before, planningMutationSnapshot(t, f))
			})
		}
	})
}

func planningAssociationSnapshot(t *testing.T, f milestoneFixture, kind string, items, resources []string) []byte {
	t.Helper()
	var snapshot []byte
	table, column := pgx.Identifier{kind + "s"}.Sanitize(), pgx.Identifier{kind + "_id"}.Sanitize()
	query := `SELECT jsonb_build_object(
 'items',(SELECT jsonb_agg(to_jsonb(i) ORDER BY id) FROM issues i WHERE id=ANY($1::uuid[])),
 'item_revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY issue_id) FROM issue_edit_revisions r WHERE issue_id=ANY($1::uuid[])),
 'resources',(SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM ` + table + ` r WHERE id=ANY($2::uuid[])),
 'revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY ` + column + `) FROM ` + pgx.Identifier{kind + "_edit_revisions"}.Sanitize() + ` r WHERE ` + column + `=ANY($2::uuid[])),
 'links',(SELECT jsonb_agg(to_jsonb(l) ORDER BY id) FROM ` + pgx.Identifier{kind + "_issues"}.Sanitize() + ` l WHERE issue_id=ANY($1::uuid[])))`
	require.NoError(t, f.pool.QueryRow(t.Context(), query, items, resources).Scan(&snapshot))
	return snapshot
}
