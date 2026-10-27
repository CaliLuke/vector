//go:build integration

package backgroundtasks

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func startMilestoneActivityRedis(t *testing.T) *redis.Client {
	t.Helper()
	container, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{Image: "valkey/valkey:9.1.2-alpine", ExposedPorts: []string{"6379/tcp"}, WaitingFor: wait.ForListeningPort("6379/tcp")}, Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, testcontainers.TerminateContainer(container)) })
	host, err := container.Host(t.Context())
	require.NoError(t, err)
	port, err := container.MappedPort(t.Context(), "6379/tcp")
	require.NoError(t, err)
	address := net.JoinHostPort(host, port.Port())
	t.Setenv("REDIS_URL", "redis://"+address+"/0")
	client := redis.NewClient(&redis.Options{Addr: address})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	require.NoError(t, client.Ping(t.Context()).Err())
	return client
}

func (f milestoneWebhookFixture) activityItem(t *testing.T) uuid.UUID {
	t.Helper()
	item := uuid.New()
	f.exec(t, `INSERT INTO issues(id,project_id,workspace_id,created_by_id,name,description_json,description_html,priority,sequence_id,sort_order,is_draft,created_at,updated_at)
 VALUES($1,$2,$3,$4,'Milestone item','{}','','medium',1,0,false,now(),now())`, item, testsupport.LabelProject, testsupport.LabelWorkspace, testsupport.LabelActor)
	return item
}

func (f milestoneWebhookFixture) subscriber(t *testing.T, item uuid.UUID) string {
	t.Helper()
	id := uuid.NewString()
	f.exec(t, `INSERT INTO users SELECT (jsonb_populate_record(NULL::users,to_jsonb(source)||jsonb_build_object('id',$1::uuid,'username',$1::text,'email',$1::text||'@example.test','display_name','Receiver'))).* FROM users source WHERE id=$2`, id, testsupport.LabelActor)
	f.exec(t, `INSERT INTO workspace_members SELECT (jsonb_populate_record(NULL::workspace_members,to_jsonb(source)||jsonb_build_object('id',gen_random_uuid(),'member_id',$1::uuid))).* FROM workspace_members source WHERE member_id=$2`, id, testsupport.LabelActor)
	f.exec(t, `INSERT INTO project_members SELECT (jsonb_populate_record(NULL::project_members,to_jsonb(source)||jsonb_build_object('id',gen_random_uuid(),'member_id',$1::uuid))).* FROM project_members source WHERE member_id=$2`, id, testsupport.LabelActor)
	f.exec(t, `INSERT INTO issue_subscribers(id,created_at,updated_at,workspace_id,project_id,issue_id,subscriber_id) VALUES(gen_random_uuid(),now(),now(),$1,$2,$3,$4)`, testsupport.LabelWorkspace, testsupport.LabelProject, item, id)
	return id
}

func (f milestoneWebhookFixture) activityEffect(t *testing.T) (string, IssueActivityInput) {
	t.Helper()
	var id string
	var raw []byte
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT id::text,payload FROM work_item_effects WHERE kind='activity' ORDER BY created_at DESC,id DESC LIMIT 1`).Scan(&id, &raw))
	var kwargs map[string]any
	require.NoError(t, json.Unmarshal(raw, &kwargs))
	input, err := issueActivityInputFromTask(kwargs)
	require.NoError(t, err)
	return id, input
}

func (f milestoneWebhookFixture) activityStore(t *testing.T) *PostgresIssueActivityStore {
	t.Helper()
	store, err := NewPostgresIssueActivityStore(f.pool)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	return store
}

func (f milestoneWebhookFixture) recordAndNotify(t *testing.T, store *PostgresIssueActivityStore, event string, input IssueActivityInput) []string {
	t.Helper()
	ids, err := store.RecordIssueActivity(f.ctx, event, input)
	require.NoError(t, err)
	require.Len(t, ids, 1)
	require.NoError(t, store.ProcessIssueNotifications(f.ctx, event, input, ids))
	return ids
}

func closeMilestoneActivityStore(t *testing.T, close func() error) {
	t.Helper()
	t.Cleanup(func() { require.NoError(t, close()) })
}

func (f milestoneWebhookFixture) activityRuntime(t *testing.T, store *PostgresIssueActivityStore) (*selvedge.App, *IssueActivityDispatcher) {
	t.Helper()
	f.exec(t, `CREATE SCHEMA plane_execution`)
	url, err := withExecutionSearchPath(f.pool.Config().ConnString())
	require.NoError(t, err)
	app := selvedge.NewApp(selvedge.WithDatabase(url), selvedge.WithServiceName("milestone-activity-proof"), selvedge.WithActivityPollInterval(10*time.Millisecond), selvedge.WithWorkflowResumptionInterval(10*time.Millisecond))
	dispatcher, err := RegisterIssueActivityWorkflow(app, store, retry.Exponential(3, time.Millisecond, time.Millisecond, 1))
	require.NoError(t, err)
	require.NoError(t, app.Start(f.ctx))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, app.Shutdown(ctx))
	})
	return app, dispatcher
}
