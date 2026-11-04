//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type releaseMutationCase struct{ tool, action string }

var releaseMutationCases = []releaseMutationCase{
	{"release", "create"}, {"release", "update"}, {"release", "delete"}, {"release", "update_changelog"}, {"release", "manage_workitems"},
	{"release_tag", "create"}, {"release_tag", "update"}, {"release_tag", "delete"},
	{"release_label", "create"}, {"release_label", "update"}, {"release_label", "delete"}, {"release_label", "attach"}, {"release_label", "detach"},
}

func releaseSnapshot(t *testing.T, f milestoneFixture) []byte {
	t.Helper()
	return releaseSnapshotQuery(t, t.Context(), f.pool.QueryRow)
}
func releaseSnapshotQuery(t *testing.T, ctx context.Context, query func(context.Context, string, ...any) pgx.Row) []byte {
	t.Helper()
	var raw string
	require.NoError(t, query(ctx, `SELECT jsonb_build_object(
 'releases',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM work_item_releases t),
 'tags',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM release_tags t),
 'labels',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM release_labels t),
 'bindings',(SELECT jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text) FROM release_item_labels t),
 'items',(SELECT jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text) FROM release_work_items t),
 'changelogs',(SELECT jsonb_agg(to_jsonb(t) ORDER BY release_id) FROM release_changelogs t),
 'events',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM release_events t),
 'revisions',(SELECT jsonb_agg(to_jsonb(t) ORDER BY workspace_id) FROM release_revisions t),
 'receipts',(SELECT jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text) FROM mutation_receipts t),
 'effects',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM work_item_effects t))::text`).Scan(&raw))
	return []byte(raw)
}
func releaseWriteInput(t *testing.T, f milestoneFixture, call catalogueCall, tc releaseMutationCase) map[string]any {
	t.Helper()
	field, selector := "name", "release_id"
	if tc.tool == "release_tag" {
		field, selector = "version", "tag_id"
	}
	if tc.tool == "release_label" {
		selector = "label_id"
	}
	input := map[string]any{"action": "create", "workspace_slug": "workspace-one", field: uuid.NewString()}
	if tc.action != "create" {
		created := milestoneObject(t, call(tc.tool, input))
		input = map[string]any{"action": tc.action, "workspace_slug": "workspace-one", selector: created["id"]}
		switch tc.action {
		case "update":
			input[field] = uuid.NewString()
		case "update_changelog":
			input["description_stripped"] = "Release notes"
		case "manage_workitems":
			input["add_ids"] = f.item(t)
		case "attach", "detach":
			release := milestoneObject(t, call("release", map[string]any{"action": "create", "workspace_slug": "workspace-one", "name": uuid.NewString()}))
			input["release_id"], input["label_ids"] = release["id"], created["id"]
			delete(input, "label_id")
			if tc.action == "detach" {
				catalogueNull(t, call(tc.tool, map[string]any{"action": "attach", "workspace_slug": "workspace-one", "release_id": release["id"], "label_ids": created["id"]}))
			}
		}
	}
	input["request_id"] = uuid.NewString()
	return input
}
