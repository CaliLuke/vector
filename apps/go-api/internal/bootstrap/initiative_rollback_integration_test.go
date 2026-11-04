//go:build integration

package bootstrap_test

import (
	_ "embed"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

//go:embed schema_migrations/000039_initiatives.down.sql
var initiativeStorageDown string

//go:embed schema_migrations/000039_initiatives.up.sql
var initiativeStorageUp string

func TestInitiativeRollbackPreservesRecordsAndRetainedAudit(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	id := uuid.NewString()
	f.exec(t, `INSERT INTO initiatives(id,workspace_id,name) VALUES($1,$2,'Retained initiative')`, id, f.workspace)
	f.exec(t, `INSERT INTO initiative_events(workspace_id,initiative_id,action,input,after_value) SELECT workspace_id,id,'create','{}',to_jsonb(i) FROM initiatives i WHERE id=$1`, id)
	snapshot := func() string {
		var raw string
		require.NoError(t, f.db.QueryRow(f.ctx, `SELECT jsonb_build_object('records',(SELECT jsonb_agg(to_jsonb(i) ORDER BY id) FROM initiatives i),'events',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM initiative_events e))::text`).Scan(&raw))
		return raw
	}
	for _, state := range []string{"live", "tombstone", "audit after physical deletion"} {
		t.Run(state, func(t *testing.T) {
			switch state {
			case "tombstone":
				f.exec(t, `UPDATE initiatives SET deleted_at=now() WHERE id=$1`, id)
			case "audit after physical deletion":
				f.exec(t, `DELETE FROM initiatives WHERE id=$1`, id)
			}
			before := snapshot()
			_, err := f.db.Exec(f.ctx, initiativeStorageDown)
			require.ErrorContains(t, err, "cannot discard initiative data")
			f.exec(t, `ROLLBACK`)
			require.Equal(t, before, snapshot())
		})
	}
}
