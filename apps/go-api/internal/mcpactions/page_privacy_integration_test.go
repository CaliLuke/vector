//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/pagereads"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPageCollectionsPreservePrivateAncestorsAndOwnerRights(t *testing.T) {
	f := templateFixture(t)
	other, _ := customerReferenceTarget(t, f, "USER")
	repo := pagereads.NewPageAPIRepository(f.pool)
	q := pagereads.PageAPIQuery{ActorID: uuid.MustParse(other), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), WorkspaceSlug: "workspace-one"}
	runCatalogue(t, f, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		collection := milestoneObject(t, call("collection", map[string]any{"action": "create", "name": "Private folder", "access": "private"}))["id"].(string)
		parent := milestoneObject(t, call("page", map[string]any{"action": "create", "name": "Private root", "description_html": "<p>Hidden</p>", "access": 1, "collection_id": collection}))["id"].(string)
		child := milestoneObject(t, call("page", map[string]any{"action": "create", "name": "Public child", "description_html": "<p>Still hidden</p>", "parent_id": parent}))["id"].(string)
		public := milestoneObject(t, call("page", map[string]any{"action": "create", "name": "Visible root", "description_html": "<p>Visible</p>", "collection_id": collection}))["id"].(string)
		cq := q
		cq.Tool = "collection"
		cq.Action = "retrieve"
		cq.CollectionID = uuid.MustParse(collection)
		_, err := repo.ReadPageAPI(t.Context(), cq)
		require.ErrorIs(t, err, projectmutations.ErrForbidden)
		// Even an administrator needs explicit private collection membership.
		_, err = f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20 WHERE member_id=$1`, other)
		require.NoError(t, err)
		_, err = repo.ReadPageAPI(t.Context(), cq)
		require.ErrorIs(t, err, projectmutations.ErrForbidden)
		member := milestoneObject(t, call("collection", map[string]any{"action": "add_member", "collection_id": collection, "user_id": other, "member_access": "view"}))
		pq := q
		pq.Tool = "page"
		pq.Action = "retrieve"
		for _, id := range []string{parent, child} {
			pq.PageID = uuid.MustParse(id)
			_, err = repo.ReadPageAPI(t.Context(), pq)
			require.ErrorIs(t, err, projectmutations.ErrForbidden)
		}
		pq.PageID = uuid.MustParse(public)
		_, err = repo.ReadPageAPI(t.Context(), pq)
		require.NoError(t, err)
		cq.Action = "list_pages"
		raw, err := repo.ReadPageAPI(t.Context(), cq)
		require.NoError(t, err)
		var listed map[string]any
		require.NoError(t, json.Unmarshal(raw, &listed))
		require.Len(t, listed["results"], 1)
		require.EqualValues(t, 1, listed["total_results"])
		update := pagereads.PageAPICommand{PageAPIQuery: pq, Fields: map[string]any{"name": "Editor title"}}
		update.Action = "update"
		before := pageSnapshot(t, f)
		_, err = repo.MutatePageAPI(t.Context(), update)
		require.ErrorIs(t, err, projectmutations.ErrForbidden)
		require.Equal(t, before, pageSnapshot(t, f))
		milestoneObject(t, call("collection", map[string]any{"action": "update_member", "collection_id": collection, "collection_member_id": member["id"], "member_access": "edit"}))
		request := uuid.New()
		update.RequestID = &request
		_, err = repo.MutatePageAPI(t.Context(), update)
		require.NoError(t, err)
		// Editor rights do not grant ownership of the folder or another private page.
		forbidden := pagereads.PageAPICommand{PageAPIQuery: cq}
		for _, action := range []string{"delete", "add_member"} {
			forbidden.Action = action
			forbidden.Fields = nil
			forbidden.UserID = uuid.Nil
			if action == "add_member" {
				forbidden.UserID = uuid.MustParse(testsupport.LabelActor)
				forbidden.Fields = map[string]any{"member_access": 2}
			}
			_, err = repo.MutatePageAPI(t.Context(), forbidden)
			require.ErrorIs(t, err, projectmutations.ErrForbidden)
		}
		update.PageID = uuid.MustParse(child)
		update.RequestID = nil
		_, err = repo.MutatePageAPI(t.Context(), update)
		require.ErrorIs(t, err, projectmutations.ErrForbidden)
		milestoneObject(t, call("collection", map[string]any{"action": "update_member", "collection_id": collection, "collection_member_id": member["id"], "member_access": "view"}))
		update.PageID = uuid.MustParse(public)
		update.RequestID = &request
		_, err = repo.MutatePageAPI(t.Context(), update)
		require.ErrorIs(t, err, projectmutations.ErrForbidden)
		var ownerMember string
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT id FROM page_collection_members WHERE collection_id=$1 AND member_id=$2`, collection, testsupport.LabelActor).Scan(&ownerMember))
		for _, action := range []string{"update_member", "remove_member"} {
			input := map[string]any{"action": action, "collection_id": collection, "collection_member_id": ownerMember}
			if action == "update_member" {
				input["member_access"] = "view"
			}
			before = pageSnapshot(t, f)
			milestoneFailure(t, call("collection", input), "invalid_params")
			require.Equal(t, before, pageSnapshot(t, f))
		}
		catalogueNull(t, call("collection", map[string]any{"action": "delete", "collection_id": collection}))
		pq.PageID = uuid.MustParse(child)
		_, err = repo.ReadPageAPI(t.Context(), pq)
		require.ErrorIs(t, err, projectmutations.ErrForbidden)
		var archived int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM pages WHERE id=ANY($1::uuid[]) AND archived_at IS NOT NULL`, []string{parent, child, public}).Scan(&archived))
		require.Equal(t, 3, archived)
	})
}
func TestPageCollectionMovesAreAtomicAcrossWholeSubtrees(t *testing.T) {
	runCatalogue(t, templateFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		collection := milestoneObject(t, call("collection", map[string]any{"action": "create", "name": "Destination"}))["id"]
		root := milestoneObject(t, call("page", map[string]any{"action": "create", "name": "Root", "description_html": "<p>Original</p>"}))["id"]
		child := milestoneObject(t, call("page", map[string]any{"action": "create", "name": "Child", "description_html": "<p>Child</p>", "parent_id": root, "is_locked": true}))["id"]
		move := map[string]any{"action": "add_pages", "collection_id": collection, "page_ids": root, "request_id": uuid.NewString()}
		before := pageSnapshot(t, f)
		milestoneFailure(t, call("collection", move), "invalid_params")
		require.Equal(t, before, pageSnapshot(t, f))
		_, err := f.pool.Exec(t.Context(), `UPDATE pages SET is_locked=false WHERE id=$1`, child)
		require.NoError(t, err)
		milestoneObject(t, call("collection", move))
		require.Equal(t, true, milestoneObject(t, call("collection", move))["replayed"])
		before = pageSnapshot(t, f)
		milestoneFailure(t, call("page", map[string]any{"action": "set_collection", "page_id": child, "collection_id": collection}), "invalid_params")
		require.Equal(t, before, pageSnapshot(t, f))
		milestoneObject(t, call("page", map[string]any{"action": "archive", "page_id": root}))
		milestoneObject(t, call("page", map[string]any{"action": "archive", "page_id": child, "archive": false}))
		restored := milestoneObject(t, call("page", map[string]any{"action": "retrieve", "page_id": child}))
		require.Nil(t, restored["parent_id"])
		require.Nil(t, restored["archived_at"])
		before = pageSnapshot(t, f)
		milestoneFailure(t, call("page", map[string]any{"action": "delete", "page_id": child}), "invalid_params")
		require.Equal(t, before, pageSnapshot(t, f))
	})
}
