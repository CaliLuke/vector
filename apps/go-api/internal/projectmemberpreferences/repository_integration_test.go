//go:build integration

package projectmemberpreferences_test

import (
	"testing"

	preferences "github.com/CaliLuke/go-plane/internal/projectmemberpreferences"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/stretchr/testify/require"
)

func TestPreferencesMergeAndKeepProjectWorkspaceOwnership(t *testing.T) {
	ctx, db := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed+`
ALTER TABLE project_members ADD COLUMN preferences jsonb DEFAULT '{"existing":true}', ADD COLUMN updated_at timestamptz, ADD COLUMN updated_by_id uuid;`)
	repo, err := preferences.NewRepository(db)
	require.NoError(t, err)
	require.NoError(t, repo.Authorize(ctx, testsupport.LabelActor, "demo", testsupport.LabelProject))
	require.NoError(t, repo.Lookup(ctx, "demo", testsupport.LabelProject, testsupport.LabelActor))
	result, err := repo.Patch(ctx, testsupport.LabelActor, "demo", testsupport.LabelProject, testsupport.LabelActor, map[string]any{"selected": "list"})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"existing": true, "selected": "list"}, result["preferences"])
	var stored string
	require.NoError(t, db.QueryRow(ctx, `SELECT preferences::text FROM project_members WHERE project_id=$1`, testsupport.LabelProject).Scan(&stored))
	require.JSONEq(t, `{"existing":true,"selected":"list"}`, stored)
	_, err = db.Exec(ctx, `UPDATE projects SET workspace_id='00000000-0000-4000-8000-000000000099' WHERE id=$1`, testsupport.LabelProject)
	require.NoError(t, err)
	require.Error(t, repo.Authorize(ctx, testsupport.LabelActor, "demo", testsupport.LabelProject), "a stale membership bridge must not authorize another workspace's project")
	require.Error(t, repo.Lookup(ctx, "demo", testsupport.LabelProject, testsupport.LabelActor))
	_, err = repo.Patch(ctx, testsupport.LabelActor, "demo", testsupport.LabelProject, testsupport.LabelActor, map[string]any{"selected": "board"})
	require.Error(t, err)
	var retained string
	require.NoError(t, db.QueryRow(ctx, `SELECT preferences::text FROM project_members WHERE project_id=$1`, testsupport.LabelProject).Scan(&retained))
	require.Equal(t, stored, retained)
}
