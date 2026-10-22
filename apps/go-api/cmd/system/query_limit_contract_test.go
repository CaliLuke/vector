package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	systemapi "github.com/CaliLuke/go-plane"
	activityapi "github.com/CaliLuke/go-plane/activityreads/gen/plane_internal_django_migration_api"
	publicprojectmetadatareadsapi "github.com/CaliLuke/go-plane/gen/public_project_metadata_reads"
	system "github.com/CaliLuke/go-plane/gen/system"
	useridentityapi "github.com/CaliLuke/go-plane/gen/user_identity"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

type djangoOriginalQueryLimitFixture struct {
	Name       string `json:"limit_case"`
	RawQuery   string `json:"limit_raw_query"`
	FieldCount int    `json:"limit_field_count"`
	Accepted   bool   `json:"accepted"`
}

func originalQueryLimitFixtures(t *testing.T) []djangoOriginalQueryLimitFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/query_compatibility.json")
	if err != nil {
		t.Fatal(err)
	}
	var all []djangoOriginalQueryLimitFixture
	if err := json.Unmarshal(data, &all); err != nil {
		t.Fatal(err)
	}
	var fixtures []djangoOriginalQueryLimitFixture
	for _, fixture := range all {
		if fixture.Name != "" {
			fixtures = append(fixtures, fixture)
		}
	}
	if len(fixtures) != 10 {
		t.Fatalf("got %d original-field fixtures; want 10", len(fixtures))
	}
	return fixtures
}

func queryLimitProductionHandler(pending pendingReplacementEndpoints) http.Handler {
	return newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pending,
	)
}

// These requests pass through the shipping mount and its complete middleware
// stack. The fixture count is from Django, before empty fields or duplicates
// are removed. Negotiation must still reject excess fields before authentication.
func TestDjangoQueryOriginalFieldLimitThroughReadMount(t *testing.T) {
	const path = "/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/issues/00000000-0000-4000-8000-000000000002/activities/"
	for _, fixture := range originalQueryLimitFixtures(t) {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			for _, mode := range []string{"authenticated", "missing credentials", "unacceptable media"} {
				t.Run(fixture.Name+"/"+strconv.Itoa(fixture.FieldCount)+"/"+method+"/"+mode, func(t *testing.T) {
					service := &activityHTTPService{}
					handler := queryLimitProductionHandler(pendingReplacementEndpoints{
						activityReads: activityapi.NewEndpoints(service), activityReadsMetadata: service,
						activityReadsBodyLimit: defaultFileSizeLimit,
					})
					request := httptest.NewRequest(method, path+"?"+fixture.RawQuery, nil)
					if mode == "authenticated" {
						request.Header.Set("X-Api-Key", "token")
					}
					if mode == "unacceptable media" {
						request.Header.Set("Accept", "application/xml")
					}
					response := httptest.NewRecorder()
					handler.ServeHTTP(response, request)
					wantStatus := http.StatusInternalServerError
					if fixture.Accepted {
						switch mode {
						case "authenticated":
							wantStatus = http.StatusOK
							if method == http.MethodHead {
								wantStatus = http.StatusMethodNotAllowed
							}
						case "missing credentials":
							wantStatus = http.StatusUnauthorized
						case "unacceptable media":
							wantStatus = http.StatusNotAcceptable
						}
					}
					if response.Code != wantStatus {
						t.Fatalf("status=%d body=%q; want %d", response.Code, response.Body.String(), wantStatus)
					}
					if !fixture.Accepted {
						if service.authCalls != 0 || service.protocolCalls != 0 {
							t.Errorf("limit check reached authentication: generated=%d protocol=%d", service.authCalls, service.protocolCalls)
						}
						if response.Header().Get("Content-Type") != "text/html; charset=utf-8" {
							t.Errorf("Content-Type=%q", response.Header().Get("Content-Type"))
						}
						if method == http.MethodGet && response.Body.String() != string(djangoServerErrorBody) {
							t.Errorf("limit body=%q", response.Body.String())
						}
					}
					if method == http.MethodHead && response.Body.Len() != 0 {
						t.Errorf("HEAD body=%q", response.Body.String())
					}
				})
			}
		}
	}
}

type queryLimitMetadataSession struct {
	publicProjectMetadataThrottleStub
	throttleCalls int
}

func (sessions *queryLimitMetadataSession) AllowRequest(ctx context.Context, request *http.Request) (bool, time.Duration, bool, bool, *djangosession.SessionRefresh, string, error) {
	sessions.throttleCalls++
	return sessions.publicProjectMetadataThrottleStub.AllowRequest(ctx, request)
}

func (sessions *queryLimitMetadataSession) AllowRequestWithoutTimezone(ctx context.Context, request *http.Request) (bool, time.Duration, bool, bool, *djangosession.SessionRefresh, string, error) {
	sessions.throttleCalls++
	return sessions.publicProjectMetadataThrottleStub.AllowRequestWithoutTimezone(ctx, request)
}

// Public metadata has a separate negotiation/session stack: excess fields
// produce its existing JSON 500, still inspect the session, and never refresh
// it. The same request under the limit can refresh a valid session.
func TestDjangoQueryOriginalFieldLimitThroughPublicMetadataMount(t *testing.T) {
	for _, fixture := range originalQueryLimitFixtures(t) {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			for _, accept := range []string{"application/json", "application/xml"} {
				t.Run(fixture.Name+"/"+strconv.Itoa(fixture.FieldCount)+"/"+method+"/"+accept, func(t *testing.T) {
					sessions := &queryLimitMetadataSession{publicProjectMetadataThrottleStub: publicProjectMetadataThrottleStub{
						allowed: true, refresh: true,
						result: djangosession.SessionRefresh{SessionKey: "valid-session-key", BrowserClose: true},
					}}
					handler := queryLimitProductionHandler(pendingReplacementEndpoints{
						publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
						publicProjectMetadataThrottle: sessions,
					})
					request := httptest.NewRequest(method, "/api/public/anchor/example/labels/?"+fixture.RawQuery, nil)
					request.Header.Set("Accept", accept)
					request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session-key"})
					response := httptest.NewRecorder()
					handler.ServeHTTP(response, request)
					wantStatus, wantRefresh := http.StatusInternalServerError, 0
					if fixture.Accepted {
						wantStatus, wantRefresh = http.StatusOK, 1
						if accept == "application/xml" {
							wantStatus = http.StatusNotAcceptable
						}
					}
					if response.Code != wantStatus {
						t.Fatalf("status=%d body=%q; want %d", response.Code, response.Body.String(), wantStatus)
					}
					if sessions.calls != 1 || sessions.refreshCalls != wantRefresh {
						t.Errorf("session inspections=%d refreshes=%d; want 1/%d", sessions.calls, sessions.refreshCalls, wantRefresh)
					}
					if (!fixture.Accepted || accept == "application/xml") && sessions.throttleCalls != 0 {
						t.Errorf("negotiation rejection reached anonymous throttle %d times", sessions.throttleCalls)
					}
					if !fixture.Accepted {
						if response.Header().Get("Set-Cookie") != "" {
							t.Errorf("500 refreshed session: %q", response.Header().Get("Set-Cookie"))
						}
						if response.Header().Get("Content-Type") != "application/json" {
							t.Errorf("Content-Type=%q", response.Header().Get("Content-Type"))
						}
						if method == http.MethodGet && strings.TrimSpace(response.Body.String()) != `{"error":"Something went wrong please try again later"}` {
							t.Errorf("limit body=%q", response.Body.String())
						}
					}
					if method == http.MethodHead && response.Body.Len() != 0 {
						t.Errorf("HEAD body=%q", response.Body.String())
					}
				})
			}
		}
	}
}

// User-me uses its own format/limit consumer and lazy session lifecycle.
func TestDjangoQueryOriginalFieldLimitThroughUserMeMount(t *testing.T) {
	for _, fixture := range originalQueryLimitFixtures(t) {
		t.Run(fixture.Name+"/"+strconv.Itoa(fixture.FieldCount), func(t *testing.T) {
			sessions := &publicProjectMetadataThrottleStub{}
			handler := queryLimitProductionHandler(pendingReplacementEndpoints{
				userIdentity:        useridentityapi.NewEndpoints(userIdentityHTTPStub{}),
				userIdentitySession: sessions,
			})
			request := httptest.NewRequest(http.MethodGet, "/api/users/me/?"+fixture.RawQuery, nil)
			request.Header.Set("Accept", "application/xml")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			wantStatus := http.StatusInternalServerError
			wantBody := `{"error":"Something went wrong please try again later"}`
			if fixture.Accepted {
				wantStatus = http.StatusNotAcceptable
				wantBody = `{"detail":"Could not satisfy the request Accept header."}`
			}
			if response.Code != wantStatus || strings.TrimSpace(response.Body.String()) != wantBody {
				t.Fatalf("response=%d %q; want %d %q", response.Code, response.Body.String(), wantStatus, wantBody)
			}
			if sessions.calls != 1 || sessions.refreshCalls != 0 || sessions.timezoneCalls != 0 {
				t.Fatalf("session lifecycle: inspect=%d refresh=%d timezone=%d", sessions.calls, sessions.refreshCalls, sessions.timezoneCalls)
			}
		})
	}
}
