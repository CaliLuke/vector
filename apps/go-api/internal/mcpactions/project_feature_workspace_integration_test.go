//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPProjectFeaturesBindTheSelectedWorkspace(t *testing.T) {
	f := newMilestoneFixture(t)
	otherWorkspace, otherProject := addPrivateProjectInOtherWorkspace(t, f)
	for _, mode := range []string{"personal token", "OAuth"} {
		for _, version := range []string{"2025-11-25", "2026-07-28"} {
			t.Run(mode+"/"+version, func(t *testing.T) {
				handler, token := f.handler(t, nil), "test-one"
				if mode == "OAuth" {
					handler, token = oauthHandler(t, f.fixture, f.oauth.Authenticate), "oauth-user"
				}
				call := func(input map[string]any) rpcResult {
					raw, marshalErr := json.Marshal(input)
					require.NoError(t, marshalErr)
					return oauthToolCall(t, handler, version, token, "", "project", string(raw), http.StatusOK)
				}
				key := uuid.NewString()
				input := map[string]any{"action": "update_features", "workspace_slug": "other", "project_id": otherProject, "cycles": true, "request_id": key}
				accepted := milestoneObject(t, call(input))
				require.Equal(t, otherProject, accepted["resource_id"])
				require.Equal(t, false, accepted["replayed"])
				before := featureMutationSnapshot(t, f)
				otherBefore := featureMutationSnapshotForWorkspace(t, f, otherWorkspace)
				for _, scope := range []struct{ workspace, project string }{{"workspace-one", otherProject}, {"other", testsupport.LabelProject}} {
					milestoneFailure(t, call(map[string]any{"action": "get_features", "workspace_slug": scope.workspace, "project_id": scope.project}), "forbidden")
					milestoneFailure(t, call(map[string]any{"action": "update_features", "workspace_slug": scope.workspace, "project_id": scope.project, "cycles": true, "request_id": key}), "forbidden")
				}
				require.JSONEq(t, string(before), string(featureMutationSnapshot(t, f)))
				require.JSONEq(t, string(otherBefore), string(featureMutationSnapshotForWorkspace(t, f, otherWorkspace)))
				replay := milestoneObject(t, call(input))
				require.Equal(t, true, replay["replayed"])
				require.Equal(t, accepted["result"], replay["result"])
				require.JSONEq(t, string(otherBefore), string(featureMutationSnapshotForWorkspace(t, f, otherWorkspace)))
			})
		}
	}
}
