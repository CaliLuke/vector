//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/pagereads"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPPageScopeAndGuestVisibility(t *testing.T) {
	f := templateFixture(t)
	foreignWorkspace, _ := addPrivateProjectInOtherWorkspace(t, f)
	repo := pagereads.NewPageAPIRepository(f.pool)
	foreign := pagereads.PageAPICommand{PageAPIQuery: pagereads.PageAPIQuery{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(foreignWorkspace), WorkspaceSlug: "other", Tool: "collection", Action: "create"}, Fields: map[string]any{"name": "Foreign folder"}}
	collection, err := repo.MutatePageAPI(t.Context(), foreign)
	require.NoError(t, err)
	foreign.Tool = "page"
	foreign.Fields = map[string]any{"name": "Foreign page", "description_html": "<p>Foreign</p>"}
	page, err := repo.MutatePageAPI(t.Context(), foreign)
	require.NoError(t, err)
	other, _ := customerReferenceTarget(t, f, "USER")
	runCatalogueCredentials(t, f, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, action := range []string{"retrieve", "list_pages", "list_members", "search_pages"} {
			milestoneFailure(t, call("collection", map[string]any{"action": action, "collection_id": collection.ID}), "not_found")
		}
		milestoneFailure(t, call("page", map[string]any{"action": "retrieve", "page_id": page.ID}), "not_found")
		before := pageSnapshot(t, f)
		milestoneFailure(t, call("page", map[string]any{"action": "update", "page_id": page.ID, "name": "Cross tenant"}), "not_found")
		milestoneFailure(t, call("collection", map[string]any{"action": "update", "collection_id": collection.ID, "name": "Cross tenant"}), "not_found")
		require.Equal(t, before, pageSnapshot(t, f))
		local := milestoneObject(t, call("page", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": "Project page", "description_html": "<p>Project</p>"}))["id"]
		milestoneFailure(t, call("page", map[string]any{"action": "retrieve", "page_id": local}), "not_found")
		_, err = f.pool.Exec(t.Context(), `UPDATE pages SET owned_by_id=$2 WHERE id=$1`, local, other)
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=5 WHERE member_id=$1`, testsupport.LabelActor)
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), `UPDATE projects SET guest_view_all_features=false WHERE id=$1`, testsupport.LabelProject)
		require.NoError(t, err)
		milestoneFailure(t, call("page", map[string]any{"action": "retrieve", "project_id": testsupport.LabelProject, "page_id": local}), "forbidden")
		listed := milestoneObject(t, call("page", map[string]any{"action": "list", "project_id": testsupport.LabelProject}))
		require.Empty(t, listed["results"])
		_, err = f.pool.Exec(t.Context(), `UPDATE projects SET guest_view_all_features=true WHERE id=$1`, testsupport.LabelProject)
		require.NoError(t, err)
		require.Equal(t, local, milestoneObject(t, call("page", map[string]any{"action": "retrieve", "project_id": testsupport.LabelProject, "page_id": local}))["id"])
		before = pageSnapshot(t, f)
		milestoneFailure(t, call("page", map[string]any{"action": "update", "project_id": testsupport.LabelProject, "page_id": local, "name": "Guest edit"}), "forbidden")
		require.Equal(t, before, pageSnapshot(t, f))
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=20 WHERE member_id=$1`, testsupport.LabelActor)
		require.NoError(t, err)
	})
}
