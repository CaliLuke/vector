//go:build integration

package bootstrap_test

import (
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestTypeLevelMigrationPreservesUnassignedTypeMetadata(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	other := otherFiniteOrderScope(t, f)
	f.exec(t, validTypeLevelsDown)
	type savedType struct{ metadata, level string }
	expected := map[string]savedType{}
	for _, scope := range []milestoneSchemaFixture{f, other} {
		for _, deleted := range []any{nil, "2020-01-04"} {
			for _, value := range []string{"NaN", "Infinity", "-Infinity", "-1", "0.5", "21", "2147483648", "1.7976931348623157e308", "0", "1", "20"} {
				id := uuid.New().String()
				f.exec(t, insertNumericType, id, scope.workspace, value, deleted)
				var saved savedType
				require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (to_jsonb(t)-'level')::text,level::text FROM issue_types t WHERE id=$1`, id).Scan(&saved.metadata, &saved.level))
				if value != "0" && value != "1" && value != "20" {
					saved.level = "0"
				}
				expected[id] = saved
			}
		}
	}
	before := map[string]int64{}
	for _, workspace := range []string{f.workspace, other.workspace} {
		var revision int64
		require.NoError(t, f.db.QueryRow(f.ctx, `SELECT revision FROM work_item_type_api_revisions WHERE workspace_id=$1`, workspace).Scan(&revision))
		before[workspace] = revision
	}
	f.exec(t, validTypeLevelsUp)
	for id, want := range expected {
		var actual savedType
		require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (to_jsonb(t)-'level')::text,level::text FROM issue_types t WHERE id=$1`, id).Scan(&actual.metadata, &actual.level))
		require.Equal(t, want, actual, id)
	}
	for workspace, revision := range before {
		var actual int64
		require.NoError(t, f.db.QueryRow(f.ctx, `SELECT revision FROM work_item_type_api_revisions WHERE workspace_id=$1`, workspace).Scan(&actual))
		require.Equal(t, revision+16, actual)
	}
	var settings int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM work_item_type_settings WHERE workspace_id IN($1,$2)`, f.workspace, other.workspace).Scan(&settings))
	require.Zero(t, settings, "repair must not enable types or hierarchy")
}
