package backgroundtasks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestWebhookModelActivityUsesRequestedFieldOrderAndSourceValues(t *testing.T) {
	store := &PostgresWebhookTaskStore{}
	input := WebhookTaskInput{
		EventID: uuid.NewString(), TaskName: webhookModelActivityTask,
		Kwargs: map[string]json.RawMessage{
			"model_name": []byte(`"issue"`), "model_id": []byte(`"` + uuid.NewString() + `"`),
			"slug": []byte(`"demo"`), "actor_id": []byte(`"` + uuid.NewString() + `"`),
			"origin":           []byte(`"https://plane.example.test"`),
			"requested_data":   []byte(`{"name":"Updated","priority":"high","unchanged":"same"}`),
			"current_instance": []byte(`"{\"name\":\"Original\",\"priority\":\"medium\",\"unchanged\":\"same\"}"`),
		},
	}
	activities, err := store.ModelActivities(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(activities) != 2 {
		t.Fatalf("changed activities = %d, want 2", len(activities))
	}
	if string(activities[0].Field) != `"name"` || string(activities[0].OldValue) != `"Original"` || string(activities[0].NewValue) != `"Updated"` {
		t.Fatalf("first field activity = %#v", activities[0])
	}
	if string(activities[1].Field) != `"priority"` || string(activities[1].OldValue) != `"medium"` || string(activities[1].NewValue) != `"high"` {
		t.Fatalf("second field activity = %#v", activities[1])
	}
	if activities[0].WorkflowEventID != input.EventID || activities[0].CurrentSite != "https://plane.example.test" {
		t.Fatalf("producer identity/origin lost: %#v", activities[0])
	}
}

func TestWebhookPayloadEncodingMatchesPythonSigningAndCompactRequestBody(t *testing.T) {
	payload := webhookPayload{
		Event: "issue", Action: "create", WebhookID: "webhook", WorkspaceID: "workspace", WorkspaceSlug: "demo",
		Data: json.RawMessage(`{"id":"issue","name":"Café"}`), Activity: json.RawMessage(`{"field":null}`),
	}
	compact, err := marshalWebhookJSON(payload, false)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(compact), `{"event":"issue","action":"create","webhook_id":"webhook","workspace_id":"workspace","workspace_slug":"demo","data":{"id":"issue","name":"Caf\u00e9"},"activity":{"field":null}}`; got != want {
		t.Fatalf("compact payload = %s, want %s", got, want)
	}
	spaced, err := marshalWebhookJSON(payload, true)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(spaced), `{"event": "issue", "action": "create", "webhook_id": "webhook", "workspace_id": "workspace", "workspace_slug": "demo", "data": {"id": "issue", "name": "Caf\u00e9"}, "activity": {"field": null}}`; got != want {
		t.Fatalf("Python-compatible signing payload = %s, want %s", got, want)
	}
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write(spaced)
	if got, want := hex.EncodeToString(mac.Sum(nil)), "c5289024ef0ff199606b765882c9047be69deb3565931b74ecbdac6a4118f68c"; got != want {
		t.Fatalf("signature = %s, want %s", got, want)
	}
}

func TestWebhookPinnedPostUsesTrustedLocalHostAndDoesNotFollowRedirects(t *testing.T) {
	t.Setenv("WEBHOOK_ALLOWED_HOSTS", "127.0.0.1")
	t.Setenv("WEBHOOK_ALLOWED_IPS", "")
	var observedHost, observedAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observedHost = r.Host
		observedAgent = r.UserAgent()
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Location", "/must-not-follow")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	store := &PostgresWebhookTaskStore{}
	response, err := store.post(context.Background(), server.URL+"/hook", http.Header{"Content-Type": {"application/json"}, "User-Agent": {"Autopilot"}}, []byte(`{"event":"issue"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Errorf("close HTTP response body: %v", closeErr)
		}
	}()
	if response.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("response status = %d, want 307", response.StatusCode)
	}
	if observedHost == "" || observedAgent != "Autopilot" {
		t.Fatalf("server observed Host %q User-Agent %q", observedHost, observedAgent)
	}
}

func TestWebhookPinnedPostRejectsPrivateTargetWithoutAllowlist(t *testing.T) {
	t.Setenv("WEBHOOK_ALLOWED_HOSTS", "")
	t.Setenv("WEBHOOK_ALLOWED_IPS", "")
	store := &PostgresWebhookTaskStore{}
	response, err := store.post(context.Background(), "http://127.0.0.1:1234/hook", nil, []byte(`{}`))
	if response != nil {
		defer func() {
			if closeErr := response.Body.Close(); closeErr != nil {
				t.Errorf("close HTTP response body: %v", closeErr)
			}
		}()
	}
	if err == nil || !strings.Contains(err.Error(), "private or reserved IP") {
		t.Fatalf("private target error = %v, want source URL policy rejection", err)
	}
}
