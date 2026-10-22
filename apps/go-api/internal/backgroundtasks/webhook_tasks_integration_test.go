//go:build integration

package backgroundtasks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresWebhookModelActivityDeliversSignedPayloadRetriesAndDeactivates(t *testing.T) {
	_, executionURL, pool := openBackgroundTaskDatabase(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	ownerID, workspaceID, projectID, webhookID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()
	username := "webhook-owner-" + ownerID.String()
	slug := "wh-" + ownerID.String()[:8]
	var requestCount atomic.Int32
	var captureMu sync.Mutex
	var receivedBody []byte
	var receivedSignature, receivedDelivery, receivedEvent, receivedAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestCount.Add(1) == 1 {
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				t.Error("test HTTP server does not support connection hijacking")
				return
			}
			conn, _, err := hijacker.Hijack()
			if err == nil {
				_ = conn.Close() // transport exception triggers one durable retry
			}
			return
		}
		captureMu.Lock()
		receivedBody, _ = io.ReadAll(r.Body)
		receivedSignature = r.Header.Get("X-Plane-Signature")
		receivedDelivery = r.Header.Get("X-Plane-Delivery")
		receivedEvent = r.Header.Get("X-Plane-Event")
		receivedAgent = r.UserAgent()
		captureMu.Unlock()
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, "accepted")
	}))
	defer server.Close()
	t.Setenv("WEBHOOK_ALLOWED_HOSTS", "127.0.0.1")
	t.Setenv("WEBHOOK_ALLOWED_IPS", "")

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM webhook_logs WHERE webhook=$1`, webhookID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM webhooks WHERE id=$1`, webhookID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM projects WHERE id=$1`, projectID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM workspaces WHERE id=$1`, workspaceID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM users WHERE id=$1`, ownerID)
	})
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,password,username,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid,email) VALUES ($1,'!', $2,'Webhook Owner','Webhook','Owner','',$3,$3,$3,'','',false,false,false,true,false,true,false,false,$2,'','','','',false,'UTC',true,$4)`, ownerID, username, now, username+"@example.test"); err != nil {
		t.Fatalf("insert webhook owner: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES ($1,'Webhook Test',$2,$3,'UTC','',$4,$4)`, workspaceID, ownerID, slug, now); err != nil {
		t.Fatalf("insert webhook workspace: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id,name,description,network,workspace_id,created_by_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES ($1,'Webhook Project','',0,$2,$3,'WHK',false,false,false,true,false,false,false,false,0,0,'{}'::jsonb,'UTC',$4,$4)`, projectID, workspaceID, ownerID, now); err != nil {
		t.Fatalf("insert webhook project: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO webhooks(id,created_at,updated_at,url,is_active,secret_key,project,issue,module,cycle,issue_comment,created_by_id,workspace_id,is_internal,version) VALUES($1,$2,$2,$3,true,'webhook-secret',true,false,false,false,false,$4,$5,false,'v1')`, webhookID, now, server.URL+"/hook", ownerID, workspaceID); err != nil {
		t.Fatalf("insert test webhook: %v", err)
	}

	mailer := &recordingWebhookEmailAdmission{}
	store, err := NewPostgresWebhookTaskStore(pool, mailer)
	if err != nil {
		t.Fatal(err)
	}
	app := selvedge.NewApp(
		selvedge.WithDatabase(executionURL),
		selvedge.WithServiceName("plane-webhook-integration-test"),
		selvedge.WithActivityPollInterval(10*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(10*time.Millisecond),
		selvedge.WithTimerCheckInterval(10*time.Millisecond),
	)
	policy := &retry.Policy{MaxAttempts: 2, InitialInterval: time.Millisecond, MaxInterval: time.Millisecond, Multiplier: 1, RandomizationFactor: 0}
	dispatcher, err := RegisterWebhookTaskWorkflows(app, store, policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := app.Shutdown(shutdownCtx); err != nil {
			t.Errorf("shutdown webhook workflow runtime: %v", err)
		}
	}()
	fallback := &recordingWebhookFallback{}
	admission, err := NewWebhookTaskAdmission(dispatcher)
	if err != nil {
		t.Fatal(err)
	}
	publisher, err := NewWebhookTaskPublisher(fallback, admission)
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.Task(ctx, webhookModelActivityTask, nil, map[string]any{
		"model_name": "project", "model_id": projectID.String(), "requested_data": map[string]any{"name": "Webhook Project"},
		"current_instance": nil, "actor_id": ownerID.String(), "slug": slug, "origin": "https://plane.example.test",
	}); err != nil {
		t.Fatalf("admit model_activity producer: %v", err)
	}
	waitForWebhookRequests(t, ctx, &requestCount, 2)
	waitForWebhookLogCount(t, ctx, pool, webhookID, 2)
	captureMu.Lock()
	receivedBody = append([]byte(nil), receivedBody...)
	observedSignature, observedDelivery, observedEvent, observedAgent := receivedSignature, receivedDelivery, receivedEvent, receivedAgent
	captureMu.Unlock()
	if fallback.calls.Load() != 0 {
		t.Fatalf("native webhook task fell back %d times", fallback.calls.Load())
	}
	if observedEvent != "project" || observedAgent != "Autopilot" || observedDelivery == "" {
		t.Fatalf("webhook headers event=%q agent=%q delivery=%q", observedEvent, observedAgent, observedDelivery)
	}
	if _, err := uuid.Parse(observedDelivery); err != nil {
		t.Fatalf("delivery header is not a UUID: %q", observedDelivery)
	}
	var payload webhookPayload
	if err := json.Unmarshal(receivedBody, &payload); err != nil {
		t.Fatalf("decode delivered payload: %v (%s)", err, receivedBody)
	}
	if payload.Event != "project" || payload.Action != "created" || payload.WebhookID != webhookID.String() || payload.WorkspaceID != workspaceID.String() || payload.WorkspaceSlug != slug {
		t.Fatalf("delivered payload identity = %#v", payload)
	}
	var projectData map[string]json.RawMessage
	if err := json.Unmarshal(payload.Data, &projectData); err != nil || string(projectData["id"]) != `"`+projectID.String()+`"` {
		t.Fatalf("serialized model data = %s, err=%v", payload.Data, err)
	}
	var signingPayload webhookPayload
	if err := json.Unmarshal(receivedBody, &signingPayload); err != nil {
		t.Fatal(err)
	}
	bytesToSign, err := marshalWebhookJSON(signingPayload, true)
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte("webhook-secret"))
	_, _ = mac.Write(bytesToSign)
	if want := hex.EncodeToString(mac.Sum(nil)); observedSignature != want {
		t.Fatalf("X-Plane-Signature = %q, want %q", observedSignature, want)
	}
	var statuses []string
	rows, err := pool.Query(ctx, `SELECT response_status FROM webhook_logs WHERE webhook=$1 ORDER BY created_at,id`, webhookID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		statuses = append(statuses, status)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	rows.Close()
	if len(statuses) != 2 || statuses[0] != "500" || statuses[1] != "202" {
		t.Fatalf("delivery log statuses = %v, want [500 202]", statuses)
	}
	var retryCount int
	if err := pool.QueryRow(ctx, `SELECT retry_count FROM webhook_logs WHERE webhook=$1 AND response_status='202' ORDER BY created_at DESC LIMIT 1`, webhookID).Scan(&retryCount); err != nil {
		t.Fatal(err)
	}
	if retryCount != 1 {
		t.Fatalf("successful retry_count = %d, want 1", retryCount)
	}

	// Exercise the terminal transport-failure branch without waiting through
	// Celery's minutes-long backoff. This still writes through the real DB store.
	store.requestClient = func(context.Context, string, http.Header, []byte) (*http.Response, error) {
		return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
	}
	terminalInput := webhookSendInput{EventID: uuid.NewString(), WebhookID: webhookID.String(), Slug: slug, Event: "project", EventData: json.RawMessage(`{"id":"` + projectID.String() + `"}`), Action: "created", CurrentSite: "https://plane.example.test", Activity: json.RawMessage(`null`)}
	if err := store.Deliver(ctx, terminalInput, webhookMaxAttempts); err != nil {
		t.Fatalf("terminal webhook attempt: %v", err)
	}
	var active bool
	if err := pool.QueryRow(ctx, `SELECT is_active FROM webhooks WHERE id=$1`, webhookID).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active {
		t.Fatal("webhook remains active after terminal request failure")
	}
	if mailer.calls.Load() != 1 {
		t.Fatalf("deactivation mail admissions = %d, want 1", mailer.calls.Load())
	}
}

type recordingWebhookEmailAdmission struct{ calls atomic.Int32 }

func (admission *recordingWebhookEmailAdmission) AdmitTemplatedEmailTask(context.Context, string, string, []any) error {
	admission.calls.Add(1)
	return nil
}

type recordingWebhookFallback struct{ calls atomic.Int32 }

func (fallback *recordingWebhookFallback) Task(context.Context, string, []any, map[string]any) error {
	fallback.calls.Add(1)
	return nil
}

func (fallback *recordingWebhookFallback) SoftDelete(context.Context, string, string, string) error {
	return nil
}

func waitForWebhookRequests(t *testing.T, ctx context.Context, count *atomic.Int32, want int32) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for count.Load() < want {
		select {
		case <-ctx.Done():
			t.Fatalf("webhook requests = %d, want %d: %v", count.Load(), want, ctx.Err())
		case <-ticker.C:
		}
	}
}

func waitForWebhookLogCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, webhookID uuid.UUID, want int) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var count int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_logs WHERE webhook=$1`, webhookID).Scan(&count); err != nil {
			t.Fatalf("count webhook logs: %v", err)
		}
		if count == want {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("webhook logs = %d, want %d: %v", count, want, ctx.Err())
		case <-ticker.C:
		}
	}
}
