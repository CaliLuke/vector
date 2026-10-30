//go:build integration

package bootstrap_test

import (
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestLabelFiniteOrderConstraintRejectsInvalidWrites(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	id := uuid.New().String()
	f.exec(t, `INSERT INTO labels(id,workspace_id,project_id,name,color,description,sort_order,created_at,updated_at)
  VALUES($1,$2,$3,'Finite','#abc','',12.5,now(),now())`, id, f.workspace, f.project)
	for _, value := range []string{"NaN", "Infinity", "-Infinity"} {
		for _, query := range []string{
			`UPDATE labels SET sort_order=$4::text::double precision WHERE id=$1 AND workspace_id=$2 AND project_id=$3`,
			`INSERT INTO labels(id,workspace_id,project_id,name,color,description,sort_order,created_at,updated_at)
    VALUES(gen_random_uuid(),$2,$3,$1::text,'#abc','',$4::text::double precision,now(),now())`,
		} {
			_, err := f.db.Exec(f.ctx, query, id, f.workspace, f.project, value)
			var failure *pgconn.PgError
			require.ErrorAs(t, err, &failure)
			require.Equal(t, "23514", failure.Code)
			require.Equal(t, "labels_sort_order_finite", failure.ConstraintName)
		}
	}
	var count int
	var order float64
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM labels`).Scan(&count))
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT sort_order FROM labels WHERE id=$1`, id).Scan(&order))
	require.Equal(t, 1, count)
	require.Equal(t, 12.5, order)
}
