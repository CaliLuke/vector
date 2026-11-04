//go:build integration

package bootstrap_test

import (
	_ "embed"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

//go:embed schema_migrations/000031_work_item_relations.down.sql
var relationStorageDown string

//go:embed schema_migrations/000031_work_item_relations.up.sql
var relationStorageUp string

func TestRelationRollbackPreservesRetainedCustomDefinitions(t *testing.T) {
	for _, retired := range []bool{false, true} {
		t.Run(map[bool]string{false: "live", true: "retained"}[retired], func(t *testing.T) {
			f := newMilestoneSchemaFixture(t)
			id := uuid.NewString()
			f.exec(t, `INSERT INTO work_item_relation_definitions(id,workspace_id,name,outward,inward,deleted_at) VALUES($1,$2,'Custom','supports','supported by',CASE WHEN $3 THEN now() ELSE NULL END)`, id, f.workspace, retired)
			_, err := f.db.Exec(f.ctx, relationStorageDown)
			require.ErrorContains(t, err, "custom relation data must be migrated before rollback")
			f.exec(t, `ROLLBACK`)
			var name string
			require.NoError(t, f.db.QueryRow(f.ctx, `SELECT name FROM work_item_relation_definitions WHERE id=$1`, id).Scan(&name))
			require.Equal(t, "Custom", name)
		})
	}
}
