//go:build integration

package mcpactions_test

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type templateScope struct{ kind, project string }

func templateFixture(t *testing.T) milestoneFixture {
	t.Helper()
	f := newMilestoneFixture(t)
	_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20; UPDATE project_members SET role=20`)
	require.NoError(t, err)
	return f
}

func templateScopes() []templateScope {
	return []templateScope{{"workitem", ""}, {"page", ""}, {"project", ""}, {"workitem", testsupport.LabelProject}, {"page", testsupport.LabelProject}}
}

func templateStorageSnapshot(t *testing.T, f milestoneFixture) []byte {
	t.Helper()
	return templateSnapshotQuery(t, t.Context(), f.pool.QueryRow)
}

func templateSnapshotQuery(t *testing.T, ctx context.Context, query func(context.Context, string, ...any) pgx.Row) []byte {
	t.Helper()
	var snapshot string
	require.NoError(t, query(ctx, `SELECT jsonb_build_object(
  'templates', (SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM resource_templates t),
  'events', (SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM template_events t),
  'settings', (SELECT jsonb_agg(to_jsonb(t) ORDER BY workspace_id) FROM workspace_template_settings t),
  'receipts', (SELECT jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text) FROM mutation_receipts t)
 )::text`).Scan(&snapshot))
	return []byte(snapshot)
}

func TestMCPTemplateReceiptsRetainAcceptedTimezone(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, scope := range templateScopes() {
			t.Run(scope.kind+"/"+scope.project, func(t *testing.T) {
				input := map[string]any{"action": "create", "kind": scope.kind, "name": "Receipt template", "template_data": `{"created_at":"2026-10-01T12:00:00+05:30","nested":{"updated_at":"unchanged"},"serial":9007199254740993}`, "request_id": uuid.NewString()}
				if scope.project != "" {
					input["project_id"] = scope.project
				}
				for _, action := range []string{"create", "update"} {
					_, err := f.pool.Exec(t.Context(), `UPDATE users SET user_timezone='Asia/Kolkata' WHERE id=$1`, testsupport.LabelActor)
					require.NoError(t, err)
					response := call("template", input)
					accepted := milestoneObject(t, response)
					require.Equal(t, false, accepted["replayed"])
					record := accepted["result"].(map[string]any)
					for _, field := range []string{"created_at", "updated_at"} {
						stamp, parseErr := time.Parse(time.RFC3339Nano, record[field].(string))
						require.NoError(t, parseErr)
						_, offset := stamp.Zone()
						require.Equal(t, 19800, offset, field)
					}
					require.Contains(t, response.Content[0].Text, "9007199254740993")
					data := record["template_data"].(map[string]any)
					require.Equal(t, "2026-10-01T12:00:00+05:30", data["created_at"])
					require.Equal(t, map[string]any{"updated_at": "unchanged"}, data["nested"])
					before := templateStorageSnapshot(t, f)
					_, err = f.pool.Exec(t.Context(), `UPDATE users SET user_timezone='UTC' WHERE id=$1`, testsupport.LabelActor)
					require.NoError(t, err)
					accepted["replayed"] = true
					require.Equal(t, accepted, milestoneObject(t, call("template", input)), action+" replay must retain the accepted timestamp representation")
					require.Equal(t, before, templateStorageSnapshot(t, f))
					input["action"], input["template_id"], input["name"] = "update", record["id"], "Updated template"
					input["request_id"] = uuid.NewString()
					delete(input, "template_data")
				}
			})
		}
	})
}
