package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	activityapi "github.com/CaliLuke/go-plane/activityreads/gen/plane_internal_django_migration_api"
	system "github.com/CaliLuke/go-plane/gen/system"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	store "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	api "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/project_membership_lifecycle"
	"github.com/google/uuid"
)

const membershipHTTPProject = "11111111-1111-4111-8111-111111111111"

type membershipHTTPRepository struct {
	events *[]string
	denied bool
}

func (r *membershipHTTPRepository) AuthorizeLifecycle(_ context.Context, c store.LifecycleCommand) error {
	*r.events = append(*r.events, "authorization:"+c.Method)
	if r.denied {
		return &djangoapi.HTTPError{Status: 403, Body: map[string]any{"detail": "You do not have permission to perform this action."}}
	}
	return nil
}
func (r *membershipHTTPRepository) ExecuteLifecycle(_ context.Context, c store.LifecycleCommand) (any, error) {
	*r.events = append(*r.events, "dispatch:"+c.Method)
	return []any{}, nil
}

type membershipHTTPKeys struct {
	events *[]string
	zone   string
}

func (k membershipHTTPKeys) Validate(_ context.Context, key string) (apikey.Principal, error) {
	*k.events = append(*k.events, "authentication")
	if key != "valid" {
		return apikey.Principal{}, apikey.ErrInvalid
	}
	return apikey.Principal{UserID: uuid.MustParse(membershipHTTPProject), Timezone: k.zone}, nil
}

type membershipHTTPThrottle struct{ events *[]string }

func (t membershipHTTPThrottle) Allow(context.Context, string) (apikey.Rate, error) {
	*t.events = append(*t.events, "throttling")
	return apikey.Rate{Allowed: true}, nil
}

type membershipHTTPSessions struct{}

func (membershipHTTPSessions) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{}, errors.New("production session middleware should supply the principal")
}

func TestProjectMembershipLifecycleMiddlewareOrderAndEffects(t *testing.T) {
	for _, test := range []struct {
		name, method, suffix, zone, key string
		denied, preflight               bool
		status, audit                   int
		events                          []string
	}{
		{name: "request-size before authentication", method: "POST", suffix: "members/", zone: "UTC", key: "valid", status: 413},
		{name: "API read", method: "GET", suffix: "members/", zone: "UTC", key: "valid", status: 200, audit: 1, events: []string{"authentication", "authorization:GET", "throttling", "dispatch:GET"}},
		{name: "API write reaches generated handler", method: "POST", suffix: "members/", zone: "UTC", key: "valid", status: 201, audit: 1, events: []string{"authentication", "authorization:POST", "throttling", "dispatch:POST"}},
		{name: "permission precedes throttle and timezone", method: "POST", suffix: "members/", zone: "Not/AZone", key: "valid", denied: true, status: 403, audit: 1, events: []string{"authentication", "authorization:POST"}},
		{name: "timezone precedes dispatch", method: "GET", suffix: "members/", zone: "Not/AZone", key: "valid", status: 400, audit: 1, events: []string{"authentication", "authorization:GET", "throttling"}},
		{name: "HEAD permission precedes 405", method: "HEAD", suffix: "members/", zone: "UTC", key: "valid", denied: true, status: 403, audit: 1, events: []string{"authentication", "authorization:HEAD"}},
		{name: "OPTIONS authenticates before 405", method: "OPTIONS", suffix: "members/", zone: "UTC", key: "valid", status: 405, audit: 1, events: []string{"authentication", "authorization:OPTIONS", "throttling"}},
		{name: "anonymous OPTIONS", method: "OPTIONS", suffix: "members/", zone: "UTC", status: 401, events: []string{"authentication"}},
		{name: "preflight bypasses auth effects", method: "OPTIONS", suffix: "members/", zone: "Not/AZone", key: "valid", preflight: true, status: 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			var events []string
			repository := &membershipHTTPRepository{events: &events, denied: test.denied}
			service, err := systemapi.NewProjectMembershipLifecycle(repository, membershipHTTPKeys{&events, test.zone}, membershipHTTPThrottle{&events}, membershipHTTPSessions{})
			if err != nil {
				t.Fatal(err)
			}
			logger := &apiV1ActivityLoggerStub{}
			activity := &activityHTTPService{}
			handler := newHTTPHandlerWithPendingReplacements(
				newLogContext(context.Background(), io.Discard, false),
				system.NewEndpoints(systemapi.NewSystem()),
				false,
				nil, nil,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{membershipLifecycle: api.NewEndpoints(service), membershipLifecycleMetadata: service.(systemapi.MembershipLifecycleProtocolAuthorizer), activityReads: activityapi.NewEndpoints(activity), activityReadsMetadata: activity, activityReadsActivity: logger, activityReadsSecret: "secret", activityReadsBodyLimit: func() int64 {
					if strings.HasPrefix(test.name, "request-size") {
						return 1
					}
					return defaultFileSizeLimit
				}(), readServicesCORS: []string{"https://app.example.test"}})
			request := httptest.NewRequest(test.method, "/api/v1/workspaces/demo/projects/"+membershipHTTPProject+"/"+test.suffix, strings.NewReader(`{}`))
			request.Header.Set("Content-Type", "application/json")
			if test.key != "" {
				request.Header.Set("X-Api-Key", test.key)
			}
			if test.preflight {
				request.Header.Set("Origin", "https://app.example.test")
				request.Header.Set("Access-Control-Request-Method", "POST")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status || len(logger.records) != test.audit || !reflect.DeepEqual(events, test.events) {
				t.Fatalf("status=%d audit=%d events=%v body=%s; want status=%d audit=%d events=%v", response.Code, len(logger.records), events, response.Body.String(), test.status, test.audit, test.events)
			}
			if len(logger.records) > 0 && logger.records[0].ResponseCode != test.status {
				t.Fatalf("audit status=%d", logger.records[0].ResponseCode)
			}
		})
	}
}

func TestProjectMembershipLifecycleSessionOptionsRetainsSourceSerializerMetadata(t *testing.T) {
	policy, err := djangoRuntimeCORSPolicy(nil)
	if err != nil {
		t.Fatal(err)
	}
	stack := func(next http.Handler) http.Handler { return next }
	middleware := newSessionServiceProtocolMiddleware(notificationProtocolMetadata, sessionProtocolAuthorizerStub{allowed: true}, policy, stack, djangoSessionCookieConfig{}, false, apiV1CompatibilityOptions{})
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected generated dispatch for %s", r.URL.Path)
	}))
	for _, test := range []struct {
		suffix string
		post   bool
	}{{"members/", true}, {"members/leave/", true}, {"members/22222222-2222-4222-8222-222222222222/", false}} {
		request := httptest.NewRequest(http.MethodOptions, "/api/workspaces/demo/projects/"+membershipHTTPProject+"/"+test.suffix, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		var body struct {
			Name    string                    `json:"name"`
			Actions map[string]map[string]any `json:"actions"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != 200 || body.Name != "Project Member" {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
		post, present := body.Actions["POST"]
		if present != test.post {
			t.Fatalf("POST metadata=%v expected=%v body=%s", present, test.post, response.Body.String())
		}
		if present {
			member := post["member"].(map[string]any)
			children := member["children"].(map[string]any)
			if member["read_only"] != true || children["last_login_medium"] == nil || post["deleted_at"] == nil {
				t.Fatalf("incomplete source serializer metadata: %v", post)
			}
		}
	}
}

func TestProjectMembershipLifecycleRetainsExistingReadAndHeadOwnership(t *testing.T) {
	policy, err := djangoRuntimeCORSPolicy(nil)
	if err != nil {
		t.Fatal(err)
	}
	stack := func(next http.Handler) http.Handler { return next }
	// Denied session protocol makes accidental interception visible: retained
	// methods must reach their existing owner, which performs its own auth.
	middleware := newSessionServiceProtocolMiddleware(notificationProtocolMetadata, sessionProtocolAuthorizerStub{allowed: false}, policy, stack, djangoSessionCookieConfig{}, false, apiV1CompatibilityOptions{})
	for _, suffix := range []string{"members/", "members/22222222-2222-4222-8222-222222222222/"} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			called := false
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(200) }))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(method, "/api/workspaces/demo/projects/"+membershipHTTPProject+"/"+suffix, nil))
			if !called || response.Code != 200 {
				t.Fatalf("retained %s %s intercepted: status=%d", method, suffix, response.Code)
			}
		}
	}
	called := false
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(200) }))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodHead, "/api/workspaces/demo/projects/"+membershipHTTPProject+"/members/leave/", nil))
	if called || response.Code != 401 {
		t.Fatalf("leave HEAD bypassed source authentication: called=%v status=%d", called, response.Code)
	}
}

func TestProjectMembershipLifecycleRetainedReadsRedirectSlashlessPaths(t *testing.T) {
	policy, err := djangoRuntimeCORSPolicy(nil)
	if err != nil {
		t.Fatal(err)
	}
	stack := func(next http.Handler) http.Handler { return next }
	middleware := newSessionServiceProtocolMiddleware(notificationProtocolMetadata, sessionProtocolAuthorizerStub{allowed: false}, policy, stack, djangoSessionCookieConfig{}, false, apiV1CompatibilityOptions{})
	for _, suffix := range []string{"members", "members/22222222-2222-4222-8222-222222222222"} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			called := false
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(404) }))
			path := "/api/workspaces/demo/projects/" + membershipHTTPProject + "/" + suffix
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(method, path+"?search=retained", nil))
			if called || response.Code != 301 || response.Header().Get("Location") != path+"/?search=retained" {
				t.Fatalf("retained %s %s status=%d location=%q oldowner=%v body=%s", method, path, response.Code, response.Header().Get("Location"), called, response.Body.String())
			}
		}
	}
}
