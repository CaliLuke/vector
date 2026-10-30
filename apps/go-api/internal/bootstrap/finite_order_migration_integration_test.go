//go:build integration

package bootstrap_test

import (
	_ "embed"
	"fmt"
	"strconv"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

//go:embed schema_migrations/000043_finite_ordering.up.sql
var finiteOrderingUp string

//go:embed schema_migrations/000043_finite_ordering.down.sql
var finiteOrderingDown string

type savedFiniteOrder struct {
	metadata string
	order    float64
}

func TestFiniteOrderingMigrationPreservesMetadataTenantsAndRevisions(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	other := otherFiniteOrderScope(t, f)
	scopes := []milestoneSchemaFixture{f, other}
	rows := [][]finiteOrderRow{seedFiniteOrders(t, f), seedFiniteOrders(t, other)}
	cases := []struct {
		value   string
		invalid bool
		target  int
	}{
		{"NaN", true, 0}, {"NaN", true, 1}, {"Infinity", true, 0}, {"Infinity", true, 1}, {"-Infinity", true, 0}, {"-Infinity", true, 1},
		{"-12.5", false, 0}, {"0", false, 1}, {"1.7976931348623157e308", false, 0},
	}
	for _, test := range cases {
		t.Run(fmt.Sprintf("%s/tenant%d", test.value, test.target), func(t *testing.T) {
			f.exec(t, finiteOrderingDown)
			for index, scope := range scopes {
				value := "42.25"
				if index == test.target {
					value = test.value
				}
				setHistoricalOrders(t, scope, rows[index], value, test.value == "-Infinity")
			}
			before := []map[string]savedFiniteOrder{readFiniteOrders(t, f, rows[0]), readFiniteOrders(t, other, rows[1])}
			revisions := []map[string]int64{readOrderRevisions(t, f, rows[0]), readOrderRevisions(t, other, rows[1])}
			f.exec(t, finiteOrderingUp)
			for index, scope := range scopes {
				want := 42.25
				if index == test.target {
					if test.invalid {
						want = 65535
					} else {
						var err error
						want, err = strconv.ParseFloat(test.value, 64)
						require.NoError(t, err)
					}
				}
				for table, saved := range readFiniteOrders(t, scope, rows[index]) {
					require.Equal(t, before[index][table].metadata, saved.metadata, table+" metadata")
					require.Equal(t, want, saved.order, table+" order")
				}
				for table, revision := range readOrderRevisions(t, scope, rows[index]) {
					increment := int64(0)
					if test.invalid && index == test.target {
						increment = orderRevisionIncrements[table]
					}
					require.Equal(t, revisions[index][table]+increment, revision, table+" revision")
				}
			}
		})
	}
	require.NoError(t, bootstrap.MigrateSchema(f.ctx, f.url))
	// Downgrade only removes enforcement; it must not recreate corrupt positions.
	before := readFiniteOrders(t, f, rows[0])
	f.exec(t, finiteOrderingDown)
	require.Equal(t, before, readFiniteOrders(t, f, rows[0]))
}

func otherFiniteOrderScope(t *testing.T, f milestoneSchemaFixture) milestoneSchemaFixture {
	t.Helper()
	other := f
	other.workspace, other.project, other.item = uuid.New().String(), uuid.New().String(), uuid.New().String()
	f.exec(t, `INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at)
 SELECT $1::uuid,'Other',owner_id,$1::text,'UTC','',now(),now() FROM workspaces WHERE id=$2`, other.workspace, f.workspace)
	f.exec(t, `INSERT INTO projects SELECT (jsonb_populate_record(NULL::projects,to_jsonb(p)||jsonb_build_object('id',$2::uuid,'workspace_id',$3::uuid))).* FROM projects p WHERE id=$1`, f.project, other.project, other.workspace)
	f.exec(t, `INSERT INTO issues(id,workspace_id,project_id,name,description_json,description_html,priority,sort_order,sequence_id,is_draft,created_at,updated_at) VALUES($1,$2,$3,'Other work','{}','','none',0,1,false,now(),now())`, other.item, other.workspace, other.project)
	return other
}

func setHistoricalOrders(t *testing.T, f milestoneSchemaFixture, rows []finiteOrderRow, value string, deleted bool) {
	t.Helper()
	for _, row := range rows {
		extra := ""
		if row.table != "page_collection_pages" {
			extra = ",deleted_at=NULL"
			if deleted {
				extra = ",deleted_at='2020-01-03'"
			}
		}
		query := fmt.Sprintf(`UPDATE %s SET %s=$2::text::double precision%s WHERE id=$1`, pgx.Identifier{"public", row.table}.Sanitize(), pgx.Identifier{row.column}.Sanitize(), extra)
		f.exec(t, query, row.id, value)
	}
}

func readFiniteOrders(t *testing.T, f milestoneSchemaFixture, rows []finiteOrderRow) map[string]savedFiniteOrder {
	t.Helper()
	result := make(map[string]savedFiniteOrder, len(rows))
	for _, row := range rows {
		var saved savedFiniteOrder
		query := fmt.Sprintf(`SELECT (to_jsonb(saved)-$2::text)::text,%s FROM %s saved WHERE id=$1`, pgx.Identifier{row.column}.Sanitize(), pgx.Identifier{"public", row.table}.Sanitize())
		require.NoError(t, f.db.QueryRow(f.ctx, query, row.id, row.column).Scan(&saved.metadata, &saved.order))
		result[row.table] = saved
	}
	return result
}

var orderRevisionIncrements = map[string]int64{
	"cycle_edit_revisions": 1, "module_edit_revisions": 1, "state_edit_revisions": 1, "issue_edit_revisions": 1,
	"work_item_relation_revisions": 1, "release_revisions": 1, "page_api_revisions": 3, "customer_revisions": 2,
}

func readOrderRevisions(t *testing.T, f milestoneSchemaFixture, rows []finiteOrderRow) map[string]int64 {
	t.Helper()
	result := make(map[string]int64, len(orderRevisionIncrements))
	for _, row := range rows {
		resource := map[string]string{"cycles": "cycle", "modules": "module", "states": "state", "issues": "issue"}[row.table]
		if resource == "" {
			continue
		}
		table := resource + "_edit_revisions"
		var revision int64
		require.NoError(t, f.db.QueryRow(f.ctx, fmt.Sprintf(`SELECT revision FROM %s WHERE %s=$1`, pgx.Identifier{"public", table}.Sanitize(), pgx.Identifier{resource + "_id"}.Sanitize()), row.id).Scan(&revision))
		result[table] = revision
	}
	for _, table := range []string{"work_item_relation_revisions", "release_revisions", "page_api_revisions", "customer_revisions"} {
		var revision int64
		require.NoError(t, f.db.QueryRow(f.ctx, fmt.Sprintf(`SELECT revision FROM %s WHERE workspace_id=$1`, pgx.Identifier{"public", table}.Sanitize()), f.workspace).Scan(&revision))
		result[table] = revision
	}
	return result
}
