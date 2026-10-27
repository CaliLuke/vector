//go:build integration

package issuequery_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/stretchr/testify/require"
)

func TestMilestoneReadScopeRequiresCurrentOwnedMembership(t *testing.T) {
	ctx, db := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed)
	for _, tc := range []struct {
		name, change string
		want         bool
	}{
		{"member", "SELECT 1", true},
		{"project administrator", "UPDATE project_members SET role=20", true},
		{"restricted guest", "UPDATE project_members SET role=5", false},
		{"full guest", "UPDATE project_members SET role=5; UPDATE workspace_members SET role=5; UPDATE projects SET guest_view_all_features=true", true},
		{"unknown project role", "UPDATE project_members SET role=99; UPDATE projects SET guest_view_all_features=true", false},
		{"unknown workspace role", "UPDATE workspace_members SET role=99", false},
		{"inactive actor", "UPDATE users SET is_active=false", false},
		{"deleted workspace", "UPDATE workspaces SET deleted_at=now()", false},
		{"deleted project", "UPDATE projects SET deleted_at=now()", false},
		{"archived project", "UPDATE projects SET archived_at=now()", false},
		{"inactive workspace membership", "UPDATE workspace_members SET is_active=false", false},
		{"deleted workspace membership", "UPDATE workspace_members SET deleted_at=now()", false},
		{"inactive project membership", "UPDATE project_members SET is_active=false", false},
		{"deleted project membership", "UPDATE project_members SET deleted_at=now()", false},
		{"foreign project membership", "UPDATE project_members SET workspace_id='00000000-0000-4000-8000-000000000099'", false},
		{"foreign workspace membership", "UPDATE workspace_members SET workspace_id='00000000-0000-4000-8000-000000000099'", false},
		{"workspace admin without project membership", "UPDATE workspace_members SET role=20; DELETE FROM project_members", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := db.Begin(ctx)
			require.NoError(t, err)
			t.Cleanup(func() { _ = tx.Rollback(context.WithoutCancel(ctx)) })
			_, err = tx.Exec(ctx, tc.change)
			require.NoError(t, err)
			var admitted bool
			require.NoError(t, tx.QueryRow(ctx, "SELECT "+issuequery.MilestoneReadScopeSQL("$1::uuid", "$2::uuid", "$3::uuid"), testsupport.LabelProject, testsupport.LabelWorkspace, testsupport.LabelActor).Scan(&admitted))
			require.Equal(t, tc.want, admitted)
		})
	}
	var admitted bool
	require.NoError(t, db.QueryRow(ctx, "SELECT "+issuequery.MilestoneReadScopeSQL("$1::uuid", "$2::uuid", "$3::uuid"), testsupport.LabelProject, "00000000-0000-4000-8000-000000000099", testsupport.LabelActor).Scan(&admitted))
	require.False(t, admitted, "workspace UUID must match the project even when the actor is otherwise admitted")
}
