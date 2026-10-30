//go:build integration

package bootstrap_test

import (
	"fmt"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestOrderingConstraintsRejectNonfiniteInsertsAndUpdates(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	checkFiniteOrderInventory(t, f)
	for _, row := range seedFiniteOrders(t, f) {
		t.Run(row.table, func(t *testing.T) {
			table := pgx.Identifier{"public", row.table}.Sanitize()
			column := pgx.Identifier{row.column}.Sanitize()
			for _, value := range []string{"NaN", "Infinity", "-Infinity"} {
				for _, insert := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/insert=%t", value, insert), func(t *testing.T) {
						query := fmt.Sprintf(`UPDATE %s SET %s=$2::text::double precision WHERE id=$1`, table, column)
						args := []any{row.id, value}
						if insert {
							query = fmt.Sprintf(`INSERT INTO %s SELECT (jsonb_populate_record(NULL::%s,to_jsonb(saved)||jsonb_build_object('id',$3::uuid,$4::text,$2::text))).* FROM %s saved WHERE id=$1`, table, table, table)
							args = append(args, uuid.New().String(), row.column)
						}
						_, err := f.db.Exec(f.ctx, query, args...)
						var failure *pgconn.PgError
						require.ErrorAs(t, err, &failure)
						require.Equal(t, "23514", failure.Code)
						require.Equal(t, row.table+"_"+row.column+"_finite", failure.ConstraintName)
					})
				}
			}
			var order float64
			require.NoError(t, f.db.QueryRow(f.ctx, fmt.Sprintf(`SELECT %s FROM %s WHERE id=$1`, column, table), row.id).Scan(&order))
			require.Equal(t, 12.5, order, "rejected writes must preserve the stored order")
		})
	}
}

func checkFiniteOrderInventory(t *testing.T, f milestoneSchemaFixture) {
	t.Helper()
	rows, err := f.db.Query(f.ctx, `SELECT table_name||'.'||column_name FROM information_schema.columns WHERE table_schema='public' AND data_type='double precision' AND column_name IN('sort_order','sequence')`)
	require.NoError(t, err)
	actual, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	expected := []string{"labels.sort_order"}
	for _, spec := range finiteOrderTables {
		expected = append(expected, spec.table+"."+spec.column)
	}
	require.ElementsMatch(t, expected, actual, "every floating-point ordering column needs storage coverage")
}
