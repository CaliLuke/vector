//go:build integration

package bootstrap_test

import (
	"fmt"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

const insertEpochActivity = `INSERT INTO issue_activities(id,created_at,updated_at,workspace_id,project_id,issue_id,verb,comment,attachments,epoch,deleted_at)
 VALUES($1,'2020-01-02','2020-01-03',$2,$3,$4,'updated','retained activity',ARRAY['kept.txt'],$5::text::double precision,$6::timestamptz)`

func TestActivityEpochConstraintRejectsNonfiniteWrites(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	id := uuid.New().String()
	f.exec(t, insertEpochActivity, id, f.workspace, f.project, f.item, "12.5", nil)
	for _, value := range []string{"NaN", "Infinity", "-Infinity"} {
		for _, insert := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/insert=%t", value, insert), func(t *testing.T) {
				query := `UPDATE issue_activities SET epoch=$2::text::double precision WHERE id=$1`
				args := []any{id, value}
				if insert {
					query = insertEpochActivity
					args = []any{uuid.New().String(), f.workspace, f.project, f.item, value, nil}
				}
				_, err := f.db.Exec(f.ctx, query, args...)
				var failure *pgconn.PgError
				require.ErrorAs(t, err, &failure)
				require.Equal(t, "23514", failure.Code)
				require.Equal(t, "issue_activities_epoch_finite", failure.ConstraintName)
			})
		}
	}
	var epoch float64
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT epoch FROM issue_activities WHERE id=$1`, id).Scan(&epoch))
	require.Equal(t, 12.5, epoch)
	for _, value := range []any{nil, "0", "-12.5", "1.7976931348623157e308"} {
		f.exec(t, insertEpochActivity, uuid.New().String(), f.workspace, f.project, f.item, value, nil)
	}
}
