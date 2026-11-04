//go:build integration

package bootstrap_test

import (
	_ "embed"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

//go:embed schema_migrations/000035_workspace_releases.down.sql
var releaseStorageDown string

//go:embed schema_migrations/000035_workspace_releases.up.sql
var releaseStorageUp string

func TestReleaseRollbackRefusesIndependentCataloguesAndRetainedAudit(t *testing.T) {
	for _, state := range []struct{ name, seed, table string }{
		{"tag", `INSERT INTO release_tags(id,workspace_id,version) VALUES($1,$2,'v1')`, "release_tags"},
		{"label", `INSERT INTO release_labels(id,workspace_id,name) VALUES($1,$2,'Label')`, "release_labels"},
		{"audit after physical deletion", `INSERT INTO release_events(id,workspace_id,resource_id,resource_kind,action,after_value) VALUES($1,$2,$1,'release','delete','{"name":"Deleted release"}')`, "release_events"},
	} {
		t.Run(state.name, func(t *testing.T) {
			f := newMilestoneSchemaFixture(t)
			id := uuid.NewString()
			f.exec(t, state.seed, id, f.workspace)
			_, err := f.db.Exec(f.ctx, releaseStorageDown)
			require.ErrorContains(t, err, "cannot discard workspace release data")
			f.exec(t, `ROLLBACK`)
			var count int
			require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM `+state.table+` WHERE id=$1`, id).Scan(&count))
			require.Equal(t, 1, count)
		})
	}
}
