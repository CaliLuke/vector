package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	systemapi "github.com/CaliLuke/go-plane"
	modulereadapi "github.com/CaliLuke/go-plane/gen/module_reads"
	"github.com/CaliLuke/go-plane/gen/system"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	modulereadsstore "github.com/CaliLuke/go-plane/internal/modulereads"
	modulelifecyclesvr "github.com/CaliLuke/go-plane/modulelifecycle/gen/http/module_lifecycle_api/server"
	moduleapi "github.com/CaliLuke/go-plane/modulelifecycle/gen/module_lifecycle_api"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

const moduleLifecycleTransportProject = "11111111-1111-4111-8111-111111111111"
const moduleLifecycleTransportModule = "22222222-2222-4222-8222-222222222222"

type moduleLifecycleTransportRepository struct {
	modulereadsstore.LifecycleRepository
	denyAppProject bool
	denyEntity     bool
	missingModule  bool
	missingLink    bool
	appAuthCalls   int
	createCalls    int
	updateCalls    int
	archiveCalls   []struct {
		id      string
		archive bool
	}
	linkErr error
}

func (repo *moduleLifecycleTransportRepository) AuthorizeEntityProject(context.Context, string, string, string) error {
	if repo.denyEntity {
		return modulereadsstore.ErrForbidden
	}
	return nil
}

func (repo *moduleLifecycleTransportRepository) AuthorizeAppProject(context.Context, string, string, string) error {
	repo.appAuthCalls++
	if repo.denyAppProject {
		return modulereadsstore.ErrForbidden
	}
	return nil
}

func (*moduleLifecycleTransportRepository) CheckModuleProject(context.Context, string, string, bool) error {
	return nil
}

func (*moduleLifecycleTransportRepository) CheckModule(context.Context, string, string, string, bool, bool) error {
	return nil
}

func (repo *moduleLifecycleTransportRepository) ModuleExists(context.Context, string, string, string) (bool, error) {
	return !repo.missingModule, nil
}

func (repo *moduleLifecycleTransportRepository) LinkExists(context.Context, string, string, string, string) (bool, error) {
	return !repo.missingLink, nil
}

func (repo *moduleLifecycleTransportRepository) CreateModule(context.Context, string, string, string, map[string]any, map[string][]string, string) (json.RawMessage, error) {
	repo.createCalls++
	return json.RawMessage(`{}`), nil
}

func (repo *moduleLifecycleTransportRepository) UpdateModule(context.Context, string, string, string, string, map[string]any, map[string][]string, bool, string) (json.RawMessage, error) {
	repo.updateCalls++
	return json.RawMessage(`{}`), nil
}

func (*moduleLifecycleTransportRepository) CreateModuleV1(context.Context, string, string, string, map[string]any, map[string][]string, string) (json.RawMessage, error) {
	return nil, errors.New("unexpected API-v1 module create in app transport test")
}

func (*moduleLifecycleTransportRepository) UpdateModuleV1(context.Context, string, string, string, string, map[string]any, map[string][]string, string) (json.RawMessage, error) {
	return nil, errors.New("unexpected API-v1 module update in app transport test")
}

func (repo *moduleLifecycleTransportRepository) SetModuleArchived(_ context.Context, _, _, _, moduleID string, archive bool) (time.Time, error) {
	repo.archiveCalls = append(repo.archiveCalls, struct {
		id      string
		archive bool
	}{moduleID, archive})
	return time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC), nil
}

func (*moduleLifecycleTransportRepository) DeleteModule(context.Context, string, string, string, string, bool, string) error {
	return nil
}

func (*moduleLifecycleTransportRepository) CreateLink(context.Context, string, string, string, string, map[string]any, map[string][]string) (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}

func (repo *moduleLifecycleTransportRepository) UpdateLink(context.Context, string, string, string, string, string, map[string]any, map[string][]string, bool) (json.RawMessage, error) {
	if repo.linkErr != nil {
		return nil, repo.linkErr
	}
	return json.RawMessage(`{}`), nil
}

func (*moduleLifecycleTransportRepository) DeleteLink(context.Context, string, string, string, string, string) error {
	return nil
}

func (*moduleLifecycleTransportRepository) CheckLink(context.Context, string, string, string, string, string) error {
	return nil
}

type moduleLifecycleTransportKeys struct{}

func (moduleLifecycleTransportKeys) Validate(_ context.Context, key string) (apikey.Principal, error) {
	if key != "module-key" {
		return apikey.Principal{}, apikey.ErrInvalid
	}
	return apikey.Principal{UserID: uuid.MustParse(assetsTransportUserID), Timezone: "UTC"}, nil
}

type moduleLifecycleTransportSessions struct{}

func (moduleLifecycleTransportSessions) Validate(_ context.Context, key string) (djangosession.Principal, error) {
	if key != "valid-session" {
		return djangosession.Principal{}, djangosession.ErrSessionNotFound
	}
	return djangosession.Principal{UserID: assetsTransportUserID, UserTimezone: "UTC"}, nil
}

type moduleLifecycleTransportThrottle struct{}

func (moduleLifecycleTransportThrottle) Allow(context.Context, string) (apikey.Rate, error) {
	return apikey.Rate{Allowed: true}, nil
}

func moduleLifecycleTransportHandler(t *testing.T, repo *moduleLifecycleTransportRepository, moduleReads *modulereadapi.Endpoints) http.Handler {
	t.Helper()
	service, err := systemapi.NewModuleLifecycle(repo, moduleLifecycleTransportSessions{}, moduleLifecycleTransportKeys{}, moduleLifecycleTransportThrottle{})
	if err != nil {
		t.Fatal(err)
	}
	publicSession := &assetsTransportSession{publicProjectMetadataThrottleStub: &publicProjectMetadataThrottleStub{
		allowed: true, userID: assetsTransportUserID, userTimezone: "UTC",
	}}
	return newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			moduleLifecycle:         moduleapi.NewEndpoints(service),
			moduleLifecycleMetadata: service,
			moduleReads:             moduleReads,
			readServicesSession:     publicSession,
			readServicesCookie:      djangoSessionCookieConfig{Name: "session-id", Configured: true},
			activityReadsBodyLimit:  1 << 20,
		},
	)
}

func TestModuleLifecycleGeneratedMountsMatchAllWriteRoutes(t *testing.T) {
	want := map[string]struct{}{
		"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/unarchive/":    {},
		"POST /api/v1/workspaces/{slug}/projects/{project_id}/modules/":                              {},
		"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/modules/{pk}/":                       {},
		"PATCH /api/v1/workspaces/{slug}/projects/{project_id}/modules/{pk}/":                        {},
		"POST /api/v1/workspaces/{slug}/projects/{project_id}/modules/{pk}/archive/":                 {},
		"DELETE /api/workspaces/{slug}/projects/{project_id}/archived-modules/":                      {},
		"POST /api/workspaces/{slug}/projects/{project_id}/archived-modules/":                        {},
		"DELETE /api/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/":                 {},
		"POST /api/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/":                   {},
		"POST /api/workspaces/{slug}/projects/{project_id}/modules/":                                 {},
		"DELETE /api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/archive/":           {},
		"POST /api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/archive/":             {},
		"POST /api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/":        {},
		"DELETE /api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/": {},
		"PATCH /api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/":  {},
		"PUT /api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/":    {},
		"DELETE /api/workspaces/{slug}/projects/{project_id}/modules/{pk}/":                          {},
		"PATCH /api/workspaces/{slug}/projects/{project_id}/modules/{pk}/":                           {},
		"PUT /api/workspaces/{slug}/projects/{project_id}/modules/{pk}/":                             {},
	}
	mux := loomhttp.NewMuxer()
	server := modulelifecyclesvr.New(&moduleapi.Endpoints{}, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
	server.Mount(mux)
	if len(server.Mounts) != len(want) {
		t.Fatalf("generated write mounts=%d, want %d", len(server.Mounts), len(want))
	}
	for _, mount := range server.Mounts {
		key := mount.Verb + " " + mount.Pattern
		if _, ok := want[key]; !ok {
			t.Errorf("unexpected generated module lifecycle mount %s (%s)", key, mount.Method)
			continue
		}
		delete(want, key)
	}
	for key := range want {
		t.Errorf("missing generated module lifecycle mount %s", key)
	}
}

func TestModuleLifecycleProductionMuxPreservesReadAndWriteMethodsOnCollection(t *testing.T) {
	reads := &moduleLifecycleTransportReads{}
	repo := &moduleLifecycleTransportRepository{}
	handler := moduleLifecycleTransportHandler(t, repo, modulereadapi.NewEndpoints(reads))
	path := "/api/workspaces/acme/projects/" + moduleLifecycleTransportProject + "/modules/"

	get := httptest.NewRequest(http.MethodGet, path, nil)
	get.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, get)
	if getResponse.Code != http.StatusOK || !strings.Contains(getResponse.Body.String(), `"source":"module-reads"`) || repo.appAuthCalls != 0 {
		t.Fatalf("GET status=%d body=%q app_auth_calls=%d; want existing read handler", getResponse.Code, getResponse.Body.String(), repo.appAuthCalls)
	}

	post := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"name":"created"}`))
	post.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	postResponse := httptest.NewRecorder()
	handler.ServeHTTP(postResponse, post)
	if postResponse.Code != http.StatusCreated || repo.appAuthCalls != 1 || reads.listCalls != 1 {
		t.Fatalf("POST status=%d app_auth_calls=%d read_calls=%d body=%q", postResponse.Code, repo.appAuthCalls, reads.listCalls, postResponse.Body.String())
	}
}

func TestModuleLifecycleAppSlashlessReadMethodsRedirectAcrossLifecyclePaths(t *testing.T) {
	base := "/api/workspaces/acme/projects/" + moduleLifecycleTransportProject
	module := moduleLifecycleTransportModule
	paths := []struct {
		name string
		path string
	}{
		{name: "module collection", path: base + "/modules/"},
		{name: "module detail", path: base + "/modules/" + module + "/"},
		{name: "module link collection", path: base + "/modules/" + module + "/module-links/"},
		{name: "module link detail", path: base + "/modules/" + module + "/module-links/" + module + "/"},
		{name: "module archive action", path: base + "/modules/" + module + "/archive/"},
		{name: "archived module collection", path: base + "/archived-modules/"},
		{name: "archived module detail", path: base + "/archived-modules/" + module + "/"},
	}
	handler := moduleLifecycleTransportHandler(t, &moduleLifecycleTransportRepository{}, nil)
	for _, path := range paths {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(path.name+"/"+method, func(t *testing.T) {
				request := httptest.NewRequest(method, strings.TrimSuffix(path.path, "/"), nil)
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)

				if response.Code != http.StatusMovedPermanently {
					t.Fatalf("status=%d body=%q; want slashless 301", response.Code, response.Body.String())
				}
				if response.Header().Get("Location") != path.path {
					t.Errorf("Location=%q, want %q", response.Header().Get("Location"), path.path)
				}
				if !strings.Contains(response.Header().Get("Vary"), "Cookie") || !strings.Contains(response.Header().Get("Vary"), "origin") {
					t.Errorf("Vary=%q, want Cookie and origin", response.Header().Get("Vary"))
				}
				if response.Header().Get("Content-Type") != "text/html; charset=utf-8" {
					t.Errorf("Content-Type=%q, want Django redirect HTML", response.Header().Get("Content-Type"))
				}
				if method == http.MethodHead && response.Body.Len() != 0 {
					t.Errorf("HEAD body=%q; want empty body", response.Body.String())
				}
			})
		}
	}
}

func TestModuleLinkLifecycleProductionMuxSelectsMethodSpecificProtocol(t *testing.T) {
	reads := &moduleLifecycleTransportReads{}
	repo := &moduleLifecycleTransportRepository{}
	handler := moduleLifecycleTransportHandler(t, repo, modulereadapi.NewEndpoints(reads))
	collection := "/api/workspaces/acme/projects/" + moduleLifecycleTransportProject + "/modules/" + moduleLifecycleTransportModule + "/module-links/"
	detail := collection + moduleLifecycleTransportModule + "/"
	request := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}

	for _, test := range []struct {
		method string
		path   string
		allow  string
	}{
		{method: http.MethodPost, path: collection, allow: "POST, OPTIONS"},
		{method: http.MethodPut, path: detail, allow: "PUT, PATCH, DELETE, OPTIONS"},
		{method: http.MethodPatch, path: detail, allow: "PUT, PATCH, DELETE, OPTIONS"},
		{method: http.MethodDelete, path: detail, allow: "PUT, PATCH, DELETE, OPTIONS"},
	} {
		body := `{"title":"ref","url":"https://example.test"}`
		response := request(test.method, test.path, body)
		wantStatus := http.StatusOK
		if test.method == http.MethodPost {
			wantStatus = http.StatusCreated
		} else if test.method == http.MethodDelete {
			wantStatus = http.StatusNoContent
		}
		if response.Code != wantStatus || response.Header().Get("Allow") != test.allow {
			t.Errorf("%s %s status=%d Allow=%q body=%q; want status %d and Allow %q", test.method, test.path, response.Code, response.Header().Get("Allow"), response.Body.String(), wantStatus, test.allow)
		}
	}

	for _, test := range []struct {
		method string
		path   string
		allow  string
	}{
		{method: http.MethodPut, path: collection, allow: "POST, OPTIONS"},
		{method: http.MethodPost, path: detail, allow: "PUT, PATCH, DELETE, OPTIONS"},
	} {
		response := request(test.method, test.path, `{}`)
		if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != test.allow {
			t.Errorf("unsupported %s %s status=%d Allow=%q body=%q; want 405 and Allow %q", test.method, test.path, response.Code, response.Header().Get("Allow"), response.Body.String(), test.allow)
		}
	}

	for _, test := range []struct {
		path       string
		allow      string
		wantSource string
	}{
		{path: collection, allow: "GET, POST, HEAD, OPTIONS", wantSource: `"source":"module-read-links"`},
		{path: detail, allow: "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", wantSource: `"source":"module-read-link-detail"`},
	} {
		getResponse := request(http.MethodGet, test.path, "")
		if getResponse.Code != http.StatusOK || getResponse.Header().Get("Allow") != test.allow || !strings.Contains(getResponse.Body.String(), test.wantSource) {
			t.Errorf("GET %s status=%d Allow=%q body=%q; want retained read response with baseline Allow %q", test.path, getResponse.Code, getResponse.Header().Get("Allow"), getResponse.Body.String(), test.allow)
		}
		headResponse := request(http.MethodHead, test.path, "")
		if headResponse.Code != http.StatusOK || headResponse.Header().Get("Allow") != test.allow || headResponse.Body.Len() != 0 {
			t.Errorf("HEAD %s status=%d Allow=%q body=%q; want retained HEAD 200 with baseline Allow %q and empty body", test.path, headResponse.Code, headResponse.Header().Get("Allow"), headResponse.Body.String(), test.allow)
		}
	}
	if reads.linksCalls != 2 || reads.linkDetailCalls != 2 {
		t.Errorf("retained read calls: links=%d link_detail=%d; want GET and HEAD on both routes", reads.linksCalls, reads.linkDetailCalls)
	}
}

func TestModuleLifecycleProductionMuxAuthenticatesBeforeParsingAndMapsParserErrors(t *testing.T) {
	path := "/api/workspaces/acme/projects/" + moduleLifecycleTransportProject + "/modules/"
	t.Run("anonymous request is denied before repository authorization", func(t *testing.T) {
		repo := &moduleLifecycleTransportRepository{}
		handler := moduleLifecycleTransportHandler(t, repo, nil)
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{"))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized || repo.appAuthCalls != 0 || repo.createCalls != 0 {
			t.Fatalf("anonymous POST status=%d auth_calls=%d create_calls=%d body=%q", response.Code, repo.appAuthCalls, repo.createCalls, response.Body.String())
		}
	})

	t.Run("invalid JSON preserves parser status and body", func(t *testing.T) {
		repo := &moduleLifecycleTransportRepository{}
		handler := moduleLifecycleTransportHandler(t, repo, nil)
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{"))
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || repo.appAuthCalls != 1 || repo.createCalls != 0 {
			t.Fatalf("invalid JSON POST status=%d auth_calls=%d create_calls=%d body=%q", response.Code, repo.appAuthCalls, repo.createCalls, response.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["detail"] == nil {
			t.Fatalf("invalid JSON response body=%q; want a JSON parser detail", response.Body.String())
		}
	})
}

func TestModuleLifecycleProductionMuxMapsErrorsAndPreservesCorrectedPUTPrecedence(t *testing.T) {
	t.Run("PUT permission precedes invalid body", func(t *testing.T) {
		repo := &moduleLifecycleTransportRepository{denyAppProject: true}
		handler := moduleLifecycleTransportHandler(t, repo, nil)
		path := "/api/workspaces/acme/projects/" + moduleLifecycleTransportProject + "/modules/" + moduleLifecycleTransportModule + "/"
		request := httptest.NewRequest(http.MethodPut, path, strings.NewReader("{"))
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden || repo.appAuthCalls != 1 || repo.updateCalls != 0 || !strings.Contains(response.Body.String(), `"error":"You don't have the required permissions."`) {
			t.Fatalf("PUT status=%d auth_calls=%d update_calls=%d body=%q", response.Code, repo.appAuthCalls, repo.updateCalls, response.Body.String())
		}
	})

	t.Run("link validation status and body survive the service boundary", func(t *testing.T) {
		repo := &moduleLifecycleTransportRepository{linkErr: &modulereadsstore.ModuleValidationError{Body: map[string]any{"url": []string{"Invalid URL format."}}}}
		handler := moduleLifecycleTransportHandler(t, repo, nil)
		path := "/api/workspaces/acme/projects/" + moduleLifecycleTransportProject + "/modules/" + moduleLifecycleTransportModule + "/module-links/" + moduleLifecycleTransportModule + "/"
		request := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(`{"url":"https://example.test"}`))
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"url":["Invalid URL format."]`) {
			t.Fatalf("PATCH link status=%d body=%q; want preserved validation response", response.Code, response.Body.String())
		}
	})
}

func TestModuleLifecycleArchivedItemAliasesUseSourceActionsAndPK(t *testing.T) {
	repo := &moduleLifecycleTransportRepository{}
	handler := moduleLifecycleTransportHandler(t, repo, nil)
	path := "/api/workspaces/acme/projects/" + moduleLifecycleTransportProject + "/archived-modules/" + moduleLifecycleTransportModule + "/"
	for _, test := range []struct {
		method  string
		archive bool
	}{
		{method: http.MethodPost, archive: true},
		{method: http.MethodDelete, archive: false},
	} {
		request := httptest.NewRequest(test.method, path, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK && !(test.method == http.MethodDelete && response.Code == http.StatusNoContent) {
			t.Fatalf("%s status=%d body=%q", test.method, response.Code, response.Body.String())
		}
		if got := response.Header().Get("Vary"); got != "Cookie, origin" {
			t.Fatalf("%s Vary=%q, want no compression variation for short response", test.method, got)
		}
	}
	if len(repo.archiveCalls) != 2 || repo.archiveCalls[0].id != moduleLifecycleTransportModule || !repo.archiveCalls[0].archive || repo.archiveCalls[1].id != moduleLifecycleTransportModule || repo.archiveCalls[1].archive {
		t.Fatalf("archive calls=%#v; want POST archive and DELETE unarchive of pk", repo.archiveCalls)
	}
}

func TestModuleLifecycleCompressionMiddlewareAddsVaryAtDjangoThreshold(t *testing.T) {
	for _, test := range []struct {
		name         string
		body         string
		wantEncoding bool
	}{
		{name: "empty", body: "", wantEncoding: false},
		{name: "below threshold", body: strings.Repeat("x", 199), wantEncoding: false},
		{name: "at threshold", body: strings.Repeat("x", 200), wantEncoding: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := readServicesGzipMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				mergeVary(response.Header(), "Cookie", "origin")
				_, _ = response.Write([]byte(test.body))
			}))
			request := httptest.NewRequest(http.MethodGet, "/api/workspaces/acme/projects/"+moduleLifecycleTransportProject+"/modules/", nil)
			request.Header.Set("Accept-Encoding", "gzip")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			got := strings.Contains(response.Header().Get("Vary"), "Accept-Encoding")
			if got != test.wantEncoding {
				t.Fatalf("Vary=%q, want Accept-Encoding present=%t", response.Header().Get("Vary"), test.wantEncoding)
			}
		})
	}
}

func TestModuleLifecycleAPIAliasErrorsRetainDjangoProtocolHeaders(t *testing.T) {
	handler := moduleLifecycleTransportHandler(t, &moduleLifecycleTransportRepository{}, nil)
	path := "/api/v1/workspaces/acme/projects/" + moduleLifecycleTransportProject + "/modules/" + moduleLifecycleTransportModule + "/archive/"
	request := httptest.NewRequest(http.MethodPost, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%q, want unauthenticated API-key response", response.Code, response.Body.String())
	}
	for name, want := range map[string]string{
		"Allow": "POST", "Cross-Origin-Opener-Policy": "same-origin", "Referrer-Policy": "same-origin",
		"Vary": "origin", "X-Content-Type-Options": "nosniff", "X-Frame-Options": "DENY",
	} {
		if got := response.Header().Get(name); got != want {
			t.Errorf("%s=%q, want %q", name, got, want)
		}
	}
}

func TestModuleLifecycleOPTIONSMetadataUsesSourceAllowAndVariants(t *testing.T) {
	tests := []struct {
		path        string
		wantAllow   string
		wantName    string
		wantActions bool
		denyEntity  bool
		missing     bool
		module      bool
	}{
		{
			path:        "/api/workspaces/acme/projects/" + moduleLifecycleTransportProject + "/modules/",
			wantAllow:   "POST, OPTIONS",
			wantName:    "Module",
			wantActions: true,
		},
		{
			path:      "/api/workspaces/acme/projects/" + moduleLifecycleTransportProject + "/modules/" + moduleLifecycleTransportModule + "/",
			wantAllow: "PUT, PATCH, DELETE, OPTIONS",
			wantName:  "Module",
			missing:   true,
			module:    true,
		},
		{
			path:        "/api/workspaces/acme/projects/" + moduleLifecycleTransportProject + "/modules/" + moduleLifecycleTransportModule + "/module-links/",
			wantAllow:   "POST, OPTIONS",
			wantName:    "Module Link",
			wantActions: true,
		},
		{
			path:       "/api/workspaces/acme/projects/" + moduleLifecycleTransportProject + "/modules/" + moduleLifecycleTransportModule + "/module-links/",
			wantAllow:  "POST, OPTIONS",
			wantName:   "Module Link",
			denyEntity: true,
		},
		{
			path:      "/api/workspaces/acme/projects/" + moduleLifecycleTransportProject + "/modules/" + moduleLifecycleTransportModule + "/module-links/" + moduleLifecycleTransportModule + "/",
			wantAllow: "PUT, PATCH, DELETE, OPTIONS",
			wantName:  "Module Link",
			missing:   true,
		},
		{
			path:      "/api/workspaces/acme/projects/" + moduleLifecycleTransportProject + "/archived-modules/",
			wantAllow: "POST, DELETE, OPTIONS",
			wantName:  "Module Archive Unarchive Endpoint",
		},
	}
	for _, test := range tests {
		t.Run(test.wantName+test.path, func(t *testing.T) {
			handler := moduleLifecycleTransportHandler(t, &moduleLifecycleTransportRepository{denyEntity: test.denyEntity, missingModule: test.missing && test.module, missingLink: test.missing && !test.module}, nil)
			request := httptest.NewRequest(http.MethodOptions, test.path, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK || response.Header().Get("Allow") != test.wantAllow {
				t.Fatalf("status=%d Allow=%q body=%q", response.Code, response.Header().Get("Allow"), response.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode metadata: %v body=%q", err, response.Body.String())
			}
			if body["name"] != test.wantName {
				t.Fatalf("metadata name=%v want %q", body["name"], test.wantName)
			}
			_, hasActions := body["actions"]
			if hasActions != test.wantActions {
				t.Fatalf("metadata actions present=%t want %t", hasActions, test.wantActions)
			}
		})
	}
}

type moduleLifecycleTransportReads struct {
	modulereadapi.Service
	listCalls       int
	linksCalls      int
	linkDetailCalls int
}

func (*moduleLifecycleTransportReads) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, nil
}

func (service *moduleLifecycleTransportReads) List(context.Context, *modulereadapi.ListPayload) (*modulereadapi.RawJSONResponseMetadata, io.ReadCloser, error) {
	service.listCalls++
	return &modulereadapi.RawJSONResponseMetadata{ContentType: "application/json"}, io.NopCloser(strings.NewReader(`{"source":"module-reads"}`)), nil
}

func (service *moduleLifecycleTransportReads) Links(context.Context, *modulereadapi.LinksPayload) (*modulereadapi.RawJSONResponseMetadata, io.ReadCloser, error) {
	service.linksCalls++
	return &modulereadapi.RawJSONResponseMetadata{ContentType: "application/json"}, io.NopCloser(strings.NewReader(`{"source":"module-read-links"}`)), nil
}

func (service *moduleLifecycleTransportReads) LinkDetail(context.Context, *modulereadapi.LinkDetailPayload) (*modulereadapi.RawJSONResponseMetadata, io.ReadCloser, error) {
	service.linkDetailCalls++
	return &modulereadapi.RawJSONResponseMetadata{ContentType: "application/json"}, io.NopCloser(strings.NewReader(`{"source":"module-read-link-detail"}`)), nil
}

// Keep imports tied to their intended generated authorization contract.
var _ = moduleapi.MethodNames
