package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	loomhttp "github.com/CaliLuke/loom/http"
)

type cycleLifecycleOptionsAuthorizerStub struct {
	exists       bool
	lookupCalls  int
	lookupTarget systemapi.SessionProtocolResource
}

func (*cycleLifecycleOptionsAuthorizerStub) AuthorizeSessionProtocol(context.Context, string) (int, map[string]any) {
	return 0, nil
}

func (*cycleLifecycleOptionsAuthorizerStub) AuthorizeSessionResourceProtocol(context.Context, string, systemapi.SessionProtocolResource) (int, map[string]any) {
	return 0, nil
}

func (*cycleLifecycleOptionsAuthorizerStub) OwnsSessionProtocolMethod(string, string) bool {
	return false
}

func (stub *cycleLifecycleOptionsAuthorizerStub) SessionProtocolResourceExists(_ context.Context, resource systemapi.SessionProtocolResource) (bool, error) {
	stub.lookupCalls++
	stub.lookupTarget = resource
	return stub.exists, nil
}

func TestCycleDetailOptionsUsesSourceMetadataAndOmitsActionsForMissingObject(t *testing.T) {
	cycles := &cycleLifecycleOptionsAuthorizerStub{exists: true}
	dispatch := labelSessionProtocolDispatch{cycles: cycles}
	stack := func(next http.Handler) http.Handler { return next }
	handler := newSessionServiceProtocolMiddleware(notificationProtocolMetadata, dispatch, loomhttp.RuntimeCORSPolicy{}, stack, djangoSessionCookieConfig{}, false, apiV1CompatibilityOptions{})(http.NotFoundHandler())
	path := "/api/workspaces/acme/projects/11111111-1111-4111-8111-111111111111/cycles/22222222-2222-4222-8222-222222222222/"

	for _, test := range []struct {
		name        string
		exists      bool
		wantActions bool
	}{
		{name: "existing cycle has PUT action metadata", exists: true, wantActions: true},
		{name: "missing cycle omits actions", exists: false, wantActions: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			cycles.exists = test.exists
			request := httptest.NewRequest(http.MethodOptions, path, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "validated"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			actions, hasActions := body["actions"].(map[string]any)
			if response.Code != http.StatusOK || hasActions != test.wantActions || body["name"] != "Cycle" {
				t.Fatalf("status=%d metadata=%s", response.Code, response.Body.String())
			}
			if test.wantActions && actions["PUT"] == nil {
				t.Fatalf("existing cycle metadata has no PUT schema: %s", response.Body.String())
			}
			if response.Header().Get("Allow") != "PUT, PATCH, DELETE, OPTIONS" {
				t.Fatalf("Allow = %q", response.Header().Get("Allow"))
			}
			if cycles.lookupTarget.ID != "22222222-2222-4222-8222-222222222222" || cycles.lookupTarget.ProjectID != "11111111-1111-4111-8111-111111111111" {
				t.Fatalf("resource lookup received %#v", cycles.lookupTarget)
			}
		})
	}
	if cycles.lookupCalls != 2 {
		t.Fatalf("detail object lookups = %d, want 2", cycles.lookupCalls)
	}
}

func TestCycleProtocolMetadataDoesNotChangeNotificationArchiveMetadata(t *testing.T) {
	protocol := newSessionServiceProtocol("/api/workspaces/{slug}/users/notifications/{pk}/archive/", "POST, DELETE, OPTIONS", "Notification", false)
	_, got := notificationProtocolMetadata(protocol)
	metadata, ok := got.(json.RawMessage)
	if !ok || !bytes.Equal(metadata, notificationActionMetadata) {
		t.Fatalf("notification archive metadata changed: %#v", got)
	}
}
