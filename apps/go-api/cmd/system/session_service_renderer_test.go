package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
)

func TestSessionServiceRendererFailurePreservesHTMLAndAudit(t *testing.T) {
	for _, marked := range []bool{false, true} {
		t.Run(map[bool]string{false: "JSON application failure", true: "HTML renderer failure"}[marked], func(t *testing.T) {
			logger := &apiV1ActivityLoggerStub{}
			options := apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit, activityLogger: logger, activitySecret: "test-secret"}
			endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Allow", "GET, POST, HEAD, OPTIONS")
				if marked {
					systemapi.MarkReadServiceDjangoHTML500(r.Context())
				}
				writeReadServiceJSON(w, 500, map[string]string{"error": "application failure"})
			})
			handler := sessionServiceMiddlewareStack(options, nil, djangoSessionCookieConfig{})(endpoint)
			request := httptest.NewRequest("POST", "/api/workspaces/demo/users/notifications/", strings.NewReader(`{}`))
			request.Header.Set("X-Api-Key", "audit-only")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != 500 || len(logger.records) != 1 || logger.records[0].ResponseCode != 500 {
				t.Fatalf("status=%d audits=%+v", response.Code, logger.records)
			}
			if logger.records[0].ResponseBody != response.Body.String() {
				t.Fatalf("audit body=%v response=%s", logger.records[0].ResponseBody, response.Body)
			}
			if marked {
				if response.Header().Get("Content-Length") != strconv.Itoa(len(djangoServerErrorBody)) {
					t.Fatalf("Content-Length=%q", response.Header().Get("Content-Length"))
				}
				if response.Body.String() != string(djangoServerErrorBody) || response.Header().Get("Content-Type") != "text/html; charset=utf-8" || response.Header().Get("Allow") != "" {
					t.Fatalf("renderer response=%s headers=%v", response.Body, response.Header())
				}
			} else if response.Body.String() != `{"error":"application failure"}` || response.Header().Get("Content-Type") != "application/json" || response.Header().Get("Allow") != "GET, POST, HEAD, OPTIONS" {
				t.Fatalf("application response=%s headers=%v", response.Body, response.Header())
			}
		})
	}
}

type scopedSessionProtocolStub struct {
	resource systemapi.SessionProtocolResource
	status   int
}

func (*scopedSessionProtocolStub) AuthorizeSessionProtocol(context.Context, string) (int, map[string]any) {
	panic("resource authorization must be used")
}

func (s *scopedSessionProtocolStub) AuthorizeSessionResourceProtocol(_ context.Context, _ string, resource systemapi.SessionProtocolResource) (int, map[string]any) {
	s.resource = resource
	return s.status, map[string]any{"detail": "denied"}
}

func TestSessionServiceAuthoredOptionsAndResourceAuthorization(t *testing.T) {
	const pattern = "/api/workspaces/{slug}/projects/{project_id}/protocol-test/{pk}/"
	const path = "/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/protocol-test/00000000-0000-4000-8000-000000000002/"
	original := readServiceProtocols
	protocol := newSessionServiceProtocol(pattern, "POST, OPTIONS", "Test", false)
	protocol.authoredOptions = true
	readServiceProtocols = append(append([]readServiceProtocol(nil), original...), protocol)
	t.Cleanup(func() { readServiceProtocols = original })
	policy, err := djangoRuntimeCORSPolicy([]string{"https://app.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, method string
		preflight    bool
		denied, want int
		called       bool
	}{
		{"authored OPTIONS", "OPTIONS", false, 0, 202, true},
		{"preflight bypass", "OPTIONS", true, 403, 200, false},
		{"denied before dispatch", "TRACE", false, 403, 403, false},
		{"authorized unsupported", "TRACE", false, 0, 405, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			authorizer := &scopedSessionProtocolStub{status: tc.denied}
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called = true; w.WriteHeader(202) })
			identity := func(h http.Handler) http.Handler { return h }
			metadata := func(readServiceProtocol) (int, any) { t.Fatal("authored metadata was intercepted"); return 0, nil }
			handler := newSessionServiceProtocolMiddleware(metadata, authorizer, policy, identity, djangoSessionCookieConfig{}, false, apiV1CompatibilityOptions{})(next)
			request := httptest.NewRequest(tc.method, path, nil)
			if tc.preflight {
				request.Header.Set("Origin", "https://app.example.test")
				request.Header.Set("Access-Control-Request-Method", "POST")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.want || called != tc.called {
				t.Fatalf("status=%d called=%v body=%s", response.Code, called, response.Body)
			}
			if tc.method == "TRACE" {
				want := systemapi.SessionProtocolResource{Method: "TRACE", Pattern: pattern, Slug: "demo", ProjectID: "00000000-0000-4000-8000-000000000001", ID: "00000000-0000-4000-8000-000000000002", PathParameters: map[string]string{"slug": "demo", "project_id": "00000000-0000-4000-8000-000000000001", "pk": "00000000-0000-4000-8000-000000000002"}}
				if !reflect.DeepEqual(authorizer.resource, want) {
					t.Fatalf("resource=%+v want=%+v", authorizer.resource, want)
				}
			} else if !reflect.DeepEqual(authorizer.resource, systemapi.SessionProtocolResource{}) {
				t.Fatalf("unexpected authorization=%+v", authorizer.resource)
			}
		})
	}
}
