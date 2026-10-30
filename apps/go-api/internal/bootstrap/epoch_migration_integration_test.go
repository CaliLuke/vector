//go:build integration

package bootstrap_test

import (
	_ "embed"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/stretchr/testify/require"
)

//go:embed schema_migrations/000044_finite_activity_epoch.up.sql
var finiteEpochUp string

//go:embed schema_migrations/000044_finite_activity_epoch.down.sql
var finiteEpochDown string

func TestActivityEpochMigrationPreservesHistory(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	other := otherFiniteOrderScope(t, f)
	f.exec(t, finiteEpochDown)
	type savedActivity struct {
		metadata string
		epoch    *string
	}
	expected := map[string]savedActivity{}
	for _, scope := range []milestoneSchemaFixture{f, other} {
		for _, deleted := range []any{nil, "2020-01-04"} {
			for _, value := range []any{nil, "NaN", "Infinity", "-Infinity", "-12.5", "0", "123.25", "1.7976931348623157e308"} {
				id := uuid.New().String()
				f.exec(t, insertEpochActivity, id, scope.workspace, scope.project, scope.item, value, deleted)
				var saved savedActivity
				require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (to_jsonb(a)-'epoch')::text,epoch::text FROM issue_activities a WHERE id=$1`, id).Scan(&saved.metadata, &saved.epoch))
				if value == "NaN" || value == "Infinity" || value == "-Infinity" {
					saved.epoch = nil
				}
				expected[id] = saved
			}
		}
	}
	f.exec(t, finiteEpochUp)
	check := func() {
		t.Helper()
		var count int
		require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM issue_activities`).Scan(&count))
		require.Equal(t, len(expected), count)
		for id, want := range expected {
			var actual savedActivity
			require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (to_jsonb(a)-'epoch')::text,epoch::text FROM issue_activities a WHERE id=$1`, id).Scan(&actual.metadata, &actual.epoch))
			require.Equal(t, want, actual, id)
		}
	}
	check()
	require.NoError(t, bootstrap.MigrateSchema(f.ctx, f.url))
	check()
	f.exec(t, finiteEpochDown)
	check()
}
