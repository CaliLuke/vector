//go:build integration

package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/stretchr/testify/require"
)

// A lost response happens after real durable admission, before the outbox row
// can be marked admitted. The next drain must reconcile the same event.
type lostMilestoneWebhookAdmission struct {
	delegate effectWebhookAdmission
	lost     bool
}

func (a *lostMilestoneWebhookAdmission) AdmitWebhookTask(ctx context.Context, id, name string, args []any, kwargs map[string]any) error {
	if err := a.delegate.AdmitWebhookTask(ctx, id, name, args, kwargs); err != nil {
		return err
	}
	if !a.lost {
		a.lost = true
		return errors.New("lost admission response")
	}
	return nil
}

func TestMilestoneWebhookOutboxReconcilesAdmissionAndRetriesDelivery(t *testing.T) {
	f := newMilestoneWebhookFixture(t)
	f.create(t)
	hook := f.hook(t, true)
	event := f.effect(t, webhookModelActivityTask)
	var requests atomic.Int32
	bodies := make(chan []byte, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		requests.Add(1)
		bodies <- body
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)
	t.Setenv("WEBHOOK_ALLOWED_HOSTS", "127.0.0.1")
	t.Setenv("WEBHOOK_ALLOWED_IPS", "")
	f.exec(t, `UPDATE webhooks SET url=$2 WHERE id=$1`, hook, server.URL)
	var attempts atomic.Int32
	sender := &PostgresWebhookTaskStore{}
	f.store.requestClient = func(ctx context.Context, url string, headers http.Header, body []byte) (*http.Response, error) {
		if attempts.Add(1) == 1 {
			return nil, errors.New("temporary transport failure")
		}
		return sender.post(ctx, url, headers, body)
	}
	f.exec(t, `CREATE SCHEMA plane_execution`)
	executionURL, err := withExecutionSearchPath(f.pool.Config().ConnString())
	require.NoError(t, err)
	app := selvedge.NewApp(selvedge.WithDatabase(executionURL), selvedge.WithServiceName("milestone-webhook-proof"), selvedge.WithActivityPollInterval(10*time.Millisecond), selvedge.WithWorkflowResumptionInterval(10*time.Millisecond), selvedge.WithTimerCheckInterval(10*time.Millisecond))
	dispatcher, err := RegisterWebhookTaskWorkflows(app, f.store, retry.Exponential(3, time.Millisecond, time.Millisecond, 1))
	require.NoError(t, err)
	require.NoError(t, app.Start(f.ctx))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, app.Shutdown(ctx))
	})
	admission, err := NewWebhookTaskAdmission(dispatcher)
	require.NoError(t, err)
	outbox := &PostgresWorkItemEffectStore{pool: f.pool, webhooks: &lostMilestoneWebhookAdmission{delegate: admission}}
	count, err := outbox.Drain(f.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	var admitted bool
	var admissions int
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT admitted_at IS NOT NULL,attempts FROM work_item_effects WHERE id=$1`, event.EventID).Scan(&admitted, &admissions))
	require.False(t, admitted)
	require.Equal(t, 1, admissions)
	f.exec(t, `UPDATE work_item_effects SET next_attempt_at=now() WHERE id=$1`, event.EventID)
	count, err = outbox.Drain(f.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	waitVersionWorkflow(t, f.ctx, app, "plane-webhook-model:"+event.EventID)
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT admitted_at IS NOT NULL,attempts FROM work_item_effects WHERE id=$1`, event.EventID).Scan(&admitted, &admissions))
	require.True(t, admitted)
	require.Equal(t, 2, admissions)
	require.EqualValues(t, 2, attempts.Load())
	require.EqualValues(t, 1, requests.Load())
	var payload webhookPayload
	require.NoError(t, json.Unmarshal(<-bodies, &payload))
	require.Equal(t, "milestone", payload.Event)
	require.Equal(t, "created", payload.Action)
	require.Equal(t, hook, payload.WebhookID)
	var failed, succeeded int
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FILTER(WHERE response_status='500'),count(*) FILTER(WHERE response_status='202' AND retry_count=1) FROM webhook_logs WHERE webhook=$1`, hook).Scan(&failed, &succeeded))
	require.Equal(t, 1, failed)
	require.Equal(t, 1, succeeded)
}
