//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPInitiativeHistorySurvivesRenameDeletionAndWorkerReplay(t *testing.T) {
	assertCatalogueAssociationHistory(t, "initiative", initiativeFixture(t), initiativeSnapshot)
}

func TestMCPReleaseHistorySurvivesRenameDeletionAndWorkerReplay(t *testing.T) {
	assertCatalogueAssociationHistory(t, "release", templateFixture(t), releaseSnapshot)
}

func assertCatalogueAssociationHistory(t *testing.T, tool string, f milestoneFixture, snapshot func(*testing.T, milestoneFixture) []byte) {
	t.Helper()
	configureCatalogueWorker(t)
	var id, item string
	var deletion map[string]any
	original, renamed := uuid.NewString(), uuid.NewString()
	runCatalogue(t, f, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		id = milestoneObject(t, call(tool, map[string]any{"action": "create", "name": original}))["id"].(string)
		item = f.item(t)
		add := map[string]any{"action": "manage_workitems", tool + "_id": id, "add_ids": item, "request_id": uuid.NewString()}
		require.Equal(t, false, milestoneObject(t, call(tool, add))["replayed"])
		require.Equal(t, true, milestoneObject(t, call(tool, add))["replayed"])
		milestoneObject(t, call(tool, map[string]any{"action": "update", tool + "_id": id, "name": renamed}))
		deletion = map[string]any{"action": "delete", tool + "_id": id, "request_id": uuid.NewString()}
		require.Equal(t, false, milestoneObject(t, call(tool, deletion))["replayed"])
	})
	events := catalogueWorkerEvents(t, f)
	require.Len(t, events, 2, "link and deletion each commit one durable history event")
	var count int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field=$2`, item, tool).Scan(&count))
	require.Zero(t, count, "history remains queued while the worker is stopped")
	runtime := startCatalogueWorker(t, f)
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	waitCatalogueWorkerEvents(t, ctx, f, runtime, events)
	readHistory := func() []byte {
		var raw []byte
		require.NoError(t, f.pool.QueryRow(ctx, `SELECT jsonb_agg(jsonb_build_array(verb,old_value,new_value,old_identifier,new_identifier,actor_id,workspace_id,project_id) ORDER BY created_at,id) FROM issue_activities WHERE issue_id=$1 AND field=$2`, item, tool).Scan(&raw))
		return raw
	}
	history := readHistory()
	var rows [][]any
	require.NoError(t, json.Unmarshal(history, &rows))
	require.ElementsMatch(t, [][]any{
		{"created", nil, original, nil, id, testsupport.LabelActor, testsupport.LabelWorkspace, testsupport.LabelProject},
		{"deleted", renamed, nil, id, nil, testsupport.LabelActor, testsupport.LabelWorkspace, testsupport.LabelProject},
	}, rows)
	before := snapshot(t, f)
	runCatalogue(t, f, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		require.Equal(t, true, milestoneObject(t, call(tool, deletion))["replayed"])
	})
	waitCatalogueWorkerEvents(t, ctx, f, runtime, events)
	require.Equal(t, before, snapshot(t, f))
	require.JSONEq(t, string(history), string(readHistory()))
	var live bool
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT deleted_at IS NULL FROM issues WHERE id=$1`, item).Scan(&live))
	require.True(t, live)
}
