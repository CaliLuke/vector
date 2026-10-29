//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDiscoveryReadersRejectIncompleteWorkspaceScope(t *testing.T) {
	f := newMilestoneFixture(t)
	apps := databaseApplications(t, f.pool)
	actor, workspace := uuid.MustParse(testsupport.LabelActor), uuid.MustParse(testsupport.LabelWorkspace)
	states, err := apps.States.VisibleActions(t.Context(), actor, workspace, "workspace-one")
	require.NoError(t, err)
	require.True(t, states.Read)
	types, err := apps.Types.APIVisibility(t.Context(), workitemtypes.APIScope{ActorID: actor, WorkspaceID: workspace, WorkspaceSlug: "workspace-one"})
	require.NoError(t, err)
	require.True(t, types.Read)

	for _, scope := range []struct {
		name             string
		actor, workspace uuid.UUID
		slug             string
	}{
		{"missing identity", uuid.Nil, workspace, "workspace-one"},
		{"missing workspace ID", actor, uuid.Nil, "workspace-one"},
		{"missing workspace slug", actor, workspace, ""},
	} {
		t.Run(scope.name, func(t *testing.T) {
			stateAccess, stateErr := apps.States.VisibleActions(t.Context(), scope.actor, scope.workspace, scope.slug)
			require.NoError(t, stateErr)
			require.Empty(t, stateAccess)
			typeAccess, typeErr := apps.Types.APIVisibility(t.Context(), workitemtypes.APIScope{ActorID: scope.actor, WorkspaceID: scope.workspace, WorkspaceSlug: scope.slug})
			require.NoError(t, typeErr)
			require.Empty(t, typeAccess)
		})
	}
}
