package workitems

import (
	"context"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/jackc/pgx/v5"
)

type collaborationParentDB struct {
	Database
	t *testing.T
}

func (db collaborationParentDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	if !strings.Contains(query, "FROM issue_comments") || len(args) != 1 || args[0] != "parent-comment" {
		db.t.Fatalf("comment parent used the wrong relation: %s %#v", query, args)
	}
	return collaborationParentRow{}
}

type collaborationParentRow struct{}

func (collaborationParentRow) Scan(values ...any) error {
	*values[0].(*[]byte) = []byte(`{"id":"parent-comment","project_id":"project"}`)
	return nil
}

func TestCollaborationAPIParentExpansionUsesCommentRelation(t *testing.T) {
	r := &Repository{db: collaborationParentDB{t: t}}
	q := Request{API: true, Method: "GET", Params: map[string]string{"fields": "id,parent", "expand": "parent"}}
	value, err := r.collaborationProjection(context.Background(), q, map[string]any{"id": "child-comment", "parent_id": "parent-comment"}, "comment", false)
	if err != nil {
		t.Fatal(err)
	}
	parent, ok := value["parent"].(map[string]any)
	if !ok || parent["id"] != "parent-comment" || parent["project_id"] != "project" || len(value) != 2 {
		t.Fatalf("comment parent projection: %#v", value)
	}
	q.Params["fields"] = "id"
	value, err = r.collaborationProjection(context.Background(), q, map[string]any{"id": "child-comment", "parent_id": "parent-comment"}, "comment", false)
	if err != nil || len(value) != 1 {
		t.Fatalf("unselected parent expanded: %#v %v", value, err)
	}
}

func TestCollaborationAPIDumpPreservesSerializerFieldOrder(t *testing.T) {
	row := map[string]any{
		"id": "comment-id", "created_at": "2026-01-02T03:04:05Z", "updated_at": "2026-01-02T03:04:06Z", "deleted_at": nil,
		"comment_html": "<p>hello</p>", "attachments": []any{}, "access": "EXTERNAL", "external_source": "fixture", "external_id": "external-1", "edited_at": nil,
		"created_by_id": nil, "updated_by_id": nil, "project_id": "project-id", "workspace_id": "workspace-id", "description_id": "description-id", "issue_id": "issue-id", "actor_id": "actor-id", "parent_id": nil,
	}
	r := &Repository{}
	got, err := r.collaborationDump(context.Background(), Request{API: true, Method: "GET", Timezone: "UTC"}, row, "comment", false)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id": "comment-id", "created_at": "2026-01-02T03:04:05Z", "updated_at": "2026-01-02T03:04:06Z", "deleted_at": null, "comment_html": "<p>hello</p>", "attachments": [], "access": "EXTERNAL", "external_source": "fixture", "external_id": "external-1", "edited_at": null, "created_by": null, "updated_by": null, "project": "project-id", "workspace": "workspace-id", "description": "description-id", "issue": "issue-id", "actor": "actor-id", "parent": null}`
	if got != want {
		t.Fatalf("serialized API activity dump differs\n got: %s\nwant: %s", got, want)
	}
}

func TestCollaborationTimestampProjectionLeavesCommentJSONUntouched(t *testing.T) {
	value := map[string]any{
		"created_at":   "2026-01-02T03:04:05Z",
		"edited_at":    "2026-01-02T03:04:06Z",
		"comment_json": map[string]any{"created_at": "2026-01-02T03:04:07Z", "nested": map[string]any{"updated_at": "2026-01-02T03:04:08Z"}},
	}
	collaborationTimestamps(value, "America/Los_Angeles")
	if value["created_at"] != "2026-01-01T19:04:05-08:00" {
		t.Fatalf("created_at was not localized: %#v", value["created_at"])
	}
	if value["edited_at"] != "2026-01-01T19:04:06-08:00" {
		t.Fatalf("edited_at was not localized: %#v", value["edited_at"])
	}
	commentJSON := value["comment_json"].(map[string]any)
	if commentJSON["created_at"] != "2026-01-02T03:04:07Z" {
		t.Fatalf("comment JSON timestamp was mutated: %#v", commentJSON)
	}
	if commentJSON["nested"].(map[string]any)["updated_at"] != "2026-01-02T03:04:08Z" {
		t.Fatalf("nested comment JSON timestamp was mutated: %#v", commentJSON)
	}
}

func TestCollaborationNestedProjectionDumpRetainsDeclaredChildOrder(t *testing.T) {
	row := map[string]any{"id": "u", "first_name": "A", "last_name": "B", "avatar": "", "avatar_url": nil, "is_bot": false, "display_name": "ab"}
	keys := []string{"id", "first_name", "last_name", "avatar", "avatar_url", "is_bot", "display_name"}
	got, err := pythoncompat.Dumps(row, jsonOrder(keys, nil))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id": "u", "first_name": "A", "last_name": "B", "avatar": "", "avatar_url": null, "is_bot": false, "display_name": "ab"}`
	if got != want {
		t.Fatalf("user-lite serializer field order changed\n got: %s\nwant: %s", got, want)
	}
}
