package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/gen/system"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	modulelifecycleclient "github.com/CaliLuke/go-plane/modulelifecycle/gen/http/module_lifecycle_api/client"
	moduleapi "github.com/CaliLuke/go-plane/modulelifecycle/gen/module_lifecycle_api"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/google/uuid"
)

const moduleLifecycleSecurityActor = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"

type moduleLifecycleSecurityRepository struct {
	moduleLifecycleTransportRepository
	authorizeCalls int
	authorizedUser string
	authorizedSlug string
	authorizedProj string
	createCalls    int
	createdUser    string
	createdSlug    string
	createdProj    string
	linkCalls      int
}

func (repo *moduleLifecycleSecurityRepository) AuthorizeEntityProject(_ context.Context, user, slug, project string) error {
	repo.authorizeCalls++
	repo.authorizedUser, repo.authorizedSlug, repo.authorizedProj = user, slug, project
	return nil
}

func (repo *moduleLifecycleSecurityRepository) CreateModuleV1(_ context.Context, user, slug, project string, _ map[string]any, _ map[string][]string, _ string) (json.RawMessage, error) {
	repo.createCalls++
	repo.createdUser, repo.createdSlug, repo.createdProj = user, slug, project
	return json.RawMessage(`{"id":"33333333-3333-4333-8333-333333333333"}`), nil
}

func (repo *moduleLifecycleSecurityRepository) CreateLink(context.Context, string, string, string, string, map[string]any, map[string][]string) (json.RawMessage, error) {
	repo.linkCalls++
	return json.RawMessage(`{"created_at":"2025-01-02T03:04:05Z","created_by":null,"deleted_at":null,"id":"44444444-4444-4444-8444-444444444444","metadata":{},"module":"22222222-2222-4222-8222-222222222222","project":"11111111-1111-4111-8111-111111111111","title":"Brief","updated_at":"2025-01-02T03:04:05Z","updated_by":null,"url":"https://example.test/brief","workspace":"55555555-5555-4555-8555-555555555555"}`), nil
}

type moduleLifecycleSecurityKeys struct {
	calls     int
	keys      []string
	principal apikey.Principal
}

func (keys *moduleLifecycleSecurityKeys) Validate(_ context.Context, key string) (apikey.Principal, error) {
	keys.calls++
	keys.keys = append(keys.keys, key)
	if key != "valid-module-key" {
		return apikey.Principal{}, apikey.ErrInvalid
	}
	return keys.principal, nil
}

type moduleLifecycleSecurityThrottle struct {
	calls int
	keys  []string
}

func (throttle *moduleLifecycleSecurityThrottle) Allow(_ context.Context, key string) (apikey.Rate, error) {
	throttle.calls++
	throttle.keys = append(throttle.keys, key)
	return apikey.Rate{Allowed: true, Remaining: "9", Reset: 1}, nil
}

func moduleLifecycleSecurityHandler(t *testing.T, repo *moduleLifecycleSecurityRepository, keys *moduleLifecycleSecurityKeys, throttle *moduleLifecycleSecurityThrottle) http.Handler {
	t.Helper()
	service, err := systemapi.NewModuleLifecycle(repo, moduleLifecycleTransportSessions{}, keys, throttle)
	if err != nil {
		t.Fatal(err)
	}
	return newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			moduleLifecycle:         moduleapi.NewEndpoints(service),
			moduleLifecycleMetadata: service,
			readServicesSession:     &assetsTransportSession{publicProjectMetadataThrottleStub: &publicProjectMetadataThrottleStub{allowed: true, userID: assetsTransportUserID, userTimezone: "UTC"}},
			readServicesCookie:      djangoSessionCookieConfig{Name: "session-id", Configured: true},
			activityReadsBodyLimit:  1 << 20,
		},
	)
}

func TestModuleLifecycleAPIKeySecurityAuthenticatesAndCarriesPrincipalThroughThrottleAndAuthorization(t *testing.T) {
	for _, test := range []struct {
		name       string
		key        string
		wantStatus int
	}{
		{name: "valid key reaches protected operation", key: "valid-module-key", wantStatus: http.StatusCreated},
		{name: "invalid key keeps source forbidden response", key: "invalid-module-key", wantStatus: http.StatusForbidden},
		{name: "missing key keeps source unauthenticated response", wantStatus: http.StatusUnauthorized},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &moduleLifecycleSecurityRepository{}
			keys := &moduleLifecycleSecurityKeys{principal: apikey.Principal{UserID: uuid.MustParse(moduleLifecycleSecurityActor), Timezone: "UTC"}}
			throttle := &moduleLifecycleSecurityThrottle{}
			handler := moduleLifecycleSecurityHandler(t, repo, keys, throttle)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/acme/projects/"+moduleLifecycleTransportProject+"/modules/", strings.NewReader(`{}`))
			if test.key != "" {
				request.Header.Set("X-Api-Key", test.key)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status=%d body=%q, want %d", response.Code, response.Body.String(), test.wantStatus)
			}
			if keys.calls != 1 || len(keys.keys) != 1 || keys.keys[0] != test.key {
				t.Fatalf("API-key validation calls=%d keys=%v, want one validation of %q", keys.calls, keys.keys, test.key)
			}
			if test.wantStatus != http.StatusCreated {
				if repo.authorizeCalls != 0 || repo.createCalls != 0 || throttle.calls != 0 {
					t.Fatalf("denied request crossed boundary: authorization=%d create=%d throttle=%d", repo.authorizeCalls, repo.createCalls, throttle.calls)
				}
				return
			}
			if repo.authorizeCalls != 1 || repo.authorizedUser != moduleLifecycleSecurityActor || repo.authorizedSlug != "acme" || repo.authorizedProj != moduleLifecycleTransportProject {
				t.Fatalf("authorization calls=%d actor=%q slug=%q project=%q", repo.authorizeCalls, repo.authorizedUser, repo.authorizedSlug, repo.authorizedProj)
			}
			if throttle.calls != 1 || len(throttle.keys) != 1 || throttle.keys[0] != test.key {
				t.Fatalf("throttle calls=%d keys=%v, want one call for credential", throttle.calls, throttle.keys)
			}
			if repo.createCalls != 1 || repo.createdUser != moduleLifecycleSecurityActor || repo.createdSlug != "acme" || repo.createdProj != moduleLifecycleTransportProject {
				t.Fatalf("create calls=%d actor=%q slug=%q project=%q", repo.createCalls, repo.createdUser, repo.createdSlug, repo.createdProj)
			}
		})
	}
}

func TestModuleLifecycleLinkCreateReturnsLocationAndJSONURL(t *testing.T) {
	repo := &moduleLifecycleSecurityRepository{}
	keys := &moduleLifecycleSecurityKeys{principal: apikey.Principal{UserID: uuid.MustParse(moduleLifecycleSecurityActor), Timezone: "UTC"}}
	throttle := &moduleLifecycleSecurityThrottle{}
	handler := moduleLifecycleSecurityHandler(t, repo, keys, throttle)
	path := "/api/workspaces/acme/projects/" + moduleLifecycleTransportProject + "/modules/" + moduleLifecycleTransportModule + "/module-links/"
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"title":"Brief","url":"https://example.test/brief"}`))
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%q, want %d", response.Code, response.Body.String(), http.StatusCreated)
	}
	wantURL := "https://example.test/brief"
	if got := response.Header().Get("Location"); got != wantURL {
		t.Errorf("Location=%q, want %q", got, wantURL)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v; body=%q", err, response.Body.String())
	}
	if got, _ := body["url"].(string); got != wantURL {
		t.Errorf("JSON url=%q, want %q; body=%q", got, wantURL, response.Body.String())
	}
	if repo.linkCalls != 1 {
		t.Errorf("link create calls=%d, want one", repo.linkCalls)
	}
}

func TestModuleLifecycleGeneratedServerClientArchiveSessionRoundTrip(t *testing.T) {
	repo := &moduleLifecycleSecurityRepository{}
	keys := &moduleLifecycleSecurityKeys{}
	throttle := &moduleLifecycleSecurityThrottle{}
	handler := moduleLifecycleSecurityHandler(t, repo, keys, throttle)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := modulelifecycleclient.NewClient(
		"http", strings.TrimPrefix(server.URL, "http://"), server.Client(),
		loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false,
	)
	credential := "valid-session"
	result, err := client.PostAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDArchive()(context.Background(), &moduleapi.PostAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDArchivePayload{
		PlaneSessionCookieCredential: &credential,
		ModuleID:                     moduleLifecycleTransportModule,
		ProjectID:                    moduleLifecycleTransportProject,
		Slug:                         "acme",
	})
	if err != nil {
		t.Fatalf("generated client archive call: %v", err)
	}
	response, ok := result.(*moduleapi.RestFrameworkSerializersArchiveresponseResponse)
	if !ok {
		t.Fatalf("generated client response type=%T", result)
	}
	if got, want := string(response.ArchivedAt), "2025-01-02 03:04:05+00:00"; got != want {
		t.Errorf("archived_at=%q, want %q", got, want)
	}
	if len(repo.archiveCalls) != 1 || repo.archiveCalls[0].id != moduleLifecycleTransportModule || !repo.archiveCalls[0].archive {
		t.Errorf("archive calls=%v, want one archive of %s", repo.archiveCalls, moduleLifecycleTransportModule)
	}
}
