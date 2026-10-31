//go:build integration

package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func webhookDeactivationInput(id string) webhookSendInput {
	return webhookSendInput{EventID: uuid.NewString(), WebhookID: id, WorkspaceID: testsupport.LabelWorkspace, Slug: "milestones", Event: "project", Action: "created", CurrentSite: "https://example.test", EventData: json.RawMessage(`{}`), Activity: json.RawMessage(`null`)}
}

func TestWebhookDeactivationRetainsEmailDuringAdmissionOutage(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		name := "workspace identity"
		if legacy {
			name = "legacy slug"
		}
		t.Run(name, func(t *testing.T) {
			f := newMilestoneWebhookFixture(t)
			id := f.hook(t, true)
			input := webhookDeactivationInput(id)
			if legacy {
				input.WorkspaceID = ""
			}
			requests := 0
			f.store.requestClient = func(context.Context, string, http.Header, []byte) (*http.Response, error) {
				requests++
				return nil, errors.New("transport unavailable")
			}
			require.NoError(t, f.store.Deliver(f.ctx, input, webhookMaxAttempts))
			var active bool
			require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT is_active FROM webhooks WHERE id=$1`, id).Scan(&active))
			require.False(t, active)

			mailer := &templatedEmailAdmissionSpy{err: errors.New("email admission unavailable")}
			effects := &PostgresWorkItemEffectStore{pool: f.pool, emails: mailer}
			count, err := effects.Drain(f.ctx)
			require.NoError(t, err)
			require.Equal(t, 1, count)
			require.Len(t, mailer.events, 1)
			eventID := mailer.events[0]
			require.Equal(t, []string{webhookDeactivationEmailTask}, mailer.names)
			require.Equal(t, []any{id, testsupport.LabelActor, input.CurrentSite, "transport unavailable"}, mailer.args[0])
			var retained, delayed bool
			var attempts int
			var lastError string
			require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT admitted_at IS NULL AND payload<>'{}',next_attempt_at>clock_timestamp(),attempts,last_error FROM work_item_effects WHERE id=$1`, eventID).Scan(&retained, &delayed, &attempts, &lastError))
			require.True(t, retained)
			require.True(t, delayed)
			require.Equal(t, 1, attempts)
			require.Equal(t, mailer.err.Error(), lastError)

			// Replaying the terminal delivery must not create another email intent.
			require.NoError(t, f.store.Deliver(f.ctx, input, webhookMaxAttempts))
			if !legacy {
				require.Equal(t, 1, requests, "native deliveries skip inactive webhooks")
			}
			f.exec(t, `UPDATE work_item_effects SET next_attempt_at=clock_timestamp() WHERE id=$1`, eventID)
			recovered := &templatedEmailAdmissionSpy{}
			effects = &PostgresWorkItemEffectStore{pool: f.pool, emails: recovered}
			count, err = effects.Drain(f.ctx)
			require.NoError(t, err)
			require.Equal(t, 1, count)
			require.Equal(t, mailer.events, recovered.events, "restart must reuse the durable admission identity")
			require.Equal(t, mailer.args, recovered.args)
			var admitted int
			require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM work_item_effects WHERE admitted_at IS NOT NULL AND payload='{}' AND attempts=2 AND last_error=''`).Scan(&admitted))
			require.Equal(t, 1, admitted)
			count, err = effects.Drain(f.ctx)
			require.NoError(t, err)
			require.Zero(t, count)
		})
	}
}

func TestWebhookDeactivationRollsBackWhenEmailCannotBeQueued(t *testing.T) {
	f := newMilestoneWebhookFixture(t)
	id := f.hook(t, true)
	f.store.requestClient = func(context.Context, string, http.Header, []byte) (*http.Response, error) {
		return nil, errors.New("transport unavailable")
	}
	f.exec(t, `CREATE FUNCTION reject_deactivation_email() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'outbox unavailable'; END $$; CREATE TRIGGER reject_deactivation_email BEFORE INSERT ON work_item_effects FOR EACH STATEMENT EXECUTE FUNCTION reject_deactivation_email()`)
	input := webhookDeactivationInput(id)
	require.ErrorContains(t, f.store.Deliver(f.ctx, input, webhookMaxAttempts), "outbox unavailable")
	var active bool
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT is_active FROM webhooks WHERE id=$1`, id).Scan(&active))
	require.True(t, active, "deactivation must roll back with its email intent")
	f.exec(t, `DROP TRIGGER reject_deactivation_email ON work_item_effects`)
	require.NoError(t, f.store.Deliver(f.ctx, input, webhookMaxAttempts))
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT is_active FROM webhooks WHERE id=$1`, id).Scan(&active))
	require.False(t, active)
	var pending int
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM work_item_effects WHERE admitted_at IS NULL`).Scan(&pending))
	require.Equal(t, 1, pending)
}

func TestWebhookConcurrentTerminalFailuresQueueOneDeactivationEmail(t *testing.T) {
	f := newMilestoneWebhookFixture(t)
	id := f.hook(t, true)
	ready, release := make(chan struct{}, 2), make(chan struct{})
	f.store.requestClient = func(ctx context.Context, _ string, _ http.Header, _ []byte) (*http.Response, error) {
		ready <- struct{}{}
		select {
		case <-release:
			return nil, errors.New("transport unavailable")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	results := make(chan error, 2)
	for range 2 {
		go func() { results <- f.store.Deliver(f.ctx, webhookDeactivationInput(id), webhookMaxAttempts) }()
	}
	for range 2 {
		select {
		case <-ready:
		case <-f.ctx.Done():
			t.Fatal(f.ctx.Err())
		}
	}
	close(release)
	for range 2 {
		select {
		case err := <-results:
			require.NoError(t, err)
		case <-f.ctx.Done():
			t.Fatal(f.ctx.Err())
		}
	}
	var pending int
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM work_item_effects WHERE payload->>'name'=$1 AND payload->'args'->>0=$2`, webhookDeactivationEmailTask, id).Scan(&pending))
	require.Equal(t, 1, pending)
}

func TestWebhookWithoutCreatorDeactivatesWithoutEmail(t *testing.T) {
	f := newMilestoneWebhookFixture(t)
	id := f.hook(t, true)
	f.exec(t, `UPDATE webhooks SET created_by_id=NULL WHERE id=$1`, id)
	f.store.requestClient = func(context.Context, string, http.Header, []byte) (*http.Response, error) {
		return nil, errors.New("transport unavailable")
	}
	require.NoError(t, f.store.Deliver(f.ctx, webhookDeactivationInput(id), webhookMaxAttempts))
	var active bool
	var pending int
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT is_active FROM webhooks WHERE id=$1`, id).Scan(&active))
	require.False(t, active)
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM work_item_effects`).Scan(&pending))
	require.Zero(t, pending)
}
