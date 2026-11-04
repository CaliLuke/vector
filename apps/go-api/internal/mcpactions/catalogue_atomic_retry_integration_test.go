//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// Reuse one fixture and retry identity through cancellation, commit rejection,
// successful retry and replay. Each domain supplies its complete storage snapshot.
func assertCatalogueAtomicRetry(t *testing.T, f milestoneFixture, eventTable string, snapshot func(*testing.T, milestoneFixture) []byte, mutate func(context.Context) (projectmutations.Result, error)) {
	t.Helper()
	before := snapshot(t, f)
	cancelScopeWriteWhileLocked(t, f, schemaMutationLock, uuid.MustParse(testsupport.LabelWorkspace), func(ctx context.Context) error { _, err := mutate(ctx); return err })
	require.Equal(t, before, snapshot(t, f))
	table := pgx.Identifier{eventTable}.Sanitize()
	_, err := f.pool.Exec(t.Context(), `CREATE FUNCTION reject_catalogue_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected catalogue commit rejection'; END $$;
CREATE CONSTRAINT TRIGGER reject_catalogue_commit AFTER INSERT ON `+table+` DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_catalogue_commit()`)
	require.NoError(t, err)
	rejected, err := mutate(t.Context())
	require.ErrorIs(t, err, mutationreceipts.ErrIndeterminate)
	require.Equal(t, projectmutations.Result{}, rejected)
	require.Equal(t, before, snapshot(t, f))
	_, err = f.pool.Exec(t.Context(), `DROP TRIGGER reject_catalogue_commit ON `+table+`; DROP FUNCTION reject_catalogue_commit()`)
	require.NoError(t, err)
	accepted, err := mutate(t.Context())
	require.NoError(t, err)
	require.NotNil(t, accepted.Replayed)
	require.False(t, *accepted.Replayed)
	after := snapshot(t, f)
	require.NotEqual(t, before, after)
	replay, err := mutate(t.Context())
	require.NoError(t, err)
	require.NotNil(t, replay.Replayed)
	require.True(t, *replay.Replayed)
	accepted.Replayed = replay.Replayed
	require.Equal(t, accepted, replay)
	require.Equal(t, after, snapshot(t, f))
}
