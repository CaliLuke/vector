//go:build integration

package projectintakewrites_test

import (
	"testing"

	intakes "github.com/CaliLuke/go-plane/internal/projectintakewrites"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/stretchr/testify/require"
)

func TestIntakeWritesRejectStaleWorkspaceBridges(t *testing.T) {
	ctx, db := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed+`
ALTER TABLE projects ADD COLUMN cover_image text, ADD COLUMN cover_image_asset_id uuid, ADD COLUMN logo_props jsonb;
ALTER TABLE file_assets ADD COLUMN workspace_id uuid, ADD COLUMN project_id uuid;
UPDATE intakes SET name='Original',is_default=false;`)
	repo, err := intakes.NewRepository(db)
	require.NoError(t, err)
	require.NoError(t, repo.Authorize(ctx, testsupport.LabelActor, "demo", testsupport.LabelProject))
	_, err = repo.Patch(ctx, testsupport.LabelActor, "demo", testsupport.LabelProject, "00000000-0000-4000-8000-000000000088", map[string]any{"name": "Original"})
	require.NoError(t, err)
	_, err = db.Exec(ctx, `UPDATE projects SET workspace_id='00000000-0000-4000-8000-000000000099' WHERE id=$1`, testsupport.LabelProject)
	require.NoError(t, err)
	require.Error(t, repo.Authorize(ctx, testsupport.LabelActor, "demo", testsupport.LabelProject))
	_, err = repo.Patch(ctx, testsupport.LabelActor, "demo", testsupport.LabelProject, "00000000-0000-4000-8000-000000000088", map[string]any{"name": "Wrong tenant"})
	require.Error(t, err)
	require.Error(t, repo.Delete(ctx, testsupport.LabelActor, "demo", testsupport.LabelProject, "00000000-0000-4000-8000-000000000088"))
	var retained bool
	require.NoError(t, db.QueryRow(ctx, `SELECT name='Original' AND deleted_at IS NULL FROM intakes`).Scan(&retained))
	require.True(t, retained)
	_, err = db.Exec(ctx, `UPDATE projects SET workspace_id=$1 WHERE id=$2`, testsupport.LabelWorkspace, testsupport.LabelProject)
	require.NoError(t, err)
	require.NoError(t, repo.Delete(ctx, testsupport.LabelActor, "demo", testsupport.LabelProject, testsupport.LabelIntake))
	require.NoError(t, db.QueryRow(ctx, `SELECT deleted_at IS NULL FROM intakes`).Scan(&retained))
	require.False(t, retained)
}
