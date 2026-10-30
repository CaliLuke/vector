//go:build integration

package bootstrap_test

import (
	_ "embed"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

//go:embed schema_migrations/000042_label_finite_order.up.sql
var labelFiniteOrderUp string

//go:embed schema_migrations/000042_label_finite_order.down.sql
var labelFiniteOrderDown string

func TestLabelFiniteOrderMigrationRepairsOnlyInvalidOrders(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	f.exec(t, labelFiniteOrderDown)
	type savedLabel struct {
		id, rest string
		order    float64
		revision int64
	}
	labels := []savedLabel{}
	for index, value := range []string{"NaN", "Infinity", "-Infinity", "-12.5", "0", "1.7976931348623157e308"} {
		id := uuid.New().String()
		var project any = f.project
		if index == 1 {
			project = nil
		}
		f.exec(t, `INSERT INTO labels(id,workspace_id,project_id,name,color,description,sort_order,created_at,updated_at,deleted_at)
   VALUES($1::uuid,$2,$3,$1::text,'#abc','preserve metadata',$4::text::double precision,'2020-01-01','2020-01-02',CASE WHEN $5 THEN '2020-01-03'::timestamptz END)`, id, f.workspace, project, value, index == 2)
		saved := savedLabel{id: id}
		require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (to_jsonb(label)-'sort_order')::text,sort_order FROM labels label WHERE id=$1`, id).Scan(&saved.rest, &saved.order))
		require.NoError(t, f.db.QueryRow(f.ctx, `SELECT revision FROM label_edit_revisions WHERE label_id=$1`, id).Scan(&saved.revision))
		if index < 3 {
			saved.order = 65535
			saved.revision++
		}
		labels = append(labels, saved)
	}
	f.exec(t, labelFiniteOrderUp)
	for _, saved := range labels {
		var rest string
		var order float64
		var revision int64
		require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (to_jsonb(label)-'sort_order')::text,sort_order FROM labels label WHERE id=$1`, saved.id).Scan(&rest, &order))
		require.NoError(t, f.db.QueryRow(f.ctx, `SELECT revision FROM label_edit_revisions WHERE label_id=$1`, saved.id).Scan(&revision))
		require.Equal(t, saved.rest, rest, "repair changed unrelated label metadata")
		require.Equal(t, saved.order, order)
		require.Equal(t, saved.revision, revision, "only repaired rows must advance their revision")
	}
}
