//go:build integration

package bootstrap_test

import (
	"fmt"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

const insertNumericType = `INSERT INTO issue_types(id,workspace_id,name,description,logo_props,is_active,is_default,is_epic,level,created_at,updated_at,external_id,external_source,deleted_at)
 VALUES($1::uuid,$2,$1::text,'retained description','{"name":"bug","color":"#123456"}',true,false,false,$3::text::double precision,'2020-01-02','2020-01-03','external-type','tracker',$4::timestamptz)`

func TestTypeLevelsRejectInvalidInsertsAndUpdates(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	id := uuid.New().String()
	f.exec(t, insertNumericType, id, f.workspace, "1", nil)
	var before int64
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT revision FROM work_item_type_api_revisions WHERE workspace_id=$1`, f.workspace).Scan(&before))
	for _, value := range []string{"NaN", "Infinity", "-Infinity", "-1", "0.5", "21", "2147483648", "1.7976931348623157e308"} {
		for _, insert := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/insert=%t", value, insert), func(t *testing.T) {
				query := `UPDATE issue_types SET level=$2::text::double precision WHERE id=$1`
				args := []any{id, value}
				if insert {
					query = insertNumericType
					args = []any{uuid.New().String(), f.workspace, value, nil}
				}
				_, err := f.db.Exec(f.ctx, query, args...)
				var failure *pgconn.PgError
				require.ErrorAs(t, err, &failure)
				require.Equal(t, "23514", failure.Code)
				require.Equal(t, "issue_types_level_valid", failure.ConstraintName)
			})
		}
	}
	var level int
	var after int64
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT level::integer FROM issue_types WHERE id=$1`, id).Scan(&level))
	require.Equal(t, 1, level)
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT revision FROM work_item_type_api_revisions WHERE workspace_id=$1`, f.workspace).Scan(&after))
	require.Equal(t, before, after, "rejected writes must not change revisions")
	for _, value := range []string{"0", "1", "20"} {
		f.exec(t, insertNumericType, uuid.New().String(), f.workspace, value, nil)
	}
}
