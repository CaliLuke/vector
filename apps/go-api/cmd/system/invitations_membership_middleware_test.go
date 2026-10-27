package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	activityapi "github.com/CaliLuke/go-plane/activityreads/gen/plane_internal_django_migration_api"
	projectmembershipreadsapi "github.com/CaliLuke/go-plane/gen/project_membership_reads"
	"github.com/CaliLuke/go-plane/gen/system"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	store "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	api "github.com/CaliLuke/go-plane/invitationsmembership/gen/invitations_membership"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/google/uuid"
)

type invitationsHTTPCache struct{}

func (invitationsHTTPCache) Invalidate(context.Context, string, string, bool) error { return nil }

type invitationsHTTPRepository struct {
	*store.InvitationsRepository
	events *[]string
	denied bool
}

func (r *invitationsHTTPRepository) AuthorizeWorkspaceRole(context.Context, uuid.UUID, string, ...int) error {
	*r.events = append(*r.events, "authorization")
	if r.denied {
		return store.ErrForbidden
	}
	return nil
}

func (r *invitationsHTTPRepository) WorkspaceExists(context.Context, string) (bool, error) {
	return true, nil
}

func (r *invitationsHTTPRepository) WorkspaceInvites(context.Context, string) (json.RawMessage, error) {
	*r.events = append(*r.events, "dispatch")
	return json.RawMessage(`[]`), nil
}

func (r *invitationsHTTPRepository) WorkspaceMemberUsers(context.Context, string) (json.RawMessage, error) {
	*r.events = append(*r.events, "dispatch")
	return json.RawMessage(`[]`), nil
}

func (r *invitationsHTTPRepository) Invitation(_ context.Context, c store.InvitationCommand) (json.RawMessage, error) {
	*r.events = append(*r.events, "dispatch")
	return json.RawMessage(`[]`), nil
}

func TestInvitationsMembershipProductionMiddleware(t *testing.T) {
	for _, test := range []struct {
		name, method, suffix, key, zone, allow string
		denied, preflight, session             bool
		limit                                  int64
		status, audit                          int
		events                                 []string
	}{
		{name: "role list rejects anonymous", method: "GET", suffix: "roles/", zone: "UTC", status: 401, events: []string{"authentication"}},
		{name: "role scope denied", method: "GET", suffix: "roles/", key: "valid", zone: "UTC", denied: true, status: 403, audit: 1, events: []string{"authentication", "authorization"}},
		{name: "role POST unsupported after auth", method: "POST", suffix: "roles/", key: "valid", zone: "UTC", status: 405, audit: 1, events: []string{"authentication", "authorization", "throttling"}, allow: "GET"},
		{name: "role identifier checked after throttle", method: "GET", suffix: "roles/not-a-uuid/", key: "valid", zone: "UTC", status: 400, audit: 1, events: []string{"authentication", "authorization", "throttling"}},
		{name: "member PATCH retains write Allow", method: "PATCH", suffix: "members/00000000-0000-4000-8000-000000000011/", key: "valid", zone: "UTC", denied: true, session: true, status: 403, audit: 1, events: []string{"authorization"}, allow: "PATCH, DELETE, OPTIONS"},
		{name: "member DELETE retains write Allow", method: "DELETE", suffix: "members/00000000-0000-4000-8000-000000000011/", key: "valid", zone: "UTC", denied: true, session: true, status: 403, audit: 1, events: []string{"authorization"}, allow: "PATCH, DELETE, OPTIONS"},
		{name: "member OPTIONS retains write Allow", method: "OPTIONS", suffix: "members/00000000-0000-4000-8000-000000000011/", key: "valid", zone: "UTC", denied: true, session: true, status: 200, audit: 1, allow: "PATCH, DELETE, OPTIONS"},
		{name: "request-size before auth", method: "POST", suffix: "invitations/", key: "valid", zone: "UTC", limit: 1, status: 413},
		{name: "invite read has no rate throttle", method: "GET", suffix: "invitations/", key: "valid", zone: "UTC", status: 200, audit: 1, events: []string{"authentication", "authorization", "dispatch"}},
		{name: "invite derived HEAD", method: "HEAD", suffix: "invitations/", key: "valid", zone: "UTC", status: 200, audit: 1, events: []string{"authentication", "authorization", "dispatch"}},
		{name: "invite OPTIONS metadata", method: "OPTIONS", suffix: "invitations/", key: "valid", zone: "UTC", status: 200, audit: 1, events: []string{"authentication", "authorization"}},
		{name: "roster read throttles", method: "GET", suffix: "members/", key: "valid", zone: "UTC", status: 200, audit: 1, events: []string{"authentication", "authorization", "throttling", "dispatch"}},
		{name: "roster HEAD is unsupported after authorization", method: "HEAD", suffix: "members/", key: "valid", zone: "UTC", status: 405, audit: 1, events: []string{"authentication", "authorization", "throttling"}},
		{name: "permission before timezone", method: "GET", suffix: "invitations/", key: "valid", zone: "Not/AZone", denied: true, status: 403, audit: 1, events: []string{"authentication", "authorization"}},
		{name: "timezone before dispatch", method: "GET", suffix: "invitations/", key: "valid", zone: "Not/AZone", status: 400, audit: 1, events: []string{"authentication", "authorization"}},
		{name: "anonymous OPTIONS", method: "OPTIONS", suffix: "invitations/", zone: "UTC", status: 401, events: []string{"authentication"}},
		{name: "session before permission-classes", method: "GET", suffix: "invitations/", zone: "Not/AZone", session: true, status: 401, audit: 1},
		{name: "session permission-classes before timezone", method: "GET", suffix: "invitations/", key: "valid", zone: "Not/AZone", denied: true, session: true, status: 403, audit: 1, events: []string{"authorization"}},
		{name: "session timezone before dispatch", method: "GET", suffix: "invitations/", key: "valid", zone: "Not/AZone", session: true, status: 400, audit: 1, events: []string{"authorization"}},
		{name: "session authorized dispatch", method: "GET", suffix: "invitations/", key: "valid", zone: "UTC", session: true, status: 200, audit: 1, events: []string{"authorization", "dispatch"}},
		{name: "session OPTIONS permission-classes", method: "OPTIONS", suffix: "invitations/", key: "valid", zone: "UTC", session: true, status: 200, audit: 1, events: []string{"authorization"}},
		{name: "preflight has no auth effects", method: "OPTIONS", suffix: "invitations/", key: "valid", zone: "Not/AZone", preflight: true, status: 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			var events []string
			repository := &invitationsHTTPRepository{events: &events, denied: test.denied}
			service, err := systemapi.NewInvitationsMembership(repository, membershipHTTPKeys{&events, test.zone}, membershipHTTPSessions{}, membershipHTTPThrottle{&events}, invitationsHTTPCache{})
			if err != nil {
				t.Fatal(err)
			}
			logger := &apiV1ActivityLoggerStub{}
			sessions := &publicProjectMetadataThrottleStub{userTimezone: test.zone}
			if test.key != "" {
				sessions.userID = membershipHTTPProject
			}
			activity := &activityHTTPService{}
			limit := test.limit
			if limit == 0 {
				limit = defaultFileSizeLimit
			}
			handler := newHTTPHandlerWithPendingReplacements(
				newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{
					invitationsMembership: api.NewEndpoints(service), invitationsMetadata: service.(systemapi.InvitationsMembershipProtocolAuthorizer),
					activityReads: activityapi.NewEndpoints(activity), activityReadsMetadata: activity,
					activityReadsActivity: logger, activityReadsSecret: "secret", activityReadsBodyLimit: limit,
					readServicesSession: sessions,
					readServicesCORS:    []string{"https://app.example.test"},
				})
			prefix := "/api/v1/workspaces/demo/"
			if test.session {
				prefix = "/api/workspaces/demo/"
			}
			request := httptest.NewRequest(test.method, prefix+test.suffix, strings.NewReader(`{}`))
			if test.session {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
				request.Header.Set("X-Api-Key", "audit-only")
			}
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
			if test.allow != "" && response.Header().Get("Allow") != test.allow {
				t.Fatalf("Allow=%q want=%q", response.Header().Get("Allow"), test.allow)
			}
			if test.method == "HEAD" && response.Body.Len() != 0 {
				t.Fatalf("HEAD body: %s", response.Body.String())
			}
		})
	}
}

// Give the pools different identities so stale replica authentication cannot
// accidentally pass when invitation routes require the primary database.
type invitationsPoolKeyValidator struct{ id uuid.UUID }

func (v invitationsPoolKeyValidator) Validate(context.Context, string) (apikey.Principal, error) {
	return apikey.Principal{UserID: v.id}, nil
}
func TestInvitationsMembershipAPIKeysUseSourceDatabase(t *testing.T) {
	primary, replica := uuid.New(), uuid.New()
	router := membershipAPIKeyRouter{writer: invitationsPoolKeyValidator{primary}, reader: invitationsPoolKeyValidator{replica}}
	for _, path := range []string{
		"/api/v1/workspaces/demo/invitations/",
		"/api/v1/workspaces/demo/invitations/lookup/",
		"/api/v1/workspaces/demo/members/",
		"/api/v1/workspaces/demo/members-lite/",
		"/api/v1/workspaces/demo/projects/project/members/",
	} {
		for _, method := range []string{"GET", "HEAD", "OPTIONS", "POST", "PATCH", "DELETE"} {
			t.Run(method+" "+path, func(t *testing.T) {
				want := replica
				if strings.Contains(path, "/invitations/") || method == "POST" || method == "PATCH" || method == "DELETE" {
					want = primary
				}
				handler := loomhttp.RequestMetadataMiddleware(requestMetadataPolicy)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
					got, err := router.Validate(request.Context(), "key")
					if err != nil || got.UserID != want {
						t.Fatalf("identity=%v err=%v want=%v", got.UserID, err, want)
					}
				}))
				handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(method, path, nil))
			})
		}
	}
}

// Mount both owners: the invitation write boundary must not reject an existing
// member GET or its derived HEAD before the retired read service can run.
func TestInvitationsMembershipPreservesExistingMemberReads(t *testing.T) {
	for _, missing := range []bool{false, true} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(fmt.Sprintf("%s/missing=%t", method, missing), func(t *testing.T) {
				var events []string
				service, err := systemapi.NewInvitationsMembership(&invitationsHTTPRepository{events: &events}, membershipHTTPKeys{&events, "UTC"}, membershipHTTPSessions{}, membershipHTTPThrottle{&events}, invitationsHTTPCache{})
				if err != nil {
					t.Fatal(err)
				}
				reads := projectMembershipReadsHTTPStub{}
				want := http.StatusOK
				if missing {
					reads.workspaceMemberError = &projectmembershipreadsapi.ProjectMembershipReadsNotFound{Message: "Workspace member not found"}
					want = http.StatusNotFound
				}
				handler := newHTTPHandlerWithPendingReplacements(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
					nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
					pendingReplacementEndpoints{invitationsMembership: api.NewEndpoints(service), invitationsMetadata: service.(systemapi.InvitationsMembershipProtocolAuthorizer), projectMembershipReads: projectmembershipreadsapi.NewEndpoints(reads), projectMembershipMetadata: reads, readServicesSession: &publicProjectMetadataThrottleStub{userID: membershipHTTPProject, userTimezone: "UTC"}})
				request := httptest.NewRequest(method, "/api/workspaces/acme/members/00000000-0000-4000-8000-000000000011/", nil)
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != want {
					t.Fatalf("status=%d body=%s want=%d", response.Code, response.Body.String(), want)
				}
				if response.Header().Get("Allow") != "GET, PATCH, DELETE, HEAD, OPTIONS" {
					t.Fatalf("Allow=%q", response.Header().Get("Allow"))
				}
				if method == http.MethodHead && response.Body.Len() != 0 {
					t.Fatalf("HEAD body=%s", response.Body.String())
				}
				if len(events) != 0 {
					t.Fatalf("read invoked invitation write service: %v", events)
				}
			})
		}
	}
}
