//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/CaliLuke/selvedge"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPPageHistoryAndComponentCleanupStayWithinOwningPage(t *testing.T) {
	configureCatalogueWorker(t)
	f := templateFixture(t)
	component, asset := uuid.NewString(), uuid.NewString()
	body := `<p>Original</p><image-component id="` + component + `" src="` + asset + `"></image-component>`
	var first, second string
	runCatalogue(t, f, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		first = milestoneObject(t, call("page", map[string]any{"action": "create", "name": "First", "description_html": body}))["id"].(string)
		second = milestoneObject(t, call("page", map[string]any{"action": "create", "name": "Copied component", "description_html": body}))["id"].(string)
	})
	var count int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM page_logs`).Scan(&count))
	require.Zero(t, count)
	var raw []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT jsonb_agg(jsonb_build_object('id',id,'input',payload)) FROM work_item_effects`).Scan(&raw))
	var original []struct {
		ID    string
		Input backgroundtasks.PageTransactionInput
	}
	require.NoError(t, json.Unmarshal(raw, &original))
	runtime := startCatalogueWorker(t, f)
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	waitPageWorkerEffects(t, ctx, f, runtime)
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM page_logs WHERE "transaction"=$1`, component).Scan(&count))
	require.Equal(t, 2, count)
	input := map[string]any{"action": "update", "page_id": first, "name": "Updated", "description_html": "<p>Replacement</p>", "request_id": uuid.NewString()}
	runCatalogue(t, f, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		require.Equal(t, false, milestoneObject(t, call("page", input))["replayed"])
	})
	waitPageWorkerEffects(t, ctx, f, runtime)
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM page_logs WHERE page_id=$1`, first).Scan(&count))
	require.Zero(t, count)
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM page_logs WHERE page_id=$1 AND "transaction"=$2`, second, component).Scan(&count))
	require.Equal(t, 1, count, "removing a copied image from one page must preserve the other page's log")
	var html string
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*),max(description_html) FROM page_versions WHERE page_id=$1`, first).Scan(&count, &html))
	require.Equal(t, 1, count)
	require.Equal(t, "<p>Replacement</p>", html)
	before := pageSnapshot(t, f)
	runCatalogue(t, f, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		require.Equal(t, true, milestoneObject(t, call("page", input))["replayed"])
	})
	require.Equal(t, before, pageSnapshot(t, f))
	for _, event := range original {
		_, err := runtime.PageTransactions.Admit(ctx, event.ID, event.Input)
		require.NoError(t, err)
	}
	waitPageWorkerEffects(t, ctx, f, runtime)
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM page_versions WHERE page_id=$1`, first).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM page_logs WHERE page_id=$1`, second).Scan(&count))
	require.Equal(t, 1, count)
}
func waitPageWorkerEffects(t *testing.T, ctx context.Context, f milestoneFixture, runtime *backgroundtasks.Runtime) {
	t.Helper()
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var pending int
		err := f.pool.QueryRow(ctx, `SELECT count(*) FROM work_item_effects WHERE admitted_at IS NULL OR last_error<>''`).Scan(&pending)
		assert.NoError(c, err)
		assert.Zero(c, pending)
	}, 30*time.Second, 25*time.Millisecond)
	rows, err := f.pool.Query(ctx, `SELECT id::text,kind FROM work_item_effects ORDER BY created_at,id`)
	require.NoError(t, err)
	var workflows []string
	for rows.Next() {
		var id, kind string
		require.NoError(t, rows.Scan(&id, &kind))
		prefix := "page-transaction:"
		if kind == "page_version" {
			prefix = "page-version:"
		} else {
			require.Equal(t, "page_transaction", kind)
		}
		workflows = append(workflows, prefix+id)
	}
	require.NoError(t, rows.Err())
	rows.Close()
	for _, id := range workflows {
		result, err := selvedge.WaitWorkflow[struct{}](ctx, runtime.App, id)
		require.NoError(t, err)
		require.Equal(t, "completed", result.Status, result.Error)
	}
}
