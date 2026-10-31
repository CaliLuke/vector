//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectconfiguration"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPEstimateNativeEditRejectsPartialCatalogueChanges(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		estimate := milestoneObject(t, call("project_estimate", map[string]any{"action": "create", "name": "Native edit", "type": "points"}))
		id := estimate["id"].(string)
		milestoneObject(t, call("project_estimate", map[string]any{"action": "link", "estimate_id": id}))
		points := catalogueArray(t, call("project_estimate", map[string]any{"action": "create_points", "estimate_id": id, "points": `[{"value":"1","key":0},{"value":"2","key":1}]`}))
		repository, err := projectconfiguration.NewRepository(f.pool)
		require.NoError(t, err)
		input := map[string]any{"estimate": map[string]any{"name": "Rejected edit"}, "estimate_points": []map[string]any{
			{"id": points[0]["id"], "value": "3"}, {"id": points[1]["id"], "value": strings.Repeat("X", 21)},
		}}
		raw, err := json.Marshal(input)
		require.NoError(t, err)
		prior := milestoneObject(t, call("project_estimate", map[string]any{"action": "retrieve", "include_revision": true}))
		before := estimateStorageSnapshot(t, f)
		_, err = repository.PatchEstimate(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject, id, strings.NewReader(string(raw)), "application/json")
		var rejected *projectconfiguration.HTTPError
		require.ErrorAs(t, err, &rejected)
		require.Equal(t, 400, rejected.Status)
		require.Equal(t, before, estimateStorageSnapshot(t, f), "failed native edits must not change rows, revisions or effects seen by MCP")
		require.Equal(t, estimate, milestoneObject(t, call("project_estimate", map[string]any{"action": "retrieve"})))

		input["estimate"].(map[string]any)["name"] = "Accepted edit"
		input["estimate_points"].([]map[string]any)[1]["value"] = "4"
		raw, err = json.Marshal(input)
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), `CREATE FUNCTION reject_native_estimate_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected estimate commit failure'; END $$;
 CREATE CONSTRAINT TRIGGER reject_native_estimate_commit AFTER UPDATE ON estimate_points DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_native_estimate_commit()`)
		require.NoError(t, err)
		result, err := repository.PatchEstimate(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject, id, strings.NewReader(string(raw)), "application/json")
		require.Error(t, err)
		require.Nil(t, result)
		require.Equal(t, before, estimateStorageSnapshot(t, f), "commit failure must roll back the complete native edit")
		_, err = f.pool.Exec(t.Context(), `DROP TRIGGER reject_native_estimate_commit ON estimate_points; DROP FUNCTION reject_native_estimate_commit()`)
		require.NoError(t, err)
		result, err = repository.PatchEstimate(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject, id, strings.NewReader(string(raw)), "application/json")
		require.NoError(t, err)
		require.NotEmpty(t, result)
		latest := milestoneObject(t, call("project_estimate", map[string]any{"action": "retrieve", "include_revision": true}))
		require.Equal(t, "Accepted edit", latest["name"])
		require.NotEqual(t, prior["revision"], latest["revision"])
		current := catalogueArray(t, call("project_estimate", map[string]any{"action": "list_points", "estimate_id": id}))
		require.Len(t, current, 2)
		require.Equal(t, "3", current[0]["value"])
		require.Equal(t, "4", current[1]["value"])
		deletion := map[string]any{"action": "delete", "request_id": uuid.NewString(), "expected_revision": prior["revision"]}
		milestoneFailure(t, call("project_estimate", deletion), "revision_conflict")
		deletion["expected_revision"] = latest["revision"]
		require.Equal(t, false, milestoneObject(t, call("project_estimate", deletion))["replayed"])

	})
}
