package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"uuid"

	systemapi "github.com/CaliLuke/go-plane"
	api "github.com/CaliLuke/go-plane/workspacesearch/gen/workspace_search"
)

type searchMountRepository struct {
	denied               bool
	permissions, queries int
	count                int
}

func (r *searchMountRepository) WorkspaceAllowed(context.Context, uuid.UUID, string) (bool, error) {
	r.permissions++
	return !r.denied, nil
}
func (r *searchMountRepository) Global(context.Context, uuid.UUID, string, string, string, string, string) (json.RawMessage, error) {
	r.queries++
	return json.RawMessage(`{"issues":[]}`), nil
}
func (r *searchMountRepository) Entity(_ context.Context, _ uuid.UUID, _, _, _, _ string, count int) (json.RawMessage, error) {
	r.queries++
	r.count = count
	return json.RawMessage(`{"user_mention":[]}`), nil
}

func TestWorkspaceSearchCountUsesPythonIntegerGrammar(t *testing.T) {
	for _, count := range []string{"  +12 ", "١٢", "1_2"} {
		t.Run(count, func(t *testing.T) {
			repository := &searchMountRepository{}
			sessions := searchMountSessions{&cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}}
			service, err := systemapi.NewWorkspaceSearch(repository, sessions)
			if err != nil {
				t.Fatal(err)
			}
			handler := extensionHTTPHandler(pendingReplacementEndpoints{workspaceSearch: api.NewEndpoints(service), workspaceSearchMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}})
			request := httptest.NewRequest("GET", "/api/workspaces/demo/entity-search/?"+url.Values{"count": {count}}.Encode(), nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != 200 || repository.count != 12 || repository.queries != 1 {
				t.Fatalf("status=%d count=%d queries=%d body=%s", response.Code, repository.count, repository.queries, response.Body.String())
			}
		})
	}
}

func TestWorkspaceSearchInvalidCountReturnsValidationErrorBeforeRepository(t *testing.T) {
	repository := &searchMountRepository{}
	sessions := searchMountSessions{&cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}}
	service, err := systemapi.NewWorkspaceSearch(repository, sessions)
	if err != nil {
		t.Fatal(err)
	}
	handler := extensionHTTPHandler(pendingReplacementEndpoints{workspaceSearch: api.NewEndpoints(service), workspaceSearchMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}})
	request := httptest.NewRequest("GET", "/api/workspaces/demo/entity-search/?count=many", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || strings.TrimSpace(response.Body.String()) != `{"count":["A valid non-negative integer is required."]}` {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if repository.permissions != 1 || repository.queries != 0 {
		t.Fatalf("permissions=%d queries=%d; invalid count must not reach search", repository.permissions, repository.queries)
	}
}

func TestWorkspaceSearchRejectsMalformedProjectUUIDBeforeRepository(t *testing.T) {
	for _, tc := range []struct {
		name, path, want string
		permissions      int
	}{
		{name: "entity", path: "/api/workspaces/demo/entity-search/?query=Member&project_id=__invalid__", want: `{"error":"Please provide valid detail"}`, permissions: 1},
		{name: "global", path: "/api/workspaces/demo/search/?search=needle&entities=issue&project_id=__invalid__&workspace_search=false", want: `{"error":"Please provide valid detail"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository := &searchMountRepository{}
			sessions := searchMountSessions{&cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}}
			service, err := systemapi.NewWorkspaceSearch(repository, sessions)
			if err != nil {
				t.Fatal(err)
			}
			handler := extensionHTTPHandler(pendingReplacementEndpoints{workspaceSearch: api.NewEndpoints(service), workspaceSearchMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}})
			request := httptest.NewRequest("GET", tc.path, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest || strings.TrimSpace(response.Body.String()) != tc.want {
				t.Fatalf("status=%d body=%s, want 400 %s", response.Code, response.Body.String(), tc.want)
			}
			if repository.permissions != tc.permissions || repository.queries != 0 {
				t.Fatalf("permissions=%d queries=%d, want %d and no search query", repository.permissions, repository.queries, tc.permissions)
			}
		})
	}
}

func TestWorkspaceSearchProductionMounts(t *testing.T) {
	for _, protocol := range workspaceSearchProtocols() {
		entity := strings.HasSuffix(protocol.pattern, "/entity-search/")
		for _, tc := range []struct {
			name, method                           string
			authenticated, denied, invalidTimezone bool
		}{
			{"anonymous", "GET", false, false, false},
			{"read", "GET", true, false, false},
			{"head", "HEAD", true, false, false},
			{"metadata", "OPTIONS", true, false, false},
			{"anonymous metadata", "OPTIONS", false, false, false},
			{"denied read", "GET", true, true, false},
			{"denied metadata", "OPTIONS", true, true, false},
			{"denied unsupported method", "POST", true, true, false},
			{"invalid timezone", "GET", true, false, true},
			{"permission before timezone", "GET", true, true, true},
			{"metadata permission before timezone", "OPTIONS", true, true, true},
		} {
			t.Run(protocol.pattern+"/"+tc.name, func(t *testing.T) {
				repository := &searchMountRepository{denied: tc.denied}
				sessions := searchMountSessions{&cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC", invalidZone: tc.invalidTimezone}}
				service, err := systemapi.NewWorkspaceSearch(repository, sessions)
				if err != nil {
					t.Fatal(err)
				}
				handler := extensionHTTPHandler(pendingReplacementEndpoints{workspaceSearch: api.NewEndpoints(service), workspaceSearchMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}, activityReadsBodyLimit: 1 << 20})
				request := httptest.NewRequest(tc.method, strings.ReplaceAll(protocol.pattern, "{slug}", "demo"), nil)
				if tc.authenticated {
					request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				status := 200
				switch {
				case !tc.authenticated:
					status = 401
				case entity && tc.denied:
					status = 403
				case tc.invalidTimezone:
					status = 400
				case tc.method == "POST":
					status = 405
				}
				if response.Code != status {
					t.Fatalf("status=%d body=%s; want %d", response.Code, response.Body.String(), status)
				}
				if tc.method == http.MethodOptions && !tc.invalidTimezone {
					want := `{"description":"","name":"Search Endpoint","parses":["application/json","application/x-www-form-urlencoded","multipart/form-data"],"renders":["application/json"]}`
					if !entity {
						want = `{"description":"Endpoint to search across multiple fields in the workspace and\nalso show related workspace if found","name":"Global Search Endpoint","parses":["application/json","application/x-www-form-urlencoded","multipart/form-data"],"renders":["application/json"]}`
					}
					if status == http.StatusForbidden {
						want = `{"detail":"You do not have permission to perform this action."}`
					} else if status == http.StatusUnauthorized {
						want = `{"detail":"Authentication credentials were not provided."}`
					}
					if strings.TrimSpace(response.Body.String()) != want {
						t.Fatalf("OPTIONS body=%s; want %s", response.Body.String(), want)
					}
				}
				permissions := 0
				if entity && tc.authenticated {
					permissions = 1
				}
				queries := 0
				if status == 200 && (tc.method == "GET" || tc.method == "HEAD") {
					queries = 1
				}
				if repository.permissions != permissions || repository.queries != queries {
					t.Fatalf("permissions=%d queries=%d; want %d,%d", repository.permissions, repository.queries, permissions, queries)
				}
				if tc.method == "HEAD" && response.Body.Len() != 0 {
					t.Fatalf("HEAD body=%s", response.Body.String())
				}
			})
		}
	}
}
