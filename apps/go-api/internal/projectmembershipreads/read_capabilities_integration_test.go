//go:build integration

package projectmembershipreads

import (
	"context"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestReadCapabilitiesUseCurrentScopeAndMembershipPolicies(t *testing.T) {
	reader, db := directoryFixture(t)
	_, err := db.Exec(t.Context(), `CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,deleted_at timestamptz);
 CREATE TABLE project_members(project_id uuid,workspace_id uuid,member_id uuid,is_active boolean,deleted_at timestamptz);
 INSERT INTO projects VALUES('40000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001',NULL);
 INSERT INTO project_members SELECT '40000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001',id,true,NULL FROM users WHERE id IN('10000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000003');`)
	require.NoError(t, err)
	actor := uuid.MustParse(directoryActor)
	workspace := uuid.MustParse("20000000-0000-4000-8000-000000000001")
	capabilities, err := reader.ReadCapabilities(t.Context(), uuid.MustParse(directoryGuest), workspace, "current")
	require.NoError(t, err)
	require.Equal(t, ReadCapabilities{Workspace: true, ProjectMembers: true}, capabilities)
	global, err := reader.ReadCapabilities(t.Context(), actor, uuid.UUID{}, "")
	require.NoError(t, err)
	require.Equal(t, ReadCapabilities{true, true, true}, global)
	mismatch, err := reader.ReadCapabilities(t.Context(), actor, workspace, "other")
	require.NoError(t, err)
	require.Empty(t, mismatch)
	for _, tc := range []struct {
		name, change string
		want         ReadCapabilities
	}{
		{"account deactivated", `UPDATE users SET is_active=false`, ReadCapabilities{}},
		{"workspace deleted", `UPDATE workspaces SET deleted_at=now()`, ReadCapabilities{}},
		{"membership deleted", `UPDATE workspace_members SET deleted_at=now()`, ReadCapabilities{}},
		{"membership inactive", `UPDATE workspace_members SET is_active=false`, ReadCapabilities{}},
		{"invalid workspace role", `UPDATE workspace_members SET role=0`, ReadCapabilities{}},
		{"guest downgrade", `UPDATE workspace_members SET role=5`, ReadCapabilities{Workspace: true, ProjectMembers: true}},
		{"project deleted", `UPDATE projects SET deleted_at=now()`, ReadCapabilities{Workspace: true, WorkspaceMembers: true}},
		{"project membership inactive", `UPDATE project_members SET is_active=false`, ReadCapabilities{Workspace: true, WorkspaceMembers: true}},
		{"project membership deleted", `UPDATE project_members SET deleted_at=now()`, ReadCapabilities{Workspace: true, WorkspaceMembers: true}},
		{"foreign project membership", `UPDATE project_members SET workspace_id='20000000-0000-4000-8000-000000000002'`, ReadCapabilities{Workspace: true, WorkspaceMembers: true}},
		{"slug reassigned", `UPDATE workspaces SET slug='renamed' WHERE slug='current'; UPDATE workspaces SET slug='current' WHERE slug='other'`, ReadCapabilities{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := db.Begin(t.Context())
			require.NoError(t, err)
			defer func() { require.NoError(t, tx.Rollback(context.Background())) }()
			current, err := NewDirectoryReader(tx)
			require.NoError(t, err)
			before, err := current.ReadCapabilities(t.Context(), actor, workspace, "current")
			require.NoError(t, err)
			require.Equal(t, ReadCapabilities{true, true, true}, before)
			_, err = tx.Exec(t.Context(), tc.change)
			require.NoError(t, err)
			after, err := current.ReadCapabilities(t.Context(), actor, workspace, "current")
			require.NoError(t, err)
			require.Equal(t, tc.want, after)
		})
	}
}
