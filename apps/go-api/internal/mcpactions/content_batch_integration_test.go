//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPContentCatalogueBatch(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		t.Run("pages and collections", func(t *testing.T) { cataloguePages(t, f, call) })
		t.Run("releases labels and tags", func(t *testing.T) { catalogueReleases(t, f, call) })
	})
}

func cataloguePages(t *testing.T, f milestoneFixture, call catalogueCall) {
	collection := milestoneObject(t, call("collection", map[string]any{"action": "create", "name": "Knowledge", "access": "private"}))
	cid := collection["id"].(string)
	changed := milestoneObject(t, call("collection", map[string]any{"action": "update", "collection_id": cid, "name": "Engineering"}))
	require.Equal(t, "Engineering", changed["name"])
	require.Equal(t, changed, milestoneObject(t, call("collection", map[string]any{"action": "retrieve", "collection_id": cid})))
	require.Equal(t, []map[string]any{changed}, catalogueArray(t, call("collection", map[string]any{"action": "list"})))
	input := map[string]any{"action": "create", "name": "Guide", "description_html": "<p>Original</p>", "collection_id": cid, "request_id": uuid.NewString()}
	receipt := milestoneObject(t, call("page", input))
	pid := receipt["resource_id"].(string)
	require.Equal(t, true, milestoneObject(t, call("page", input))["replayed"])
	page := milestoneObject(t, call("page", map[string]any{"action": "retrieve", "page_id": pid}))
	require.Equal(t, receipt["result"], page)
	updated := milestoneObject(t, call("page", map[string]any{"action": "update", "page_id": pid, "description_html": "<p>Updated</p>"}))
	require.Equal(t, "<p>Updated</p>", updated["description_html"])
	listed := milestoneObject(t, call("page", map[string]any{"action": "list"}))
	require.Equal(t, []any{updated}, listed["results"])
	child := milestoneObject(t, call("page", map[string]any{"action": "create", "name": "Child", "description_html": "<p>Child</p>", "parent_id": pid}))
	require.Equal(t, cid, child["collection_id"])
	pages := milestoneObject(t, call("collection", map[string]any{"action": "list_pages", "collection_id": cid}))
	require.Len(t, pages["results"], 1)
	require.Empty(t, catalogueArray(t, call("collection", map[string]any{"action": "search_pages", "collection_id": cid, "search": "Guide"})))
	second := milestoneObject(t, call("collection", map[string]any{"action": "create", "name": "Other"}))
	secondID := second["id"].(string)
	search := catalogueArray(t, call("collection", map[string]any{"action": "search_pages", "collection_id": secondID, "search": "Guide"}))
	require.Len(t, search, 1)
	require.Equal(t, pid, search[0]["id"])
	filed := milestoneObject(t, call("page", map[string]any{"action": "set_collection", "page_id": pid, "collection_id": secondID}))
	require.Equal(t, secondID, filed["collection_id"])
	movedChild := milestoneObject(t, call("page", map[string]any{"action": "retrieve", "page_id": child["id"]}))
	require.Equal(t, secondID, movedChild["collection_id"])
	catalogueNull(t, call("collection", map[string]any{"action": "remove_page", "collection_id": secondID, "page_collection_id": filed["page_collection_id"]}))
	unfiled := milestoneObject(t, call("page", map[string]any{"action": "retrieve", "page_id": child["id"]}))
	require.Nil(t, unfiled["collection_id"])
	added := catalogueArray(t, call("collection", map[string]any{"action": "add_pages", "collection_id": cid, "page_ids": pid}))
	require.Len(t, added, 1)
	catalogueCollectionMembers(t, f, call, cid)
	item := f.item(t)
	link := milestoneObject(t, call("page", map[string]any{"action": "attach_to_workitem", "project_id": testsupport.LabelProject, "workitem_id": item, "page_id": pid}))
	links := catalogueArray(t, call("page", map[string]any{"action": "list_workitem_pages", "project_id": testsupport.LabelProject, "workitem_id": item}))
	require.Equal(t, []map[string]any{link}, links)
	catalogueNull(t, call("page", map[string]any{"action": "detach_from_workitem", "project_id": testsupport.LabelProject, "workitem_id": item, "workitem_page_id": link["id"]}))
	archived := milestoneObject(t, call("page", map[string]any{"action": "archive", "page_id": pid}))
	require.Equal(t, true, archived["archived"])
	restored := milestoneObject(t, call("page", map[string]any{"action": "archive", "page_id": pid, "archive": false}))
	require.Equal(t, false, restored["archived"])
	milestoneObject(t, call("page", map[string]any{"action": "archive", "page_id": pid}))
	catalogueNull(t, call("page", map[string]any{"action": "delete", "page_id": pid}))
	catalogueNull(t, call("collection", map[string]any{"action": "delete", "collection_id": cid, "archive_pages": false}))
	catalogueNull(t, call("collection", map[string]any{"action": "delete", "collection_id": secondID}))
	var transactions, versions int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FILTER(WHERE kind='page_transaction'),count(*) FILTER(WHERE kind='page_version') FROM work_item_effects WHERE kind IN('page_transaction','page_version')`).Scan(&transactions, &versions))
	require.Positive(t, transactions)
	require.Positive(t, versions)
}

func catalogueCollectionMembers(t *testing.T, f milestoneFixture, call catalogueCall, collection string) {
	t.Helper()
	member := uuid.NewString()
	_, err := f.pool.Exec(t.Context(), `INSERT INTO users SELECT (jsonb_populate_record(NULL::users,to_jsonb(u)||jsonb_build_object('id',$1::text,'username',$1::text,'email',$1::text||'@example.test'))).* FROM users u LIMIT 1`, member)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `INSERT INTO workspace_members SELECT (jsonb_populate_record(NULL::workspace_members,to_jsonb(wm)||jsonb_build_object('id',gen_random_uuid(),'member_id',$1::text,'role',15))).* FROM workspace_members wm WHERE member_id=$2 LIMIT 1`, member, testsupport.LabelActor)
	require.NoError(t, err)
	added := milestoneObject(t, call("collection", map[string]any{"action": "add_member", "collection_id": collection, "user_id": member, "member_access": "view"}))
	require.Equal(t, float64(0), added["access"])
	updated := milestoneObject(t, call("collection", map[string]any{"action": "update_member", "collection_id": collection, "collection_member_id": added["id"], "member_access": "edit"}))
	require.Equal(t, float64(2), updated["access"])
	listed := catalogueArray(t, call("collection", map[string]any{"action": "list_members", "collection_id": collection}))
	require.Contains(t, listed, updated)
	require.Len(t, listed, 2)
	catalogueNull(t, call("collection", map[string]any{"action": "remove_member", "collection_id": collection, "collection_member_id": added["id"]}))
	require.Len(t, catalogueArray(t, call("collection", map[string]any{"action": "list_members", "collection_id": collection})), 1)
}

func catalogueReleases(t *testing.T, f milestoneFixture, call catalogueCall) {
	tag := milestoneObject(t, call("release_tag", map[string]any{"action": "create", "version": uuid.NewString()}))
	tagID := tag["id"].(string)
	updatedTag := milestoneObject(t, call("release_tag", map[string]any{"action": "update", "tag_id": tagID, "commit_hash": "abc123"}))
	require.Equal(t, "abc123", updatedTag["commit_hash"])
	require.Equal(t, updatedTag, milestoneObject(t, call("release_tag", map[string]any{"action": "retrieve", "tag_id": tagID})))
	tags := milestoneObject(t, call("release_tag", map[string]any{"action": "list"}))
	require.Equal(t, []any{updatedTag}, tags["results"])
	label := milestoneObject(t, call("release_label", map[string]any{"action": "create", "name": uuid.NewString(), "color": "#aabbcc"}))
	labelID := label["id"].(string)
	updatedLabel := milestoneObject(t, call("release_label", map[string]any{"action": "update", "label_id": labelID, "color": "#123456"}))
	require.Equal(t, "#123456", updatedLabel["color"])
	input := map[string]any{"action": "create", "name": uuid.NewString(), "tag_id": tagID, "request_id": uuid.NewString()}
	receipt := milestoneObject(t, call("release", input))
	id := receipt["resource_id"].(string)
	require.Equal(t, true, milestoneObject(t, call("release", input))["replayed"])
	updated := milestoneObject(t, call("release", map[string]any{"action": "update", "release_id": id, "status": "released", "release_date": "2026-10-27"}))
	require.Equal(t, true, updated["is_latest"])
	require.Equal(t, updated, milestoneObject(t, call("release", map[string]any{"action": "retrieve", "release_id": id})))
	listed := milestoneObject(t, call("release", map[string]any{"action": "list"}))
	require.Equal(t, []any{updated}, listed["results"])
	changelog := milestoneObject(t, call("release", map[string]any{"action": "update_changelog", "release_id": id, "description_stripped": "Fixed <escaping>"}))
	require.Equal(t, changelog, milestoneObject(t, call("release", map[string]any{"action": "get_changelog", "release_id": id})))
	require.Contains(t, changelog["changelog"].(map[string]any)["description_html"], "&lt;escaping&gt;")
	catalogueNull(t, call("release_label", map[string]any{"action": "attach", "release_id": id, "label_ids": labelID}))
	labels := milestoneObject(t, call("release_label", map[string]any{"action": "list", "release_id": id}))
	require.Equal(t, []any{updatedLabel}, labels["results"])
	catalogueNull(t, call("release_label", map[string]any{"action": "detach", "release_id": id, "label_ids": labelID}))
	item := f.item(t)
	catalogueNull(t, call("release", map[string]any{"action": "manage_workitems", "release_id": id, "add_ids": item}))
	items := milestoneObject(t, call("release", map[string]any{"action": "list_workitems", "release_id": id}))
	require.Len(t, items["results"], 1)
	catalogueNull(t, call("release", map[string]any{"action": "manage_workitems", "release_id": id, "add_ids": item, "remove_ids": item}))
	empty := milestoneObject(t, call("release", map[string]any{"action": "list_workitems", "release_id": id}))
	require.Empty(t, empty["results"])
	catalogueNull(t, call("release_tag", map[string]any{"action": "delete", "tag_id": tagID}))
	require.Nil(t, milestoneObject(t, call("release", map[string]any{"action": "retrieve", "release_id": id}))["tag"])
	catalogueNull(t, call("release_label", map[string]any{"action": "delete", "label_id": labelID}))
	catalogueNull(t, call("release", map[string]any{"action": "delete", "release_id": id}))
	var retained bool
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT deleted_at IS NULL FROM issues WHERE id=$1`, item).Scan(&retained))
	require.True(t, retained)
}
