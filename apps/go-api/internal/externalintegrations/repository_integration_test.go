//go:build integration

package externalintegrations_test

import (
	"errors"
	"testing"

	integrations "github.com/CaliLuke/go-plane/internal/externalintegrations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type configurationCipher struct{}

func (configurationCipher) Decrypt(value string) (string, error) {
	if value == "sealed" {
		return "decrypted", nil
	}
	return "", errors.New("invalid ciphertext")
}

func TestConfigurationPrecedenceAndProjectOwnership(t *testing.T) {
	ctx, db := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed+`
CREATE TABLE instance_configurations(key text, value text, is_encrypted boolean, deleted_at timestamptz);
INSERT INTO instance_configurations VALUES('secret','sealed',true,NULL),('empty',NULL,false,NULL),('retired','old',false,now());
ALTER TABLE workspaces ADD COLUMN logo text, ADD COLUMN logo_asset_id uuid;
ALTER TABLE projects ADD COLUMN cover_image text, ADD COLUMN cover_image_asset_id uuid, ADD COLUMN logo_props jsonb;`)
	pool, err := pgxpool.New(ctx, db.Config().ConnString())
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	repo, err := integrations.NewRepository(pool, configurationCipher{})
	require.NoError(t, err)
	environment := map[string]string{"secret": "environment", "empty": "fallback", "retired": "current", "missing": "default"}
	keys := []string{"secret", "empty", "retired", "missing"}
	values, err := repo.Configuration(ctx, keys, environment, false)
	require.NoError(t, err)
	require.Equal(t, environment, values)
	values, err = repo.Configuration(ctx, keys, environment, true)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"secret": "decrypted", "empty": "", "retired": "current", "missing": "default"}, values)
	role, err := repo.WorkspaceRole(ctx, testsupport.LabelActor, "demo")
	require.NoError(t, err)
	require.Equal(t, 15, role)
	allowed, err := repo.ProjectAllowed(ctx, testsupport.LabelActor, "demo", testsupport.LabelProject)
	require.NoError(t, err)
	require.True(t, allowed)
	workspace, err := repo.WorkspaceLite(ctx, "demo")
	require.NoError(t, err)
	require.Contains(t, string(workspace), "Demo")
	project, err := repo.ProjectLite(ctx, testsupport.LabelProject)
	require.NoError(t, err)
	require.Contains(t, string(project), "Current")
	_, err = db.Exec(ctx, `UPDATE instance_configurations SET value='corrupted' WHERE key='secret'`)
	require.NoError(t, err)
	_, err = repo.Configuration(ctx, keys, environment, true)
	require.ErrorContains(t, err, "decrypt")
	_, err = db.Exec(ctx, `UPDATE projects SET workspace_id='00000000-0000-4000-8000-000000000099' WHERE id=$1`, testsupport.LabelProject)
	require.NoError(t, err)
	allowed, err = repo.ProjectAllowed(ctx, testsupport.LabelActor, "demo", testsupport.LabelProject)
	require.NoError(t, err)
	require.False(t, allowed, "membership and project must belong to the selected workspace")
}
