package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	projectmembershipreadsapi "github.com/CaliLuke/go-plane/gen/project_membership_reads"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	intakeStore "github.com/CaliLuke/go-plane/internal/projectintakewrites"
	preferencesstore "github.com/CaliLuke/go-plane/internal/projectmemberpreferences"
	intakeapi "github.com/CaliLuke/go-plane/projectintakewrites/gen/project_intake_writes"
	preferencesapi "github.com/CaliLuke/go-plane/projectmemberpreferences/gen/project_member_preferences"
)

type extensionPreferencesRepository struct {
	recorder *cycleMiddlewareRecorder
}

func (r extensionPreferencesRepository) Authorize(context.Context, string, string, string) error {
	r.recorder.add("authorization")
	if r.recorder.denied {
		return &preferencesstore.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	return nil
}
func (r extensionPreferencesRepository) Patch(_ context.Context, _, _, _, _ string, values map[string]any) (map[string]any, error) {
	r.recorder.add("dispatch")
	return map[string]any{"preferences": values}, nil
}

func TestProjectMemberPreferencesProductionMount(t *testing.T) {
	const path = "/api/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/preferences/member/22222222-2222-4222-8222-222222222222/"
	for _, tc := range []struct {
		name, method, body    string
		authenticated, denied bool
		status                int
	}{
		{name: "anonymous malformed patch", method: "PATCH", body: `{`, status: 401},
		{name: "denied malformed patch", method: "PATCH", body: `{`, authenticated: true, denied: true, status: 403},
		{name: "allowed malformed patch", method: "PATCH", body: `{`, authenticated: true, status: 400},
		{name: "array rejects without writing", method: "PATCH", body: `[1]`, authenticated: true, status: 400},
		{name: "null rejects without writing", method: "PATCH", body: `null`, authenticated: true, status: 400},
		{name: "saved preferences", method: "PATCH", body: `{"view":"list"}`, authenticated: true, status: 200},
		{name: "metadata skips method role", method: "OPTIONS", authenticated: true, denied: true, status: 200},
		{name: "unsupported method skips method role", method: "POST", authenticated: true, denied: true, status: 405},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &cycleMiddlewareRecorder{denied: tc.denied}
			sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
			service, err := systemapi.NewProjectMemberPreferences(extensionPreferencesRepository{recorder}, sessions)
			if err != nil {
				t.Fatal(err)
			}
			pending := pendingReplacementEndpoints{projectMemberPreferences: preferencesapi.NewEndpoints(service), projectMemberPreferencesMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}, activityReadsBodyLimit: 1 << 20}
			handler := extensionHTTPHandler(pending)
			request := httptest.NewRequest(tc.method, path, strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			if tc.authenticated {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if (tc.method == http.MethodPatch || tc.method == http.MethodOptions) && response.Header().Get("Allow") != "PATCH, OPTIONS" {
				t.Fatalf("write handler Allow = %q", response.Header().Get("Allow"))
			}
			if response.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s events=%v", response.Code, tc.status, response.Body.String(), recorder.events)
			}
			if tc.status != 200 {
				for _, event := range recorder.events {
					if event == "dispatch" {
						t.Fatalf("rejected request wrote preferences: %v", recorder.events)
					}
				}
			}
			if tc.status == 200 && tc.method == "PATCH" && !strings.Contains(response.Body.String(), `"view":"list"`) {
				t.Fatalf("saved preferences absent: %s", response.Body.String())
			}
		})
	}
}

type extensionIntakeRepository struct {
	calls, authorizations int
	denied                bool
}

func (r *extensionIntakeRepository) Authorize(context.Context, string, string, string) error {
	r.authorizations++
	if r.denied {
		return &intakeStore.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	return nil
}
func (r *extensionIntakeRepository) ValidateCreate(input djangoinput.Input) (map[string]any, error) {
	return (&intakeStore.Repository{}).ValidateCreate(input)
}
func (r *extensionIntakeRepository) Create(context.Context, string, string, string, map[string]any) (json.RawMessage, error) {
	r.calls++
	return json.RawMessage(`{"name":"New intake"}`), nil
}
func (r *extensionIntakeRepository) Patch(context.Context, string, string, string, string, map[string]any) (json.RawMessage, error) {
	r.calls++
	return json.RawMessage(`{"name":"Updated intake"}`), nil
}
func (r *extensionIntakeRepository) Delete(context.Context, string, string, string, string) error {
	r.calls++
	return nil
}

func TestProjectIntakeWritesProductionMounts(t *testing.T) {
	for _, collection := range []string{"inboxes", "intakes"} {
		for _, method := range []string{"POST", "PATCH", "DELETE"} {
			t.Run(collection+"/"+method, func(t *testing.T) {
				repository := &extensionIntakeRepository{}
				sessions := &cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}
				service, err := systemapi.NewProjectIntakeWrites(repository, sessions)
				if err != nil {
					t.Fatal(err)
				}
				pending := pendingReplacementEndpoints{projectIntakeWrites: intakeapi.NewEndpoints(service), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}, activityReadsBodyLimit: 1 << 20}
				handler := extensionHTTPHandler(pending)
				path := "/api/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/" + collection + "/"
				status := 201
				if method != "POST" {
					path += "22222222-2222-4222-8222-222222222222/"
					status = 200
				}
				if method == "DELETE" {
					status = 204
				}
				for _, authenticated := range []bool{false, true} {
					request := httptest.NewRequest(method, path, strings.NewReader(`{"name":"New intake"}`))
					request.Header.Set("Content-Type", "application/json")
					if authenticated {
						request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
					}
					response := httptest.NewRecorder()
					handler.ServeHTTP(response, request)
					want := 401
					if authenticated {
						want = status
					}
					if response.Code != want {
						t.Fatalf("authenticated=%v status=%d want=%d body=%s", authenticated, response.Code, want, response.Body.String())
					}
					if !authenticated && repository.calls != 0 {
						t.Fatal("anonymous request reached repository")
					}
				}
				if repository.calls != 1 {
					t.Fatalf("dispatches=%d want1", repository.calls)
				}
			})
		}
	}
}

func TestProjectIntakeSerializerValidationPrecedesCreatePermission(t *testing.T) {
	repository := &extensionIntakeRepository{denied: true}
	sessions := &cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}
	service, err := systemapi.NewProjectIntakeWrites(repository, sessions)
	if err != nil {
		t.Fatal(err)
	}
	handler := extensionHTTPHandler(pendingReplacementEndpoints{projectIntakeWrites: intakeapi.NewEndpoints(service), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}, activityReadsBodyLimit: 1 << 20})
	request := httptest.NewRequest("POST", "/api/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/intakes/", strings.NewReader(`{"name":""}`))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 400 || repository.authorizations != 0 || repository.calls != 0 {
		t.Fatalf("status=%d body=%s auth=%d writes=%d", response.Code, response.Body.String(), repository.authorizations, repository.calls)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if string(body["name"]) != `["This field may not be blank."]` {
		t.Fatalf("source field errors missing: %s", response.Body.String())
	}
}

func (r extensionPreferencesRepository) Lookup(context.Context, string, string, string) error {
	return nil
}
func TestProjectMemberPreferenceWritesRetainReadAndHead(t *testing.T) {
	recorder := &cycleMiddlewareRecorder{}
	sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
	service, err := systemapi.NewProjectMemberPreferences(extensionPreferencesRepository{recorder}, sessions)
	if err != nil {
		t.Fatal(err)
	}
	stub := projectMembershipReadsHTTPStub{}
	handler := extensionHTTPHandler(pendingReplacementEndpoints{projectMemberPreferences: preferencesapi.NewEndpoints(service), projectMemberPreferencesMetadata: service.(sessionResourceProtocolAuthorizer), projectMembershipReads: projectmembershipreadsapi.NewEndpoints(stub), projectMembershipMetadata: stub, readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}})
	for _, method := range []string{"GET", "HEAD"} {
		request := httptest.NewRequest(method, "/api/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/preferences/member/22222222-2222-4222-8222-222222222222/", nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != 200 {
			t.Fatalf("%s retained read status=%d body=%s", method, response.Code, response.Body.String())
		}
	}
}
