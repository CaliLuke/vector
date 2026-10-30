//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPLabelsBindWorkspaceProjectAndResource(t *testing.T) {
	f := newMilestoneFixture(t)
	otherWorkspace, otherProject := addPrivateProjectInOtherWorkspace(t, f)
	for _, mode := range []string{"personal token", "OAuth"} {
		for _, version := range []string{"2025-11-25", "2026-07-28"} {
			t.Run(mode+"/"+version, func(t *testing.T) {
				handler, token := f.handler(t, nil), "test-one"
				if mode == "OAuth" {
					handler, token = oauthHandler(t, f.fixture, f.oauth.Authenticate), "oauth-user"
				}
				call := func(tool string, input map[string]any) rpcResult {
					raw, err := json.Marshal(input)
					require.NoError(t, err)
					return oauthToolCall(t, handler, version, token, "", tool, string(raw), http.StatusOK)
				}
				project := milestoneObject(t, call("project", map[string]any{"action": "create", "workspace_slug": "workspace-one",
					"name": "Label scope " + strings.ReplaceAll(uuid.NewString(), "-", ""), "identifier": strings.ToUpper(uuid.NewString()[:8])}))
				_, err := f.pool.Exec(t.Context(), `UPDATE projects SET network=0`)
				require.NoError(t, err)
				key, name := uuid.NewString(), uuid.NewString()
				scopes := []struct{ workspace, workspaceID, project string }{
					{"workspace-one", testsupport.LabelWorkspace, testsupport.LabelProject},
					{"workspace-one", testsupport.LabelWorkspace, project["id"].(string)},
					{"other", otherWorkspace, otherProject},
				}
				inputs := make([]map[string]any, len(scopes))
				created := make([]map[string]any, len(scopes))
				for i, scope := range scopes {
					inputs[i] = map[string]any{"action": "create", "workspace_slug": scope.workspace, "project_id": scope.project, "name": name, "request_id": key}
					created[i] = milestoneObject(t, call("label", inputs[i]))
					require.Equal(t, false, created[i]["replayed"])
					require.Equal(t, scope.project, created[i]["result"].(map[string]any)["project"])
					require.Equal(t, scope.workspaceID, created[i]["result"].(map[string]any)["workspace"])
					var savedWorkspace, savedProject, savedName string
					require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT workspace_id,project_id,name FROM labels WHERE id=$1`, created[i]["label_id"]).Scan(&savedWorkspace, &savedProject, &savedName))
					require.Equal(t, scope.workspaceID, savedWorkspace)
					require.Equal(t, scope.project, savedProject)
					require.Equal(t, name, savedName)
				}
				require.NotEqual(t, created[0]["label_id"], created[1]["label_id"])
				require.NotEqual(t, created[0]["label_id"], created[2]["label_id"])
				require.NotEqual(t, created[1]["label_id"], created[2]["label_id"])
				before := labelMutationSnapshot(t, f)
				for _, wrong := range []struct{ workspace, project, label string }{
					{"workspace-one", otherProject, created[2]["label_id"].(string)},
					{"other", testsupport.LabelProject, created[0]["label_id"].(string)},
				} {
					base := map[string]any{"workspace_slug": wrong.workspace, "project_id": wrong.project}
					scoped := catalogueScoped(call, base)
					milestoneFailure(t, scoped("label", map[string]any{"action": "list"}), "forbidden")
					milestoneFailure(t, scoped("label", map[string]any{"action": "retrieve", "label_id": wrong.label}), "forbidden")
					milestoneFailure(t, scoped("label", map[string]any{"action": "create", "name": name, "request_id": key}), "forbidden")
					milestoneFailure(t, scoped("label", map[string]any{"action": "update", "label_id": wrong.label, "description": "foreign", "request_id": uuid.NewString()}), "forbidden")
					milestoneFailure(t, scoped("label", map[string]any{"action": "delete", "label_id": wrong.label, "request_id": uuid.NewString()}), "forbidden")
				}
				scoped := catalogueScoped(call, map[string]any{"workspace_slug": "workspace-one", "project_id": testsupport.LabelProject})
				for _, foreign := range []string{created[1]["label_id"].(string), created[2]["label_id"].(string)} {
					milestoneFailure(t, scoped("label", map[string]any{"action": "retrieve", "label_id": foreign}), "not_found")
					milestoneFailure(t, scoped("label", map[string]any{"action": "update", "label_id": foreign, "description": "foreign", "request_id": uuid.NewString()}), "not_found")
					milestoneFailure(t, scoped("label", map[string]any{"action": "delete", "label_id": foreign, "request_id": uuid.NewString()}), "not_found")
					milestoneFailure(t, scoped("label", map[string]any{"action": "create", "name": uuid.NewString(), "parent": foreign, "request_id": uuid.NewString()}), "invalid_params")
				}
				require.Equal(t, before, labelMutationSnapshot(t, f))
				for i, input := range inputs {
					replay := milestoneObject(t, call("label", input))
					require.Equal(t, true, replay["replayed"])
					require.Equal(t, created[i]["result"], replay["result"])
				}
				require.Equal(t, before, labelMutationSnapshot(t, f))
			})
		}
	}
}
