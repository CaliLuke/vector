//go:build integration

package bootstrap_test

import (
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

const insertSizedAsset = `INSERT INTO file_assets(id,created_at,updated_at,attributes,asset,workspace_id,project_id,issue_id,entity_type,size,is_uploaded,is_deleted,is_archived,storage_metadata,deleted_at)
 VALUES($1::uuid,'2020-01-02','2020-01-03','{"name":"kept.txt","type":"text/plain","size":17.5}',$1::text,$2,$3,$4,'ISSUE_ATTACHMENT',$5::text::double precision,true,false,false,'{"retained":true}',$6::timestamptz)`

func TestAssetSizeConstraintRejectsNonfiniteWrites(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	id := uuid.New().String()
	f.exec(t, insertSizedAsset, id, f.workspace, f.project, f.item, "37.5", nil)
	for _, value := range []string{"NaN", "Infinity", "-Infinity"} {
		t.Run(value, func(t *testing.T) {
			for _, insert := range []bool{false, true} {
				query := `UPDATE file_assets SET size=$2::text::double precision WHERE id=$1`
				args := []any{id, value}
				if insert {
					query = insertSizedAsset
					args = []any{uuid.New().String(), f.workspace, f.project, f.item, value, nil}
				}
				_, err := f.db.Exec(f.ctx, query, args...)
				var failure *pgconn.PgError
				require.ErrorAs(t, err, &failure)
				require.Equal(t, "23514", failure.Code)
				require.Equal(t, "file_assets_size_finite", failure.ConstraintName)
			}
		})
	}
	var size float64
	var revision int64
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT size,revision FROM file_assets JOIN file_asset_edit_revisions ON asset_id=id WHERE id=$1`, id).Scan(&size, &revision))
	require.Equal(t, 37.5, size)
	require.Equal(t, int64(1), revision)
}
