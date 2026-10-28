//go:build integration

package backgroundtasks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CaliLuke/selvedge"
	"github.com/google/uuid"
)

func TestPostgresEventTrackingWorkflowDeliversPostHogCaptureOverHTTP(t *testing.T) {
	_, executionURL, pool := openBackgroundTaskDatabase(t)
	t.Cleanup(pool.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	captures := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		select {
		case captures <- body:
		default:
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()
	t.Setenv("SKIP_ENV_VAR", "0")
	t.Setenv("POSTHOG_API_KEY", "local-proof-key")
	t.Setenv("POSTHOG_HOST", server.URL)

	store, err := NewPostgresEventTrackingStore(pool, eventTrackingCipherStub{})
	if err != nil {
		t.Fatal(err)
	}
	app := selvedge.NewApp(
		selvedge.WithDatabase(executionURL),
		selvedge.WithServiceName("plane-event-tracking-integration-test"),
		selvedge.WithActivityPollInterval(20*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(20*time.Millisecond),
		selvedge.WithTimerCheckInterval(20*time.Millisecond),
	)
	dispatcher, err := RegisterEventTrackingWorkflow(app, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	if startErr := app.Start(ctx); startErr != nil {
		t.Fatal(startErr)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if shutdownErr := app.Shutdown(shutdownCtx); shutdownErr != nil {
			t.Errorf("shutdown event tracking runtime: %v", shutdownErr)
		}
	}()

	userID := uuid.NewString()
	workspaceID := uuid.NewString()
	slug := "event-proof-" + userID[:8]
	now := time.Now().UTC()
	username := "event-proof-" + userID
	if _, execErr := pool.Exec(ctx, `INSERT INTO users (id,password,username,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ($1,'!',$2,'Event Owner','Event','Owner','',$3,$3,$3,'','',false,false,false,true,false,true,false,false,$2,'','','','',false,'UTC',true)`, userID, username, now); execErr != nil {
		t.Fatalf("insert event owner: %v", execErr)
	}
	if _, execErr := pool.Exec(ctx, `INSERT INTO workspaces (id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES ($1,'Event Workspace',$2,$3,'UTC','',$4,$4)`, workspaceID, userID, slug, now); execErr != nil {
		t.Fatalf("insert event workspace: %v", execErr)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM workspaces WHERE id=$1`, workspaceID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID)
	})
	instanceID, err := dispatcher.Admit(ctx, uuid.NewString(), EventTrackingInput{
		UserID: userID, EventName: "workspace_deleted", Slug: slug,
		EventProperties: map[string]any{"workspace_id": workspaceID, "role": "admin"},
	})
	if err != nil {
		t.Fatalf("admit event tracking workflow: %v", err)
	}
	result, err := selvedge.WaitWorkflow[struct{}](ctx, app, instanceID)
	if err != nil {
		t.Fatalf("wait for event tracking workflow: %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("event tracking workflow status=%q error=%v", result.Status, result.Error)
	}
	select {
	case body := <-captures:
		batch, ok := body["batch"].([]any)
		if !ok || len(batch) != 1 {
			t.Fatalf("PostHog payload batch=%#v", body["batch"])
		}
		capture, ok := batch[0].(map[string]any)
		if !ok || capture["event"] != "workspace_deleted" || capture["distinct_id"] != userID {
			t.Fatalf("PostHog capture=%#v", batch[0])
		}
		properties, ok := capture["properties"].(map[string]any)
		groups, groupsOK := properties["$groups"].(map[string]any)
		if !ok || !groupsOK || groups["workspace"] != slug || properties["role"] != "owner" {
			t.Fatalf("PostHog capture properties=%#v", capture["properties"])
		}
	case <-ctx.Done():
		t.Fatal("PostHog SDK did not deliver the capture to the local HTTP receiver")
	}
}

type eventTrackingCipherStub struct{}

func (eventTrackingCipherStub) Decrypt(string) (string, error) { return "", nil }
