//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	states "github.com/CaliLuke/go-plane/internal/projectstatereads"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	googleuuid "github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func stateToolCall(f milestoneFixture, t *testing.T, mode, version string) func(map[string]any) rpcResult {
	t.Helper()
	pat, oauth := f.handler(t, nil), oauthHandler(t, f.fixture, f.oauth.Authenticate)
	return func(input map[string]any) rpcResult {
		body, err := json.Marshal(input)
		require.NoError(t, err)
		if mode == "OAuth" {
			return oauthToolCall(t, oauth, version, "oauth-user", "workspace-one", "state", string(body), http.StatusOK)
		}
		return rpcVersion(t, pat, "test-one", "tools/call", map[string]any{"name": "state", "arguments": input}, version)
	}
}
func TestStateMCPActionsAcrossCredentialsAndProtocols(t *testing.T) {
	f := newMilestoneFixture(t)
	native, err := states.NewWriteRepository(f.pool)
	require.NoError(t, err)
	workspace := uuid.MustParse(testsupport.LabelWorkspace)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, mode := range []string{"personal token", "OAuth"} {
			t.Run(version+"/"+mode, func(t *testing.T) {
				call := stateToolCall(f, t, mode, version)
				project := func(input map[string]any) rpcResult {
					input["project_id"] = testsupport.LabelProject
					return call(input)
				}
				created := milestoneObject(t, project(map[string]any{"action": "create", "name": "Ready", "color": "#123456", "sequence": nil, "default": nil}))
				require.Len(t, created, 18)
				require.Equal(t, "ready", created["slug"])
				require.Equal(t, "backlog", created["group"])
				require.Equal(t, false, created["default"])
				id := created["id"].(string)
				detail := milestoneObject(t, project(map[string]any{"action": "retrieve", "state_id": id}))
				require.Equal(t, created, detail)
				value, readErr := native.ReadAPIV1(t.Context(), testsupport.LabelActor, states.APIV1Request{Operation: "detail", ExpectedWorkspaceID: &workspace, Params: map[string]string{"slug": "workspace-one", "project_id": testsupport.LabelProject, "state_id": id}})
				require.NoError(t, readErr)
				body, marshalErr := json.Marshal(djangoapi.Timestamps(value, "UTC"))
				require.NoError(t, marshalErr)
				require.JSONEq(t, string(body), string(project(map[string]any{"action": "retrieve", "state_id": id}).StructuredContent))
				workspaceDetail := milestoneObject(t, call(map[string]any{"action": "retrieve", "state_id": id, "project_id": ""}))
				require.Equal(t, detail, workspaceDetail)
				page := milestoneObject(t, call(map[string]any{"action": "list", "per_page": 0, "cursor": ""}))
				require.Len(t, page, 7)
				require.Equal(t, float64(1), page["total_count"])
				require.Equal(t, []any{created}, page["results"])
				changed := milestoneObject(t, project(map[string]any{"action": "update", "state_id": id, "name": "", "sequence": 0, "default": false, "external_source": "import", "external_id": "1"}))
				require.Equal(t, "Ready", changed["name"])
				require.Equal(t, float64(0), changed["sequence"])
				require.Equal(t, false, changed["default"])
				require.Equal(t, "import", changed["external_source"])
				require.Equal(t, "1", changed["external_id"])
				result := project(map[string]any{"action": "delete", "state_id": id})
				require.False(t, result.IsError, result.Content)
				require.Empty(t, result.StructuredContent)
				require.Empty(t, result.Content)
				milestoneFailure(t, project(map[string]any{"action": "retrieve", "state_id": id}), "not_found")
				empty := milestoneObject(t, call(map[string]any{"action": "list"}))
				require.Empty(t, empty["results"])
				milestoneFailure(t, call(map[string]any{"action": "create", "name": "Workspace", "color": "#123456", "group": "backlog", "sequence": nil, "default": nil}), "workspace_not_managed")
				milestoneFailure(t, call(map[string]any{"action": "update", "state_id": id}), "workspace_not_managed")
				milestoneFailure(t, call(map[string]any{"action": "delete", "state_id": id}), "workspace_not_managed")
			})
		}
	}
}
func TestStateMCPReceiptsAndRevisionsAcrossCredentialsAndProtocols(t *testing.T) {
	f := newMilestoneFixture(t)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, mode := range []string{"personal token", "OAuth"} {
			t.Run(version+"/"+mode, func(t *testing.T) {
				call := stateToolCall(f, t, mode, version)
				project := func(input map[string]any) rpcResult {
					input["project_id"] = testsupport.LabelProject
					return call(input)
				}
				input := map[string]any{"action": "create", "name": googleuuid.NewString(), "color": "#123456", "request_id": googleuuid.NewString()}
				created := milestoneObject(t, project(input))
				require.Len(t, created, 6)
				require.False(t, created["replayed"].(bool))
				id, rev := created["state_id"].(string), created["revision"].(string)
				input["sequence"], input["default"], input["external_source"] = nil, nil, ""
				replay := milestoneObject(t, project(input))
				require.True(t, replay["replayed"].(bool))
				require.Equal(t, created["result"], replay["result"])
				input["color"] = "#ffffff"
				milestoneFailure(t, project(input), "request_conflict")
				detail := milestoneObject(t, project(map[string]any{"action": "retrieve", "state_id": id, "include_revision": true}))
				require.Len(t, detail, 19)
				require.Equal(t, rev, detail["revision"])
				update := map[string]any{"action": "update", "state_id": id, "sequence": 0, "default": false, "expected_revision": rev, "request_id": googleuuid.NewString()}
				changed := milestoneObject(t, project(update))
				require.NotEqual(t, rev, changed["revision"])
				repeated := milestoneObject(t, project(update))
				require.True(t, repeated["replayed"].(bool))
				require.Equal(t, changed["result"], repeated["result"])
				milestoneFailure(t, project(map[string]any{"action": "update", "state_id": id, "expected_revision": rev}), "revision_conflict")
				deletion := map[string]any{"action": "delete", "state_id": id, "request_id": googleuuid.NewString(), "expected_revision": changed["revision"]}
				deleted := milestoneObject(t, project(deletion))
				require.Nil(t, deleted["result"])
				again := milestoneObject(t, project(deletion))
				require.True(t, again["replayed"].(bool))
				require.Equal(t, deleted["revision"], again["revision"])
				var effects int
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects WHERE payload->>'id'=$1`, id).Scan(&effects))
				require.Equal(t, 1, effects)
			})
		}
	}
}
