//go:build integration

package bootstrap_test

import (
	_ "embed"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/stretchr/testify/require"
)

//go:embed schema_migrations/000045_finite_asset_size.up.sql
var finiteAssetSizeUp string

//go:embed schema_migrations/000045_finite_asset_size.down.sql
var finiteAssetSizeDown string

func TestAssetSizeMigrationPreservesObjectsMetadataAndRevisions(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	other := otherFiniteOrderScope(t, f)
	f.exec(t, finiteAssetSizeDown)
	type savedAsset struct {
		metadata string
		size     float64
		revision int64
	}
	expected := map[string]savedAsset{}
	scopes := [][3]any{{f.workspace, f.project, f.item}, {other.workspace, other.project, other.item}, {nil, nil, nil}}
	for _, scope := range scopes {
		for _, deleted := range []any{nil, "2020-01-04"} {
			for _, value := range []string{"NaN", "Infinity", "-Infinity", "-12.5", "0", "37.5", "1.7976931348623157e308"} {
				id := uuid.New().String()
				f.exec(t, insertSizedAsset, id, scope[0], scope[1], scope[2], value, deleted)
				var saved savedAsset
				require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (to_jsonb(a)-'size')::text,size,revision FROM file_assets a JOIN file_asset_edit_revisions ON asset_id=id WHERE id=$1`, id).Scan(&saved.metadata, &saved.size, &saved.revision))
				if value == "NaN" || value == "Infinity" || value == "-Infinity" {
					saved.size = 0
					saved.revision++
				}
				expected[id] = saved
			}
		}
	}
	f.exec(t, finiteAssetSizeUp)
	check := func() {
		t.Helper()
		var count int
		require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM file_assets`).Scan(&count))
		require.Equal(t, len(expected), count)
		for id, want := range expected {
			var actual savedAsset
			require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (to_jsonb(a)-'size')::text,size,revision FROM file_assets a JOIN file_asset_edit_revisions ON asset_id=id WHERE id=$1`, id).Scan(&actual.metadata, &actual.size, &actual.revision))
			require.Equal(t, want, actual, id)
		}
	}
	check()
	require.NoError(t, bootstrap.MigrateSchema(f.ctx, f.url))
	check()
	f.exec(t, finiteAssetSizeDown)
	check()
}
