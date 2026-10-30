//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPWorkLogActivitiesUseCommittedSnapshots(t *testing.T) {
	f := newMilestoneFixture(t)
	_, err := f.pool.Exec(t.Context(), `CREATE SCHEMA plane_execution`)
	require.NoError(t, err)
	executionURL, err := url.Parse(f.pool.Config().ConnString())
	require.NoError(t, err)
	query := executionURL.Query()
	query.Set("options", "-c search_path=plane_execution,public")
	executionURL.RawQuery = strings.ReplaceAll(query.Encode(), "+", "%20")
	app := selvedge.NewApp(selvedge.WithDatabase(executionURL.String()), selvedge.WithServiceName("work-log-activity"), selvedge.WithActivityPollInterval(10*time.Millisecond), selvedge.WithWorkflowResumptionInterval(10*time.Millisecond))
	store, err := backgroundtasks.NewPostgresIssueActivityStore(f.pool)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	dispatcher, err := backgroundtasks.RegisterIssueActivityWorkflow(app, store, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
	require.NoError(t, err)
	require.NoError(t, app.Start(t.Context()))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, app.Shutdown(ctx))
	})
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		item := f.item(t)
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item})
		created := milestoneObject(t, call("work_log", map[string]any{"action": "create", "duration": 30, "description": "Before"}))
		id := created["id"].(string)
		milestoneObject(t, call("work_log", map[string]any{"action": "update", "work_log_id": id, "duration": 45, "description": "After"}))
		deletion := map[string]any{"action": "delete", "work_log_id": id, "request_id": uuid.NewString()}
		milestoneObject(t, call("work_log", deletion))
		require.Equal(t, true, milestoneObject(t, call("work_log", deletion))["replayed"])
		before, err := json.Marshal(map[string]any{"id": id, "duration": 30, "description": "Before"})
		require.NoError(t, err)
		after, err := json.Marshal(map[string]any{"id": id, "duration": 45, "description": "After"})
		require.NoError(t, err)
		for _, change := range []struct {
			verb     string
			old, new []byte
		}{
			{"created", nil, before}, {"updated", before, after}, {"deleted", after, nil},
		} {
			var eventID string
			var payload []byte
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT id::text,payload FROM work_item_effects WHERE payload->>'issue_id'=$1 AND payload->>'type'=$2`, item, "work_log.activity."+change.verb).Scan(&eventID, &payload))
			var input backgroundtasks.IssueActivityInput
			require.NoError(t, json.Unmarshal(payload, &input))
			require.False(t, input.Notification)
			ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
			run, err := dispatcher.Admit(ctx, eventID, input)
			require.NoError(t, err)
			result, err := selvedge.WaitWorkflow[struct{}](ctx, app, run)
			cancel()
			require.NoError(t, err)
			require.Equal(t, "completed", result.Status, result.Error)
			_, err = store.RecordIssueActivity(t.Context(), eventID, input)
			require.NoError(t, err, "recovery after activity commit must be idempotent")
			var oldValue, newValue *string
			var actor, workspace, project string
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT old_value,new_value,actor_id::text,workspace_id::text,project_id::text FROM issue_activities WHERE issue_id=$1 AND field='work_log' AND verb=$2`, item, change.verb).Scan(&oldValue, &newValue, &actor, &workspace, &project))
			require.Equal(t, testsupport.LabelActor, actor)
			require.Equal(t, testsupport.LabelWorkspace, workspace)
			require.Equal(t, testsupport.LabelProject, project)
			if change.old == nil {
				require.Nil(t, oldValue)
			} else {
				require.NotNil(t, oldValue)
				require.JSONEq(t, string(change.old), *oldValue)
			}
			if change.new == nil {
				require.Nil(t, newValue)
			} else {
				require.NotNil(t, newValue)
				require.JSONEq(t, string(change.new), *newValue)
			}
		}
		var activities, notifications, emails int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='work_log'),(SELECT count(*) FROM notifications WHERE entity_identifier=$1),(SELECT count(*) FROM email_notification_logs WHERE entity_identifier=$1)`, item).Scan(&activities, &notifications, &emails))
		require.Equal(t, 3, activities)
		require.Zero(t, notifications)
		require.Zero(t, emails)
	})
}
