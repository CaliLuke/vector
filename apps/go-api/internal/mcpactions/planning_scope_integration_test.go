//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestMCPPlanningBindsWorkspaceProjectResourceAndReceipt(t *testing.T) {
	f := newMilestoneFixture(t)
	otherWorkspace, otherProject := addPrivateProjectInOtherWorkspace(t, f)
	runSelectedWorkspaceMatrix(t, f, func(t *testing.T, call catalogueCall) {
		project := milestoneObject(t, call("project", map[string]any{"action": "create", "workspace_slug": "workspace-one",
			"name": "Planning scope " + strings.ReplaceAll(uuid.NewString(), "-", ""), "identifier": strings.ToUpper(uuid.NewString()[:8])}))
		_, err := f.pool.Exec(t.Context(), `UPDATE projects SET network=0,cycle_view=true,module_view=true`)
		require.NoError(t, err)
		scopes := []struct{ workspace, workspaceID, project string }{
			{"workspace-one", testsupport.LabelWorkspace, testsupport.LabelProject},
			{"workspace-one", testsupport.LabelWorkspace, project["id"].(string)},
			{"other", otherWorkspace, otherProject},
		}
		for _, kind := range []string{"cycle", "module"} {
			t.Run(kind, func(t *testing.T) {
				key, name := uuid.NewString(), uuid.NewString()
				inputs := make([]map[string]any, len(scopes))
				created := make([]map[string]any, len(scopes))
				for i, scope := range scopes {
					input := map[string]any{"action": "create", "workspace_slug": scope.workspace, "project_id": scope.project, "name": name, "request_id": key}
					if kind == "cycle" {
						input["owned_by"] = testsupport.LabelActor
					}
					inputs[i], created[i] = input, milestoneObject(t, call(kind, input))
					require.Equal(t, false, created[i]["replayed"])
					var workspace, project, savedName string
					require.NoError(t, f.pool.QueryRow(t.Context(), "SELECT workspace_id,project_id,name FROM "+pgx.Identifier{kind + "s"}.Sanitize()+" WHERE id=$1", created[i][kind+"_id"]).Scan(&workspace, &project, &savedName))
					require.Equal(t, scope.workspaceID, workspace)
					require.Equal(t, scope.project, project)
					require.Equal(t, name, savedName)
				}
				require.NotEqual(t, created[0][kind+"_id"], created[1][kind+"_id"])
				require.NotEqual(t, created[0][kind+"_id"], created[2][kind+"_id"])
				require.NotEqual(t, created[1][kind+"_id"], created[2][kind+"_id"])
				primary := catalogueScoped(call, map[string]any{"workspace_slug": "workspace-one", "project_id": testsupport.LabelProject})
				item := f.item(t)
				milestoneObject(t, primary(kind, map[string]any{"action": "manage_workitems", kind + "_id": created[0][kind+"_id"], "add_ids": item, "request_id": uuid.NewString()}))
				before := planningMutationSnapshot(t, f)
				for _, wrong := range []struct{ workspace, project string }{
					{"workspace-one", otherProject}, {"other", testsupport.LabelProject},
				} {
					scoped := catalogueScoped(call, map[string]any{"workspace_slug": wrong.workspace, "project_id": wrong.project})
					milestoneFailure(t, scoped(kind, map[string]any{"action": "list"}), "forbidden")
					// Keep the original receipt key but deliberately cross its scope.
					fresh := map[string]any{"action": "create", "name": name, "request_id": key}
					if kind == "cycle" {
						fresh["owned_by"] = testsupport.LabelActor
					}
					milestoneFailure(t, scoped(kind, fresh), "forbidden")
					milestoneFailure(t, scoped(kind, map[string]any{"action": "update", kind + "_id": created[0][kind+"_id"], "description": "foreign", "request_id": uuid.NewString()}), "forbidden")
				}
				for _, foreign := range []any{created[1][kind+"_id"], created[2][kind+"_id"]} {
					for _, action := range []string{"retrieve", "update", "delete", "archive", "unarchive", "manage_workitems"} {
						input := map[string]any{"action": action, kind + "_id": foreign}
						switch action {
						case "update":
							input["description"] = "foreign"
						case "manage_workitems":
							input["add_ids"] = item
						}
						if action != "retrieve" {
							input["request_id"] = uuid.NewString()
						}
						milestoneFailure(t, primary(kind, input), "not_found")
					}
					if kind == "cycle" {
						milestoneFailure(t, primary(kind, map[string]any{"action": "complete", "cycle_id": foreign, "request_id": uuid.NewString()}), "not_found")
						milestoneFailure(t, primary(kind, map[string]any{"action": "transfer_workitems", "cycle_id": created[0]["cycle_id"], "new_cycle_id": foreign, "request_id": uuid.NewString()}), "not_found")
					}
				}
				require.Equal(t, before, planningMutationSnapshot(t, f))
				for i, input := range inputs {
					replay := milestoneObject(t, call(kind, input))
					require.Equal(t, true, replay["replayed"])
					require.Equal(t, created[i]["result"], replay["result"])
					require.Equal(t, created[i]["revision"], replay["revision"])
				}
				require.Equal(t, before, planningMutationSnapshot(t, f))
			})
		}
	})
}

// Each call selects its workspace; no header fixes a default workspace.
func runSelectedWorkspaceMatrix(t *testing.T, f milestoneFixture, exercise func(*testing.T, catalogueCall)) {
	t.Helper()
	for _, mode := range []string{"personal token", "OAuth"} {
		for _, version := range []string{"2025-11-25", "2026-07-28"} {
			t.Run(mode+"/"+version, func(t *testing.T) {
				handler, token := f.handler(t, nil), "test-one"
				if mode == "OAuth" {
					handler, token = oauthHandler(t, f.fixture, f.oauth.Authenticate), "oauth-user"
				}
				exercise(t, func(tool string, input map[string]any) rpcResult {
					raw, err := json.Marshal(input)
					require.NoError(t, err)
					return oauthToolCall(t, handler, version, token, "", tool, string(raw), http.StatusOK)
				})
			})
		}
	}
}
