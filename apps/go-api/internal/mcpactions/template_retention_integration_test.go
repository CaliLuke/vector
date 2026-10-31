//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/templates"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/selvedge"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTemplateRetentionPreservesRecentDefinitionsAuditAndReceipts(t *testing.T) {
	f := templateFixture(t)
	configureCatalogueWorker(t)
	repository := templates.NewRepository(f.pool)
	query := templates.Query{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), WorkspaceSlug: "workspace-one", Kind: "project"}
	var ids []string
	for _, name := range []string{"Old deletion", "Recent deletion", "Active definition"} {
		created, err := repository.Mutate(t.Context(), templates.Command{Query: query, Action: "create", Name: &name, Data: map[string]any{"name": "Reusable project"}})
		require.NoError(t, err)
		ids = append(ids, created.ID)
	}
	request := uuid.New()
	query.ID = uuid.MustParse(ids[0])
	remove := templates.Command{Query: query, Action: "delete", RequestID: &request}
	accepted, err := repository.Mutate(t.Context(), remove)
	require.NoError(t, err)
	query.ID = uuid.MustParse(ids[1])
	_, err = repository.Mutate(t.Context(), templates.Command{Query: query, Action: "delete"})
	require.NoError(t, err)
	now := time.Now().UTC()
	_, err = f.pool.Exec(t.Context(), `UPDATE resource_templates SET deleted_at=$2 WHERE id=$1`, ids[0], now.Add(-61*24*time.Hour))
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `UPDATE resource_templates SET created_at=$2 WHERE id=$1`, ids[2], now.Add(-61*24*time.Hour))
	require.NoError(t, err)
	audit := func() json.RawMessage {
		var raw json.RawMessage
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM template_events e`).Scan(&raw))
		return raw
	}
	before := audit()
	runtime := startCatalogueWorker(t, f)
	instance, err := runtime.Deletion.AdmitHardDelete(t.Context(), now)
	require.NoError(t, err)
	outcome, err := selvedge.WaitWorkflow[struct{}](t.Context(), runtime.App, instance)
	require.NoError(t, err)
	require.Equal(t, "completed", outcome.Status, outcome.Error)
	var remaining []string
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT array_agg(id::text ORDER BY id) FROM resource_templates WHERE id=ANY($1::uuid[])`, ids).Scan(&remaining))
	require.ElementsMatch(t, ids[1:], remaining, "retention uses deletion age and preserves live definitions")
	require.Equal(t, before, audit())
	snapshot := templateStorageSnapshot(t, f)
	replay, err := repository.Mutate(t.Context(), remove)
	require.NoError(t, err)
	require.NotNil(t, replay.Replayed)
	require.True(t, *replay.Replayed)
	accepted.Replayed = replay.Replayed
	require.Equal(t, accepted, replay, "purging a tombstone must not invalidate its unexpired deletion receipt")
	require.Equal(t, snapshot, templateStorageSnapshot(t, f))
}
