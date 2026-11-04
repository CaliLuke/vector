//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func assertCatalogueNativeRevision(t *testing.T, f milestoneFixture, call catalogueCall, tool, key string, target any, query string, args pgx.NamedArgs, snapshot func(*testing.T, milestoneFixture) []byte) {
	t.Helper()
	read := map[string]any{"action": "retrieve", key: target, "include_revision": true}
	before := milestoneObject(t, call(tool, read))
	tag, err := f.pool.Exec(t.Context(), query, args)
	require.NoError(t, err)
	require.EqualValues(t, 1, tag.RowsAffected())
	after := milestoneObject(t, call(tool, read))
	require.NotEqual(t, before["revision"], after["revision"])
	stored := snapshot(t, f)
	milestoneFailure(t, call(tool, map[string]any{"action": "update", key: target, "name": "Stale edit", "expected_revision": before["revision"], "request_id": uuid.NewString()}), "revision_conflict")
	require.Equal(t, stored, snapshot(t, f))
}
