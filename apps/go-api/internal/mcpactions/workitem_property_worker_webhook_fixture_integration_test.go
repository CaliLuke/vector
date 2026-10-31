//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

type propertyWebhook struct {
	Event, Action string
	WorkspaceID   string `json:"workspace_id"`
	Data          struct{ ID string }
	Activity      struct {
		Field    string
		OldValue json.RawMessage `json:"old_value"`
		NewValue json.RawMessage `json:"new_value"`
		Actor    struct{ ID string }
	}
}

type propertyWebhookReceiver struct {
	mu       sync.Mutex
	received []propertyWebhook
}

func newPropertyWebhookReceiver(t *testing.T, f milestoneFixture) *propertyWebhookReceiver {
	t.Helper()
	receiver := &propertyWebhookReceiver{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event propertyWebhook
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("decode delivered webhook: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if event.Activity.Field == "custom_properties" {
			receiver.mu.Lock()
			receiver.received = append(receiver.received, event)
			receiver.mu.Unlock()
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)
	t.Setenv("WEBHOOK_ALLOWED_HOSTS", "127.0.0.1")
	t.Setenv("WEBHOOK_ALLOWED_IPS", "")
	_, err := f.pool.Exec(t.Context(), `INSERT INTO webhooks(id,created_at,updated_at,url,is_active,secret_key,project,issue,module,cycle,issue_comment,created_by_id,workspace_id,is_internal,version)
 VALUES(gen_random_uuid(),now(),now(),$1,true,'property-worker-secret',false,true,false,false,false,$2,$3,false,'v1')`, server.URL, testsupport.LabelActor, testsupport.LabelWorkspace)
	require.NoError(t, err)
	return receiver
}

func (r *propertyWebhookReceiver) assertChanges(t *testing.T, value propertyWorkerCase) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	var actual []map[string]any
	for _, event := range r.received {
		if event.Data.ID != value.itemID {
			continue
		}
		require.Equal(t, "issue", event.Event)
		require.Equal(t, "updated", event.Action)
		require.Equal(t, testsupport.LabelWorkspace, event.WorkspaceID)
		require.Equal(t, testsupport.LabelActor, event.Activity.Actor.ID)
		var oldValue, newValue map[string]any
		require.NoError(t, json.Unmarshal(event.Activity.OldValue, &oldValue))
		require.NoError(t, json.Unmarshal(event.Activity.NewValue, &newValue))
		actual = append(actual, map[string]any{"old": oldValue, "new": newValue})
	}
	first := map[string]any{value.propertyID: "Original value"}
	second := map[string]any{value.propertyID: "Changed value"}
	cleared := map[string]any{value.propertyID: nil}
	require.ElementsMatch(t, []map[string]any{
		{"old": map[string]any{}, "new": first},
		{"old": first, "new": second},
		{"old": second, "new": cleared},
	}, actual, "deliver each committed value transition exactly once, retaining its snapshot")
}
