//go:build integration

package mcpactions_test

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPProjectDeleteReceiptSurvivesWorkerCascade(t *testing.T) {
	f := newMilestoneFixture(t)
	_, err := f.pool.Exec(t.Context(), `CREATE SCHEMA plane_execution`)
	require.NoError(t, err)
	executionURL, err := url.Parse(f.pool.Config().ConnString())
	require.NoError(t, err)
	query := executionURL.Query()
	query.Set("options", "-c search_path=plane_execution,public")
	executionURL.RawQuery = strings.ReplaceAll(query.Encode(), "+", "%20")
	app := selvedge.NewApp(selvedge.WithDatabase(executionURL.String()), selvedge.WithServiceName("project-delete-recovery"), selvedge.WithActivityPollInterval(10*time.Millisecond), selvedge.WithWorkflowResumptionInterval(10*time.Millisecond))
	store, err := backgroundtasks.NewPostgresDeletionStore(f.pool)
	require.NoError(t, err)
	dispatcher, err := backgroundtasks.RegisterDeletionWorkflows(app, store, 60, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
	require.NoError(t, err)
	require.NoError(t, app.Start(t.Context()))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, app.Shutdown(ctx))
	})
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		created := milestoneObject(t, call("project", map[string]any{"action": "create", "name": "Delete recovery", "identifier": strings.ToUpper(uuid.NewString()[:8])}))
		id := created["id"].(string)
		input := map[string]any{"action": "delete", "project_id": id, "request_id": uuid.NewString()}
		deleted := milestoneObject(t, call("project", input))
		require.Equal(t, false, deleted["replayed"])
		var eventID string
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT id FROM work_item_effects WHERE kind='soft_delete' AND payload->>'model'='project' AND payload->>'id'=$1`, id).Scan(&eventID))
		ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
		defer cancel()
		instance, err := dispatcher.Admit(ctx, backgroundtasks.DeletionInput{EventID: eventID, App: "db", Model: "project", ID: id})
		require.NoError(t, err)
		result, err := selvedge.WaitWorkflow[struct{}](ctx, app, instance)
		require.NoError(t, err)
		require.Equal(t, "completed", result.Status, result.Error)
		var identifierID string
		var identifierDeleted bool
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT id::text,deleted_at IS NOT NULL FROM project_identifiers WHERE project_id=$1`, id).Scan(&identifierID, &identifierDeleted))
		require.True(t, identifierDeleted, "integer-keyed project identifiers must finish their child cascade")
		identifierRun, err := dispatcher.Admit(ctx, backgroundtasks.DeletionInput{EventID: uuid.NewString(), App: "db", Model: "projectidentifier", ID: identifierID})
		require.NoError(t, err, "direct admission must accept the model's integer key")
		identifierResult, err := selvedge.WaitWorkflow[struct{}](ctx, app, identifierRun)
		require.NoError(t, err)
		require.Equal(t, "completed", identifierResult.Status, identifierResult.Error)
		replay := milestoneObject(t, call("project", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, deleted["revision"], replay["revision"])
		require.Equal(t, deleted["replay_expires_at"], replay["replay_expires_at"])
		milestoneFailure(t, call("project", map[string]any{"action": "retrieve", "project_id": id}), "not_found")
		var effects int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects WHERE kind='soft_delete' AND payload->>'model'='project' AND payload->>'id'=$1`, id).Scan(&effects))
		require.Equal(t, 1, effects, "receipt replay must not enqueue deletion again")
	})
}
