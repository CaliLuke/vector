//go:build integration

package bootstrap_test

import (
	_ "embed"
	"testing"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

//go:embed schema_migrations/000046_valid_type_levels.up.sql
var validTypeLevelsUp string

//go:embed schema_migrations/000046_valid_type_levels.down.sql
var validTypeLevelsDown string

func TestTypeLevelMigrationRecoversProjectCopiesWithoutChangingLinks(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	h := seedNumericHierarchy(t, f)
	other := seedNumericHierarchy(t, otherFiniteOrderScope(t, f))
	control := readNumericHierarchy(t, other)
	for _, value := range []string{"NaN", "Infinity", "-Infinity"} {
		t.Run(value, func(t *testing.T) {
			f.exec(t, validTypeLevelsDown)
			corruptHierarchyLevels(t, h, value)
			before := readNumericHierarchy(t, h)
			f.exec(t, validTypeLevelsUp)
			after := readNumericHierarchy(t, h)
			require.Equal(t, before.metadata, after.metadata)
			require.Equal(t, "0", after.childLevel)
			require.Equal(t, "1", after.parentLevel)
			require.Equal(t, before.nativeRevision+1, after.nativeRevision)
			require.Equal(t, before.apiRevision+3, after.apiRevision)
			require.Equal(t, control, readNumericHierarchy(t, other))
		})
	}
	before := readNumericHierarchy(t, h)
	require.NoError(t, bootstrap.MigrateSchema(f.ctx, f.url))
	require.Equal(t, before, readNumericHierarchy(t, h))
	f.exec(t, validTypeLevelsDown)
	require.Equal(t, before, readNumericHierarchy(t, h))
}

func TestTypeLevelMigrationRollsBackAmbiguousOrInvalidRecovery(t *testing.T) {
	for _, test := range []struct{ name, change, message string }{
		{"conflicting project copies", `INSERT INTO project_issue_types(id,workspace_id,project_id,issue_type_id,is_default,level,is_active,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,false,2,true,now(),now())`, "project copies disagree"},
		{"inconsistent parent levels", `UPDATE project_issue_types SET level=2 WHERE workspace_id=$1 AND issue_type_id=$3 AND project_id<>$2::uuid`, "without changing parent relationships"},
		{"out-of-range project copy", `UPDATE project_issue_types SET level=21 WHERE workspace_id=$1 AND issue_type_id=$3 AND project_id<>$2::uuid`, "outside the supported range"},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newMilestoneSchemaFixture(t)
			h := seedNumericHierarchy(t, f)
			f.exec(t, validTypeLevelsDown)
			corruptHierarchyLevels(t, h, "NaN")
			f.exec(t, test.change, h.workspace, h.otherProject, h.parentType)
			before := readNumericHierarchy(t, h)
			_, err := f.db.Exec(f.ctx, validTypeLevelsUp)
			var failure *pgconn.PgError
			require.ErrorAs(t, err, &failure)
			require.Equal(t, "23514", failure.Code)
			require.Contains(t, failure.Message, test.message)
			f.exec(t, `ROLLBACK`)
			require.Equal(t, before, readNumericHierarchy(t, h), "failed repair must preserve levels, links, metadata and revisions")
		})
	}
}
