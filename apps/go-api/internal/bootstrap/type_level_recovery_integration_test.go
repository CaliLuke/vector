//go:build integration

package bootstrap_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/stretchr/testify/require"
)

// The guarded journal reset matches the documented recovery procedure. A
// successful migration must never be rewound through this recovery path.
const retryTypeLevelMigration = `UPDATE public.plane_schema_migrations SET version=45,dirty=false
 WHERE version=46 AND dirty
 AND NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='public.issue_types'::regclass AND conname='issue_types_level_valid')`

func TestTypeLevelUpgradeFailurePreservesDataAndSupportsReviewedRetry(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	h := seedNumericHierarchy(t, f)
	f.exec(t, validTypeLevelsDown)
	f.exec(t, `UPDATE public.plane_schema_migrations SET version=45,dirty=false`)
	corruptHierarchyLevels(t, h, "NaN")
	f.exec(t, `UPDATE project_issue_types SET level=2 WHERE issue_type_id=$1`, h.parentType)
	before := readNumericHierarchy(t, h)
	require.ErrorContains(t, bootstrap.MigrateSchema(f.ctx, f.url), "without changing parent relationships")
	require.Equal(t, before, readNumericHierarchy(t, h))
	var version int
	var dirty bool
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT version,dirty FROM public.plane_schema_migrations`).Scan(&version, &dirty))
	require.Equal(t, 46, version)
	require.True(t, dirty)
	// The operator restores the reviewed project copy; no work-item link changes.
	f.exec(t, `UPDATE project_issue_types SET level=1 WHERE issue_type_id=$1`, h.parentType)
	reviewed := readNumericHierarchy(t, h)
	result, err := f.db.Exec(f.ctx, retryTypeLevelMigration)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.RowsAffected())
	require.NoError(t, bootstrap.MigrateSchema(f.ctx, f.url))
	after := readNumericHierarchy(t, h)
	require.Equal(t, reviewed.metadata, after.metadata)
	require.Equal(t, "0", after.childLevel)
	require.Equal(t, "1", after.parentLevel)
	require.Equal(t, reviewed.nativeRevision+1, after.nativeRevision)
	require.Equal(t, reviewed.apiRevision+3, after.apiRevision)
	// A lost journal acknowledgement can leave a completed migration dirty.
	f.exec(t, `UPDATE public.plane_schema_migrations SET dirty=true WHERE version=46`)
	result, err = f.db.Exec(f.ctx, retryTypeLevelMigration)
	require.NoError(t, err)
	require.Zero(t, result.RowsAffected())
	f.exec(t, `UPDATE public.plane_schema_migrations SET dirty=false WHERE version=46`)
	require.NoError(t, bootstrap.MigrateSchema(f.ctx, f.url))
	require.Equal(t, after, readNumericHierarchy(t, h))
}
