//go:build integration

package backgroundtasks

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"

	"github.com/CaliLuke/selvedge"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// Exercise the new immutable snapshots through the real durable worker and
// PostgreSQL activity store, including replay after the activity commit.
func assertCatalogueActivityWorkflows(t *testing.T, ctx context.Context, app *selvedge.App, dispatcher *IssueActivityDispatcher, store *PostgresIssueActivityStore, pool *pgxpool.Pool, base IssueActivityInput) {
	t.Helper()
	logID, relatedID := uuid.NewString(), uuid.NewString()
	before := `{"id":"` + logID + `","duration":30,"description":"Before"}`
	after := `{"id":"` + logID + `","duration":45,"description":"After"}`
	relation := `{"related_id":"` + relatedID + `","label":"supports","identifier":"REL-2"}`
	for _, tc := range []struct {
		kind, verb, field, requested, current string
		oldValue, newValue                    *string
		oldID, newID                          *string
	}{
		{"work_log", "created", "work_log", before, "null", nil, &before, nil, &logID},
		{"work_log", "updated", "work_log", after, before, &before, &after, &logID, &logID},
		{"work_log", "deleted", "work_log", "null", after, &after, nil, &logID, nil},
		{"workitem_relation", "created", "supports", relation, "null", nil, stringPointer("REL-2"), nil, &relatedID},
		{"workitem_relation", "deleted", "supports", relation, "null", stringPointer("REL-2"), nil, &relatedID, nil},
	} {
		t.Run(tc.kind+"/"+tc.verb, func(t *testing.T) {
			input := base
			input.Type = tc.kind + ".activity." + tc.verb
			input.RequestedData, input.CurrentInstance = json.RawMessage(tc.requested), json.RawMessage(tc.current)
			eventID := uuid.NewString()
			workflow, err := dispatcher.Admit(ctx, eventID, input)
			require.NoError(t, err)
			waitVersionWorkflow(t, ctx, app, workflow)
			_, err = store.RecordIssueActivity(ctx, eventID, input)
			require.NoError(t, err)
			var count int
			require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field=$2 AND verb=$3`, input.IssueID, tc.field, tc.verb).Scan(&count))
			require.Equal(t, 1, count, "worker replay must not duplicate history")
			var oldValue, newValue, oldID, newID *string
			require.NoError(t, pool.QueryRow(ctx, `SELECT old_value,new_value,old_identifier::text,new_identifier::text FROM issue_activities WHERE issue_id=$1 AND field=$2 AND verb=$3`, input.IssueID, tc.field, tc.verb).Scan(&oldValue, &newValue, &oldID, &newID))
			require.Equal(t, tc.oldValue, oldValue)
			require.Equal(t, tc.newValue, newValue)
			require.Equal(t, tc.oldID, oldID)
			require.Equal(t, tc.newID, newID)
		})
	}
}

// Own the database so the activity/replay proof runs in the ordinary full gate.
func openCatalogueActivityDatabase(t *testing.T) (string, *pgxpool.Pool) {
	t.Helper()
	startMilestoneActivityRedis(t)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	pair, err := postgresfixture.StartUncloned(ctx, bootstrap.MigrateSchema)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pair.Close()) })
	pool, err := pgxpool.New(ctx, pair.SeedURL())
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	_, err = pool.Exec(ctx, `CREATE SCHEMA plane_execution`)
	require.NoError(t, err)
	executionURL, err := withExecutionSearchPath(pair.SeedURL())
	require.NoError(t, err)
	return executionURL, pool
}
