//go:build integration

package mcpactions_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectconfiguration"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func labelMutationSnapshot(t *testing.T, f milestoneFixture) []byte {
	t.Helper()
	// Capture all tenants in this isolated fixture so rejected cross-scope calls
	// cannot hide changes to another workspace's labels or receipts.
	var snapshot []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT jsonb_build_object(
 'labels',COALESCE((SELECT jsonb_agg(to_jsonb(l) ORDER BY id) FROM labels l),'[]'::jsonb),
 'revisions',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY label_id) FROM label_edit_revisions r),'[]'::jsonb),
 'effects',COALESCE((SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM work_item_effects e),'[]'::jsonb),
 'receipts',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY workspace_id,action,project_id,request_id) FROM mutation_receipts r WHERE tool='label'),'[]'::jsonb))`).Scan(&snapshot))
	return snapshot
}

func TestMCPLabelReceiptInputNormalization(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		input := map[string]any{"action": "create", "name": uuid.NewString(), "request_id": uuid.NewString()}
		created := milestoneObject(t, call("label", input))
		require.Len(t, created, 6)
		require.Equal(t, false, created["replayed"])
		require.Equal(t, input["request_id"], created["request_id"])
		record := created["result"].(map[string]any)
		require.Len(t, record, 15)
		require.Equal(t, testsupport.LabelProject, record["project"])
		require.Equal(t, testsupport.LabelWorkspace, record["workspace"])

		before := labelMutationSnapshot(t, f)
		input["sort_order"], input["description"], input["external_source"] = nil, "", ""
		replay := milestoneObject(t, call("label", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, created["result"], replay["result"])
		require.Equal(t, created["revision"], replay["revision"])
		require.Equal(t, before, labelMutationSnapshot(t, f))

		input["sort_order"] = 0
		milestoneFailure(t, call("label", input), "request_conflict")
		require.Equal(t, before, labelMutationSnapshot(t, f))
	})
}

func TestMCPLabelReceiptsPreserveNativeEdits(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		input := map[string]any{"action": "create", "name": uuid.NewString(), "request_id": uuid.NewString()}
		created := milestoneObject(t, call("label", input))
		id := created["label_id"].(string)
		native, err := projectconfiguration.NewRepository(f.pool)
		require.NoError(t, err)
		_, err = native.WriteAPILabel(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject,
			id, http.MethodPatch, strings.NewReader(`{"description":"native edit"}`), "application/json")
		require.NoError(t, err)
		detail := milestoneObject(t, call("label", map[string]any{"action": "retrieve", "label_id": id, "include_revision": true}))
		require.Equal(t, "native edit", detail["description"])
		require.NotEqual(t, created["revision"], detail["revision"])

		before := labelMutationSnapshot(t, f)
		replay := milestoneObject(t, call("label", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, created["result"], replay["result"])
		require.Equal(t, before, labelMutationSnapshot(t, f))
		milestoneFailure(t, call("label", map[string]any{"action": "update", "label_id": id,
			"description": "stale edit", "expected_revision": created["revision"], "request_id": uuid.NewString()}), "revision_conflict")
		require.Equal(t, before, labelMutationSnapshot(t, f))

		update := map[string]any{"action": "update", "label_id": id, "sort_order": 0,
			"expected_revision": detail["revision"], "request_id": uuid.NewString()}
		changed := milestoneObject(t, call("label", update))
		require.Equal(t, float64(0), changed["result"].(map[string]any)["sort_order"])
		require.Equal(t, "native edit", changed["result"].(map[string]any)["description"])
		require.NotEqual(t, detail["revision"], changed["revision"])
		before = labelMutationSnapshot(t, f)
		replayedUpdate := milestoneObject(t, call("label", update))
		require.Equal(t, true, replayedUpdate["replayed"])
		require.Equal(t, changed["result"], replayedUpdate["result"])
		require.Equal(t, before, labelMutationSnapshot(t, f))
	})
}

func TestMCPLabelCascadeFailurePreservesRevisionAndReceipt(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		parent := milestoneObject(t, call("label", map[string]any{"action": "create", "name": uuid.NewString(), "request_id": uuid.NewString()}))
		id := parent["label_id"].(string)
		child := milestoneObject(t, call("label", map[string]any{"action": "create", "name": uuid.NewString(), "parent": id}))
		input := map[string]any{"action": "delete", "label_id": id, "request_id": uuid.NewString(), "expected_revision": parent["revision"]}
		before := labelMutationSnapshot(t, f)
		_, err := f.pool.Exec(t.Context(), `CREATE FUNCTION reject_mcp_label_cleanup() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected cleanup failure'; END $$;
CREATE TRIGGER reject_mcp_label_cleanup BEFORE INSERT ON work_item_effects FOR EACH STATEMENT EXECUTE FUNCTION reject_mcp_label_cleanup()`)
		require.NoError(t, err)
		milestoneFailure(t, call("label", input), "internal_error")
		require.Equal(t, before, labelMutationSnapshot(t, f))
		_, err = f.pool.Exec(t.Context(), `DROP TRIGGER reject_mcp_label_cleanup ON work_item_effects; DROP FUNCTION reject_mcp_label_cleanup()`)
		require.NoError(t, err)

		deleted := milestoneObject(t, call("label", input))
		require.Equal(t, false, deleted["replayed"])
		require.Nil(t, deleted["result"])
		require.NotEqual(t, parent["revision"], deleted["revision"])
		var deletedCount, effects int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM labels WHERE id IN ($1,$2) AND deleted_at IS NOT NULL`, id, child["id"]).Scan(&deletedCount))
		require.Equal(t, 2, deletedCount)
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects WHERE payload->>'id' IN ($1,$2)`, id, child["id"]).Scan(&effects))
		require.Equal(t, 2, effects)
		before = labelMutationSnapshot(t, f)
		replay := milestoneObject(t, call("label", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, deleted["revision"], replay["revision"])
		require.Nil(t, replay["result"])
		require.Equal(t, before, labelMutationSnapshot(t, f))
	})
}
