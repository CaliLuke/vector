//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPWorkspaceReleasePickerValidationAndPreservation(t *testing.T) {
	runCatalogue(t, templateFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		enableTypeCatalogue(t, f, "workspace-one")
		kind := milestoneObject(t, call("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": uuid.NewString()}))
		property := milestoneObject(t, call("workitem_property", map[string]any{"action": "create", "workitem_type_id": kind["id"], "display_name": "Delivery", "property_type": "RELATION", "relation_type": "RELEASE"}))
		release := milestoneObject(t, call("release", map[string]any{"action": "create", "name": "Workspace delivery"}))
		item := milestoneObject(t, call("workitem", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": "Release selection", "type_id": kind["id"]}))
		repo := workitemtypes.NewRepository(f.pool)
		schemaInput := workitemtypes.ItemSchemaInput{Mode: "edit", ProjectID: testsupport.LabelProject, IssueID: item["id"].(string)}
		schema, err := repo.ItemSchema(t.Context(), testsupport.LabelActor, "workspace-one", schemaInput)
		require.NoError(t, err)
		found := false
		for _, choice := range schema.Releases {
			if choice.ID == release["id"] {
				found = true
				require.True(t, choice.Eligible)
			}
		}
		require.True(t, found, "native picker must include the workspace catalogue")
		values := catalogueScoped(call, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item["id"], "property_id": property["id"]})
		accepted := milestoneObject(t, values("workitem_property", map[string]any{"action": "set_value", "value": release["id"]}))
		require.Equal(t, release["id"], accepted["value"])
		deletion := map[string]any{"action": "delete", "release_id": release["id"], "request_id": uuid.NewString()}
		before := releaseSnapshot(t, f)
		milestoneFailure(t, call("release", deletion), "invalid_params")
		require.Equal(t, before, releaseSnapshot(t, f))
		catalogueNull(t, values("workitem_property", map[string]any{"action": "delete_value"}))
		require.Equal(t, false, milestoneObject(t, call("release", deletion))["replayed"])
		schema, err = repo.ItemSchema(t.Context(), testsupport.LabelActor, "workspace-one", schemaInput)
		require.NoError(t, err)
		for _, choice := range schema.Releases {
			require.NotEqual(t, release["id"], choice.ID)
		}
		milestoneFailure(t, values("workitem_property", map[string]any{"action": "set_value", "value": release["id"]}), "invalid_params")
	})
}
