//go:build integration

package milestones_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMilestoneCapabilitiesRequireImmutableLiveScope(t *testing.T) {
	f := newReadFixture(t)
	for _, tc := range []struct {
		actor, workspace uuid.UUID
		slug             string
	}{
		{uuid.Nil, f.scope.WorkspaceID, f.scope.WorkspaceSlug},
		{f.scope.ActorID, uuid.Nil, f.scope.WorkspaceSlug},
		{f.scope.ActorID, f.scope.WorkspaceID, ""},
		{f.scope.ActorID, f.scope.WorkspaceID, "wrong"},
		{f.scope.ActorID, uuid.New(), f.scope.WorkspaceSlug},
	} {
		value, err := f.repo.VisibleActions(f.ctx, tc.actor, tc.workspace, tc.slug)
		require.NoError(t, err)
		require.Equal(t, milestones.Capabilities{}, value)
	}
	value, err := f.repo.VisibleActions(f.ctx, f.scope.ActorID, f.scope.WorkspaceID, f.scope.WorkspaceSlug)
	require.NoError(t, err)
	require.True(t, value.Read)
	require.True(t, value.Write)
	_, err = f.db.Exec(f.ctx, `UPDATE users SET is_active=false`)
	require.NoError(t, err)
	value, err = f.repo.VisibleActions(f.ctx, f.scope.ActorID, f.scope.WorkspaceID, f.scope.WorkspaceSlug)
	require.NoError(t, err)
	require.Equal(t, milestones.Capabilities{}, value)
	_, err = f.db.Exec(f.ctx, `ALTER TABLE projects RENAME TO unavailable_projects`)
	require.NoError(t, err)
	_, err = f.repo.VisibleActions(f.ctx, f.scope.ActorID, f.scope.WorkspaceID, f.scope.WorkspaceSlug)
	require.Error(t, err)
}
