//go:build integration

package bootstrap_test

import (
	_ "embed"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

//go:embed schema_migrations/000020_state_mutations.up.sql
var stateMutationMigrationUp string

//go:embed schema_migrations/000020_state_mutations.down.sql
var stateMutationMigrationDown string

func TestStateRevisionMigrationBackfillsAndTracksEveryRowWrite(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	// Recreate the preceding schema using the actual down migration, then place
	// a preexisting state before applying the actual upgrade SQL.
	f.exec(t, stateMutationMigrationDown)
	state := uuid.New().String()
	f.exec(t, `INSERT INTO states(id,workspace_id,project_id,name,color,slug,description,sequence,"group","default",is_triage,created_at,updated_at)
 VALUES($1,$2,$3,'Existing','#abc','existing','',1,'unstarted',false,false,now(),now())`, state, f.workspace, f.project)
	f.exec(t, stateMutationMigrationUp)
	var revision int64
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT revision FROM state_edit_revisions WHERE state_id=$1`, state).Scan(&revision))
	require.EqualValues(t, 1, revision)
	f.exec(t, `UPDATE states SET name=name WHERE id=$1`, state)
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT revision FROM state_edit_revisions WHERE state_id=$1`, state).Scan(&revision))
	require.EqualValues(t, 2, revision)
	tx, err := f.db.Begin(f.ctx)
	require.NoError(t, err)
	defer func() { testsupport.Rollback(t, tx, f.ctx) }()
	_, err = tx.Exec(f.ctx, `UPDATE states SET "default"=true WHERE id=$1`, state)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback(f.ctx))
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT revision FROM state_edit_revisions WHERE state_id=$1`, state).Scan(&revision))
	require.EqualValues(t, 2, revision)
	f.exec(t, `UPDATE states SET deleted_at=now() WHERE id=$1`, state)
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT revision FROM state_edit_revisions WHERE state_id=$1`, state).Scan(&revision))
	require.EqualValues(t, 3, revision)
	f.exec(t, `DELETE FROM states WHERE id=$1`, state)
	var count int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM state_edit_revisions WHERE state_id=$1`, state).Scan(&count))
	require.Zero(t, count)
}
