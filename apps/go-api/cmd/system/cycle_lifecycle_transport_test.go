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
	activityapi "github.com/CaliLuke/go-plane/activityreads/gen/plane_internal_django_migration_api"
	cyclelifecycleapi "github.com/CaliLuke/go-plane/cyclelifecycle/gen/cycle_lifecycle_api"
	cyclereads "github.com/CaliLuke/go-plane/gen/cycle_reads"
	"github.com/CaliLuke/go-plane/gen/system"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

const cycleLifecycleTransportPath = "/api/workspaces/acme/projects/11111111-1111-4111-8111-111111111111/cycles/22222222-2222-4222-8222-222222222222/"

type cycleLifecycleTransportReads struct {
	cyclereads.Service
	detailCalls int
}

func (service *cycleLifecycleTransportReads) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	if key == "" {
		return ctx, &cyclereads.CycleReadsUnauthenticated{Detail: readServicesAuthenticationDetail}
	}
	return ctx, nil
}

func (service *cycleLifecycleTransportReads) Detail(context.Context, *cyclereads.DetailPayload) (*cyclereads.RawJSONResponseMetadata, io.ReadCloser, error) {
	service.detailCalls++
	return &cyclereads.RawJSONResponseMetadata{ContentType: "application/json"}, io.NopCloser(strings.NewReader(`{"source":"cycle-reads"}`)), nil
}

type cycleLifecycleTransportWrites struct {
	cyclelifecycleapi.Service
	patchCalls     int
	dateCheckCalls int
}

type cycleLifecycleUnsupportedAPIKeyAuthorizer struct {
	methods []string
}

func (authorizer *cycleLifecycleUnsupportedAPIKeyAuthorizer) AuthorizeAPIActivityProtocol(ctx context.Context, key, _, _, method string) (int, map[string]any) {
	authorizer.methods = append(authorizer.methods, method)
	if key != "valid-api-key" {
		return http.StatusUnauthorized, map[string]any{"detail": readServicesAuthenticationDetail}
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Authenticated = true
	}
	return 0, nil
}

func (*cycleLifecycleUnsupportedAPIKeyAuthorizer) AuthorizeUserActivityProtocol(context.Context, string) (int, map[string]any) {
	return 0, nil
}

func (service *cycleLifecycleTransportWrites) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	if key == "" {
		return ctx, &cyclelifecycleapi.CycleLifecycleError{Kind: "status_401", Body: loom.MustJSONValueFrom(map[string]any{"detail": readServicesAuthenticationDetail})}
	}
	return ctx, nil
}

func (service *cycleLifecycleTransportWrites) AuthorizeSessionProtocol(_ context.Context, credential string) (int, map[string]any) {
	if credential != "valid-session" {
		return http.StatusUnauthorized, map[string]any{"detail": readServicesAuthenticationDetail}
	}
	return 0, nil
}

func (service *cycleLifecycleTransportWrites) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, _ systemapi.SessionProtocolResource) (int, map[string]any) {
	return service.AuthorizeSessionProtocol(ctx, credential)
}

func (*cycleLifecycleTransportWrites) OwnsSessionProtocolMethod(pattern, method string) bool {
	return pattern == "/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/" &&
		(method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete)
}

func (service *cycleLifecycleTransportWrites) PatchAPIWorkspacesBySlugProjectsByProjectIDCyclesByPk(context.Context, *cyclelifecycleapi.PatchAPIWorkspacesBySlugProjectsByProjectIDCyclesByPkPayload, io.ReadCloser) (*cyclelifecycleapi.PlaneAppSerializersCycleCycleserializerResponse, error) {
	service.patchCalls++
	return &cyclelifecycleapi.PlaneAppSerializersCycleCycleserializerResponse{}, nil
}

func (service *cycleLifecycleTransportWrites) PostAPIWorkspacesBySlugProjectsByProjectIDCyclesDateCheck(context.Context, *cyclelifecycleapi.PostAPIWorkspacesBySlugProjectsByProjectIDCyclesDateCheckPayload, io.ReadCloser) (*cyclelifecycleapi.RestFrameworkSerializersCycledatecheckresponseResponse, error) {
	service.dateCheckCalls++
	return &cyclelifecycleapi.RestFrameworkSerializersCycledatecheckresponseResponse{Status: true}, nil
}

func cycleLifecycleTransportHandler(reads *cycleLifecycleTransportReads, writes *cycleLifecycleTransportWrites) http.Handler {
	session := &assetsTransportSession{publicProjectMetadataThrottleStub: &publicProjectMetadataThrottleStub{
		allowed: true, userID: assetsTransportUserID, userTimezone: "UTC",
	}}
	return newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false),
		system.NewEndpoints(systemapi.NewSystem()), false, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			activityReads:          activityapi.NewEndpoints(&activityHTTPService{}),
			cycleReads:             cyclereads.NewEndpoints(reads),
			cycleLifecycle:         cyclelifecycleapi.NewEndpoints(writes),
			cycleLifecycleMetadata: writes,
			activityReadsMetadata:  &cycleLifecycleUnsupportedAPIKeyAuthorizer{},
			activityReadsBodyLimit: 1 << 20,
			readServicesSession:    session,
			readServicesCookie:     djangoSessionCookieConfig{Name: "session-id", Configured: true},
		},
	)
}

func TestCycleLifecycleAPIUnsupportedMethodsUseProjectAPIKeyBoundary(t *testing.T) {
	reads := &cycleLifecycleTransportReads{}
	writes := &cycleLifecycleTransportWrites{}
	handler := cycleLifecycleTransportHandler(reads, writes)

	for _, test := range []struct {
		name   string
		method string
		path   string
		allow  string
	}{
		{
			name:   "detail HEAD is unsupported",
			method: http.MethodHead,
			path:   "/api/v1/workspaces/acme/projects/11111111-1111-4111-8111-111111111111/cycles/22222222-2222-4222-8222-222222222222/",
			allow:  "PATCH, DELETE",
		},
		{
			name:   "archive GET is unsupported",
			method: http.MethodGet,
			path:   "/api/v1/workspaces/acme/projects/11111111-1111-4111-8111-111111111111/cycles/22222222-2222-4222-8222-222222222222/archive/",
			allow:  "POST",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			request.Header.Set("X-Api-Key", "valid-api-key")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != test.allow || !strings.Contains(response.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("status=%d Allow=%q Content-Type=%q body=%q", response.Code, response.Header().Get("Allow"), response.Header().Get("Content-Type"), response.Body.String())
			}
			if !headerHasToken(response.Header(), "Vary", "cookie") {
				t.Fatalf("Vary=%q, want cookie included", response.Header().Get("Vary"))
			}
		})
	}
}

func TestCycleLifecycleProductionMuxPreservesReadOwnershipAndRoutesWrites(t *testing.T) {
	reads := &cycleLifecycleTransportReads{}
	writes := &cycleLifecycleTransportWrites{}
	handler := cycleLifecycleTransportHandler(reads, writes)

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		request := httptest.NewRequest(method, cycleLifecycleTransportPath, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s detail status=%d body=%q", method, response.Code, response.Body.String())
		}
		if method == http.MethodGet && !strings.Contains(response.Body.String(), `"source":"cycle-reads"`) {
			t.Fatalf("GET detail body=%q; expected cycle-reads response", response.Body.String())
		}
	}
	if reads.detailCalls != 2 {
		t.Fatalf("cycle read detail called %d times, want GET and HEAD", reads.detailCalls)
	}

	request := httptest.NewRequest(http.MethodPatch, cycleLifecycleTransportPath, strings.NewReader(`{"name":"changed"}`))
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || writes.patchCalls != 1 || reads.detailCalls != 2 {
		t.Fatalf("PATCH status=%d patch_calls=%d read_calls=%d body=%q", response.Code, writes.patchCalls, reads.detailCalls, response.Body.String())
	}
}

func TestCycleLifecycleProductionMuxAppliesLiteralAndSessionProtocolRules(t *testing.T) {
	reads := &cycleLifecycleTransportReads{}
	writes := &cycleLifecycleTransportWrites{}
	handler := cycleLifecycleTransportHandler(reads, writes)

	dateCheckPath := "/api/workspaces/acme/projects/11111111-1111-4111-8111-111111111111/cycles/date-check/"
	getProtocol, matched := resolveReadServiceProtocolForMethod(dateCheckPath, http.MethodGet)
	if !matched || getProtocol.pattern != "/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/" || !readServiceProtocolHasMalformedUUID(dateCheckPath, getProtocol) {
		t.Fatalf("GET date-check resolver = (%#v, %t), want retained UUID detail converter for malformed literal", getProtocol, matched)
	}
	postProtocol, matched := resolveReadServiceProtocolForMethod(dateCheckPath, http.MethodPost)
	if !matched || postProtocol.pattern != "/api/workspaces/{slug}/projects/{project_id}/cycles/date-check/" {
		t.Fatalf("POST date-check resolver = (%#v, %t), want literal date-check endpoint", postProtocol, matched)
	}
	request := httptest.NewRequest(http.MethodPost, dateCheckPath, strings.NewReader(`{"start_date":"2025-01-01","end_date":"2025-01-02"}`))
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || writes.dateCheckCalls != 1 || reads.detailCalls != 0 {
		t.Fatalf("date-check status=%d date_check_calls=%d read_calls=%d body=%q", response.Code, writes.dateCheckCalls, reads.detailCalls, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPatch, cycleLifecycleTransportPath, strings.NewReader(`{"name":"changed"}`))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || writes.patchCalls != 0 {
		t.Fatalf("anonymous PATCH status=%d patch_calls=%d body=%q; want authentication denial before service", response.Code, writes.patchCalls, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodOptions, cycleLifecycleTransportPath, nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Allow") != "PUT, PATCH, DELETE, OPTIONS" {
		t.Fatalf("authenticated OPTIONS status=%d Allow=%q body=%q", response.Code, response.Header().Get("Allow"), response.Body.String())
	}

	malformedPath := "/api/workspaces/acme/projects/11111111-1111-4111-8111-111111111111/cycles/not-a-uuid/"
	request = httptest.NewRequest(http.MethodGet, malformedPath, nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("unauthenticated malformed UUID status=%d body=%q; want route-level 404 before auth", response.Code, response.Body.String())
	}

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		request = httptest.NewRequest(method, dateCheckPath, nil)
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound || response.Header().Get("Content-Type") != "text/html; charset=utf-8" || response.Header().Get("Allow") != "" || headerHasToken(response.Header(), "Vary", "cookie") {
			t.Fatalf("date-check %s status=%d Allow=%q Content-Type=%q Vary=%q body=%q", method, response.Code, response.Header().Get("Allow"), response.Header().Get("Content-Type"), response.Header().Get("Vary"), response.Body.String())
		}
	}
	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
		request = httptest.NewRequest(method, dateCheckPath, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "POST, OPTIONS" {
			t.Fatalf("date-check %s status=%d Allow=%q body=%q", method, response.Code, response.Header().Get("Allow"), response.Body.String())
		}
	}
}

func TestCycleLifecycleAPIDateCheckRouteRemainsUnregisteredAnd404(t *testing.T) {
	reads := &cycleLifecycleTransportReads{}
	writes := &cycleLifecycleTransportWrites{}
	handler := cycleLifecycleTransportHandler(reads, writes)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := server.Client()
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }

	canonical := "/api/v1/workspaces/acme/projects/11111111-1111-4111-8111-111111111111/cycles/date-check/"
	for _, path := range []string{canonical, strings.TrimSuffix(canonical, "/")} {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodTrace} {
			t.Run(method+" "+path, func(t *testing.T) {
				request, err := http.NewRequest(method, server.URL+path, nil)
				if err != nil {
					t.Fatal(err)
				}
				response, err := client.Do(request)
				if err != nil {
					t.Fatal(err)
				}
				defer func() {
					if closeErr := response.Body.Close(); closeErr != nil {
						t.Errorf("close HTTP response body: %v", closeErr)
					}
				}()
				body, err := io.ReadAll(response.Body)
				if err != nil {
					t.Fatal(err)
				}
				if response.StatusCode != http.StatusNotFound || response.Header.Get("Content-Type") != "application/json" || response.Header.Get("Allow") != "" || response.Header.Get("Location") != "" {
					t.Fatalf("status=%d Allow=%q Content-Type=%q Location=%q body=%q, want resolver JSON 404", response.StatusCode, response.Header.Get("Allow"), response.Header.Get("Content-Type"), response.Header.Get("Location"), body)
				}
				if method == http.MethodHead {
					if len(body) != 0 {
						t.Fatalf("HEAD body=%q, want empty", body)
					}
					return
				}
				var payload map[string]string
				if err := json.Unmarshal(body, &payload); err != nil || payload["error"] != "Page not found." {
					t.Fatalf("body=%q, want JSON Page not found error", body)
				}
			})
		}
	}
	for _, protocol := range readServiceProtocols {
		if protocol.pattern == "/api/v1/workspaces/{slug}/projects/{project_id}/cycles/date-check/" {
			t.Fatal("API-v1 date-check unexpectedly owns a protocol route")
		}
	}
}

func headerHasToken(header http.Header, name, token string) bool {
	for _, value := range header.Values(name) {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}
