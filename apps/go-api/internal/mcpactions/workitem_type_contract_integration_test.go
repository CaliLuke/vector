//go:build integration

package mcpactions_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// SDK 0.3.1 fields and the pinned handler's omitted-string semantics coexist
// with Vector's native rule that new definitions start inactive.
func TestMCPWorkitemTypeSDKFieldsAndOmittedValues(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		enableTypeCatalogue(t, f, "workspace-one")
		created := milestoneObject(t, call("workitem_type", map[string]any{"action": "create", "name": "  Type " + uuid.NewString() + "  ", "description": "Retained description", "external_source": "fixture", "external_id": "external"}))
		require.ElementsMatch(t, strings.Fields("id deleted_at project_ids created_at updated_at name description logo_props is_epic is_default is_active level external_source external_id created_by updated_by workspace"), slices.Collect(maps.Keys(created)))
		require.Equal(t, strings.TrimSpace(created["name"].(string)), created["name"])
		require.Equal(t, false, created["is_active"])
		require.Equal(t, false, created["is_default"])
		require.Equal(t, false, created["is_epic"])
		require.Equal(t, float64(0), created["level"])
		require.Equal(t, testsupport.LabelWorkspace, created["workspace"])
		require.Equal(t, testsupport.LabelActor, created["created_by"])
		require.Nil(t, created["deleted_at"])
		require.Empty(t, created["project_ids"])
		id := created["id"]
		require.Equal(t, created, milestoneObject(t, call("workitem_type", map[string]any{"action": "retrieve", "workitem_type_id": id})))
		updated := milestoneObject(t, call("workitem_type", map[string]any{"action": "update", "workitem_type_id": id, "name": "", "description": "", "external_source": "", "external_id": "", "is_active": false}))
		for _, key := range []string{"name", "description", "external_source", "external_id", "logo_props", "is_active", "level"} {
			require.Equal(t, created[key], updated[key], key)
		}
		catalogueNull(t, call("workitem_type", map[string]any{"action": "delete", "workitem_type_id": id}))
		milestoneFailure(t, call("workitem_type", map[string]any{"action": "retrieve", "workitem_type_id": id}), "not_found")
	})
}

func TestMCPWorkitemTypeProjectListsAndResolve(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		repository := enableTypeCatalogue(t, f, "workspace-one")
		name := "Resolved " + uuid.NewString()
		created := milestoneObject(t, call("workitem_type", map[string]any{"action": "create", "name": name}))
		project := catalogueScoped(call, map[string]any{"project_id": testsupport.LabelProject})
		milestoneFailure(t, project("workitem_type", map[string]any{"action": "retrieve", "workitem_type_id": created["id"]}), "not_found")
		resolved := milestoneObject(t, project("workitem_type", map[string]any{"action": "resolve", "name": "  " + name + "  "}))
		require.Equal(t, created["id"], resolved["id"])
		require.Equal(t, true, resolved["is_active"])
		require.Equal(t, []any{testsupport.LabelProject}, resolved["project_ids"])
		before := typeStorageSnapshot(t, f)
		require.Equal(t, resolved, milestoneObject(t, project("workitem_type", map[string]any{"action": "resolve", "name": name})))
		catalogueNull(t, project("workitem_type", map[string]any{"action": "import_to_project", "workitem_type_ids": created["id"]}))
		require.Equal(t, before, typeStorageSnapshot(t, f), "already available types need no catalogue writes")
		all := catalogueArray(t, project("workitem_type", map[string]any{"action": "list"}))
		require.GreaterOrEqual(t, len(all), 2)
		require.Equal(t, all[:1], catalogueArray(t, project("workitem_type", map[string]any{"action": "list", "per_page": 1})))
		require.Equal(t, all[1:2], catalogueArray(t, project("workitem_type", map[string]any{"action": "list", "per_page": 1, "cursor": "1:1:0"})))
		workspace := catalogueArray(t, call("workitem_type", map[string]any{"action": "list"}))
		require.Equal(t, workspace, catalogueArray(t, call("workitem_type", map[string]any{"action": "list", "per_page": 1, "cursor": "ignored"})))
		catalogue, err := repository.ProjectCatalog(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject)
		require.NoError(t, err)
		_, err = repository.MutateProject(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject, workitemtypes.ProjectInput{Revision: catalogue.Revision, Action: "pause", TypeIDs: []string{created["id"].(string)}})
		require.NoError(t, err)
		catalogueNull(t, project("workitem_type", map[string]any{"action": "import_to_project", "workitem_type_ids": created["id"]}))
		var available bool
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT is_active FROM project_issue_types WHERE issue_type_id=$1 AND project_id=$2 AND deleted_at IS NULL`, created["id"], testsupport.LabelProject).Scan(&available))
		require.True(t, available, "import resumes a paused native link")
	})
}

func TestMCPWorkitemTypeImportAndProjectReplacementAreAtomic(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		enableTypeCatalogue(t, f, "workspace-one")
		created := milestoneObject(t, call("workitem_type", map[string]any{"action": "create", "name": uuid.NewString(), "is_active": true}))
		id := created["id"].(string)
		before := typeStorageSnapshot(t, f)
		for _, input := range []map[string]any{
			{"action": "import_to_project", "project_id": testsupport.LabelProject, "workitem_type_ids": id + "," + uuid.NewString()},
			{"action": "update", "workitem_type_id": id, "description": "Rejected", "project_ids": testsupport.LabelProject + "," + uuid.NewString()},
			{"action": "create", "name": uuid.NewString(), "is_active": true, "project_ids": testsupport.LabelProject + "," + uuid.NewString()},
		} {
			input["request_id"] = uuid.NewString()
			require.True(t, call("workitem_type", input).IsError)
			require.Equal(t, before, typeStorageSnapshot(t, f))
		}
		linked := milestoneObject(t, call("workitem_type", map[string]any{"action": "update", "workitem_type_id": id, "project_ids": testsupport.LabelProject + "," + testsupport.LabelProject}))
		require.Equal(t, []any{testsupport.LabelProject}, linked["project_ids"])
		unlinked := milestoneObject(t, call("workitem_type", map[string]any{"action": "update", "workitem_type_id": id, "project_ids": "[]"}))
		require.Empty(t, unlinked["project_ids"])
	})
}
