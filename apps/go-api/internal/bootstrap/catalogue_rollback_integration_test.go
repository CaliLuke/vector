//go:build integration

package bootstrap_test

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestEmptyCatalogueStorageCanRollBackAndUpgrade(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	for _, tc := range []struct{ name, down, up, first, second, revisions string }{
		{"initiative", initiativeStorageDown, initiativeStorageUp, "initiatives", "initiative_events", "initiative_revisions"},
		{"release", releaseStorageDown, releaseStorageUp, "release_tags", "release_events", "release_revisions"},
		{"page", pageStorageDown, pageStorageUp, "page_collections", "page_api_events", "page_api_revisions"},
		{"relation", relationStorageDown, relationStorageUp, "work_item_relation_definitions", "work_item_custom_relations", "work_item_relation_revisions"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f.exec(t, tc.down)
			var gone bool
			require.NoError(t, f.db.QueryRow(f.ctx, `SELECT to_regclass($1) IS NULL AND to_regclass($2) IS NULL`, "public."+tc.first, "public."+tc.second).Scan(&gone))
			require.True(t, gone)
			f.exec(t, tc.up)
			var seeded int
			require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM `+pgx.Identifier{tc.revisions}.Sanitize()+` WHERE workspace_id=$1`, f.workspace).Scan(&seeded))
			require.Equal(t, 1, seeded)
		})
	}
}
