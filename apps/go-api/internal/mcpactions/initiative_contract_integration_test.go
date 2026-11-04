//go:build integration

package mcpactions_test

import (
	"context"
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func initiativeFixture(t *testing.T) milestoneFixture {
	t.Helper()
	f := templateFixture(t)
	_, err := f.pool.Exec(t.Context(), `UPDATE workspace_features SET initiatives=true`)
	require.NoError(t, err)
	return f
}

func initiativeSnapshot(t *testing.T, f milestoneFixture) []byte {
	t.Helper()
	return initiativeSnapshotQuery(t, t.Context(), f.pool.QueryRow)
}

func initiativeSnapshotQuery(t *testing.T, ctx context.Context, query func(context.Context, string, ...any) pgx.Row) []byte {
	t.Helper()
	var raw string
	require.NoError(t, query(ctx, `SELECT jsonb_build_object(
 'records',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM initiatives t),
 'projects',(SELECT jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text) FROM initiative_projects t),
 'items',(SELECT jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text) FROM initiative_work_items t),
 'events',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM initiative_events t),
 'revisions',(SELECT jsonb_agg(to_jsonb(t) ORDER BY workspace_id) FROM initiative_revisions t),
 'receipts',(SELECT jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text) FROM mutation_receipts t),
 'effects',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM work_item_effects t))::text`).Scan(&raw))
	return []byte(raw)
}

func initiativeWriteInput(t *testing.T, f milestoneFixture, call catalogueCall, action string) map[string]any {
	t.Helper()
	input := map[string]any{"action": "create", "workspace_slug": "workspace-one", "name": uuid.NewString()}
	if action != "create" {
		created := milestoneObject(t, call("initiative", input))
		input = map[string]any{"action": action, "workspace_slug": "workspace-one", "initiative_id": created["id"]}
		switch action {
		case "update":
			input["name"] = "Updated initiative"
		case "add_projects", "remove_projects":
			input["project_ids"] = testsupport.LabelProject
			if action == "remove_projects" {
				catalogueNull(t, call("initiative", map[string]any{"action": "add_projects", "workspace_slug": "workspace-one", "initiative_id": created["id"], "project_ids": testsupport.LabelProject}))
			}
		case "manage_workitems":
			input["add_ids"] = f.item(t)
		}
	}
	input["request_id"] = uuid.NewString()
	return input
}

func TestMCPInitiativeFieldsValidationAndAtomicAssociations(t *testing.T) {
	runCatalogue(t, initiativeFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		_, otherProject := addPrivateProjectInOtherWorkspace(t, f)
		created := milestoneObject(t, call("initiative", map[string]any{"action": "create", "name": "  Delivery  ", "description_html": "<p>Plan</p><script>alert(1)</script>", "start_date": "2026-11-01", "end_date": "2026-11-30", "lead": testsupport.LabelActor}))
		require.Equal(t, "Delivery", created["name"])
		require.Equal(t, "DRAFT", created["state"])
		require.Equal(t, testsupport.LabelActor, created["lead"])
		require.Equal(t, testsupport.LabelWorkspace, created["workspace"])
		require.Equal(t, "2026-11-01", created["start_date"])
		require.Equal(t, "2026-11-30", created["end_date"])
		require.NotContains(t, created["description_html"], "<script")
		require.Contains(t, created["description_stripped"], "Plan")
		for _, state := range []string{"PLANNED", "ACTIVE", "COMPLETED", "CLOSED"} {
			result := milestoneObject(t, call("initiative", map[string]any{"action": "update", "initiative_id": created["id"], "state": state}))
			require.Equal(t, state, result["state"])
			for _, key := range []string{"name", "description_html", "description_stripped", "start_date", "end_date", "lead"} {
				require.Equal(t, created[key], result[key], key)
			}
		}
		valid := map[string]any{"action": "update", "initiative_id": created["id"]}
		for _, invalid := range []map[string]any{
			{"name": "  "}, {"start_date": "2026-02-30"}, {"end_date": "2026-10-01"}, {"lead": uuid.NewString()},
			{"action": "add_projects", "project_ids": testsupport.LabelProject + "," + otherProject},
			{"action": "manage_workitems", "add_ids": f.item(t) + "," + uuid.NewString()},
		} {
			input := maps.Clone(valid)
			maps.Copy(input, invalid)
			input["request_id"] = uuid.NewString()
			before := initiativeSnapshot(t, f)
			result := call("initiative", input)
			require.True(t, result.IsError, result)
			require.Equal(t, before, initiativeSnapshot(t, f), "rejected batches must preserve rows, revisions, receipts and effects")
			corrected := maps.Clone(valid)
			corrected["name"], corrected["request_id"] = "Corrected", input["request_id"]
			require.Equal(t, false, milestoneObject(t, call("initiative", corrected))["replayed"])
		}
	})
}
