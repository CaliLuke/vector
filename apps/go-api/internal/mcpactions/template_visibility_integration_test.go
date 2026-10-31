//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/templates"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func templateIDs(rows []map[string]any) []string {
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row["id"].(string)
	}
	return ids
}

func TestMCPTemplateListsBindKindWorkspaceAndProjectVisibility(t *testing.T) {
	for _, kind := range []string{"workitem", "page"} {
		t.Run(kind, func(t *testing.T) {
			f := newMilestoneFixture(t)
			otherWorkspace, otherProject := addPrivateProjectInOtherWorkspace(t, f)
			foreignName := "Other workspace template"
			foreign, err := templates.NewRepository(f.pool).Mutate(t.Context(), templates.Command{
				Query:  templates.Query{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(otherWorkspace), WorkspaceSlug: "other", ProjectID: uuid.MustParse(otherProject), Kind: kind},
				Action: "create", Name: &foreignName, Data: map[string]any{},
			})
			require.NoError(t, err)
			foreignID := foreign.ID
			runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, call catalogueCall) {
				create := func(project string) string {
					input := templateScopedInput(templateScope{kind, project}, "create")
					input["name"], input["template_data"] = uuid.NewString(), `{}`
					return milestoneObject(t, call("template", input))["id"].(string)
				}
				workspaceID := create("")
				projectID := create(testsupport.LabelProject)
				otherKind := milestoneObject(t, call("template", map[string]any{"action": "create", "kind": "project", "name": uuid.NewString(), "template_data": `{}`}))["id"].(string)
				list := templateScopedInput(templateScope{kind, ""}, "list")
				ids := templateIDs(catalogueArray(t, call("template", list)))
				require.Contains(t, ids, workspaceID)
				require.Contains(t, ids, projectID)
				require.NotContains(t, ids, foreignID)
				require.NotContains(t, ids, otherKind)
				projectList := templateScopedInput(templateScope{kind, testsupport.LabelProject}, "list")
				ids = templateIDs(catalogueArray(t, call("template", projectList)))
				require.Contains(t, ids, projectID)
				require.NotContains(t, ids, workspaceID)
				require.NotContains(t, ids, foreignID)
				require.NotContains(t, ids, otherKind)
				_, err := f.pool.Exec(t.Context(), `UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, testsupport.LabelProject, testsupport.LabelActor)
				require.NoError(t, err)
				ids = templateIDs(catalogueArray(t, call("template", list)))
				require.Contains(t, ids, workspaceID)
				require.NotContains(t, ids, projectID)
				milestoneFailure(t, call("template", projectList), "forbidden")
				_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET is_active=true WHERE project_id=$1 AND member_id=$2`, testsupport.LabelProject, testsupport.LabelActor)
				require.NoError(t, err)
				_, err = f.pool.Exec(t.Context(), `UPDATE projects SET archived_at=now() WHERE id=$1`, testsupport.LabelProject)
				require.NoError(t, err)
				ids = templateIDs(catalogueArray(t, call("template", list)))
				require.NotContains(t, ids, projectID)
				milestoneFailure(t, call("template", projectList), "forbidden")
				_, err = f.pool.Exec(t.Context(), `UPDATE projects SET archived_at=NULL WHERE id=$1`, testsupport.LabelProject)
				require.NoError(t, err)
				require.Contains(t, templateIDs(catalogueArray(t, call("template", projectList))), projectID)
				// Even an actor with access to both workspaces cannot address a foreign
				// template through the selected workspace's catalogue.
				wrongScope := templateScopedInput(templateScope{kind, ""}, "update")
				wrongScope["template_id"], wrongScope["name"] = foreignID, "Forbidden change"
				before := templateStorageSnapshot(t, f)
				milestoneFailure(t, call("template", wrongScope), "not_found")
				require.Equal(t, before, templateStorageSnapshot(t, f))
			})
		})
	}
}
