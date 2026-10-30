package main

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/projectlabelreads"
	labelssvr "github.com/CaliLuke/go-plane/project_label_writes/gen/http/project_label_writes/server"
	labelsapi "github.com/CaliLuke/go-plane/project_label_writes/gen/project_label_writes"
	loomhttp "github.com/CaliLuke/loom/http"
)

const labelWriteHTTPProject = "11111111-1111-4111-8111-111111111111"
const labelWriteHTTPID = "22222222-2222-4222-8222-222222222222"

type labelWriteHTTPRepository struct {
	events         *[]string
	baseError      error
	methodError    error
	label          *store.Label
	authorizations []string
	dispatches     int
}

func (r *labelWriteHTTPRepository) Authorize(_ context.Context, user, slug, project, method string) error {
	r.authorizations = append(r.authorizations, method)
	*r.events = append(*r.events, "authorize:"+method)
	if len(r.authorizations) == 1 {
		return r.baseError
	}
	return r.methodError
}

func (*labelWriteHTTPRepository) Load(context.Context, string, string, string, string) (*store.Label, error) {
	panic("unexpected Load")
}

func (r *labelWriteHTTPRepository) Write(_ context.Context, user, slug, project, id, method string, body io.Reader, contentType string) (*store.Label, error) {
	r.dispatches++
	*r.events = append(*r.events, "dispatch:"+method)
	return r.label, nil
}

type labelWriteHTTPCache struct {
	events *[]string
	calls  int
	all    []bool
}

func (c *labelWriteHTTPCache) Invalidate(_ context.Context, slug string, all bool) error {
	c.calls++
	c.all = append(c.all, all)
	*c.events = append(*c.events, "cache")
	return nil
}

type labelWriteHTTPSessionValidator struct{}

func (labelWriteHTTPSessionValidator) Validate(context.Context, string) (djangosession.Principal, error) {
	panic("session middleware must provide the inspected principal")
}

func TestProjectLabelWritesMiddlewareOrderAndEffects(t *testing.T) {
	for _, scenario := range []struct {
		name, method, user, timezone, body string
		bodyLimit                          int64
		baseDenied, methodDenied           bool
		status                             int
		inspections, cache, dispatches     int
		wantAuth                           []string
		wantEvents                         []string
		wantAudit                          int
	}{
		{"request-size before session", http.MethodPost, "project-admin", "UTC", "oversized", 1, false, false, 413, 0, 0, 0, nil, nil, 0},
		{"session before permission-classes", http.MethodPost, "", "Not/AZone", `{}`, defaultFileSizeLimit, false, false, 401, 1, 0, 0, nil, nil, 1},
		{"permission-classes before timezone", http.MethodPost, "project-admin", "Not/AZone", `{}`, defaultFileSizeLimit, true, false, 403, 1, 0, 0, []string{http.MethodPost}, []string{"authorize:POST"}, 1},
		{"timezone before cache", http.MethodPost, "project-admin", "Not/AZone", `{}`, defaultFileSizeLimit, false, false, 400, 1, 0, 0, []string{http.MethodPost}, []string{"authorize:POST"}, 1},
		{"method permission after cache", http.MethodPost, "project-admin", "UTC", `{}`, defaultFileSizeLimit, false, true, 403, 1, 1, 0, []string{http.MethodPost, http.MethodDelete}, []string{"authorize:POST", "cache", "authorize:DELETE"}, 1},
		{"project admin without workspace membership", http.MethodPost, "project-admin", "UTC", `{}`, defaultFileSizeLimit, false, false, 201, 1, 1, 1, []string{http.MethodPost, http.MethodDelete}, []string{"authorize:POST", "cache", "authorize:DELETE", "dispatch:POST"}, 1},
		{"PUT dispatch without cache invalidation", http.MethodPut, "project-admin", "UTC", `{}`, defaultFileSizeLimit, false, false, 200, 1, 0, 1, []string{http.MethodPut}, []string{"authorize:PUT", "dispatch:PUT"}, 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			var events []string
			project := labelWriteHTTPProject
			repository := &labelWriteHTTPRepository{events: &events, label: &store.Label{ID: labelWriteHTTPID, Name: "Alpha", Color: "#111111", ProjectID: &project, WorkspaceID: "workspace", SortOrder: 10}}
			if scenario.baseDenied {
				repository.baseError = store.Failure(403, "detail", "You do not have permission to perform this action.")
			}
			if scenario.methodDenied {
				repository.methodError = store.Failure(403, "detail", "You do not have permission to perform this action.")
			}
			cache := &labelWriteHTTPCache{events: &events}
			sessions := &publicProjectMetadataThrottleStub{userID: scenario.user, userTimezone: scenario.timezone}
			logger := &apiV1ActivityLoggerStub{beforeLog: func() {
				if !reflect.DeepEqual(events, scenario.wantEvents) {
					t.Errorf("audit ran before expected sequence: %v", events)
				}
			}}
			mux := newProjectLabelWriteTestMux(t, repository, cache, sessions, logger, scenario.bodyLimit)
			path := "/api/workspaces/demo/projects/" + labelWriteHTTPProject + "/issue-labels/"
			if scenario.method == http.MethodPut {
				path += labelWriteHTTPID + "/"
			}
			request := httptest.NewRequest(scenario.method, path, strings.NewReader(scenario.body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Api-Key", "audit-only")
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
			response := httptest.NewRecorder()

			mux.ServeHTTP(response, request)

			if response.Code != scenario.status || sessions.calls != scenario.inspections || cache.calls != scenario.cache || repository.dispatches != scenario.dispatches || len(logger.records) != scenario.wantAudit {
				t.Fatalf("status=%d inspections=%d cache=%d dispatch=%d audit=%d body=%s", response.Code, sessions.calls, cache.calls, repository.dispatches, len(logger.records), response.Body.String())
			}
			if !reflect.DeepEqual(repository.authorizations, scenario.wantAuth) || !reflect.DeepEqual(events, scenario.wantEvents) {
				t.Fatalf("auth=%v events=%v, want auth=%v events=%v", repository.authorizations, events, scenario.wantAuth, scenario.wantEvents)
			}
			if scenario.cache == 1 && !reflect.DeepEqual(cache.all, []bool{true}) {
				t.Fatalf("cache scope=%v", cache.all)
			}
			if scenario.wantAudit == 1 && logger.records[0].ResponseCode != scenario.status {
				t.Fatalf("audit status=%d", logger.records[0].ResponseCode)
			}
		})
	}
}

func TestProjectLabelWritesUnexpectedNonfiniteResultUsesJSONError(t *testing.T) {
	for _, test := range []struct {
		name  string
		order float64
	}{
		{"NaN", math.NaN()}, {"positive infinity", math.Inf(1)}, {"negative infinity", math.Inf(-1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			events := []string{}
			project := labelWriteHTTPProject
			repository := &labelWriteHTTPRepository{events: &events, label: &store.Label{ID: labelWriteHTTPID, Name: "Alpha", ProjectID: &project, WorkspaceID: "workspace", SortOrder: test.order}}
			cache := &labelWriteHTTPCache{events: &events}
			sessions := &publicProjectMetadataThrottleStub{userID: "project-admin", userTimezone: "UTC"}
			logger := &apiV1ActivityLoggerStub{}
			mux := newProjectLabelWriteTestMux(t, repository, cache, sessions, logger, defaultFileSizeLimit)
			request := httptest.NewRequest(http.MethodPost, "/api/workspaces/demo/projects/"+labelWriteHTTPProject+"/issue-labels/", strings.NewReader(`{"name":"Alpha"}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Api-Key", "audit-only")
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
			response := httptest.NewRecorder()

			mux.ServeHTTP(response, request)

			var result map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatalf("error response must be JSON: %s: %v", response.Body.String(), err)
			}
			if response.Code != 500 || !strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") || !reflect.DeepEqual(result, map[string]any{"error": "Something went wrong please try again later"}) {
				t.Fatalf("status=%d content-type=%s body=%s", response.Code, response.Header().Get("Content-Type"), response.Body.String())
			}
			if !reflect.DeepEqual(events, []string{"authorize:POST", "cache", "authorize:DELETE", "dispatch:POST"}) || len(logger.records) != 1 || logger.records[0].ResponseCode != 500 {
				t.Fatalf("events=%v audit=%v", events, logger.records)
			}
		})
	}
}

func newProjectLabelWriteTestMux(t *testing.T, repository *labelWriteHTTPRepository, cache *labelWriteHTTPCache, sessions *publicProjectMetadataThrottleStub, logger *apiV1ActivityLoggerStub, limit int64) http.Handler {
	t.Helper()
	service, err := systemapi.NewProjectLabelWrites(repository, labelWriteHTTPSessionValidator{}, cache)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := djangoRuntimeCORSPolicy(nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := loomhttp.NewMuxer()
	server := labelssvr.New(labelsapi.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, policy)
	server.Use(sessionServiceMiddlewareStack(apiV1CompatibilityOptions{bodySizeLimit: limit, activityLogger: logger, activitySecret: "secret"}, sessions, djangoSessionCookieConfig{}))
	labelssvr.Mount(mux, server)
	return mux
}
