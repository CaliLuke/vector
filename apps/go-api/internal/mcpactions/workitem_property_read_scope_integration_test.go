//go:build integration

package mcpactions_test

import (
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"

	"github.com/CaliLuke/go-plane/internal/mcpactions"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/stretchr/testify/require"
)

func TestMCPPropertyReadsRecheckScopeAfterAdmission(t *testing.T) {
	f := newMilestoneFixture(t)
	otherWorkspace, _ := addPrivateProjectInOtherWorkspace(t, f)
	apps := databaseApplications(t, f.pool)
	var pending func()
	apps.Properties = changingPropertyReader{Repository: workitemtypes.NewRepository(f.pool), change: func() {
		if pending != nil {
			pending()
		}
	}}
	service, err := mcpactions.New(apps)
	require.NoError(t, err)
	f.service = service
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		fixture := newPropertyReadFixture(t, f, call, "workspace-one", testsupport.LabelProject)
		for _, input := range fixture.requests() {
			require.False(t, call("workitem_property", input).IsError, "the initial scope must be authorized")
			for _, change := range propertyReadScopeChanges(otherWorkspace) {
				t.Run(input["action"].(string)+"/"+change.name, func(t *testing.T) {
					assertPropertyReadScopeChange(t, f, call, input, change, &pending, "forbidden")
				})
			}
		}
		for _, change := range propertyValueReadScopeChanges(fixture) {
			t.Run("get_value/"+change.name, func(t *testing.T) {
				assertPropertyReadScopeChange(t, f, call, fixture.valueRead(), change, &pending, "not_found")
			})
		}
	})
}

func TestMCPPropertyActionsRejectCrossWorkspaceIdentifiers(t *testing.T) {
	f := newMilestoneFixture(t)
	_, foreignProject := addPrivateProjectInOtherWorkspace(t, f)
	runSelectedWorkspaceMatrix(t, f, func(t *testing.T, root catalogueCall) {
		own := catalogueScoped(root, map[string]any{"workspace_slug": "workspace-one"})
		for _, input := range newPropertyReadFixture(t, f, own, "workspace-one", testsupport.LabelProject).requests() {
			require.False(t, own("workitem_property", input).IsError, "the original scope must be usable")
		}
		other := catalogueScoped(root, map[string]any{"workspace_slug": "other"})
		foreign := newPropertyReadFixture(t, f, other, "other", foreignProject)
		for _, input := range foreign.requests() {
			require.False(t, other("workitem_property", input).IsError, "the same actor can read the resource through its actual workspace")
			crossed := maps.Clone(input)
			crossed["project_id"] = testsupport.LabelProject
			before := propertyStorageSnapshot(t, f)
			milestoneFailure(t, own("workitem_property", crossed), "not_found")
			require.Equal(t, before, propertyStorageSnapshot(t, f))
		}
		for _, input := range foreignPropertyMutationInputs(foreign) {
			before := propertyStorageSnapshot(t, f)
			milestoneFailure(t, own("workitem_property", input), "not_found")
			require.Equal(t, before, propertyStorageSnapshot(t, f), "cross-workspace mutations cannot change either workspace")
		}
	})
}
