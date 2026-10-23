package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"uuid"

	systemapi "github.com/CaliLuke/go-plane"
	assetslifecycleserver "github.com/CaliLuke/go-plane/assetslifecycle/gen/http/plane_internal_django_migration_api/server"
	analyticviewsapi "github.com/CaliLuke/go-plane/gen/analytic_views"
	csrftoken "github.com/CaliLuke/go-plane/gen/csrf_token"
	csrftokensvr "github.com/CaliLuke/go-plane/gen/http/csrf_token/server"
	projectstatereadsclient "github.com/CaliLuke/go-plane/gen/http/project_state_reads/client"
	instanceadminsessionapi "github.com/CaliLuke/go-plane/gen/instance_admin_session"
	instanceworkspaceapi "github.com/CaliLuke/go-plane/gen/instance_workspace_slug_check"
	lastvisitedworkspaceapi "github.com/CaliLuke/go-plane/gen/last_visited_workspace"
	projectlabelreadsapi "github.com/CaliLuke/go-plane/gen/project_label_reads"
	projectmembershipreadsapi "github.com/CaliLuke/go-plane/gen/project_membership_reads"
	projectstatereadsapi "github.com/CaliLuke/go-plane/gen/project_state_reads"
	publicprojectmetadatareadsapi "github.com/CaliLuke/go-plane/gen/public_project_metadata_reads"
	sticky "github.com/CaliLuke/go-plane/gen/stickies"
	system "github.com/CaliLuke/go-plane/gen/system"
	useraccountsapi "github.com/CaliLuke/go-plane/gen/user_accounts"
	userapitokensapi "github.com/CaliLuke/go-plane/gen/user_api_tokens"
	useridentityapi "github.com/CaliLuke/go-plane/gen/user_identity"
	userinstanceadminapi "github.com/CaliLuke/go-plane/gen/user_instance_admin"
	usernotificationpreferencesapi "github.com/CaliLuke/go-plane/gen/user_notification_preferences"
	userprofileflagsapi "github.com/CaliLuke/go-plane/gen/user_profile_flags"
	workspacefavoritesapi "github.com/CaliLuke/go-plane/gen/workspace_favorites"
	workspacehomepreferencesapi "github.com/CaliLuke/go-plane/gen/workspace_home_preferences"
	workspacemembermeapi "github.com/CaliLuke/go-plane/gen/workspace_member_me"
	workspacequicklinksapi "github.com/CaliLuke/go-plane/gen/workspace_quick_links"
	workspacerecentvisitsapi "github.com/CaliLuke/go-plane/gen/workspace_recent_visits"
	workspaceapi "github.com/CaliLuke/go-plane/gen/workspace_slug_check"
	workspacethemesapi "github.com/CaliLuke/go-plane/gen/workspace_themes"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/platform"
	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/CaliLuke/loom/clue/log"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

func TestEntityFavoritesAllowProtocol(t *testing.T) {
	tests := []struct{ path, want string }{
		{"/api/workspaces/x/user-favorite-projects/", "GET, POST, HEAD, OPTIONS"},
		{"/api/workspaces/x/projects/p/user-favorite-cycles/", "GET, POST, HEAD, OPTIONS"},
		{"/api/workspaces/x/user-favorite-projects/id/", "DELETE, OPTIONS"},
		{"/api/workspaces/x/projects/p/user-favorite-modules/id/", "DELETE, OPTIONS"},
		{"/api/workspaces/x/projects/p/favorite-pages/id/", "POST, DELETE, OPTIONS"},
	}
	for _, test := range tests {
		if got := entityFavoritesAllow(test.path); got != test.want {
			t.Errorf("Allow(%q) = %q, want %q", test.path, got, test.want)
		}
	}
}

func TestAssetLifecycleProtocolHeadersAndAllow(t *testing.T) {
	mounts := []*assetslifecycleserver.MountPoint{
		{Verb: http.MethodPost, Pattern: "/api/v1/assets/user-assets/"},
		{Verb: http.MethodGet, Pattern: "/api/assets/v2/static/{asset_id}/"},
	}
	var got http.Header
	handler := assetLifecycleProtocolHeaders(mounts)(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		got = writer.Header().Clone()
		writer.WriteHeader(http.StatusOK)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/assets/user-assets/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
	for name, want := range map[string]string{
		"Allow":                      "POST",
		"Cross-Origin-Opener-Policy": "same-origin",
		"Referrer-Policy":            "same-origin",
		"X-Content-Type-Options":     "nosniff",
		"X-Frame-Options":            "DENY",
	} {
		if got.Get(name) != want {
			t.Errorf("%s=%q, want %q", name, got.Get(name), want)
		}
	}
	if !strings.Contains(strings.ToLower(got.Get("Vary")), "cookie") || !strings.Contains(strings.ToLower(got.Get("Vary")), "origin") {
		t.Errorf("Vary=%q, want Cookie and origin", got.Get("Vary"))
	}
	if got := assetLifecycleAllowedMethods(mounts, "/api/assets/v2/static/00000000-0000-0000-0000-000000000001/"); got != "GET, HEAD, OPTIONS" {
		t.Fatalf("static Allow=%q, want GET, HEAD, OPTIONS", got)
	}
	for path, want := range map[string]string{
		"/api/assets/v2/user-assets/":                                      "POST, OPTIONS",
		"/api/assets/v2/user-assets/00000000-0000-0000-0000-000000000001/": "PATCH, DELETE, OPTIONS",
		"/api/assets/v2/workspaces/ws/projects/00000000-0000-0000-0000-000000000001/issues/00000000-0000-0000-0000-000000000002/attachments/":                                  "GET, POST, HEAD, OPTIONS",
		"/api/workspaces/ws/projects/00000000-0000-0000-0000-000000000001/issues/00000000-0000-0000-0000-000000000002/issue-attachments/00000000-0000-0000-0000-000000000003/": "DELETE, OPTIONS",
		"/api/public/assets/v2/anchor/shared-board/00000000-0000-0000-0000-000000000001/":                                                                                      "GET, PATCH, DELETE, HEAD, OPTIONS",
		"/api/v1/assets/user-assets/00000000-0000-0000-0000-000000000001/":                                                                                                     "PATCH, DELETE",
	} {
		if got := assetLifecycleAllowedMethods(nil, path); got != want {
			t.Errorf("Allow(%s)=%q, want %q", path, got, want)
		}
	}
}

type workspaceSlugHTTPStub struct {
	unauthenticated bool
	failure         bool
}

type instanceWorkspaceSlugHTTPStub struct {
	apiKey       string
	forbidden    bool
	checkFailure bool
}

type userInstanceAdminHTTPStub struct {
	apiKey  string
	isAdmin bool
	failure bool
}

type userProfileFlagsHTTPStub struct {
	apiKey  string
	value   bool
	missing bool
	failure bool
}

type userNotificationPreferencesHTTPStub struct {
	apiKey string
	patch  *usernotificationpreferencesapi.PatchPayload
}

type userAccountsHTTPStub struct {
	apiKey  string
	deleted string
}

type userAPITokensHTTPStub struct {
	apiKey string
}

type userIdentityHTTPStub struct{}

type userIdentityUnauthenticatedHTTPStub struct{ userIdentityHTTPStub }

type userIdentityTimezoneNotFoundHTTPStub struct{ userIdentityHTTPStub }

type userIdentityInternalErrorHTTPStub struct{ userIdentityHTTPStub }

func (userIdentityHTTPStub) Session(context.Context, *useridentityapi.SessionPayload) (*useridentityapi.UserSession, error) {
	return &useridentityapi.UserSession{IsAuthenticated: false}, nil
}

func (userIdentityTimezoneNotFoundHTTPStub) Session(context.Context, *useridentityapi.SessionPayload) (*useridentityapi.UserSession, error) {
	return nil, &useridentityapi.UserIdentityTimezoneNotFound{Message: "The required key does not exist."}
}

func (userIdentityInternalErrorHTTPStub) Session(context.Context, *useridentityapi.SessionPayload) (*useridentityapi.UserSession, error) {
	return nil, &useridentityapi.UserIdentityInternalError{Message: "Something went wrong please try again later"}
}

func (userIdentityHTTPStub) Me(context.Context, *useridentityapi.MePayload) (*useridentityapi.SessionUser, error) {
	return &useridentityapi.SessionUser{
		ID:                "00000000-0000-4000-8000-000000000b01",
		Avatar:            loom.NullableValue("https://example.com/avatar.png"),
		AvatarURL:         loom.NullableValue("https://example.com/avatar.png"),
		CoverImage:        loom.NullableValue("https://example.com/cover.png"),
		CoverImageURL:     loom.NullableValue("https://example.com/cover.png"),
		DateJoined:        "2026-07-01T20:04:05.123456-07:00",
		DisplayName:       "Current User",
		Email:             loom.NullableValue("user-me@example.com"),
		FirstName:         "Current",
		LastName:          "User",
		IsActive:          true,
		IsBot:             false,
		IsEmailVerified:   true,
		UserTimezone:      "America/Los_Angeles",
		Username:          "user-me@example.com",
		IsPasswordAutoset: true,
		LastLoginMedium:   loom.NullableValue("magic-code"),
		LastLoginTime:     loom.NullableValue("2026-07-01T20:04:05.123456-07:00"),
	}, nil
}

func (userIdentityHTTPStub) Profile(context.Context, *useridentityapi.ProfilePayload) (*useridentityapi.UserProfile, error) {
	return nil, errors.New("not implemented")
}

func (userIdentityHTTPStub) UpdateProfile(context.Context, *useridentityapi.UpdateProfilePayload) (*useridentityapi.UserProfile, error) {
	return nil, errors.New("not implemented")
}

func (userIdentityHTTPStub) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	if key == "" {
		return ctx, &useridentityapi.UserIdentityUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	return ctx, nil
}

func (userIdentityUnauthenticatedHTTPStub) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, &useridentityapi.UserIdentityUnauthenticated{Detail: "Authentication credentials were not provided."}
}

type instanceAdminSessionHTTPStub struct {
	apiKey        string
	sessionKey    string
	forbidden     bool
	authenticated bool
}

type workspaceQuickLinksHTTPStub struct{}

func (workspaceQuickLinksHTTPStub) List(context.Context, *workspacequicklinksapi.ListPayload) ([]*workspacequicklinksapi.WorkspaceQuickLink, error) {
	return []*workspacequicklinksapi.WorkspaceQuickLink{}, nil
}
func (workspaceQuickLinksHTTPStub) link() *workspacequicklinksapi.WorkspaceQuickLink {
	return &workspacequicklinksapi.WorkspaceQuickLink{
		ID: "00000000-0000-4000-8000-000000000041", CreatedAt: "2026-08-26T21:00:00Z", UpdatedAt: "2026-08-26T21:00:00Z",
		Workspace: "00000000-0000-4000-8000-000000000031", Owner: "00000000-0000-4000-8000-000000000010", URL: "https://plane.so", Metadata: loom.JSONValue(`{}`),
	}
}
func (stub workspaceQuickLinksHTTPStub) Create(context.Context, *workspacequicklinksapi.CreatePayload) (*workspacequicklinksapi.WorkspaceQuickLink, error) {
	return stub.link(), nil
}
func (stub workspaceQuickLinksHTTPStub) Retrieve(context.Context, *workspacequicklinksapi.RetrievePayload) (*workspacequicklinksapi.WorkspaceQuickLink, error) {
	return stub.link(), nil
}
func (stub workspaceQuickLinksHTTPStub) Patch(context.Context, *workspacequicklinksapi.PatchPayload) (*workspacequicklinksapi.WorkspaceQuickLink, error) {
	return stub.link(), nil
}
func (workspaceQuickLinksHTTPStub) Delete(context.Context, *workspacequicklinksapi.DeletePayload) error {
	return nil
}
func (workspaceQuickLinksHTTPStub) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	if key == "" {
		return ctx, &workspacequicklinksapi.WorkspaceQuickLinkUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	return ctx, nil
}

type workspaceThemesHTTPStub struct{}

type workspaceRecentVisitsHTTPStub struct{}

func (workspaceRecentVisitsHTTPStub) List(context.Context, *workspacerecentvisitsapi.ListPayload) ([]*workspacerecentvisitsapi.WorkspaceRecentVisit, error) {
	return []*workspacerecentvisitsapi.WorkspaceRecentVisit{}, nil
}

type lastVisitedWorkspaceHTTPStub struct{ unauthenticated bool }

func (lastVisitedWorkspaceHTTPStub) Get(context.Context, *lastvisitedworkspaceapi.GetPayload) (*lastvisitedworkspaceapi.LastVisitedWorkspaceResult, error) {
	return &lastvisitedworkspaceapi.LastVisitedWorkspaceResult{
		ProjectDetails:   loom.JSONValue(`[]`),
		WorkspaceDetails: loom.JSONValue(`{}`),
	}, nil
}
func (stub lastVisitedWorkspaceHTTPStub) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	if stub.unauthenticated {
		return ctx, &lastvisitedworkspaceapi.LastVisitedWorkspaceUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	return ctx, nil
}
func (workspaceRecentVisitsHTTPStub) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, nil
}

type workspaceHomePreferencesHTTPStub struct{}

type workspaceMemberMeHTTPStub struct{}

func (workspaceMemberMeHTTPStub) Get(context.Context, *workspacemembermeapi.GetPayload) (*workspacemembermeapi.WorkspaceMemberMe, error) {
	return &workspacemembermeapi.WorkspaceMemberMe{
		CompanyRole: loom.NullValue[string](),
		Member:      loom.NullableValue("00000000-0000-4000-8000-000000000001"),
		Role:        loom.NullableValue(20),
		Workspace:   loom.NullableValue("00000000-0000-4000-8000-000000000002"),
	}, nil
}
func (workspaceMemberMeHTTPStub) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, nil
}

type projectMembershipReadsHTTPStub struct {
	searchCapture                *string
	readCalls                    *int
	metadataCalls                *int
	metadataError                error
	rolesError                   error
	workspaceMembersError        error
	workspaceMemberError         error
	workspaceProjectMembersError error
}

type projectStateReadsHTTPStub struct {
	metadataCalls *int
}

type projectLabelReadsHTTPStub struct{}

type publicProjectMetadataReadsHTTPStub struct{}

type publicProjectMetadataReadsUserContextHTTPStub struct {
	publicProjectMetadataReadsHTTPStub
	userID *string
}

type publicProjectMetadataReadsRenderErrorRepositoryStub struct{}

type publicProjectMetadataReadsQueryErrorRepositoryStub struct {
	publicProjectMetadataReadsRenderErrorRepositoryStub
}

type failingRequestBody struct{}

func (failingRequestBody) Read([]byte) (int, error) { return 0, errors.New("request body unavailable") }
func (failingRequestBody) Close() error             { return nil }

func (publicProjectMetadataReadsRenderErrorRepositoryStub) Cycles(context.Context, string) (json.RawMessage, error) {
	return nil, fmt.Errorf("%w: database unavailable", publicprojectmetadatareadsstore.ErrRender)
}

func (publicProjectMetadataReadsRenderErrorRepositoryStub) Labels(context.Context, string) (json.RawMessage, error) {
	return nil, fmt.Errorf("%w: database unavailable", publicprojectmetadatareadsstore.ErrRender)
}

func (publicProjectMetadataReadsRenderErrorRepositoryStub) Members(context.Context, string) (json.RawMessage, error) {
	return nil, fmt.Errorf("%w: database unavailable", publicprojectmetadatareadsstore.ErrRender)
}

func (publicProjectMetadataReadsRenderErrorRepositoryStub) Meta(context.Context, string) (json.RawMessage, error) {
	return nil, fmt.Errorf("%w: database unavailable", publicprojectmetadatareadsstore.ErrRender)
}

func (publicProjectMetadataReadsRenderErrorRepositoryStub) Modules(context.Context, string) (json.RawMessage, error) {
	return nil, fmt.Errorf("%w: database unavailable", publicprojectmetadatareadsstore.ErrRender)
}

func (publicProjectMetadataReadsRenderErrorRepositoryStub) ProjectAnchor(context.Context, string, uuid.UUID) (json.RawMessage, error) {
	return nil, fmt.Errorf("%w: database unavailable", publicprojectmetadatareadsstore.ErrRender)
}

func (publicProjectMetadataReadsQueryErrorRepositoryStub) ProjectAnchor(context.Context, string, uuid.UUID) (json.RawMessage, error) {
	return nil, errors.New("multiple public project settings objects")
}

func (publicProjectMetadataReadsRenderErrorRepositoryStub) Settings(context.Context, string) (json.RawMessage, error) {
	return nil, fmt.Errorf("%w: database unavailable", publicprojectmetadatareadsstore.ErrRender)
}

func (publicProjectMetadataReadsRenderErrorRepositoryStub) States(context.Context, string) (json.RawMessage, error) {
	return nil, fmt.Errorf("%w: database unavailable", publicprojectmetadatareadsstore.ErrRender)
}

func (publicProjectMetadataReadsHTTPStub) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, nil
}

type publicProjectMetadataThrottleStub struct {
	allowed               bool
	wait                  time.Duration
	expire                bool
	refresh               bool
	err                   error
	refreshErr            error
	calls                 int
	allowCalls            int
	throttleErr           error
	refreshCalls          int
	result                djangosession.SessionRefresh
	rotated               *djangosession.SessionRefresh
	userID                string
	earlyExpire           bool
	earlyRefresh          *djangosession.SessionRefresh
	earlyErr              error
	earlyCalls            int
	earlyStatus           int
	timezoneCalls         int
	deferredTimezoneCalls int
	timezoneErr           error
	inspectedCookie       string
	userTimezone          string
}

type publicProjectMetadataActivityStub struct {
	records chan publicprojectmetadatareadsstore.APIActivityLog
	err     error
}

func (stub *publicProjectMetadataActivityStub) Log(_ context.Context, record publicprojectmetadatareadsstore.APIActivityLog) error {
	stub.records <- record
	return stub.err
}

func (stub *publicProjectMetadataThrottleStub) AllowRequest(context.Context, *http.Request) (bool, time.Duration, bool, bool, *djangosession.SessionRefresh, string, error) {
	stub.calls++
	stub.allowCalls++
	if stub.throttleErr != nil {
		return false, 0, stub.expire, stub.refresh, stub.rotated, stub.userID, stub.throttleErr
	}
	if stub.timezoneErr != nil {
		return false, 0, stub.expire, stub.refresh, stub.rotated, stub.userID, stub.timezoneErr
	}
	return stub.allowed, stub.wait, stub.expire, stub.refresh, stub.rotated, stub.userID, stub.err
}

func (stub *publicProjectMetadataThrottleStub) AllowRequestWithoutTimezone(context.Context, *http.Request) (bool, time.Duration, bool, bool, *djangosession.SessionRefresh, string, error) {
	stub.calls++
	stub.allowCalls++
	if stub.throttleErr != nil {
		return false, 0, stub.expire, stub.refresh, stub.rotated, stub.userID, stub.throttleErr
	}
	return stub.allowed, stub.wait, stub.expire, stub.refresh, stub.rotated, stub.userID, stub.err
}

func (stub *publicProjectMetadataThrottleStub) FinalizeUnaccessedSession(_ context.Context, _ *http.Request, status int) (bool, *djangosession.SessionRefresh, error) {
	stub.earlyCalls++
	stub.earlyStatus = status
	return stub.earlyExpire, stub.earlyRefresh, stub.earlyErr
}

func (stub *publicProjectMetadataThrottleStub) InspectSession(context.Context, *http.Request) (bool, bool, *djangosession.SessionRefresh, string, error) {
	stub.calls++
	return stub.expire, stub.refresh, stub.rotated, stub.userID, stub.err
}

func (stub *publicProjectMetadataThrottleStub) InspectSessionWithTimezone(ctx context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, string, error) {
	stub.timezoneCalls++
	if stub.timezoneErr != nil {
		return stub.expire, stub.refresh, stub.rotated, stub.userID, stub.timezoneErr
	}
	return stub.InspectSession(ctx, request)
}
func (stub *publicProjectMetadataThrottleStub) InspectReadServiceSession(_ context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	stub.timezoneCalls++
	stub.calls++
	if cookie, err := request.Cookie("session-id"); err == nil {
		stub.inspectedCookie = cookie.Value
	}
	return stub.expire, stub.refresh, stub.rotated, djangosession.Principal{
		UserID:       stub.userID,
		UserTimezone: stub.userTimezone,
	}, stub.err
}

func (stub *publicProjectMetadataThrottleStub) InspectReadServiceSessionWithoutTimezone(_ context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	stub.deferredTimezoneCalls++
	stub.calls++
	if cookie, err := request.Cookie("session-id"); err == nil {
		stub.inspectedCookie = cookie.Value
	}
	return stub.expire, stub.refresh, stub.rotated, djangosession.Principal{
		UserID:       stub.userID,
		UserTimezone: stub.userTimezone,
	}, stub.err
}

func (stub *publicProjectMetadataThrottleStub) RefreshSession(_ context.Context, _ *http.Request, prepared djangosession.SessionRefresh) (djangosession.SessionRefresh, error) {
	stub.refreshCalls++
	if stub.result.SessionKey == "" {
		stub.result.SessionKey = prepared.SessionKey
	}
	return stub.result, stub.refreshErr
}

func (publicProjectMetadataReadsHTTPStub) Cycles(context.Context, *publicprojectmetadatareadsapi.CyclesPayload) ([]*publicprojectmetadatareadsapi.PublicProjectCollectionItem, error) {
	return []*publicprojectmetadatareadsapi.PublicProjectCollectionItem{{
		ID: "00000000-0000-4000-8000-000000000003", Name: "Sprint 1",
	}}, nil
}

func (publicProjectMetadataReadsHTTPStub) Labels(context.Context, *publicprojectmetadatareadsapi.LabelsPayload) ([]*publicprojectmetadatareadsapi.PublicProjectLabel, error) {
	return []*publicprojectmetadatareadsapi.PublicProjectLabel{{
		ID: "00000000-0000-4000-8000-000000000001", Name: "Bug", Color: "#FF0000", Parent: loom.NullValue[string](),
	}}, nil
}

func (publicProjectMetadataReadsHTTPStub) Members(context.Context, *publicprojectmetadatareadsapi.MembersPayload) ([]*publicprojectmetadatareadsapi.PublicProjectMember, error) {
	return []*publicprojectmetadatareadsapi.PublicProjectMember{{
		ID:                "00000000-0000-4000-8000-000000000005",
		Member:            loom.NullValue[string](),
		MemberDisplayName: loom.NullValue[string](),
		MemberAvatar:      loom.NullValue[string](),
	}}, nil
}

func (publicProjectMetadataReadsHTTPStub) Meta(context.Context, *publicprojectmetadatareadsapi.MetaPayload) (*publicprojectmetadatareadsapi.PublicProjectMeta, error) {
	return &publicprojectmetadatareadsapi.PublicProjectMeta{
		ID:          "00000000-0000-4000-8000-000000000006",
		Identifier:  "PMETA",
		Name:        "Public Project",
		CoverImage:  loom.NullValue[string](),
		IconProp:    loom.NullableValue(loom.JSONValue(`{"name":"rocket"}`)),
		Emoji:       loom.NullValue[string](),
		Description: "",
	}, nil
}

func (publicProjectMetadataReadsHTTPStub) Modules(context.Context, *publicprojectmetadatareadsapi.ModulesPayload) ([]*publicprojectmetadatareadsapi.PublicProjectCollectionItem, error) {
	return []*publicprojectmetadatareadsapi.PublicProjectCollectionItem{{
		ID: "00000000-0000-4000-8000-000000000004", Name: "Launch",
	}}, nil
}

func (stub publicProjectMetadataReadsHTTPStub) ProjectAnchor(ctx context.Context, _ *publicprojectmetadatareadsapi.ProjectAnchorPayload) (*publicprojectmetadatareadsapi.PublicProjectSettings, error) {
	return stub.Settings(ctx, nil)
}

func (publicProjectMetadataReadsHTTPStub) Settings(context.Context, *publicprojectmetadatareadsapi.SettingsPayload) (*publicprojectmetadatareadsapi.PublicProjectSettings, error) {
	return &publicprojectmetadatareadsapi.PublicProjectSettings{
		ID:             "00000000-0000-4000-8000-000000000007",
		ProjectDetails: loom.NullValue[publicprojectmetadatareadsapi.PublicProjectSettingsProject](),
		WorkspaceDetail: &publicprojectmetadatareadsapi.PublicProjectSettingsWorkspace{
			Name: "Demo", Slug: "demo", ID: "00000000-0000-4000-8000-000000000001", LogoURL: loom.NullValue[string](),
		},
		CreatedAt: "2026-01-02T03:04:05Z", UpdatedAt: "2026-01-02T03:04:05Z",
		DeletedAt: loom.NullValue[string](), EntityIdentifier: loom.NullValue[string](), EntityName: loom.NullableValue("project"),
		Anchor: "example", ViewProps: loom.JSONValue(`{}`),
		CreatedBy: loom.NullValue[string](), UpdatedBy: loom.NullValue[string](),
		Workspace: "00000000-0000-4000-8000-000000000001", Project: loom.NullValue[string](), Intake: loom.NullValue[string](),
	}, nil
}

func (stub publicProjectMetadataReadsUserContextHTTPStub) Settings(ctx context.Context, payload *publicprojectmetadatareadsapi.SettingsPayload) (*publicprojectmetadatareadsapi.PublicProjectSettings, error) {
	*stub.userID = publicprojectmetadatareadsstore.UserID(ctx)
	return stub.publicProjectMetadataReadsHTTPStub.Settings(ctx, payload)
}

func (publicProjectMetadataReadsHTTPStub) States(context.Context, *publicprojectmetadatareadsapi.StatesPayload) ([]*publicprojectmetadatareadsapi.PublicProjectState, error) {
	return []*publicprojectmetadatareadsapi.PublicProjectState{{
		ID: "00000000-0000-4000-8000-000000000002", Name: "Backlog", Group: "backlog", Color: "#60646C", Sequence: 100,
	}}, nil
}

func (projectLabelReadsHTTPStub) label() *projectlabelreadsapi.ProjectLabelRead {
	return &projectlabelreadsapi.ProjectLabelRead{
		Parent:      loom.NullValue[string](),
		Name:        "Bug",
		Color:       "#FF0000",
		ID:          "00000000-0000-4000-8000-000000000001",
		ProjectID:   "00000000-0000-4000-8000-000000000002",
		WorkspaceID: "00000000-0000-4000-8000-000000000003",
		SortOrder:   100,
	}
}

func (stub projectLabelReadsHTTPStub) List(context.Context, *projectlabelreadsapi.ListPayload) ([]*projectlabelreadsapi.ProjectLabelRead, error) {
	return []*projectlabelreadsapi.ProjectLabelRead{stub.label()}, nil
}

func (stub projectLabelReadsHTTPStub) Detail(context.Context, *projectlabelreadsapi.DetailPayload) (*projectlabelreadsapi.ProjectLabelRead, error) {
	return stub.label(), nil
}

func (projectLabelReadsHTTPStub) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, nil
}

func (projectStateReadsHTTPStub) state() *projectstatereadsapi.ProjectStateRead {
	order := 1.0
	return &projectstatereadsapi.ProjectStateRead{
		ID:          "00000000-0000-4000-8000-000000000001",
		ProjectID:   "00000000-0000-4000-8000-000000000002",
		WorkspaceID: "00000000-0000-4000-8000-000000000003",
		Name:        "Backlog",
		Color:       "#60646C",
		Group:       "backlog",
		Default:     true,
		Description: "First backlog state",
		Sequence:    100,
		Order:       &order,
	}
}

func (stub projectStateReadsHTTPStub) List(_ context.Context, payload *projectstatereadsapi.ListPayload) (loom.JSONValue, error) {
	if payload.Grouped != nil && *payload.Grouped == "true" {
		return json.Marshal(map[string][]*projectstatereadsapi.ProjectStateRead{"backlog": {stub.state()}})
	}
	return json.Marshal([]*projectstatereadsapi.ProjectStateRead{stub.state()})
}

func (stub projectStateReadsHTTPStub) Detail(context.Context, *projectstatereadsapi.DetailPayload) (*projectstatereadsapi.ProjectStateRead, error) {
	state := stub.state()
	state.Order = nil
	return state, nil
}

func (stub projectStateReadsHTTPStub) Intake(context.Context, *projectstatereadsapi.IntakePayload) (*projectstatereadsapi.ProjectStateRead, error) {
	state := stub.state()
	state.Group = "triage"
	state.Order = nil
	return state, nil
}

func (stub projectStateReadsHTTPStub) Create(context.Context, *projectstatereadsapi.CreatePayload, io.ReadCloser) (*projectstatereadsapi.ProjectStateWritesAppResponseMetadata, io.ReadCloser, error) {
	return &projectstatereadsapi.ProjectStateWritesAppResponseMetadata{ContentType: "application/json"}, io.NopCloser(strings.NewReader(`{"name":"Created"}`)), nil
}

func (stub projectStateReadsHTTPStub) Update(context.Context, *projectstatereadsapi.UpdatePayload, io.ReadCloser) (*projectstatereadsapi.ProjectStateWritesAppResponseMetadata, io.ReadCloser, error) {
	return &projectstatereadsapi.ProjectStateWritesAppResponseMetadata{ContentType: "application/json"}, io.NopCloser(strings.NewReader(`{"name":"Updated"}`)), nil
}

func (projectStateReadsHTTPStub) Delete(context.Context, *projectstatereadsapi.DeletePayload) error {
	return nil
}

func (projectStateReadsHTTPStub) MarkDefault(context.Context, *projectstatereadsapi.MarkDefaultPayload) error {
	return nil
}

func (projectStateReadsHTTPStub) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, nil
}

func (stub projectStateReadsHTTPStub) AuthorizeMetadata(context.Context, string) error {
	if stub.metadataCalls != nil {
		*stub.metadataCalls++
	}
	return nil
}

func (stub projectMembershipReadsHTTPStub) Membership(context.Context, *projectmembershipreadsapi.MembershipPayload) (*projectmembershipreadsapi.ProjectMembership, error) {
	if stub.readCalls != nil {
		*stub.readCalls++
	}
	return &projectmembershipreadsapi.ProjectMembership{
		ID:           "00000000-0000-4000-8000-000000000001",
		CreatedAt:    "2026-09-01T00:00:00Z",
		UpdatedAt:    "2026-09-01T00:00:00Z",
		CreatedBy:    loom.NullValue[string](),
		UpdatedBy:    loom.NullValue[string](),
		DeletedAt:    loom.NullValue[string](),
		Comment:      loom.NullValue[string](),
		Role:         15,
		ViewProps:    loom.JSONValue(`{}`),
		DefaultProps: loom.JSONValue(`{}`),
		Preferences:  loom.JSONValue(`{}`),
		SortOrder:    65535,
		IsActive:     true,
		Member: &projectmembershipreadsapi.ProjectMembershipUser{
			ID:          "00000000-0000-4000-8000-000000000002",
			FirstName:   "Project",
			LastName:    "Member",
			Avatar:      "",
			AvatarURL:   loom.NullValue[string](),
			DisplayName: "Project Member",
		},
		Project: &projectmembershipreadsapi.ProjectMembershipProject{
			ID:            "00000000-0000-4000-8000-000000000003",
			Identifier:    "MEM",
			Name:          "Membership Project",
			CoverImage:    loom.NullValue[string](),
			CoverImageURL: loom.NullValue[string](),
			LogoProps:     loom.JSONValue(`{}`),
			Description:   "Membership read contract",
		},
		Workspace: &projectmembershipreadsapi.ProjectMembershipWorkspace{
			ID:      "00000000-0000-4000-8000-000000000004",
			Name:    "Membership",
			Slug:    "acme",
			LogoURL: loom.NullValue[string](),
		},
	}, nil
}

func (stub projectMembershipReadsHTTPStub) Roles(context.Context, *projectmembershipreadsapi.RolesPayload) (map[string]int, error) {
	if stub.readCalls != nil {
		*stub.readCalls++
	}
	return map[string]int{"00000000-0000-4000-8000-000000000003": 15}, stub.rolesError
}

func projectMembershipWorkspaceDirectoryHTTPValue() *projectmembershipreadsapi.WorkspaceMembershipDirectory {
	return &projectmembershipreadsapi.WorkspaceMembershipDirectory{
		ID:        "00000000-0000-4000-8000-000000000011",
		CreatedAt: "2026-09-01T00:00:00Z", UpdatedAt: "2026-09-01T00:00:00Z",
		CreatedBy: loom.NullValue[string](), UpdatedBy: loom.NullValue[string](), DeletedAt: loom.NullValue[string](),
		Workspace: "00000000-0000-4000-8000-000000000004", Role: 15, IsActive: true,
		Member: &projectmembershipreadsapi.ProjectMembershipUser{
			ID: "00000000-0000-4000-8000-000000000012", FirstName: "Project", LastName: "Member",
			Avatar: "", AvatarURL: loom.NullValue[string](), IsBot: false, DisplayName: "Project Member",
		},
		CompanyRole: loom.NullValue[string](), ViewProps: loom.JSONValue(`{}`),
		DefaultProps: loom.JSONValue(`{}`), IssueProps: loom.JSONValue(`{}`),
		ExploredFeatures:        loom.JSONValue(`{}`),
		GettingStartedChecklist: loom.JSONValue(`{}`), Tips: loom.JSONValue(`{}`),
	}
}

func projectMembershipRoleHTTPValue() *projectmembershipreadsapi.ProjectMembershipRole {
	projectID := "00000000-0000-4000-8000-000000000003"
	return &projectmembershipreadsapi.ProjectMembershipRole{
		ID: "00000000-0000-4000-8000-000000000012", Role: 15,
		Member: "00000000-0000-4000-8000-000000000011", Project: projectID,
		OriginalRole: 15, CreatedAt: "2026-09-01T00:00:00Z",
	}
}

func (stub projectMembershipReadsHTTPStub) WorkspaceMembers(_ context.Context, payload *projectmembershipreadsapi.WorkspaceMembersPayload) ([]*projectmembershipreadsapi.WorkspaceMembershipDirectory, error) {
	if stub.readCalls != nil {
		*stub.readCalls++
	}
	if stub.searchCapture != nil && payload.Search != nil {
		*stub.searchCapture = *payload.Search
	}
	return []*projectmembershipreadsapi.WorkspaceMembershipDirectory{projectMembershipWorkspaceDirectoryHTTPValue()}, stub.workspaceMembersError
}

func (stub projectMembershipReadsHTTPStub) WorkspaceMember(context.Context, *projectmembershipreadsapi.WorkspaceMemberPayload) (*projectmembershipreadsapi.WorkspaceMembershipDirectory, error) {
	return projectMembershipWorkspaceDirectoryHTTPValue(), stub.workspaceMemberError
}

func (stub projectMembershipReadsHTTPStub) WorkspaceProjectMembers(context.Context, *projectmembershipreadsapi.WorkspaceProjectMembersPayload) (map[string][]*projectmembershipreadsapi.WorkspaceProjectMembershipRole, error) {
	if stub.readCalls != nil {
		*stub.readCalls++
	}
	return map[string][]*projectmembershipreadsapi.WorkspaceProjectMembershipRole{"00000000-0000-4000-8000-000000000003": {}}, stub.workspaceProjectMembersError
}

func (projectMembershipReadsHTTPStub) ProjectMembers(context.Context, *projectmembershipreadsapi.ProjectMembersPayload) ([]*projectmembershipreadsapi.ProjectMembershipRole, error) {
	return []*projectmembershipreadsapi.ProjectMembershipRole{projectMembershipRoleHTTPValue()}, nil
}

func (projectMembershipReadsHTTPStub) ProjectMember(context.Context, *projectmembershipreadsapi.ProjectMemberPayload) (*projectmembershipreadsapi.ProjectMembershipFullDirectoryOrProjectMembershipGuestDirectory, error) {
	result := projectmembershipreadsapi.NewProjectMembershipFullDirectoryOrProjectMembershipGuestDirectoryProjectMembershipGuestDirectory(
		&projectmembershipreadsapi.ProjectMembershipGuestDirectory{
			ID: "00000000-0000-4000-8000-000000000012", Role: 15,
			Member:       "00000000-0000-4000-8000-000000000011",
			Project:      "00000000-0000-4000-8000-000000000003",
			OriginalRole: 15, CreatedAt: "2026-09-01T00:00:00Z",
		},
	)
	return &result, nil
}

func (projectMembershipReadsHTTPStub) Preference(context.Context, *projectmembershipreadsapi.PreferencePayload) (*projectmembershipreadsapi.ProjectMembershipPreference, error) {
	return &projectmembershipreadsapi.ProjectMembershipPreference{
		Preferences: loom.JSONValue(`{}`),
		ProjectID:   "00000000-0000-4000-8000-000000000003",
		MemberID:    "00000000-0000-4000-8000-000000000011",
		WorkspaceID: "00000000-0000-4000-8000-000000000004",
	}, nil
}

func (projectMembershipReadsHTTPStub) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	if key == "" {
		return ctx, &projectmembershipreadsapi.ProjectMembershipReadsUnauthenticated{
			Detail: "Authentication credentials were not provided.",
		}
	}
	return ctx, nil
}

func (stub projectMembershipReadsHTTPStub) AuthorizeMetadata(
	ctx context.Context,
	sessionID string,
	_ string,
	requireWorkspaceMembership bool,
) error {
	if stub.metadataCalls != nil {
		*stub.metadataCalls++
	}
	if _, err := stub.APIKeyAuth(ctx, sessionID, &security.APIKeyScheme{}); err != nil {
		return err
	}
	if requireWorkspaceMembership {
		return stub.metadataError
	}
	return nil
}

type analyticViewsHTTPStub struct{}

type workspaceFavoritesHTTPStub struct{}

func workspaceFavoriteHTTPValue() *workspacefavoritesapi.WorkspaceFavorite {
	return &workspacefavoritesapi.WorkspaceFavorite{
		ID:          "00000000-0000-4000-8000-000000000011",
		EntityType:  "folder",
		Name:        loom.NullableValue("Projects"),
		IsFolder:    true,
		Sequence:    65535,
		WorkspaceID: "00000000-0000-4000-8000-000000000012",
	}
}
func (workspaceFavoritesHTTPStub) List(context.Context, *workspacefavoritesapi.ListPayload) ([]*workspacefavoritesapi.WorkspaceFavorite, error) {
	return []*workspacefavoritesapi.WorkspaceFavorite{workspaceFavoriteHTTPValue()}, nil
}
func (workspaceFavoritesHTTPStub) Create(context.Context, *workspacefavoritesapi.CreatePayload) (*workspacefavoritesapi.WorkspaceFavorite, error) {
	return workspaceFavoriteHTTPValue(), nil
}
func (workspaceFavoritesHTTPStub) PatchCollection(context.Context, *workspacefavoritesapi.PatchCollectionPayload) (*workspacefavoritesapi.WorkspaceFavorite, error) {
	return nil, &workspacefavoritesapi.WorkspaceFavoriteInternal{Message: "Something went wrong please try again later"}
}
func (workspaceFavoritesHTTPStub) DeleteCollection(context.Context, *workspacefavoritesapi.DeleteCollectionPayload) error {
	return &workspacefavoritesapi.WorkspaceFavoriteInternal{Message: "Something went wrong please try again later"}
}
func (workspaceFavoritesHTTPStub) Retrieve(context.Context, *workspacefavoritesapi.RetrievePayload) (*workspacefavoritesapi.WorkspaceFavorite, error) {
	return nil, &workspacefavoritesapi.WorkspaceFavoriteInternal{Message: "Something went wrong please try again later"}
}
func (workspaceFavoritesHTTPStub) CreateDetail(context.Context, *workspacefavoritesapi.CreateDetailPayload) (*workspacefavoritesapi.WorkspaceFavorite, error) {
	return nil, &workspacefavoritesapi.WorkspaceFavoriteInternal{Message: "Something went wrong please try again later"}
}
func (workspaceFavoritesHTTPStub) Patch(context.Context, *workspacefavoritesapi.PatchPayload) (*workspacefavoritesapi.WorkspaceFavorite, error) {
	return workspaceFavoriteHTTPValue(), nil
}
func (workspaceFavoritesHTTPStub) Delete(context.Context, *workspacefavoritesapi.DeletePayload) error {
	return nil
}
func (workspaceFavoritesHTTPStub) Group(context.Context, *workspacefavoritesapi.GroupPayload) ([]*workspacefavoritesapi.WorkspaceFavorite, error) {
	return []*workspacefavoritesapi.WorkspaceFavorite{workspaceFavoriteHTTPValue()}, nil
}
func (workspaceFavoritesHTTPStub) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	if key == "" {
		return ctx, &workspacefavoritesapi.WorkspaceFavoriteUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	return ctx, nil
}

func analyticHTTPView() *analyticviewsapi.AnalyticView {
	return &analyticviewsapi.AnalyticView{
		ID:          "00000000-0000-4000-8000-000000000011",
		CreatedAt:   "2026-08-26T00:00:00Z",
		UpdatedAt:   "2026-08-26T00:00:00Z",
		Workspace:   "00000000-0000-4000-8000-000000000002",
		Name:        "Issues by state",
		Description: "Parity",
		Query:       loom.JSONValue(`{}`),
		QueryDict:   loom.JSONValue(`{"x_axis":"state_id","y_axis":"issue_count"}`),
	}
}
func (analyticViewsHTTPStub) List(context.Context, *analyticviewsapi.ListPayload) ([]*analyticviewsapi.AnalyticView, error) {
	return []*analyticviewsapi.AnalyticView{analyticHTTPView()}, nil
}
func (analyticViewsHTTPStub) Create(context.Context, *analyticviewsapi.CreatePayload) (*analyticviewsapi.AnalyticView, error) {
	return analyticHTTPView(), nil
}
func (analyticViewsHTTPStub) Retrieve(context.Context, *analyticviewsapi.RetrievePayload) (*analyticviewsapi.AnalyticView, error) {
	return analyticHTTPView(), nil
}
func (analyticViewsHTTPStub) Patch(context.Context, *analyticviewsapi.PatchPayload) (*analyticviewsapi.AnalyticView, error) {
	return analyticHTTPView(), nil
}
func (analyticViewsHTTPStub) Delete(context.Context, *analyticviewsapi.DeletePayload) error {
	return nil
}
func (analyticViewsHTTPStub) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, nil
}

func (workspaceHomePreferencesHTTPStub) List(context.Context, *workspacehomepreferencesapi.ListPayload) ([]*workspacehomepreferencesapi.WorkspaceHomePreferenceList, error) {
	return []*workspacehomepreferencesapi.WorkspaceHomePreferenceList{}, nil
}
func (workspaceHomePreferencesHTTPStub) PatchCollection(context.Context, *workspacehomepreferencesapi.PatchCollectionPayload) ([]*workspacehomepreferencesapi.WorkspaceHomePreferenceList, error) {
	return nil, &workspacehomepreferencesapi.WorkspaceHomePreferenceInternal{Message: "Something went wrong please try again later"}
}
func (workspaceHomePreferencesHTTPStub) Retrieve(context.Context, *workspacehomepreferencesapi.RetrievePayload) (*workspacehomepreferencesapi.WorkspaceHomePreference, error) {
	return nil, &workspacehomepreferencesapi.WorkspaceHomePreferenceInternal{Message: "Something went wrong please try again later"}
}
func (workspaceHomePreferencesHTTPStub) Patch(context.Context, *workspacehomepreferencesapi.PatchPayload) (*workspacehomepreferencesapi.WorkspaceHomePreference, error) {
	return &workspacehomepreferencesapi.WorkspaceHomePreference{Key: "quick_links"}, nil
}
func (workspaceHomePreferencesHTTPStub) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	if key == "" {
		return ctx, &workspacehomepreferencesapi.WorkspaceHomePreferenceUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	return ctx, nil
}

func (workspaceThemesHTTPStub) List(context.Context, *workspacethemesapi.ListPayload) ([]*workspacethemesapi.WorkspaceTheme, error) {
	return []*workspacethemesapi.WorkspaceTheme{}, nil
}
func (workspaceThemesHTTPStub) theme() *workspacethemesapi.WorkspaceTheme {
	return &workspacethemesapi.WorkspaceTheme{
		ID: "00000000-0000-4000-8000-000000000042", Workspace: "00000000-0000-4000-8000-000000000031", Actor: "00000000-0000-4000-8000-000000000010",
		CreatedAt: "2026-08-26T21:00:00Z", UpdatedAt: "2026-08-26T21:00:00Z", Name: "Dark", Colors: loom.JSONValue(`{"primary":"#111111"}`),
	}
}
func (stub workspaceThemesHTTPStub) Create(context.Context, *workspacethemesapi.CreatePayload) (*workspacethemesapi.WorkspaceTheme, error) {
	return stub.theme(), nil
}
func (stub workspaceThemesHTTPStub) Retrieve(context.Context, *workspacethemesapi.RetrievePayload) (*workspacethemesapi.WorkspaceTheme, error) {
	return stub.theme(), nil
}
func (stub workspaceThemesHTTPStub) Patch(context.Context, *workspacethemesapi.PatchPayload) (*workspacethemesapi.WorkspaceTheme, error) {
	return stub.theme(), nil
}
func (workspaceThemesHTTPStub) Delete(context.Context, *workspacethemesapi.DeletePayload) error {
	return nil
}
func (workspaceThemesHTTPStub) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	if key == "" {
		return ctx, &workspacethemesapi.WorkspaceThemeUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	return ctx, nil
}

func (stub *userAPITokensHTTPStub) token() *userapitokensapi.APIToken {
	return &userapitokensapi.APIToken{
		Token: "plane_api_secret", ID: "00000000-0000-4000-8000-000000000031",
		CreatedAt: "2026-08-26T21:00:00Z", UpdatedAt: "2026-08-26T21:00:00Z",
		Label: "token", Description: "", IsActive: true,
		User: "00000000-0000-4000-8000-000000000010", UserType: 0,
		IsService: false, AllowedRateLimit: "60/min",
	}
}

func (stub *userAPITokensHTTPStub) List(context.Context, *userapitokensapi.ListPayload) ([]*userapitokensapi.APITokenRead, error) {
	return []*userapitokensapi.APITokenRead{}, nil
}
func (stub *userAPITokensHTTPStub) Create(context.Context, *userapitokensapi.CreatePayload) (*userapitokensapi.APIToken, error) {
	return stub.token(), nil
}
func (stub *userAPITokensHTTPStub) PatchCollection(context.Context, *userapitokensapi.PatchCollectionPayload) error {
	return &userapitokensapi.APITokenInternalError{Message: "Something went wrong please try again later"}
}
func (stub *userAPITokensHTTPStub) DeleteCollection(context.Context, *userapitokensapi.DeleteCollectionPayload) error {
	return &userapitokensapi.APITokenInternalError{Message: "Something went wrong please try again later"}
}
func (stub *userAPITokensHTTPStub) Retrieve(context.Context, *userapitokensapi.RetrievePayload) (*userapitokensapi.APITokenRead, error) {
	token := stub.token()
	return &userapitokensapi.APITokenRead{ID: token.ID, CreatedAt: token.CreatedAt, UpdatedAt: token.UpdatedAt, Label: token.Label, Description: token.Description, IsActive: token.IsActive, User: token.User, UserType: token.UserType, IsService: token.IsService, AllowedRateLimit: token.AllowedRateLimit}, nil
}
func (stub *userAPITokensHTTPStub) Patch(context.Context, *userapitokensapi.PatchPayload) (*userapitokensapi.APIToken, error) {
	return stub.token(), nil
}
func (stub *userAPITokensHTTPStub) Delete(context.Context, *userapitokensapi.DeletePayload) error {
	return nil
}
func (stub *userAPITokensHTTPStub) PostDetail(context.Context, *userapitokensapi.PostDetailPayload) error {
	return &userapitokensapi.APITokenInternalError{Message: "Something went wrong please try again later"}
}
func (stub *userAPITokensHTTPStub) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	stub.apiKey = key
	if key == "" {
		return ctx, &userapitokensapi.APITokenUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	return ctx, nil
}

func (stub *instanceAdminSessionHTTPStub) user() *instanceadminsessionapi.InstanceAdminUser {
	return &instanceadminsessionapi.InstanceAdminUser{
		ID: "00000000-0000-4000-8000-000000000010", Avatar: "", AvatarURL: loom.NullValue[string](), CoverImage: loom.NullValue[string](),
		DateJoined: "2026-08-26T21:00:00Z", DisplayName: "Admin", Email: loom.NullableValue("admin@example.com"),
		FirstName: "Admin", LastName: "User", IsActive: true, UserTimezone: "UTC", Username: "admin@example.com",
	}
}
func (stub *instanceAdminSessionHTTPStub) Me(context.Context, *instanceadminsessionapi.MePayload) (*instanceadminsessionapi.InstanceAdminUser, error) {
	if stub.forbidden {
		return nil, &instanceadminsessionapi.InstanceAdminSessionForbidden{Detail: "You do not have permission to perform this action."}
	}
	return stub.user(), nil
}
func (stub *instanceAdminSessionHTTPStub) Session(_ context.Context, payload *instanceadminsessionapi.SessionPayload) (*instanceadminsessionapi.SessionResult, error) {
	if payload.SessionID != nil {
		stub.sessionKey = *payload.SessionID
	}
	if !stub.authenticated {
		return &instanceadminsessionapi.SessionResult{IsAuthenticated: false}, nil
	}
	return &instanceadminsessionapi.SessionResult{IsAuthenticated: true, User: stub.user()}, nil
}
func (stub *instanceAdminSessionHTTPStub) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	stub.apiKey = key
	if key == "" {
		return ctx, &instanceadminsessionapi.InstanceAdminSessionUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	return ctx, nil
}

func (stub *userAccountsHTTPStub) account() *useraccountsapi.UserAccount {
	return &useraccountsapi.UserAccount{
		ID:                    "00000000-0000-4000-8000-000000000021",
		CreatedAt:             "2026-08-26T21:00:00Z",
		UpdatedAt:             "2026-08-26T21:00:00Z",
		User:                  "00000000-0000-4000-8000-000000000010",
		ProviderAccountID:     "provider-1",
		Provider:              "github",
		AccessToken:           "secret",
		AccessTokenExpiredAt:  loom.NullValue[string](),
		RefreshToken:          loom.NullValue[string](),
		RefreshTokenExpiredAt: loom.NullValue[string](),
		LastConnectedAt:       "2026-08-26T21:00:00Z",
		IDToken:               "",
		Metadata:              loom.JSONValue(`{}`),
	}
}

func (stub *userAccountsHTTPStub) List(context.Context, *useraccountsapi.ListPayload) ([]*useraccountsapi.UserAccount, error) {
	return []*useraccountsapi.UserAccount{stub.account()}, nil
}

func (stub *userAccountsHTTPStub) Retrieve(_ context.Context, payload *useraccountsapi.RetrievePayload) (*useraccountsapi.UserAccount, error) {
	if payload.Pk != "00000000-0000-4000-8000-000000000021" {
		return nil, &useraccountsapi.UserAccountNotFound{Message: "The required object does not exist."}
	}
	return stub.account(), nil
}

func (stub *userAccountsHTTPStub) Delete(_ context.Context, payload *useraccountsapi.DeletePayload) error {
	stub.deleted = payload.Pk
	return nil
}

func (stub *userAccountsHTTPStub) DeleteCollection(context.Context, *useraccountsapi.DeleteCollectionPayload) error {
	return &useraccountsapi.UserAccountInternalError{Message: "Something went wrong please try again later"}
}

func (stub *userAccountsHTTPStub) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	stub.apiKey = key
	if key == "" {
		return ctx, &useraccountsapi.UserAccountUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	return ctx, nil
}

func (stub *userNotificationPreferencesHTTPStub) Get(context.Context, *usernotificationpreferencesapi.GetPayload) (*usernotificationpreferencesapi.UserNotificationPreference, error) {
	return &usernotificationpreferencesapi.UserNotificationPreference{ID: "00000000-0000-0000-0000-000000000001"}, nil
}

func (stub *userNotificationPreferencesHTTPStub) Patch(_ context.Context, payload *usernotificationpreferencesapi.PatchPayload) (*usernotificationpreferencesapi.UserNotificationPreference, error) {
	stub.patch = payload
	return &usernotificationpreferencesapi.UserNotificationPreference{ID: "00000000-0000-0000-0000-000000000001"}, nil
}

func (stub *userNotificationPreferencesHTTPStub) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	stub.apiKey = key
	if key == "" {
		return ctx, &usernotificationpreferencesapi.UserNotificationPreferenceUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	return ctx, nil
}

func (stub *userProfileFlagsHTTPStub) UpdateOnboarded(_ context.Context, payload *userprofileflagsapi.UpdateOnboardedPayload) (*userprofileflagsapi.UserProfileFlagResult, error) {
	if payload.IsOnboarded != nil {
		stub.value = *payload.IsOnboarded
	}
	return stub.profileFlagResult()
}

func (stub *userProfileFlagsHTTPStub) UpdateTourCompleted(_ context.Context, payload *userprofileflagsapi.UpdateTourCompletedPayload) (*userprofileflagsapi.UserProfileFlagResult, error) {
	if payload.IsTourCompleted != nil {
		stub.value = *payload.IsTourCompleted
	}
	return stub.profileFlagResult()
}

func (stub *userProfileFlagsHTTPStub) profileFlagResult() (*userprofileflagsapi.UserProfileFlagResult, error) {
	if stub.missing {
		return nil, &userprofileflagsapi.UserProfileFlagNotFound{Message: "The required object does not exist."}
	}
	if stub.failure {
		return nil, &userprofileflagsapi.UserProfileFlagInternalError{Message: "Something went wrong please try again later"}
	}
	return &userprofileflagsapi.UserProfileFlagResult{Message: "Updated successfully"}, nil
}

func (stub *userProfileFlagsHTTPStub) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	stub.apiKey = key
	if key == "" {
		return ctx, &userprofileflagsapi.UserProfileFlagUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	return ctx, nil
}

func (stub *userInstanceAdminHTTPStub) Retrieve(context.Context, *userinstanceadminapi.RetrievePayload) (*userinstanceadminapi.UserInstanceAdminResult, error) {
	if stub.failure {
		return nil, &userinstanceadminapi.UserInstanceAdminInternalError{Message: "Something went wrong please try again later"}
	}
	return &userinstanceadminapi.UserInstanceAdminResult{IsInstanceAdmin: stub.isAdmin}, nil
}

func (stub *userInstanceAdminHTTPStub) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	stub.apiKey = key
	if key == "" {
		return ctx, &userinstanceadminapi.UserInstanceAdminUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	return ctx, nil
}

func (stub *instanceWorkspaceSlugHTTPStub) Check(_ context.Context, payload *instanceworkspaceapi.CheckPayload) (*instanceworkspaceapi.InstanceWorkspaceSlugAvailability, error) {
	if payload.Slug == nil || *payload.Slug == "" {
		return nil, &instanceworkspaceapi.InstanceWorkspaceSlugError{Message: "Workspace Slug is required"}
	}
	if stub.checkFailure {
		return nil, &instanceworkspaceapi.InstanceWorkspaceSlugInternalError{Message: "Something went wrong please try again later"}
	}
	return &instanceworkspaceapi.InstanceWorkspaceSlugAvailability{Status: true}, nil
}

func (stub *instanceWorkspaceSlugHTTPStub) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	stub.apiKey = key
	if key == "" {
		return ctx, &instanceworkspaceapi.InstanceWorkspaceSlugUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	if stub.forbidden {
		return ctx, &instanceworkspaceapi.InstanceWorkspaceSlugForbidden{Detail: "You do not have permission to perform this action."}
	}
	return ctx, nil
}

func (stub workspaceSlugHTTPStub) Check(_ context.Context, payload *workspaceapi.CheckPayload) (*workspaceapi.WorkspaceSlugAvailability, error) {
	if payload.Slug == nil || *payload.Slug == "" {
		return nil, &workspaceapi.WorkspaceSlugError{Message: "Workspace Slug is required"}
	}
	if stub.failure {
		return nil, &workspaceapi.WorkspaceSlugInternalError{Message: "Something went wrong please try again later"}
	}
	return &workspaceapi.WorkspaceSlugAvailability{Status: true}, nil
}

func (stub workspaceSlugHTTPStub) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	if stub.unauthenticated {
		return ctx, &workspaceapi.WorkspaceSlugUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	return ctx, nil
}

func TestWorkspaceSlugCheckHTTP(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	stub := workspaceSlugHTTPStub{}
	handler := newHTTPHandlerWithStickiesAndWorkspaceSlugCheck(ctx, endpoints, false, nil, nil, workspaceapi.NewEndpoints(stub))
	for _, test := range []struct {
		name, query, wantBody string
		wantStatus            int
	}{
		{name: "missing", wantStatus: http.StatusBadRequest, wantBody: `{"error":"Workspace Slug is required"}`},
		{name: "last repeated value wins empty", query: "?slug=next&slug=", wantStatus: http.StatusBadRequest, wantBody: `{"error":"Workspace Slug is required"}`},
		{name: "last repeated value wins nonempty", query: "?slug=&slug=next", wantStatus: http.StatusOK, wantBody: `{"status":true}`},
		{name: "available", query: "?slug=next", wantStatus: http.StatusOK, wantBody: `{"status":true}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/workspace-slug-check/"+test.query, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || strings.TrimSpace(response.Body.String()) != test.wantBody {
				t.Fatalf("response = %d %q, want %d %q", response.Code, response.Body.String(), test.wantStatus, test.wantBody)
			}
			for name, want := range map[string]string{"Allow": "GET, HEAD, OPTIONS", "Cross-Origin-Opener-Policy": "same-origin", "Referrer-Policy": "same-origin", "Vary": "Cookie, origin", "X-Content-Type-Options": "nosniff", "X-Frame-Options": "DENY"} {
				if got := response.Header().Get(name); got != want {
					t.Errorf("%s = %q, want %q", name, got, want)
				}
			}
			if got := response.Header().Get("loom-error"); got != "" {
				t.Errorf("loom-error = %q, want empty", got)
			}
		})
	}
}

func TestWorkspaceSlugCheckHTTPUnauthenticated(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	handler := newHTTPHandlerWithStickiesAndWorkspaceSlugCheck(ctx, endpoints, false, nil, nil, workspaceapi.NewEndpoints(workspaceSlugHTTPStub{unauthenticated: true}))
	request := httptest.NewRequest(http.MethodGet, "/api/workspace-slug-check/?slug=next", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || strings.TrimSpace(response.Body.String()) != `{"detail":"Authentication credentials were not provided."}` {
		t.Fatalf("response = %d %q, want 401 Django detail", response.Code, response.Body.String())
	}
	if response.Header().Get("Allow") != "GET, HEAD, OPTIONS" || response.Header().Get("Vary") != "Cookie, origin" || response.Header().Get("loom-error") != "" {
		t.Fatalf("unauthenticated headers = %#v", response.Header())
	}
}

func TestWorkspaceSlugCheckHTTPHeadAndOptions(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	handler := newHTTPHandlerWithStickiesAndWorkspaceSlugCheck(ctx, endpoints, false, nil, nil, workspaceapi.NewEndpoints(workspaceSlugHTTPStub{}))
	for _, method := range []string{http.MethodHead, http.MethodOptions} {
		t.Run(method, func(t *testing.T) {
			request := httptest.NewRequest(method, "/api/workspace-slug-check/?slug=next", nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK || response.Body.Len() != 0 {
				t.Fatalf("response = %d %q, want 200 with empty body", response.Code, response.Body.String())
			}
			if response.Header().Get("Allow") != "GET, HEAD, OPTIONS" || response.Header().Get("Vary") != "Cookie, origin" {
				t.Fatalf("headers = %#v", response.Header())
			}
		})
	}
}

func TestWorkspaceSlugCheckHTTPInternalError(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	handler := newHTTPHandlerWithStickiesAndWorkspaceSlugCheck(ctx, endpoints, false, nil, nil, workspaceapi.NewEndpoints(workspaceSlugHTTPStub{failure: true}))
	request := httptest.NewRequest(http.MethodGet, "/api/workspace-slug-check/?slug=next", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || strings.TrimSpace(response.Body.String()) != `{"error":"Something went wrong please try again later"}` {
		t.Fatalf("response = %d %q, want Django-compatible 500", response.Code, response.Body.String())
	}
}

func TestInstanceWorkspaceSlugCheckHTTP(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	stub := &instanceWorkspaceSlugHTTPStub{}
	handler := newHTTPHandlerWithSlugChecks(ctx, endpoints, false, nil, nil, nil, instanceworkspaceapi.NewEndpoints(stub))
	for _, test := range []struct {
		name, query, wantBody string
		wantStatus            int
	}{
		{name: "missing", wantStatus: http.StatusBadRequest, wantBody: `{"error":"Workspace Slug is required"}`},
		{name: "last repeated value wins empty", query: "?slug=next&slug=", wantStatus: http.StatusBadRequest, wantBody: `{"error":"Workspace Slug is required"}`},
		{name: "last repeated value wins nonempty", query: "?slug=&slug=next", wantStatus: http.StatusOK, wantBody: `{"status":true}`},
		{name: "available", query: "?slug=next", wantStatus: http.StatusOK, wantBody: `{"status":true}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/instances/workspace-slug-check/"+test.query, nil)
			request.AddCookie(&http.Cookie{Name: "admin-session-id", Value: "admin-token"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || strings.TrimSpace(response.Body.String()) != test.wantBody {
				t.Fatalf("response = %d %q, want %d %q", response.Code, response.Body.String(), test.wantStatus, test.wantBody)
			}
			for name, want := range map[string]string{"Allow": "GET, HEAD, OPTIONS", "Cross-Origin-Opener-Policy": "same-origin", "Referrer-Policy": "same-origin", "Vary": "Cookie, origin", "X-Content-Type-Options": "nosniff", "X-Frame-Options": "DENY"} {
				if got := response.Header().Get(name); got != want {
					t.Errorf("%s = %q, want %q", name, got, want)
				}
			}
			if got := response.Header().Get("loom-error"); got != "" {
				t.Errorf("loom-error = %q, want empty", got)
			}
		})
	}
	if stub.apiKey != "admin-token" {
		t.Fatalf("API key = %q, want admin session cookie", stub.apiKey)
	}
}

func TestInstanceWorkspaceSlugCheckHTTPAuthentication(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	for _, test := range []struct {
		name, cookieName, cookieValue, wantBody string
		forbidden                               bool
		wantStatus                              int
	}{
		{name: "anonymous", wantStatus: http.StatusUnauthorized, wantBody: `{"detail":"Authentication credentials were not provided."}`},
		{name: "ordinary session cookie is ignored", cookieName: "session-id", cookieValue: "ordinary-token", wantStatus: http.StatusUnauthorized, wantBody: `{"detail":"Authentication credentials were not provided."}`},
		{name: "authenticated non-admin", cookieName: "admin-session-id", cookieValue: "ordinary-token", forbidden: true, wantStatus: http.StatusForbidden, wantBody: `{"detail":"You do not have permission to perform this action."}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			stub := &instanceWorkspaceSlugHTTPStub{forbidden: test.forbidden}
			handler := newHTTPHandlerWithSlugChecks(ctx, endpoints, false, nil, nil, nil, instanceworkspaceapi.NewEndpoints(stub))
			request := httptest.NewRequest(http.MethodGet, "/api/instances/workspace-slug-check/?slug=next", nil)
			if test.cookieName != "" {
				request.AddCookie(&http.Cookie{Name: test.cookieName, Value: test.cookieValue})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || strings.TrimSpace(response.Body.String()) != test.wantBody {
				t.Fatalf("response = %d %q, want %d %q", response.Code, response.Body.String(), test.wantStatus, test.wantBody)
			}
			if response.Header().Get("Allow") != "GET, HEAD, OPTIONS" || response.Header().Get("Vary") != "Cookie, origin" || response.Header().Get("loom-error") != "" {
				t.Fatalf("headers = %#v", response.Header())
			}
		})
	}
}

func TestInstanceWorkspaceSlugCheckHTTPHeadAndOptions(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	handler := newHTTPHandlerWithSlugChecks(ctx, endpoints, false, nil, nil, nil, instanceworkspaceapi.NewEndpoints(&instanceWorkspaceSlugHTTPStub{}))
	for _, method := range []string{http.MethodHead, http.MethodOptions} {
		t.Run(method, func(t *testing.T) {
			request := httptest.NewRequest(method, "/api/instances/workspace-slug-check/?slug=next", nil)
			request.AddCookie(&http.Cookie{Name: "admin-session-id", Value: "admin-token"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK || response.Body.Len() != 0 {
				t.Fatalf("response = %d %q, want 200 with empty body", response.Code, response.Body.String())
			}
			if response.Header().Get("Allow") != "GET, HEAD, OPTIONS" || response.Header().Get("Vary") != "Cookie, origin" {
				t.Fatalf("headers = %#v", response.Header())
			}
		})
	}
}

func TestInstanceWorkspaceSlugCheckHTTPInternalError(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	handler := newHTTPHandlerWithSlugChecks(ctx, endpoints, false, nil, nil, nil, instanceworkspaceapi.NewEndpoints(&instanceWorkspaceSlugHTTPStub{checkFailure: true}))
	request := httptest.NewRequest(http.MethodGet, "/api/instances/workspace-slug-check/?slug=next", nil)
	request.AddCookie(&http.Cookie{Name: "admin-session-id", Value: "admin-token"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || strings.TrimSpace(response.Body.String()) != `{"error":"Something went wrong please try again later"}` {
		t.Fatalf("response = %d %q, want Django-compatible 500", response.Code, response.Body.String())
	}
}

func TestUserInstanceAdminHTTP(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	for _, test := range []struct {
		name, wantBody string
		isAdmin        bool
	}{
		{name: "ordinary user", wantBody: `{"is_instance_admin":false}`},
		{name: "instance admin", wantBody: `{"is_instance_admin":true}`, isAdmin: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			stub := &userInstanceAdminHTTPStub{isAdmin: test.isAdmin}
			handler := newHTTPHandlerWithBackendReplacements(ctx, endpoints, false, nil, nil, nil, nil, userinstanceadminapi.NewEndpoints(stub))
			request := httptest.NewRequest(http.MethodGet, "/api/users/me/instance-admin/?ignored=value", nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "user-token"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != test.wantBody {
				t.Fatalf("response = %d %q, want 200 %q", response.Code, response.Body.String(), test.wantBody)
			}
			if stub.apiKey != "user-token" {
				t.Fatalf("API key = %q, want normal session cookie", stub.apiKey)
			}
			for name, want := range map[string]string{"Allow": "GET, HEAD, OPTIONS", "Cross-Origin-Opener-Policy": "same-origin", "Referrer-Policy": "same-origin", "Vary": "Cookie, origin", "X-Content-Type-Options": "nosniff", "X-Frame-Options": "DENY"} {
				if got := response.Header().Get(name); got != want {
					t.Errorf("%s = %q, want %q", name, got, want)
				}
			}
			if got := response.Header().Get("loom-error"); got != "" {
				t.Errorf("loom-error = %q, want empty", got)
			}
		})
	}
}

func TestUserInstanceAdminHTTPAuthentication(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	for _, test := range []struct {
		name, cookieName, cookieValue string
	}{
		{name: "anonymous"},
		{name: "admin session cookie is ignored", cookieName: "admin-session-id", cookieValue: "admin-token"},
	} {
		t.Run(test.name, func(t *testing.T) {
			stub := &userInstanceAdminHTTPStub{}
			handler := newHTTPHandlerWithBackendReplacements(ctx, endpoints, false, nil, nil, nil, nil, userinstanceadminapi.NewEndpoints(stub))
			request := httptest.NewRequest(http.MethodGet, "/api/users/me/instance-admin/", nil)
			if test.cookieName != "" {
				request.AddCookie(&http.Cookie{Name: test.cookieName, Value: test.cookieValue})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized || strings.TrimSpace(response.Body.String()) != `{"detail":"Authentication credentials were not provided."}` {
				t.Fatalf("response = %d %q, want Django-compatible 401", response.Code, response.Body.String())
			}
			if response.Header().Get("Allow") != "GET, HEAD, OPTIONS" || response.Header().Get("Vary") != "Cookie, origin" || response.Header().Get("loom-error") != "" {
				t.Fatalf("headers = %#v", response.Header())
			}
		})
	}
}

func TestUserInstanceAdminHTTPHeadAndOptions(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	handler := newHTTPHandlerWithBackendReplacements(ctx, endpoints, false, nil, nil, nil, nil, userinstanceadminapi.NewEndpoints(&userInstanceAdminHTTPStub{}))
	for _, method := range []string{http.MethodHead, http.MethodOptions} {
		t.Run(method, func(t *testing.T) {
			request := httptest.NewRequest(method, "/api/users/me/instance-admin/", nil)
			if method == http.MethodHead {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "user-token"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK || response.Body.Len() != 0 {
				t.Fatalf("response = %d %q, want 200 with empty body", response.Code, response.Body.String())
			}
			if response.Header().Get("Allow") != "GET, HEAD, OPTIONS" || response.Header().Get("Vary") != "Cookie, origin" {
				t.Fatalf("headers = %#v", response.Header())
			}
		})
	}
}

func TestUserInstanceAdminHTTPInternalError(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	handler := newHTTPHandlerWithBackendReplacements(ctx, endpoints, false, nil, nil, nil, nil, userinstanceadminapi.NewEndpoints(&userInstanceAdminHTTPStub{failure: true}))
	request := httptest.NewRequest(http.MethodGet, "/api/users/me/instance-admin/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "user-token"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || strings.TrimSpace(response.Body.String()) != `{"error":"Something went wrong please try again later"}` {
		t.Fatalf("response = %d %q, want Django-compatible 500", response.Code, response.Body.String())
	}
}

func TestUserProfileFlagsHTTP(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	for _, test := range []struct {
		name, path, body string
		wantValue        bool
	}{
		{name: "onboard true", path: "/api/users/me/onboard/", body: `{"is_onboarded":true}`, wantValue: true},
		{name: "onboard false", path: "/api/users/me/onboard/", body: `{"is_onboarded":false}`},
		{name: "onboard missing defaults false", path: "/api/users/me/onboard/", body: `{}`},
		{name: "tour true", path: "/api/users/me/tour-completed/", body: `{"is_tour_completed":true}`, wantValue: true},
		{name: "tour missing defaults false", path: "/api/users/me/tour-completed/", body: `{}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			stub := &userProfileFlagsHTTPStub{}
			handler := newHTTPHandlerWithProfileFlags(ctx, endpoints, false, nil, nil, nil, nil, nil, userprofileflagsapi.NewEndpoints(stub))
			request := httptest.NewRequest(http.MethodPatch, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "user-token"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != `{"message":"Updated successfully"}` || stub.value != test.wantValue {
				t.Fatalf("response = %d %q, value = %v", response.Code, response.Body.String(), stub.value)
			}
			if stub.apiKey != "user-token" || response.Header().Get("Allow") != "PATCH, OPTIONS" || response.Header().Get("Vary") != "Cookie, origin" || response.Header().Get("loom-error") != "" {
				t.Fatalf("API key = %q, headers = %#v", stub.apiKey, response.Header())
			}
		})
	}
}

func TestUserProfileFlagsHTTPPayloadCompatibility(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	for _, test := range []struct {
		name, body, wantBody string
		wantStatus           int
		wantValue            bool
	}{
		{name: "null", body: `{"is_onboarded":null}`, wantStatus: http.StatusBadRequest, wantBody: `{"error":"The payload is not valid"}`},
		{name: "lowercase false string", body: `{"is_onboarded":"false"}`, wantStatus: http.StatusBadRequest, wantBody: `{"error":"Please provide valid detail"}`},
		{name: "Django false string", body: `{"is_onboarded":"0"}`, wantStatus: http.StatusOK, wantBody: `{"message":"Updated successfully"}`},
		{name: "numeric true", body: `{"is_onboarded":1}`, wantStatus: http.StatusOK, wantBody: `{"message":"Updated successfully"}`, wantValue: true},
		{name: "array body", body: `[]`, wantStatus: http.StatusInternalServerError, wantBody: `{"error":"Something went wrong please try again later"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			stub := &userProfileFlagsHTTPStub{}
			handler := newHTTPHandlerWithProfileFlags(ctx, endpoints, false, nil, nil, nil, nil, nil, userprofileflagsapi.NewEndpoints(stub))
			request := httptest.NewRequest(http.MethodPatch, "/api/users/me/onboard/", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "user-token"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || strings.TrimSpace(response.Body.String()) != test.wantBody || stub.value != test.wantValue {
				t.Fatalf("response = %d %q, value = %v; want %d %q, %v", response.Code, response.Body.String(), stub.value, test.wantStatus, test.wantBody, test.wantValue)
			}
		})
	}
}

func TestUserProfileFlagsHTTPAuthenticationAndErrors(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	for _, test := range []struct {
		name, wantBody string
		stub           userProfileFlagsHTTPStub
		wantStatus     int
		cookie         bool
	}{
		{name: "anonymous", wantStatus: http.StatusUnauthorized, wantBody: `{"detail":"Authentication credentials were not provided."}`},
		{name: "missing profile", cookie: true, stub: userProfileFlagsHTTPStub{missing: true}, wantStatus: http.StatusNotFound, wantBody: `{"error":"The required object does not exist."}`},
		{name: "repository failure", cookie: true, stub: userProfileFlagsHTTPStub{failure: true}, wantStatus: http.StatusInternalServerError, wantBody: `{"error":"Something went wrong please try again later"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			stub := test.stub
			handler := newHTTPHandlerWithProfileFlags(ctx, endpoints, false, nil, nil, nil, nil, nil, userprofileflagsapi.NewEndpoints(&stub))
			request := httptest.NewRequest(http.MethodPatch, "/api/users/me/onboard/", strings.NewReader(`{"is_onboarded":true}`))
			request.Header.Set("Content-Type", "application/json")
			if test.cookie {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "user-token"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || strings.TrimSpace(response.Body.String()) != test.wantBody || response.Header().Get("loom-error") != "" {
				t.Fatalf("response = %d %q, headers = %#v", response.Code, response.Body.String(), response.Header())
			}
		})
	}
}

func TestUserProfileFlagsHTTPHeadAndOptions(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	handler := newHTTPHandlerWithProfileFlags(ctx, endpoints, false, nil, nil, nil, nil, nil, userprofileflagsapi.NewEndpoints(&userProfileFlagsHTTPStub{}))
	head := httptest.NewRecorder()
	handler.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/api/users/me/onboard/", nil))
	if head.Code != http.StatusMethodNotAllowed || head.Body.Len() != 0 || head.Header().Get("Allow") != "PATCH, OPTIONS" {
		t.Fatalf("HEAD response = %d %q, headers = %#v", head.Code, head.Body.String(), head.Header())
	}
	options := httptest.NewRecorder()
	handler.ServeHTTP(options, httptest.NewRequest(http.MethodOptions, "/api/users/me/tour-completed/", nil))
	want := `{"description":"","name":"Update User Tour Completed Endpoint","parses":["application/json","application/x-www-form-urlencoded","multipart/form-data"],"renders":["application/json"]}`
	if options.Code != http.StatusOK || strings.TrimSpace(options.Body.String()) != want || options.Header().Get("Allow") != "PATCH, OPTIONS" {
		t.Fatalf("OPTIONS response = %d %q, headers = %#v", options.Code, options.Body.String(), options.Header())
	}
}

func TestUserNotificationPreferencesHTTPMount(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	systemEndpoints := system.NewEndpoints(systemapi.NewSystem())
	stub := &userNotificationPreferencesHTTPStub{}
	handler := newHTTPHandlerWithNotificationPreferences(ctx, systemEndpoints, false, nil, nil, nil, nil, nil, nil, usernotificationpreferencesapi.NewEndpoints(stub))

	request := httptest.NewRequest(http.MethodGet, "/api/users/me/notification-preferences/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || stub.apiKey != "session-cookie" {
		t.Fatalf("GET response = %d, key = %q, body = %q", response.Code, stub.apiKey, response.Body.String())
	}
	if response.Header().Get("Allow") != "GET, PATCH, HEAD, OPTIONS" || response.Header().Get("Vary") != "Cookie, origin" {
		t.Fatalf("GET headers = %#v", response.Header())
	}

	request = httptest.NewRequest(http.MethodPatch, "/api/users/me/notification-preferences/", strings.NewReader(`{"property_change":"false"}`))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || stub.patch == nil || stub.patch.Body == nil {
		t.Fatalf("PATCH response = %d, body = %q, payload = %#v", response.Code, response.Body.String(), stub.patch)
	}
	jsonValue, present := stub.patch.Body.PropertyChange.Value()
	var value string
	if err := json.Unmarshal(jsonValue, &value); !present || err != nil || value != "false" {
		t.Fatalf("PATCH bool representation = %s, present=%t, err=%v", jsonValue, present, err)
	}

	request = httptest.NewRequest(http.MethodPatch, "/api/users/me/notification-preferences/", strings.NewReader("{"))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("malformed PATCH status = %d, body = %q", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPatch, "/api/users/me/notification-preferences/", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("empty PATCH status = %d, body = %q", response.Code, response.Body.String())
	}
}

func TestUserNotificationPreferencesHTTPHeadAndOptions(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	systemEndpoints := system.NewEndpoints(systemapi.NewSystem())
	handler := newHTTPHandlerWithNotificationPreferences(ctx, systemEndpoints, false, nil, nil, nil, nil, nil, nil, usernotificationpreferencesapi.NewEndpoints(&userNotificationPreferencesHTTPStub{}))

	head := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodHead, "/api/users/me/notification-preferences/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	handler.ServeHTTP(head, request)
	if head.Code != http.StatusOK || head.Body.Len() != 0 || head.Header().Get("Allow") != "GET, PATCH, HEAD, OPTIONS" {
		t.Fatalf("HEAD response = %d %q, headers = %#v", head.Code, head.Body.String(), head.Header())
	}

	options := httptest.NewRecorder()
	handler.ServeHTTP(options, httptest.NewRequest(http.MethodOptions, "/api/users/me/notification-preferences/", nil))
	var body map[string]any
	if options.Code != http.StatusOK || json.Unmarshal(options.Body.Bytes(), &body) != nil {
		t.Fatalf("OPTIONS response = %d %q", options.Code, options.Body.String())
	}
	if body["name"] != "User Notification Preference Endpoint" || options.Header().Get("Allow") != "GET, PATCH, HEAD, OPTIONS" {
		t.Fatalf("OPTIONS body/headers = %#v %#v", body, options.Header())
	}
}

func TestUserAccountsHTTPMount(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	systemEndpoints := system.NewEndpoints(systemapi.NewSystem())
	stub := &userAccountsHTTPStub{}
	handler := newHTTPHandlerWithUserAccounts(ctx, systemEndpoints, false, nil, nil, nil, nil, nil, nil, nil, useraccountsapi.NewEndpoints(stub))
	id := "00000000-0000-4000-8000-000000000021"

	for _, testCase := range []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/api/users/me/accounts/", http.StatusOK},
		{http.MethodDelete, "/api/users/me/accounts/", http.StatusInternalServerError},
		{http.MethodGet, "/api/users/me/accounts/" + id + "/", http.StatusOK},
		{http.MethodDelete, "/api/users/me/accounts/" + id + "/", http.StatusNoContent},
	} {
		request := httptest.NewRequest(testCase.method, testCase.path, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != testCase.status {
			t.Errorf("%s %s status = %d, want %d; body %q", testCase.method, testCase.path, response.Code, testCase.status, response.Body.String())
		}
		if response.Header().Get("Allow") != "GET, DELETE, HEAD, OPTIONS" || response.Header().Get("Vary") != "Cookie, origin" {
			t.Errorf("%s %s headers = %#v", testCase.method, testCase.path, response.Header())
		}
	}
	if stub.apiKey != "session-cookie" || stub.deleted != id {
		t.Fatalf("key = %q, deleted = %q", stub.apiKey, stub.deleted)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/users/me/accounts/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || response.Body.String() != "{\"detail\":\"Authentication credentials were not provided.\"}\n" {
		t.Fatalf("anonymous response = %d %q", response.Code, response.Body.String())
	}
}

func TestUserAccountsHTTPHeadAndOptions(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	systemEndpoints := system.NewEndpoints(systemapi.NewSystem())
	handler := newHTTPHandlerWithUserAccounts(ctx, systemEndpoints, false, nil, nil, nil, nil, nil, nil, nil, useraccountsapi.NewEndpoints(&userAccountsHTTPStub{}))
	id := "00000000-0000-4000-8000-000000000021"

	for _, path := range []string{"/api/users/me/accounts/", "/api/users/me/accounts/" + id + "/"} {
		head := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodHead, path, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
		handler.ServeHTTP(head, request)
		if head.Code != http.StatusOK || head.Body.Len() != 0 || head.Header().Get("Allow") != "GET, DELETE, HEAD, OPTIONS" {
			t.Errorf("HEAD %s = %d %q, headers = %#v", path, head.Code, head.Body.String(), head.Header())
		}

		options := httptest.NewRecorder()
		handler.ServeHTTP(options, httptest.NewRequest(http.MethodOptions, path, nil))
		var body map[string]any
		if options.Code != http.StatusOK || json.Unmarshal(options.Body.Bytes(), &body) != nil {
			t.Errorf("OPTIONS %s = %d %q", path, options.Code, options.Body.String())
		}
		if body["name"] != "Account Endpoint" || options.Header().Get("Allow") != "GET, DELETE, HEAD, OPTIONS" {
			t.Errorf("OPTIONS %s body/headers = %#v %#v", path, body, options.Header())
		}
	}
}

func TestUserAPITokensAndInstanceAdminSessionHTTPMounts(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	systemEndpoints := system.NewEndpoints(systemapi.NewSystem())
	tokens := &userAPITokensHTTPStub{}
	admin := &instanceAdminSessionHTTPStub{authenticated: true}
	handler := newHTTPHandlerWithAPITokensAndInstanceAdminSession(
		ctx, systemEndpoints, false, nil, nil, nil, nil, nil, nil, nil, nil,
		userapitokensapi.NewEndpoints(tokens), instanceadminsessionapi.NewEndpoints(admin),
	)

	for _, testCase := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/api/users/api-tokens/", http.StatusOK},
		{http.MethodPost, "/api/users/api-tokens/", http.StatusCreated},
		{http.MethodPatch, "/api/users/api-tokens/", http.StatusInternalServerError},
		{http.MethodDelete, "/api/users/api-tokens/", http.StatusInternalServerError},
		{http.MethodGet, "/api/users/api-tokens/00000000-0000-4000-8000-000000000031/", http.StatusOK},
		{http.MethodPatch, "/api/users/api-tokens/00000000-0000-4000-8000-000000000031/", http.StatusOK},
		{http.MethodDelete, "/api/users/api-tokens/00000000-0000-4000-8000-000000000031/", http.StatusNoContent},
		{http.MethodPost, "/api/users/api-tokens/00000000-0000-4000-8000-000000000031/", http.StatusInternalServerError},
	} {
		var body io.Reader
		if testCase.method == http.MethodPost || testCase.method == http.MethodPatch {
			body = strings.NewReader(`{}`)
		}
		request := httptest.NewRequest(testCase.method, testCase.path, body)
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "user-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != testCase.status {
			t.Errorf("%s %s status = %d, want %d; body %q", testCase.method, testCase.path, response.Code, testCase.status, response.Body.String())
		}
		if response.Header().Get("Allow") != "GET, POST, PATCH, DELETE, HEAD, OPTIONS" {
			t.Errorf("%s %s Allow = %q", testCase.method, testCase.path, response.Header().Get("Allow"))
		}
	}
	if tokens.apiKey != "user-session" {
		t.Fatalf("API token session key = %q", tokens.apiKey)
	}

	for _, path := range []string{"/api/instances/admins/me/", "/api/instances/admins/session/"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.AddCookie(&http.Cookie{Name: "admin-session-id", Value: "admin-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Header().Get("Allow") != "GET, HEAD, OPTIONS" {
			t.Errorf("GET %s = %d %q, headers = %#v", path, response.Code, response.Body.String(), response.Header())
		}
	}
	if admin.apiKey != "admin-session" || admin.sessionKey != "admin-session" {
		t.Fatalf("admin keys = auth %q session %q", admin.apiKey, admin.sessionKey)
	}
}

func TestUserAPITokensAndInstanceAdminSessionHTTPAuthentication(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	systemEndpoints := system.NewEndpoints(systemapi.NewSystem())
	admin := &instanceAdminSessionHTTPStub{forbidden: true}
	handler := newHTTPHandlerWithAPITokensAndInstanceAdminSession(
		ctx, systemEndpoints, false, nil, nil, nil, nil, nil, nil, nil, nil,
		userapitokensapi.NewEndpoints(&userAPITokensHTTPStub{}), instanceadminsessionapi.NewEndpoints(admin),
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/users/api-tokens/", nil))
	if response.Code != http.StatusUnauthorized || strings.TrimSpace(response.Body.String()) != `{"detail":"Authentication credentials were not provided."}` {
		t.Fatalf("anonymous API tokens = %d %q", response.Code, response.Body.String())
	}

	request := httptest.NewRequest(http.MethodGet, "/api/instances/admins/me/", nil)
	request.AddCookie(&http.Cookie{Name: "admin-session-id", Value: "ordinary-session"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || strings.TrimSpace(response.Body.String()) != `{"detail":"You do not have permission to perform this action."}` {
		t.Fatalf("ordinary instance admin me = %d %q", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/instances/admins/session/", nil))
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != `{"is_authenticated":false}` {
		t.Fatalf("anonymous instance admin session = %d %q", response.Code, response.Body.String())
	}
}

func TestWorkspaceContentHTTPMounts(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	systemEndpoints := system.NewEndpoints(systemapi.NewSystem())
	handler := newHTTPHandlerWithWorkspaceContent(
		ctx, systemEndpoints, false, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		workspacequicklinksapi.NewEndpoints(workspaceQuickLinksHTTPStub{}),
		workspacethemesapi.NewEndpoints(workspaceThemesHTTPStub{}),
		workspacehomepreferencesapi.NewEndpoints(workspaceHomePreferencesHTTPStub{}),
	)
	id := "00000000-0000-4000-8000-000000000041"
	for _, testCase := range []struct {
		method string
		path   string
		body   string
		status int
		allow  string
	}{
		{http.MethodGet, "/api/workspaces/acme/quick-links/", "", http.StatusOK, "GET, POST, HEAD, OPTIONS"},
		{http.MethodPost, "/api/workspaces/acme/quick-links/", `{"url":"plane.so"}`, http.StatusCreated, "GET, POST, HEAD, OPTIONS"},
		{http.MethodGet, "/api/workspaces/acme/quick-links/" + id + "/", "", http.StatusOK, "GET, PATCH, DELETE, HEAD, OPTIONS"},
		{http.MethodPatch, "/api/workspaces/acme/quick-links/" + id + "/", `{"title":"Updated"}`, http.StatusOK, "GET, PATCH, DELETE, HEAD, OPTIONS"},
		{http.MethodDelete, "/api/workspaces/acme/quick-links/" + id + "/", "", http.StatusNoContent, "GET, PATCH, DELETE, HEAD, OPTIONS"},
		{http.MethodGet, "/api/workspaces/acme/workspace-themes/", "", http.StatusOK, "GET, POST, HEAD, OPTIONS"},
		{http.MethodPost, "/api/workspaces/acme/workspace-themes/", `{"name":"Dark","colors":{}}`, http.StatusCreated, "GET, POST, HEAD, OPTIONS"},
		{http.MethodGet, "/api/workspaces/acme/workspace-themes/" + id + "/", "", http.StatusOK, "GET, PATCH, DELETE, HEAD, OPTIONS"},
		{http.MethodPatch, "/api/workspaces/acme/workspace-themes/" + id + "/", `{"colors":{}}`, http.StatusOK, "GET, PATCH, DELETE, HEAD, OPTIONS"},
		{http.MethodDelete, "/api/workspaces/acme/workspace-themes/" + id + "/", "", http.StatusNoContent, "GET, PATCH, DELETE, HEAD, OPTIONS"},
		{http.MethodGet, "/api/workspaces/acme/home-preferences/", "", http.StatusOK, "GET, PATCH, HEAD, OPTIONS"},
		{http.MethodPatch, "/api/workspaces/acme/home-preferences/", "{}", http.StatusInternalServerError, "GET, PATCH, HEAD, OPTIONS"},
		{http.MethodGet, "/api/workspaces/acme/home-preferences/quick_links/", "", http.StatusInternalServerError, "GET, PATCH, HEAD, OPTIONS"},
		{http.MethodPatch, "/api/workspaces/acme/home-preferences/quick_links/", `{"key":"recents"}`, http.StatusOK, "GET, PATCH, HEAD, OPTIONS"},
	} {
		request := httptest.NewRequest(testCase.method, testCase.path, strings.NewReader(testCase.body))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != testCase.status || response.Header().Get("Allow") != testCase.allow {
			t.Errorf("%s %s = %d %q Allow=%q", testCase.method, testCase.path, response.Code, response.Body.String(), response.Header().Get("Allow"))
		}
	}

	for _, path := range []string{"/api/workspaces/acme/quick-links/", "/api/workspaces/acme/workspace-themes/"} {
		head := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodHead, path, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
		handler.ServeHTTP(head, request)
		if head.Code != http.StatusOK || head.Body.Len() != 0 {
			t.Errorf("HEAD %s = %d %q", path, head.Code, head.Body.String())
		}
		options := httptest.NewRecorder()
		handler.ServeHTTP(options, httptest.NewRequest(http.MethodOptions, path, nil))
		if options.Code != http.StatusOK || !strings.Contains(options.Body.String(), `"actions"`) {
			t.Errorf("OPTIONS %s = %d %q", path, options.Code, options.Body.String())
		}
	}
	for path, expectedStatus := range map[string]int{
		"/api/workspaces/acme/home-preferences/":             http.StatusOK,
		"/api/workspaces/acme/home-preferences/quick_links/": http.StatusInternalServerError,
	} {
		head := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodHead, path, nil)
		req.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
		handler.ServeHTTP(head, req)
		if head.Code != expectedStatus {
			t.Errorf("HEAD %s = %d", path, head.Code)
		}
		options := httptest.NewRecorder()
		handler.ServeHTTP(options, httptest.NewRequest(http.MethodOptions, path, nil))
		if options.Code != http.StatusOK || options.Header().Get("Allow") != "GET, PATCH, HEAD, OPTIONS" || !strings.Contains(options.Body.String(), `"PATCH"`) {
			t.Errorf("OPTIONS %s = %d %q", path, options.Code, options.Body.String())
		}
	}
}

func TestWorkspaceMemberMeAndAnalyticViewsHTTPMounts(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	systemEndpoints := system.NewEndpoints(systemapi.NewSystem())
	handler := newHTTPHandlerWithWorkspaceReplacements(
		ctx, systemEndpoints, false, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		workspacemembermeapi.NewEndpoints(workspaceMemberMeHTTPStub{}),
		analyticviewsapi.NewEndpoints(analyticViewsHTTPStub{}),
		workspacerecentvisitsapi.NewEndpoints(workspaceRecentVisitsHTTPStub{}),
	)
	id := "00000000-0000-4000-8000-000000000011"
	for _, testCase := range []struct {
		method string
		path   string
		body   string
		status int
		allow  string
	}{
		{http.MethodGet, "/api/workspaces/acme/workspace-members/me/", "", http.StatusOK, "GET, HEAD, OPTIONS"},
		{http.MethodGet, "/api/workspaces/acme/recent-visits/", "", http.StatusOK, "GET, HEAD, OPTIONS"},
		{http.MethodPost, "/api/workspaces/acme/recent-visits/", `{}`, http.StatusMethodNotAllowed, "GET, HEAD, OPTIONS"},
		{http.MethodGet, "/api/workspaces/acme/analytic-view/", "", http.StatusOK, "GET, POST, HEAD, OPTIONS"},
		{http.MethodPost, "/api/workspaces/acme/analytic-view/", `{"name":"Issues by state"}`, http.StatusCreated, "GET, POST, HEAD, OPTIONS"},
		{http.MethodGet, "/api/workspaces/acme/analytic-view/" + id + "/", "", http.StatusOK, "GET, PATCH, DELETE, HEAD, OPTIONS"},
		{http.MethodPatch, "/api/workspaces/acme/analytic-view/" + id + "/", `{"name":"Updated"}`, http.StatusOK, "GET, PATCH, DELETE, HEAD, OPTIONS"},
		{http.MethodDelete, "/api/workspaces/acme/analytic-view/" + id + "/", "", http.StatusNoContent, "GET, PATCH, DELETE, HEAD, OPTIONS"},
	} {
		request := httptest.NewRequest(testCase.method, testCase.path, strings.NewReader(testCase.body))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != testCase.status || response.Header().Get("Allow") != testCase.allow {
			t.Errorf("%s %s = %d %q Allow=%q", testCase.method, testCase.path, response.Code, response.Body.String(), response.Header().Get("Allow"))
		}
		if testCase.path == "/api/workspaces/acme/recent-visits/" && testCase.method == http.MethodPost && response.Body.String() != "{\"detail\":\"Method \\\"POST\\\" not allowed.\"}\n" {
			t.Errorf("POST recent visits body = %q", response.Body.String())
		}
		if testCase.status != http.StatusNoContent && !strings.Contains(response.Header().Get("Vary"), "origin") {
			t.Errorf("%s %s Vary=%q", testCase.method, testCase.path, response.Header().Get("Vary"))
		}
	}

	for _, path := range []string{
		"/api/workspaces/acme/workspace-members/me/",
		"/api/workspaces/acme/recent-visits/",
		"/api/workspaces/acme/analytic-view/",
		"/api/workspaces/acme/analytic-view/" + id + "/",
	} {
		head := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodHead, path, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
		handler.ServeHTTP(head, request)
		if head.Code != http.StatusOK || head.Body.Len() != 0 {
			t.Errorf("HEAD %s = %d %q", path, head.Code, head.Body.String())
		}
		options := httptest.NewRecorder()
		handler.ServeHTTP(options, httptest.NewRequest(http.MethodOptions, path, nil))
		if options.Code != http.StatusOK {
			t.Errorf("OPTIONS %s = %d %q", path, options.Code, options.Body.String())
		}
	}
}

func TestProjectMembershipReadsHTTPMountsAndDjangoProtocol(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	systemEndpoints := system.NewEndpoints(systemapi.NewSystem())
	build := func(stub projectMembershipReadsHTTPStub) http.Handler {
		return newHTTPHandlerWithPendingReplacements(
			ctx,
			systemEndpoints,
			false,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			pendingReplacementEndpoints{
				projectMembershipReads:    projectmembershipreadsapi.NewEndpoints(stub),
				projectMembershipMetadata: stub,
			},
		)
	}
	handler := build(projectMembershipReadsHTTPStub{})

	projectID := "00000000-0000-4000-8000-000000000003"
	routes := []struct {
		path               string
		wantAcceptEncoding bool
		bodyFragment       string
		allow              string
		optionsInGo        bool
		optionsName        string
	}{
		{
			path:               "/api/workspaces/acme/projects/" + projectID + "/project-members/me/",
			wantAcceptEncoding: true,
			bodyFragment:       `"role":15`,
			allow:              "GET, HEAD, OPTIONS",
			optionsInGo:        true,
			optionsName:        "Project Member User Endpoint",
		},
		{
			path:         "/api/users/me/workspaces/acme/project-roles/",
			bodyFragment: `"00000000-0000-4000-8000-000000000003":15`,
			allow:        "GET, HEAD, OPTIONS",
			optionsInGo:  true,
			optionsName:  "User Project Roles Endpoint",
		},
		{
			path:               "/api/workspaces/acme/members/",
			wantAcceptEncoding: true,
			bodyFragment:       `"role":15`,
			allow:              "GET, HEAD, OPTIONS",
			optionsInGo:        true,
			optionsName:        "Work Space Member",
		},
		{
			path:               "/api/workspaces/acme/members/00000000-0000-4000-8000-000000000011/",
			wantAcceptEncoding: true,
			bodyFragment:       `"role":15`,
			allow:              "GET, PATCH, DELETE, HEAD, OPTIONS",
		},
		{
			path:               "/api/workspaces/acme/project-members/",
			wantAcceptEncoding: true,
			bodyFragment:       `00000000-0000-4000-8000-000000000003`,
			allow:              "GET, HEAD, OPTIONS",
			optionsInGo:        true,
			optionsName:        "Workspace Project Member Endpoint",
		},
		{
			path:               "/api/workspaces/acme/projects/" + projectID + "/members/",
			wantAcceptEncoding: true,
			bodyFragment:       `"role":15`,
			allow:              "GET, POST, HEAD, OPTIONS",
		},
		{
			path:               "/api/workspaces/acme/projects/" + projectID + "/members/00000000-0000-4000-8000-000000000012/",
			wantAcceptEncoding: true,
			bodyFragment:       `"role":15`,
			allow:              "GET, PATCH, DELETE, HEAD, OPTIONS",
		},
		{
			path:               "/api/workspaces/acme/projects/" + projectID + "/preferences/member/00000000-0000-4000-8000-000000000012/",
			wantAcceptEncoding: true,
			bodyFragment:       `"preferences"`,
			allow:              "GET, PATCH, HEAD, OPTIONS",
		},
	}
	for _, route := range routes {
		request := httptest.NewRequest(http.MethodGet, route.path, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), route.bodyFragment) {
			t.Errorf("GET %s = %d %q", route.path, response.Code, response.Body.String())
		}
		if response.Header().Get("Allow") != route.allow ||
			!strings.Contains(response.Header().Get("Vary"), "origin") ||
			!strings.Contains(response.Header().Get("Vary"), "Cookie") {
			t.Errorf("GET %s headers = %v", route.path, response.Header())
		}
		hasAcceptEncoding := strings.Contains(response.Header().Get("Vary"), "Accept-Encoding")
		if hasAcceptEncoding != route.wantAcceptEncoding {
			t.Errorf("GET %s Vary=%q, want Accept-Encoding=%v", route.path, response.Header().Get("Vary"), route.wantAcceptEncoding)
		}

		head := httptest.NewRecorder()
		headRequest := httptest.NewRequest(http.MethodHead, route.path, nil)
		headRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
		handler.ServeHTTP(head, headRequest)
		if head.Code != http.StatusOK || head.Body.Len() != 0 ||
			!strings.Contains(head.Header().Get("Vary"), "Cookie") {
			t.Errorf("HEAD %s = %d %q headers=%v", route.path, head.Code, head.Body.String(), head.Header())
		}

		if route.optionsInGo {
			options := httptest.NewRecorder()
			optionsRequest := httptest.NewRequest(http.MethodOptions, route.path, nil)
			optionsRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
			handler.ServeHTTP(options, optionsRequest)
			if options.Code != http.StatusOK || options.Header().Get("Allow") != route.allow ||
				!strings.Contains(options.Header().Get("Vary"), "Cookie") {
				t.Errorf("OPTIONS %s = %d %q headers=%v", route.path, options.Code, options.Body.String(), options.Header())
			}
			var metadata map[string]any
			if err := json.Unmarshal(options.Body.Bytes(), &metadata); err != nil || metadata["name"] != route.optionsName {
				t.Errorf("OPTIONS %s metadata = %#v, err=%v", route.path, metadata, err)
			}
			if _, exists := metadata["actions"]; exists {
				t.Errorf("OPTIONS %s unexpectedly describes actions: %#v", route.path, metadata)
			}

			anonymousOptions := httptest.NewRecorder()
			handler.ServeHTTP(anonymousOptions, httptest.NewRequest(http.MethodOptions, route.path, nil))
			if anonymousOptions.Code != http.StatusUnauthorized ||
				anonymousOptions.Body.String() != "{\"detail\":\"Authentication credentials were not provided.\"}\n" ||
				!strings.Contains(anonymousOptions.Header().Get("Vary"), "Cookie") ||
				!strings.Contains(anonymousOptions.Header().Get("Vary"), "origin") {
				t.Errorf(
					"anonymous OPTIONS %s = %d %q headers=%v",
					route.path,
					anonymousOptions.Code,
					anonymousOptions.Body.String(),
					anonymousOptions.Header(),
				)
			}
		}

		unauthenticated := httptest.NewRecorder()
		handler.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, route.path, nil))
		if unauthenticated.Code != http.StatusUnauthorized ||
			!strings.Contains(unauthenticated.Header().Get("Vary"), "Cookie") ||
			!strings.Contains(unauthenticated.Header().Get("Vary"), "origin") {
			t.Errorf("unauthenticated GET %s = %d %q headers=%v", route.path, unauthenticated.Code, unauthenticated.Body.String(), unauthenticated.Header())
		}
	}

	unsupportedAcceptRequest := httptest.NewRequest(http.MethodGet, "/api/workspaces/acme/members/", nil)
	unsupportedAcceptRequest.Header.Set("Accept", "application/xml")
	unsupportedAcceptRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	unsupportedAcceptResponse := httptest.NewRecorder()
	handler.ServeHTTP(unsupportedAcceptResponse, unsupportedAcceptRequest)
	if unsupportedAcceptResponse.Code != http.StatusNotAcceptable ||
		unsupportedAcceptResponse.Header().Get("Content-Type") != "application/json" ||
		unsupportedAcceptResponse.Body.String() != "{\"detail\":\"Could not satisfy the request Accept header.\"}\n" ||
		strings.Contains(unsupportedAcceptResponse.Header().Get("Vary"), "Cookie") {
		t.Fatalf("unsupported Accept response = %d %q headers=%v", unsupportedAcceptResponse.Code, unsupportedAcceptResponse.Body.String(), unsupportedAcceptResponse.Header())
	}

	indentedSuccessRequest := httptest.NewRequest(http.MethodGet, "/api/workspaces/acme/members/", nil)
	indentedSuccessRequest.Header.Set("Accept", "application/json; indent=2")
	indentedSuccessRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	indentedSuccessResponse := httptest.NewRecorder()
	handler.ServeHTTP(indentedSuccessResponse, indentedSuccessRequest)
	if indentedSuccessResponse.Code != http.StatusOK ||
		!strings.HasPrefix(indentedSuccessResponse.Body.String(), "[\n  {") ||
		!strings.Contains(indentedSuccessResponse.Body.String(), "\n    \"role\": 15") {
		t.Fatalf("indented success response = %d %q", indentedSuccessResponse.Code, indentedSuccessResponse.Body.String())
	}

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		unsupportedFormatResponse := httptest.NewRecorder()
		handler.ServeHTTP(
			unsupportedFormatResponse,
			httptest.NewRequest(method, "/api/workspaces/acme/members/?format=xml", nil),
		)
		if unsupportedFormatResponse.Code != http.StatusNotFound ||
			unsupportedFormatResponse.Header().Get("Content-Type") != "application/json" ||
			strings.Contains(unsupportedFormatResponse.Header().Get("Vary"), "Cookie") {
			t.Fatalf("unsupported %s format response = %d %q headers=%v", method, unsupportedFormatResponse.Code, unsupportedFormatResponse.Body.String(), unsupportedFormatResponse.Header())
		}
		if method != http.MethodHead && unsupportedFormatResponse.Body.String() != "{\"detail\":\"Not found.\"}\n" {
			t.Fatalf("unsupported %s format body = %q", method, unsupportedFormatResponse.Body.String())
		}
	}

	jsonFormatRequest := httptest.NewRequest(http.MethodGet, "/api/workspaces/acme/members/?format=xml&format=json", nil)
	jsonFormatRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	jsonFormatResponse := httptest.NewRecorder()
	handler.ServeHTTP(jsonFormatResponse, jsonFormatRequest)
	if jsonFormatResponse.Code != http.StatusOK {
		t.Fatalf("json format response = %d %q", jsonFormatResponse.Code, jsonFormatResponse.Body.String())
	}

	for _, query := range []string{"format=", "format=xml&format="} {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
			request := httptest.NewRequest(method, "/api/workspaces/acme/members/?"+query, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("empty final format %s %q = %d %q", method, query, response.Code, response.Body.String())
			}
		}
	}

	lastSearch := ""
	duplicateSearchHandler := build(projectMembershipReadsHTTPStub{searchCapture: &lastSearch})
	duplicateSearchRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/workspaces/acme/members/?search=first&search=last",
		nil,
	)
	duplicateSearchRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	duplicateSearchHandler.ServeHTTP(httptest.NewRecorder(), duplicateSearchRequest)
	if lastSearch != "last" {
		t.Fatalf("duplicate search decoded %q, want last value", lastSearch)
	}

	permissionHandler := build(projectMembershipReadsHTTPStub{
		metadataError: &projectmembershipreadsapi.ProjectMembershipReadsForbidden{
			Detail: "You do not have permission to perform this action.",
		},
	})
	permissionRequest := httptest.NewRequest(http.MethodOptions, "/api/workspaces/acme/project-members/", nil)
	permissionRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	permissionResponse := httptest.NewRecorder()
	permissionHandler.ServeHTTP(permissionResponse, permissionRequest)
	if permissionResponse.Code != http.StatusForbidden ||
		permissionResponse.Body.String() != "{\"detail\":\"You do not have permission to perform this action.\"}\n" ||
		!strings.Contains(permissionResponse.Header().Get("Vary"), "Cookie") {
		t.Fatalf("forbidden OPTIONS = %d %q headers=%v", permissionResponse.Code, permissionResponse.Body.String(), permissionResponse.Header())
	}

	workspaceMemberPermissionHandler := build(projectMembershipReadsHTTPStub{
		workspaceMembersError: &projectmembershipreadsapi.ProjectMembershipReadsPermissionDenied{
			Message: "You don't have the required permissions.",
		},
	})
	workspaceMemberOptionsRequest := httptest.NewRequest(http.MethodOptions, "/api/workspaces/acme/members/", nil)
	workspaceMemberOptionsRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	workspaceMemberOptionsResponse := httptest.NewRecorder()
	workspaceMemberPermissionHandler.ServeHTTP(workspaceMemberOptionsResponse, workspaceMemberOptionsRequest)
	if workspaceMemberOptionsResponse.Code != http.StatusOK {
		t.Fatalf("authenticated workspace-member OPTIONS = %d %q", workspaceMemberOptionsResponse.Code, workspaceMemberOptionsResponse.Body.String())
	}
	workspaceMemberReadRequest := httptest.NewRequest(http.MethodGet, "/api/workspaces/acme/members/", nil)
	workspaceMemberReadRequest.Header.Set("Accept", "application/json; indent=2")
	workspaceMemberReadRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	workspaceMemberReadResponse := httptest.NewRecorder()
	workspaceMemberPermissionHandler.ServeHTTP(workspaceMemberReadResponse, workspaceMemberReadRequest)
	if workspaceMemberReadResponse.Code != http.StatusForbidden ||
		!strings.Contains(workspaceMemberReadResponse.Header().Get("Vary"), "Cookie") ||
		!strings.HasPrefix(workspaceMemberReadResponse.Body.String(), "{\n  ") {
		t.Fatalf(
			"authenticated forbidden GET = %d %q headers=%v",
			workspaceMemberReadResponse.Code,
			workspaceMemberReadResponse.Body.String(),
			workspaceMemberReadResponse.Header(),
		)
	}

	notFoundHandler := build(projectMembershipReadsHTTPStub{
		workspaceMemberError: &projectmembershipreadsapi.ProjectMembershipReadsNotFound{
			Message: "Workspace member not found",
		},
	})
	notFoundRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/workspaces/acme/members/00000000-0000-4000-8000-000000000011/",
		nil,
	)
	notFoundRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	notFoundResponse := httptest.NewRecorder()
	notFoundHandler.ServeHTTP(notFoundResponse, notFoundRequest)
	if notFoundResponse.Code != http.StatusNotFound ||
		!strings.Contains(notFoundResponse.Header().Get("Vary"), "Cookie") {
		t.Fatalf(
			"authenticated not-found GET = %d %q headers=%v",
			notFoundResponse.Code,
			notFoundResponse.Body.String(),
			notFoundResponse.Header(),
		)
	}

	rolesPermissionHandler := build(projectMembershipReadsHTTPStub{
		metadataError: &projectmembershipreadsapi.ProjectMembershipReadsForbidden{
			Detail: "You do not have permission to perform this action.",
		},
	})
	rolesOptionsRequest := httptest.NewRequest(http.MethodOptions, "/api/users/me/workspaces/acme/project-roles/", nil)
	rolesOptionsRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	rolesOptionsResponse := httptest.NewRecorder()
	rolesPermissionHandler.ServeHTTP(rolesOptionsResponse, rolesOptionsRequest)
	if rolesOptionsResponse.Code != http.StatusForbidden ||
		rolesOptionsResponse.Body.String() != "{\"detail\":\"You do not have permission to perform this action.\"}\n" ||
		!strings.Contains(rolesOptionsResponse.Header().Get("Vary"), "Cookie") {
		t.Fatalf("forbidden project-roles OPTIONS = %d %q headers=%v", rolesOptionsResponse.Code, rolesOptionsResponse.Body.String(), rolesOptionsResponse.Header())
	}
}

func TestUserMeReadHTTPMountAndDjangoProtocol(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	handler := newHTTPHandlerWithPendingReplacements(
		ctx,
		system.NewEndpoints(systemapi.NewSystem()),
		false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{userIdentity: useridentityapi.NewEndpoints(userIdentityHTTPStub{})},
	)

	get := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/api/users/me/", nil)
	getRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	handler.ServeHTTP(get, getRequest)
	if get.Code != http.StatusOK {
		t.Fatalf("GET user me = %d %q", get.Code, get.Body.String())
	}
	if get.Header().Get("Allow") != "GET, PATCH, DELETE, HEAD, OPTIONS" ||
		get.Header().Get("Cache-Control") != "private, max-age=12" ||
		get.Header().Get("Vary") != "Cookie, Accept-Encoding, origin" ||
		!strings.Contains(get.Body.String(), `"date_joined":"2026-07-01T20:04:05.123456-07:00"`) {
		t.Fatalf("GET user me compatibility = body %q headers=%v", get.Body.String(), get.Header())
	}

	head := httptest.NewRecorder()
	headRequest := httptest.NewRequest(http.MethodHead, "/api/users/me/", nil)
	headRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	handler.ServeHTTP(head, headRequest)
	if head.Code != http.StatusOK || head.Body.Len() != 0 || head.Header().Get("Cache-Control") != "private, max-age=12" {
		t.Fatalf("HEAD user me = %d %q headers=%v", head.Code, head.Body.String(), head.Header())
	}

	options := httptest.NewRecorder()
	handler.ServeHTTP(options, httptest.NewRequest(http.MethodOptions, "/api/users/me/", nil))
	if options.Code != http.StatusOK || options.Header().Get("Allow") != "GET, PATCH, DELETE, HEAD, OPTIONS" ||
		options.Body.String() != "{\"name\":\"User Endpoint\",\"description\":\"\",\"renders\":[\"application/json\"],\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"]}\n" {
		t.Fatalf("OPTIONS user me = %d %q headers=%v", options.Code, options.Body.String(), options.Header())
	}

	unauthenticated := httptest.NewRecorder()
	handler.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/api/users/me/", nil))
	if unauthenticated.Code != http.StatusUnauthorized || unauthenticated.Header().Get("Cache-Control") != "" ||
		unauthenticated.Header().Get("Vary") != "Cookie, origin" ||
		unauthenticated.Body.String() != "{\"detail\":\"Authentication credentials were not provided.\"}" {
		t.Fatalf("anonymous GET user me = %d %q headers=%v", unauthenticated.Code, unauthenticated.Body.String(), unauthenticated.Header())
	}
}

func TestUserSessionUnknownTimezoneMatchesDjangoBadRequest(t *testing.T) {
	handler := newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			userIdentity: useridentityapi.NewEndpoints(userIdentityTimezoneNotFoundHTTPStub{}),
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/users/session/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || response.Body.String() != "{\"error\":\"The required key does not exist.\"}\n" ||
		response.Header().Get("Allow") != "GET, HEAD, OPTIONS" || response.Header().Get("loom-error") != "" {
		t.Fatalf("unknown timezone response=%d %q headers=%v", response.Code, response.Body.String(), response.Header())
	}
}

func TestUserSessionInvalidTimezoneMatchesDjangoInternalError(t *testing.T) {
	handler := newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			userIdentity: useridentityapi.NewEndpoints(userIdentityInternalErrorHTTPStub{}),
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/users/session/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || response.Body.String() != "{\"error\":\"Something went wrong please try again later\"}\n" ||
		response.Header().Get("Content-Type") != "application/json" || response.Header().Get("loom-error") != "" {
		t.Fatalf("invalid timezone response=%d %q headers=%v", response.Code, response.Body.String(), response.Header())
	}
}

func TestUserMeReadMatchesDjangoNegotiationCORSAndGZip(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	handler := newHTTPHandlerWithPendingReplacements(
		ctx,
		system.NewEndpoints(systemapi.NewSystem()),
		false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			userIdentity:     useridentityapi.NewEndpoints(userIdentityHTTPStub{}),
			userIdentityCORS: []string{"http://localhost:3000"},
		},
	)

	for _, testCase := range []struct {
		name       string
		path       string
		accept     string
		wantStatus int
		wantBody   string
	}{
		{name: "unsupported format", path: "/api/users/me/?format=xml", wantStatus: http.StatusNotFound, wantBody: `{"detail":"Not found."}`},
		{name: "empty format", path: "/api/users/me/?format=", wantStatus: http.StatusOK},
		{name: "json format", path: "/api/users/me/?format=json", wantStatus: http.StatusOK},
		{name: "unsupported accept", path: "/api/users/me/", accept: "text/html", wantStatus: http.StatusNotAcceptable, wantBody: `{"detail":"Could not satisfy the request Accept header."}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
			if testCase.accept != "" {
				request.Header.Set("Accept", testCase.accept)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != testCase.wantStatus || (testCase.wantBody != "" && response.Body.String() != testCase.wantBody) {
				t.Fatalf("response=%d %q headers=%v", response.Code, response.Body.String(), response.Header())
			}
		})
	}

	corsRequest := httptest.NewRequest(http.MethodGet, "/api/users/me/", nil)
	corsRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	corsRequest.Header.Set("Origin", "http://localhost:3000")
	corsResponse := httptest.NewRecorder()
	handler.ServeHTTP(corsResponse, corsRequest)
	if corsResponse.Code != http.StatusOK ||
		corsResponse.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" ||
		corsResponse.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("CORS response=%d headers=%v", corsResponse.Code, corsResponse.Header())
	}

	gzipRequest := httptest.NewRequest(http.MethodGet, "/api/users/me/", nil)
	gzipRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	gzipRequest.Header.Set("Accept-Encoding", "gzip")
	gzipResponse := httptest.NewRecorder()
	handler.ServeHTTP(gzipResponse, gzipRequest)
	if gzipResponse.Code != http.StatusOK || gzipResponse.Header().Get("Content-Encoding") != "gzip" ||
		gzipResponse.Header().Get("Vary") != "Cookie, Accept-Encoding, origin" {
		t.Fatalf("gzip response=%d headers=%v", gzipResponse.Code, gzipResponse.Header())
	}
	reader, err := gzip.NewReader(gzipResponse.Body)
	if err != nil {
		t.Fatal(err)
	}
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(decompressed), `"date_joined":"2026-07-01T20:04:05.123456-07:00"`) {
		t.Fatalf("decompressed body=%q", decompressed)
	}

	headRequest := httptest.NewRequest(http.MethodHead, "/api/users/me/", nil)
	headRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	headRequest.Header.Set("Accept-Encoding", "gzip")
	headResponse := httptest.NewRecorder()
	handler.ServeHTTP(headResponse, headRequest)
	if headResponse.Code != http.StatusOK || headResponse.Body.Len() != 0 ||
		headResponse.Header().Get("Content-Encoding") != "gzip" ||
		headResponse.Header().Get("Content-Length") == "" {
		t.Fatalf("HEAD gzip response=%d body=%d headers=%v", headResponse.Code, headResponse.Body.Len(), headResponse.Header())
	}
}

func TestUserMeReadRejectsOversizedBodyBeforeSessionOrDispatch(t *testing.T) {
	throttle := &publicProjectMetadataThrottleStub{userID: "00000000-0000-4000-8000-000000000b01", earlyExpire: true}
	handler := newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false),
		system.NewEndpoints(systemapi.NewSystem()),
		false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			userIdentity:          useridentityapi.NewEndpoints(userIdentityHTTPStub{}),
			userIdentitySession:   throttle,
			userIdentityCookie:    djangoSessionCookieConfig{Name: "session-id", Configured: true},
			userIdentityBodyLimit: 8,
			userIdentityBodySet:   true,
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/users/me/", strings.NewReader("123456789"))
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "short"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge || response.Body.String() != string(djangoRequestTooLargeBody) ||
		response.Header().Get("Content-Type") != "application/json" ||
		response.Header().Get("Content-Length") != strconv.Itoa(len(djangoRequestTooLargeBody)) ||
		response.Header().Get("Allow") != "" || response.Header().Get("Cache-Control") != "" ||
		response.Header().Get("Vary") != "origin, Cookie" || response.Header().Get("X-Frame-Options") != "DENY" ||
		!strings.HasPrefix(response.Header().Get("Set-Cookie"), `session-id="";`) {
		t.Fatalf("oversized response=%d %q headers=%v", response.Code, response.Body.String(), response.Header())
	}
	if throttle.calls != 0 || throttle.earlyCalls != 1 {
		t.Fatalf("oversized request session calls: inspect=%d finalize=%d", throttle.calls, throttle.earlyCalls)
	}

	now := time.Date(2026, 7, 2, 3, 4, 5, 0, time.UTC)
	refreshThrottle := &publicProjectMetadataThrottleStub{earlyRefresh: &djangosession.SessionRefresh{
		SessionKey: "refreshed", MaxAgeSeconds: 3600, ExpiresAt: now.Add(time.Hour), CookieExpiresAt: now.Add(time.Hour),
	}}
	refreshHandler := newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			userIdentity:          useridentityapi.NewEndpoints(userIdentityHTTPStub{}),
			userIdentitySession:   refreshThrottle,
			userIdentityCookie:    djangoSessionCookieConfig{Name: "session-id", Configured: true},
			userIdentityBodyLimit: 8,
			userIdentityBodySet:   true,
		},
	)
	refreshRequest := httptest.NewRequest(http.MethodGet, "/api/users/me/", strings.NewReader("123456789"))
	refreshRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	refreshResponse := httptest.NewRecorder()
	refreshHandler.ServeHTTP(refreshResponse, refreshRequest)
	if refreshResponse.Code != http.StatusRequestEntityTooLarge ||
		!strings.HasPrefix(refreshResponse.Header().Get("Set-Cookie"), "session-id=refreshed;") ||
		refreshThrottle.earlyCalls != 1 || refreshThrottle.calls != 0 {
		t.Fatalf("refresh response=%d headers=%v inspect=%d finalize=%d", refreshResponse.Code, refreshResponse.Header(), refreshThrottle.calls, refreshThrottle.earlyCalls)
	}

	readErrorThrottle := &publicProjectMetadataThrottleStub{earlyExpire: true}
	readErrorHandler := newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			userIdentity:        useridentityapi.NewEndpoints(userIdentityHTTPStub{}),
			userIdentitySession: readErrorThrottle,
			userIdentityCookie:  djangoSessionCookieConfig{Name: "session-id", Configured: true},
		},
	)
	readErrorRequest := httptest.NewRequest(http.MethodGet, "/api/users/me/", nil)
	readErrorRequest.Body = failingRequestBody{}
	readErrorRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "short"})
	readErrorResponse := httptest.NewRecorder()
	readErrorHandler.ServeHTTP(readErrorResponse, readErrorRequest)
	if readErrorResponse.Code != http.StatusInternalServerError || readErrorResponse.Header().Get("Allow") != "" ||
		!strings.HasPrefix(readErrorResponse.Header().Get("Set-Cookie"), `session-id="";`) ||
		readErrorThrottle.earlyCalls != 1 || readErrorThrottle.calls != 0 {
		t.Fatalf("read error response=%d headers=%v inspect=%d finalize=%d", readErrorResponse.Code, readErrorResponse.Header(), readErrorThrottle.calls, readErrorThrottle.earlyCalls)
	}
}

func TestUserMeReadNegotiationRunsRequestLoggerSessionInspection(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		throttle   *publicProjectMetadataThrottleStub
		wantStatus int
		wantBody   string
	}{
		{
			name:       "negotiation response survives successful lazy authentication",
			throttle:   &publicProjectMetadataThrottleStub{},
			wantStatus: http.StatusNotFound,
			wantBody:   `{"detail":"Not found."}`,
		},
		{
			name:       "lazy authentication failure replaces negotiation response",
			throttle:   &publicProjectMetadataThrottleStub{err: fmt.Errorf("session store unavailable: %w", djangosession.ErrSessionStore)},
			wantStatus: http.StatusInternalServerError,
			wantBody:   string(djangoServerErrorBody),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			activity := &publicProjectMetadataActivityStub{records: make(chan publicprojectmetadatareadsstore.APIActivityLog, 1)}
			handler := newHTTPHandlerWithPendingReplacements(
				newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{
					userIdentity:         useridentityapi.NewEndpoints(userIdentityHTTPStub{}),
					userIdentitySession:  testCase.throttle,
					userIdentityActivity: activity,
					userIdentitySecret:   "activity-secret",
				},
			)
			request := httptest.NewRequest(http.MethodGet, "/api/users/me/?format=xml", nil)
			request.Header.Set("X-Api-Key", "external-token")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != testCase.wantStatus || response.Body.String() != testCase.wantBody ||
				response.Header().Get("Vary") != "origin, Cookie" || testCase.throttle.calls != 1 ||
				testCase.throttle.timezoneCalls != 0 || testCase.throttle.earlyCalls != 0 {
				t.Fatalf("response=%d %q headers=%v inspect=%d timezone=%d finalize=%d", response.Code, response.Body.String(), response.Header(), testCase.throttle.calls, testCase.throttle.timezoneCalls, testCase.throttle.earlyCalls)
			}
			record := <-activity.records
			if record.ResponseCode != testCase.wantStatus || record.ResponseBody != testCase.wantBody {
				t.Fatalf("activity record=%#v", record)
			}
		})
	}
}

func TestUserMeReadLogsAPIKeyActivityForSuccessAndAuthenticationFailure(t *testing.T) {
	activity := &publicProjectMetadataActivityStub{records: make(chan publicprojectmetadatareadsstore.APIActivityLog, 2)}
	buildHandler := func(service useridentityapi.Service) http.Handler {
		return newHTTPHandlerWithPendingReplacements(
			newLogContext(context.Background(), io.Discard, false),
			system.NewEndpoints(systemapi.NewSystem()),
			false,
			nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
			pendingReplacementEndpoints{
				userIdentity:         useridentityapi.NewEndpoints(service),
				userIdentityActivity: activity,
				userIdentitySecret:   "activity-secret",
			},
		)
	}
	for _, testCase := range []struct {
		name       string
		service    useridentityapi.Service
		cookie     bool
		wantStatus int
	}{
		{name: "success", service: userIdentityHTTPStub{}, cookie: true, wantStatus: http.StatusOK},
		{name: "unauthenticated", service: userIdentityUnauthenticatedHTTPStub{}, wantStatus: http.StatusUnauthorized},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/users/me/?format=json", nil)
			request.Header.Set("X-Api-Key", "external-token")
			if testCase.cookie {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
			}
			response := httptest.NewRecorder()
			buildHandler(testCase.service).ServeHTTP(response, request)
			if response.Code != testCase.wantStatus {
				t.Fatalf("response=%d %q", response.Code, response.Body.String())
			}
			record := <-activity.records
			if record.TokenIdentifier != apiActivityTokenIdentifier("activity-secret", "external-token") ||
				record.Path != "/api/users/me/" || record.Method != http.MethodGet ||
				record.QueryParams != "format=json" || record.ResponseCode != testCase.wantStatus ||
				record.ResponseBody != response.Body.String() {
				t.Fatalf("activity record=%#v response=%q", record, response.Body.String())
			}
		})
	}
}

func TestUserMeReadAppliesDjangoSessionLifecycle(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	now := time.Date(2026, 7, 2, 3, 4, 5, 0, time.UTC)
	throttle := &publicProjectMetadataThrottleStub{
		userID:  "00000000-0000-4000-8000-000000000b01",
		rotated: &djangosession.SessionRefresh{SessionKey: "prepared"},
		result: djangosession.SessionRefresh{
			SessionKey:      "refreshed",
			MaxAgeSeconds:   3600,
			ExpiresAt:       now.Add(time.Hour),
			CookieExpiresAt: now.Add(time.Hour),
		},
	}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx,
		system.NewEndpoints(systemapi.NewSystem()),
		false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			userIdentity:        useridentityapi.NewEndpoints(userIdentityHTTPStub{}),
			userIdentitySession: throttle,
			userIdentityCookie:  djangoSessionCookieConfig{Name: "session-id", Configured: true},
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/users/me/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || throttle.calls != 1 || throttle.refreshCalls != 1 ||
		!strings.HasPrefix(response.Header().Get("Set-Cookie"), "session-id=refreshed;") {
		t.Fatalf("response=%d headers=%v inspect=%d refresh=%d", response.Code, response.Header(), throttle.calls, throttle.refreshCalls)
	}

	throttle.err = errors.New("session store unavailable")
	errorResponse := httptest.NewRecorder()
	handler.ServeHTTP(errorResponse, request)
	if errorResponse.Code != http.StatusInternalServerError || errorResponse.Header().Get("Cache-Control") != "" {
		t.Fatalf("session error=%d %q headers=%v", errorResponse.Code, errorResponse.Body.String(), errorResponse.Header())
	}

	throttle.err = fmt.Errorf("fallback rotation unavailable: %w", djangosession.ErrStore)
	rotationErrorResponse := httptest.NewRecorder()
	handler.ServeHTTP(rotationErrorResponse, request)
	if rotationErrorResponse.Code != http.StatusInternalServerError || rotationErrorResponse.Body.String() != string(djangoServerErrorBody) ||
		rotationErrorResponse.Header().Get("Allow") != "" || rotationErrorResponse.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("rotation error=%d %q headers=%v", rotationErrorResponse.Code, rotationErrorResponse.Body.String(), rotationErrorResponse.Header())
	}

	throttle.err = djangosession.ErrTimezoneNotFound
	timezoneErrorResponse := httptest.NewRecorder()
	handler.ServeHTTP(timezoneErrorResponse, request)
	if timezoneErrorResponse.Code != http.StatusBadRequest ||
		timezoneErrorResponse.Body.String() != `{"error":"The required key does not exist."}` ||
		timezoneErrorResponse.Header().Get("Cache-Control") != "" ||
		timezoneErrorResponse.Header().Get("Allow") != "GET, PATCH, DELETE, HEAD, OPTIONS" || throttle.timezoneCalls == 0 {
		t.Fatalf("timezone error=%d %q headers=%v timezone calls=%d", timezoneErrorResponse.Code, timezoneErrorResponse.Body.String(), timezoneErrorResponse.Header(), throttle.timezoneCalls)
	}

	throttle.err = nil
	throttle.refreshErr = errors.New("session refresh unavailable")
	refreshErrorResponse := httptest.NewRecorder()
	handler.ServeHTTP(refreshErrorResponse, request)
	if refreshErrorResponse.Code != http.StatusInternalServerError || refreshErrorResponse.Header().Get("Cache-Control") != "" {
		t.Fatalf("refresh error=%d %q headers=%v", refreshErrorResponse.Code, refreshErrorResponse.Body.String(), refreshErrorResponse.Header())
	}

	expiredThrottle := &publicProjectMetadataThrottleStub{expire: true}
	expiredHandler := newHTTPHandlerWithPendingReplacements(
		ctx,
		system.NewEndpoints(systemapi.NewSystem()),
		false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			userIdentity:        useridentityapi.NewEndpoints(userIdentityUnauthenticatedHTTPStub{}),
			userIdentitySession: expiredThrottle,
			userIdentityCookie:  djangoSessionCookieConfig{Name: "session-id", Configured: true},
		},
	)
	expiredResponse := httptest.NewRecorder()
	expiredHandler.ServeHTTP(expiredResponse, request)
	if expiredResponse.Code != http.StatusUnauthorized ||
		expiredResponse.Header().Get("Set-Cookie") != `session-id=""; expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=0; Path=/; SameSite=Lax` {
		t.Fatalf("expired session=%d %q headers=%v", expiredResponse.Code, expiredResponse.Body.String(), expiredResponse.Header())
	}
}

func TestProjectMembershipReadsOptionsDoesNotInvokeReadEndpoints(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	readCalls := 0
	metadataCalls := 0
	stub := projectMembershipReadsHTTPStub{readCalls: &readCalls, metadataCalls: &metadataCalls}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx,
		system.NewEndpoints(systemapi.NewSystem()),
		false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			projectMembershipReads:    projectmembershipreadsapi.NewEndpoints(stub),
			projectMembershipMetadata: stub,
		},
	)
	projectID := "00000000-0000-4000-8000-000000000003"
	for _, path := range []string{
		"/api/workspaces/acme/projects/" + projectID + "/project-members/me/",
		"/api/users/me/workspaces/acme/project-roles/",
		"/api/workspaces/acme/members/",
		"/api/workspaces/acme/project-members/",
	} {
		request := httptest.NewRequest(http.MethodOptions, path, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("OPTIONS %s = %d %q", path, response.Code, response.Body.String())
		}
	}
	if readCalls != 0 {
		t.Fatalf("OPTIONS invoked %d full read endpoints", readCalls)
	}
	if metadataCalls != 4 {
		t.Fatalf("metadata authorization calls = %d, want 4", metadataCalls)
	}
}

func TestProjectStateReadsHTTPMountsAndDjangoProtocol(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	metadataCalls := 0
	stub := projectStateReadsHTTPStub{metadataCalls: &metadataCalls}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx,
		system.NewEndpoints(systemapi.NewSystem()),
		false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			projectStateReads:    projectstatereadsapi.NewEndpoints(stub),
			projectStateMetadata: stub,
		},
	)
	projectID := "00000000-0000-4000-8000-000000000002"
	stateID := "00000000-0000-4000-8000-000000000001"
	listPath := "/api/workspaces/acme/projects/" + projectID + "/states/"
	detailPath := listPath + stateID + "/"
	intakePath := "/api/workspaces/acme/projects/" + projectID + "/intake-state/"

	for _, testCase := range []struct {
		path         string
		bodyFragment string
		allow        string
	}{
		{path: listPath, bodyFragment: `"order":1`, allow: "GET, POST, HEAD, OPTIONS"},
		{path: detailPath, bodyFragment: `"name":"Backlog"`, allow: "GET, PATCH, DELETE, HEAD, OPTIONS"},
		{path: intakePath, bodyFragment: `"group":"triage"`, allow: "GET, HEAD, OPTIONS"},
	} {
		request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), testCase.bodyFragment) {
			t.Errorf("GET %s = %d %q", testCase.path, response.Code, response.Body.String())
		}
		if got := response.Header().Get("Allow"); got != testCase.allow {
			t.Errorf("GET %s Allow=%q want=%q", testCase.path, got, testCase.allow)
		}
		if !strings.Contains(response.Header().Get("Vary"), "Cookie") || !strings.Contains(response.Header().Get("Vary"), "origin") {
			t.Errorf("GET %s Vary=%q", testCase.path, response.Header().Get("Vary"))
		}

		headRequest := httptest.NewRequest(http.MethodHead, testCase.path, nil)
		headRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
		headResponse := httptest.NewRecorder()
		handler.ServeHTTP(headResponse, headRequest)
		if headResponse.Code != http.StatusOK || headResponse.Body.Len() != 0 {
			t.Errorf("HEAD %s = %d %q", testCase.path, headResponse.Code, headResponse.Body.String())
		}
	}

	groupedRequest := httptest.NewRequest(http.MethodGet, listPath+"?grouped=false&grouped=true", nil)
	groupedResponse := httptest.NewRecorder()
	handler.ServeHTTP(groupedResponse, groupedRequest)
	if groupedResponse.Code != http.StatusOK || !strings.HasPrefix(groupedResponse.Body.String(), `{"backlog":`) {
		t.Fatalf("duplicate grouped query = %d %q", groupedResponse.Code, groupedResponse.Body.String())
	}

	optionsRequest := httptest.NewRequest(http.MethodOptions, intakePath, nil)
	optionsRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
	optionsResponse := httptest.NewRecorder()
	handler.ServeHTTP(optionsResponse, optionsRequest)
	if optionsResponse.Code != http.StatusOK || !strings.Contains(optionsResponse.Body.String(), `"name":"Intake State Endpoint"`) {
		t.Fatalf("OPTIONS intake = %d %q", optionsResponse.Code, optionsResponse.Body.String())
	}
	if metadataCalls != 1 {
		t.Fatalf("metadata authorization calls=%d want=1", metadataCalls)
	}

	formatResponse := httptest.NewRecorder()
	handler.ServeHTTP(formatResponse, httptest.NewRequest(http.MethodGet, listPath+"?format=xml", nil))
	if formatResponse.Code != http.StatusNotFound || formatResponse.Body.String() != "{\"detail\":\"Not found.\"}\n" {
		t.Fatalf("unsupported format = %d %q", formatResponse.Code, formatResponse.Body.String())
	}

	acceptRequest := httptest.NewRequest(http.MethodGet, listPath, nil)
	acceptRequest.Header.Set("Accept", "application/xml")
	acceptResponse := httptest.NewRecorder()
	handler.ServeHTTP(acceptResponse, acceptRequest)
	if acceptResponse.Code != http.StatusNotAcceptable {
		t.Fatalf("unsupported Accept = %d %q", acceptResponse.Code, acceptResponse.Body.String())
	}
}

func TestGeneratedProjectStateListClientDecodesDjangoWireShapes(t *testing.T) {
	decode := projectstatereadsclient.DecodeListResponse(loomhttp.ResponseDecoder, false)
	for _, testCase := range []struct {
		name string
		body string
		kind string
	}{
		{name: "list", body: `[{"name":"Backlog"}]`, kind: "array"},
		{name: "grouped", body: `{"backlog":[{"name":"Backlog"}]}`, kind: "map"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			response := &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(testCase.body)),
			}
			result, err := decode(response)
			if err != nil {
				t.Fatal(err)
			}
			switch testCase.kind {
			case "array":
				var decoded []json.RawMessage
				if err := json.Unmarshal(result.(loom.JSONValue), &decoded); err != nil || decoded == nil {
					t.Fatalf("decoded list type=%T value=%v", result, result)
				}
			case "map":
				var decoded map[string][]json.RawMessage
				if err := json.Unmarshal(result.(loom.JSONValue), &decoded); err != nil || decoded == nil {
					t.Fatalf("decoded grouped type=%T value=%v", result, result)
				}
			}
		})
	}
}

func TestProjectLabelReadsHTTPMountsAndDjangoProtocol(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	stub := projectLabelReadsHTTPStub{}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx,
		system.NewEndpoints(systemapi.NewSystem()),
		false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{projectLabelReads: projectlabelreadsapi.NewEndpoints(stub)},
	)
	projectID := "00000000-0000-4000-8000-000000000002"
	labelID := "00000000-0000-4000-8000-000000000001"
	listPath := "/api/workspaces/acme/projects/" + projectID + "/issue-labels/"
	detailPath := listPath + labelID + "/"

	for _, testCase := range []struct {
		path  string
		allow string
	}{
		{path: listPath, allow: "GET, POST, HEAD, OPTIONS"},
		{path: detailPath, allow: "GET, PUT, PATCH, DELETE, HEAD, OPTIONS"},
	} {
		request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"name":"Bug"`) || !strings.Contains(response.Body.String(), `"parent":null`) {
			t.Errorf("GET %s = %d %q", testCase.path, response.Code, response.Body.String())
		}
		if got := response.Header().Get("Allow"); got != testCase.allow {
			t.Errorf("GET %s Allow=%q want=%q", testCase.path, got, testCase.allow)
		}
		if !strings.Contains(response.Header().Get("Vary"), "Cookie") || !strings.Contains(response.Header().Get("Vary"), "origin") {
			t.Errorf("GET %s Vary=%q", testCase.path, response.Header().Get("Vary"))
		}

		headRequest := httptest.NewRequest(http.MethodHead, testCase.path, nil)
		headRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
		headResponse := httptest.NewRecorder()
		handler.ServeHTTP(headResponse, headRequest)
		if headResponse.Code != http.StatusOK || headResponse.Body.Len() != 0 {
			t.Errorf("HEAD %s = %d %q", testCase.path, headResponse.Code, headResponse.Body.String())
		}
		for _, header := range []string{"Allow", "Content-Type", "Vary", "X-Frame-Options"} {
			if got, want := headResponse.Header().Get(header), response.Header().Get(header); got != want {
				t.Errorf("HEAD %s %s=%q want GET header %q", testCase.path, header, got, want)
			}
		}
		for _, query := range []string{"?format=xml", "?format=json"} {
			head := httptest.NewRecorder()
			get := httptest.NewRecorder()
			handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, testCase.path+query, nil))
			handler.ServeHTTP(head, httptest.NewRequest(http.MethodHead, testCase.path+query, nil))
			if head.Code != get.Code || head.Body.Len() != 0 || head.Header().Get("Allow") != get.Header().Get("Allow") {
				t.Errorf("HEAD %s%s = %d %q; GET status=%d", testCase.path, query, head.Code, head.Body.String(), get.Code)
			}
		}
	}

	formatResponse := httptest.NewRecorder()
	handler.ServeHTTP(formatResponse, httptest.NewRequest(http.MethodGet, listPath+"?format=xml", nil))
	if formatResponse.Code != http.StatusNotFound || formatResponse.Body.String() != "{\"detail\":\"Not found.\"}\n" {
		t.Fatalf("unsupported format = %d %q", formatResponse.Code, formatResponse.Body.String())
	}

	acceptRequest := httptest.NewRequest(http.MethodGet, listPath, nil)
	acceptRequest.Header.Set("Accept", "application/xml")
	acceptResponse := httptest.NewRecorder()
	handler.ServeHTTP(acceptResponse, acceptRequest)
	if acceptResponse.Code != http.StatusNotAcceptable {
		t.Fatalf("unsupported Accept = %d %q", acceptResponse.Code, acceptResponse.Body.String())
	}
}

func TestProjectLabelReadsResponseWriterUsesCapturedRepositoryJSON(t *testing.T) {
	response := httptest.NewRecorder()
	raw := json.RawMessage(`[{"sort_order":100.0}]`)
	writer := &projectLabelReadsResponseWriter{
		ResponseWriter: response,
		rawResponse:    &systemapi.ProjectLabelReadsResponseCapture{Data: raw},
	}
	if _, err := writer.Write([]byte(`[{"sort_order":100}]`)); err != nil {
		t.Fatal(err)
	}
	if response.Body.String() != string(raw) {
		t.Fatalf("body=%q want=%q", response.Body.String(), raw)
	}
}

func TestPublicProjectMetadataReadsHTTPMountsAndDjangoProtocol(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	stub := publicProjectMetadataReadsHTTPStub{}
	throttle := &publicProjectMetadataThrottleStub{allowed: true}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx,
		system.NewEndpoints(systemapi.NewSystem()),
		false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(stub),
			publicProjectMetadataThrottle: throttle,
		},
	)
	for _, testCase := range []struct {
		path               string
		bodyFragment       string
		optionsName        string
		wantAcceptEncoding bool
	}{
		{path: "/api/public/anchor/example/cycles/", bodyFragment: `"name":"Sprint 1"`, optionsName: "Project Cycles Endpoint"},
		{path: "/api/public/anchor/example/labels/", bodyFragment: `"name":"Bug"`, optionsName: "Project Labels Endpoint"},
		{path: "/api/public/anchor/example/members/", bodyFragment: `"member":null`, optionsName: "Project Members Endpoint"},
		{path: "/api/public/anchor/example/meta/", bodyFragment: `"identifier":"PMETA"`, optionsName: "Project Meta Data Endpoint"},
		{path: "/api/public/anchor/example/modules/", bodyFragment: `"name":"Launch"`, optionsName: "Project Modules Endpoint"},
		{path: "/api/public/anchor/example/settings/", bodyFragment: `"anchor":"example"`, optionsName: "Project Deploy Board Public Settings Endpoint", wantAcceptEncoding: true},
		{path: "/api/public/anchor/example/states/", bodyFragment: `"name":"Backlog"`, optionsName: "Project States Endpoint"},
		{path: "/api/public/workspaces/demo/projects/00000000-0000-0000-0000-000000000000/anchor/", bodyFragment: `"anchor":"example"`, optionsName: "Workspace Project Anchor Endpoint", wantAcceptEncoding: true},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, testCase.path, nil))
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), testCase.bodyFragment) {
			t.Errorf("GET %s = %d %q", testCase.path, response.Code, response.Body.String())
		}
		if response.Header().Get("Allow") != "GET, HEAD, OPTIONS" || !strings.Contains(response.Header().Get("Vary"), "Cookie") {
			t.Errorf("GET %s headers=%v", testCase.path, response.Header())
		}
		hasAcceptEncoding := strings.Contains(response.Header().Get("Vary"), "Accept-Encoding")
		if hasAcceptEncoding != testCase.wantAcceptEncoding {
			t.Errorf("GET %s Vary=%q, want Accept-Encoding=%v", testCase.path, response.Header().Get("Vary"), testCase.wantAcceptEncoding)
		}
		if response.Header().Get("Content-Length") != strconv.Itoa(response.Body.Len()) {
			t.Errorf("GET %s Content-Length=%q body=%d", testCase.path, response.Header().Get("Content-Length"), response.Body.Len())
		}

		headResponse := httptest.NewRecorder()
		handler.ServeHTTP(headResponse, httptest.NewRequest(http.MethodHead, testCase.path, nil))
		if headResponse.Code != http.StatusOK || headResponse.Body.Len() != 0 || headResponse.Header().Get("Content-Length") != response.Header().Get("Content-Length") {
			t.Errorf("HEAD %s = %d %q", testCase.path, headResponse.Code, headResponse.Body.String())
		}

		optionsResponse := httptest.NewRecorder()
		handler.ServeHTTP(optionsResponse, httptest.NewRequest(http.MethodOptions, testCase.path, nil))
		wantOptions := `{"name":"` + testCase.optionsName + `","description":"","renders":["application/json"],"parses":["application/json","application/x-www-form-urlencoded","multipart/form-data"]}`
		if optionsResponse.Code != http.StatusOK || optionsResponse.Body.String() != wantOptions {
			t.Errorf("OPTIONS %s = %d %q", testCase.path, optionsResponse.Code, optionsResponse.Body.String())
		}
		if strings.Contains(optionsResponse.Header().Get("Vary"), "Accept-Encoding") {
			t.Errorf("small OPTIONS %s unexpectedly varies on Accept-Encoding: %v", testCase.path, optionsResponse.Header())
		}
		if optionsResponse.Header().Get("Content-Length") != strconv.Itoa(optionsResponse.Body.Len()) {
			t.Errorf("OPTIONS %s Content-Length=%q body=%d", testCase.path, optionsResponse.Header().Get("Content-Length"), optionsResponse.Body.Len())
		}

		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodTrace, "BREW"} {
			methodResponse := httptest.NewRecorder()
			handler.ServeHTTP(methodResponse, httptest.NewRequest(method, testCase.path, nil))
			wantBody := fmt.Sprintf(`{"detail":"Method \"%s\" not allowed."}`, method)
			if methodResponse.Code != http.StatusMethodNotAllowed || methodResponse.Body.String() != wantBody ||
				methodResponse.Header().Get("Allow") != "GET, HEAD, OPTIONS" {
				t.Errorf("%s %s = %d %q headers=%v", method, testCase.path, methodResponse.Code, methodResponse.Body.String(), methodResponse.Header())
			}
		}

		indentedRequest := httptest.NewRequest(http.MethodOptions, testCase.path, nil)
		indentedRequest.Header.Set("Accept", "application/json; indent=4")
		indentedResponse := httptest.NewRecorder()
		handler.ServeHTTP(indentedResponse, indentedRequest)
		if indentedResponse.Code != http.StatusOK ||
			indentedResponse.Header().Get("Vary") != "Accept-Encoding, Cookie, origin" ||
			!strings.Contains(indentedResponse.Body.String(), `"name": "`+testCase.optionsName+`"`) ||
			indentedResponse.Header().Get("Content-Length") != strconv.Itoa(indentedResponse.Body.Len()) {
			t.Errorf("indented OPTIONS %s = %d Vary=%q body=%q", testCase.path, indentedResponse.Code, indentedResponse.Header().Get("Vary"), indentedResponse.Body.String())
		}
	}
	for _, testCase := range []struct {
		method string
		body   string
	}{
		{method: "get", body: `"name":"Bug"`},
		{method: "Head", body: ""},
		{method: "oPtIoNs", body: `"name":"Project Labels Endpoint"`},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(testCase.method, "/api/public/anchor/example/labels/", nil))
		if response.Code != http.StatusOK || (testCase.body != "" && !strings.Contains(response.Body.String(), testCase.body)) ||
			(testCase.body == "" && response.Body.Len() != 0) {
			t.Errorf("case-variant %s = %d %q", testCase.method, response.Code, response.Body.String())
		}
	}
	if throttle.calls != 83 {
		t.Fatalf("throttle calls=%d want=83", throttle.calls)
	}
}

func TestPublicProjectSettingsHTTPPropagatesAuthenticatedUserForTimezoneRendering(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	var userID string
	stub := publicProjectMetadataReadsUserContextHTTPStub{userID: &userID}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx,
		system.NewEndpoints(systemapi.NewSystem()),
		false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads: publicprojectmetadatareadsapi.NewEndpoints(stub),
			publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{
				allowed: true,
				userID:  "00000000-0000-4000-8000-000000000050",
			},
		},
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/settings/", nil))
	if response.Code != http.StatusOK || userID != "00000000-0000-4000-8000-000000000050" {
		t.Fatalf("status=%d userID=%q body=%q", response.Code, userID, response.Body.String())
	}
}

func TestPublicProjectMetadataReadsHTTPMatchesDjangoThrottleResponse(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	throttle := &publicProjectMetadataThrottleStub{wait: 17 * time.Second}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx,
		system.NewEndpoints(systemapi.NewSystem()),
		false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: throttle,
		},
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", nil))
	want := `{"error_code":5900,"error_message":"RATE_LIMIT_EXCEEDED"}`
	if response.Code != http.StatusTooManyRequests || response.Body.String() != want ||
		response.Header().Get("Retry-After") != "17" ||
		response.Header().Get("Content-Length") != strconv.Itoa(len(want)) {
		t.Fatalf("throttle response=%d %q headers=%v", response.Code, response.Body.String(), response.Header())
	}
}

func TestPublicProjectMetadataReadsHTTPMatchesDjangoAcceptParsing(t *testing.T) {
	var logs bytes.Buffer
	ctx := newLogContext(context.Background(), &logs, false)
	throttle := &publicProjectMetadataThrottleStub{allowed: true}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx, system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: throttle,
		},
	)
	path := "/api/public/anchor/example/labels/"

	malformedRequest := httptest.NewRequest(http.MethodGet, path, nil)
	malformedRequest.Header.Set("Accept", "application/json; indent")
	malformedResponse := httptest.NewRecorder()
	handler.ServeHTTP(malformedResponse, malformedRequest)
	if malformedResponse.Code != http.StatusOK || !strings.Contains(malformedResponse.Body.String(), `"name":"Bug"`) {
		t.Fatalf("malformed parameter response=%d %q", malformedResponse.Code, malformedResponse.Body.String())
	}

	extendedRequest := httptest.NewRequest(http.MethodGet, path, nil)
	extendedRequest.Header.Set("Accept", "application/json; indent*=UTF-8''4")
	extendedResponse := httptest.NewRecorder()
	handler.ServeHTTP(extendedResponse, extendedRequest)
	if extendedResponse.Code != http.StatusOK || !strings.Contains(extendedResponse.Body.String(), `"name": "Bug"`) {
		t.Fatalf("extended parameter response=%d %q", extendedResponse.Code, extendedResponse.Body.String())
	}

	emptyRequest := httptest.NewRequest(http.MethodGet, path, nil)
	emptyRequest.Header["Accept"] = []string{""}
	emptyResponse := httptest.NewRecorder()
	handler.ServeHTTP(emptyResponse, emptyRequest)
	if emptyResponse.Code != http.StatusNotAcceptable || emptyResponse.Body.String() != `{"detail":"Could not satisfy the request Accept header."}` {
		t.Fatalf("empty Accept response=%d %q", emptyResponse.Code, emptyResponse.Body.String())
	}

	invalidEncodingRequest := httptest.NewRequest(http.MethodGet, path, nil)
	invalidEncodingRequest.Header.Set("Accept", "application/json; indent*=bogus''%34")
	invalidEncodingResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidEncodingResponse, invalidEncodingRequest)
	if invalidEncodingResponse.Code != http.StatusInternalServerError ||
		invalidEncodingResponse.Body.String() != `{"error":"Something went wrong please try again later"}` ||
		invalidEncodingResponse.Header().Get("Content-Type") != "application/json" ||
		invalidEncodingResponse.Header().Get("Allow") != "GET, HEAD, OPTIONS" {
		t.Fatalf("invalid encoding response=%d %q headers=%v", invalidEncodingResponse.Code, invalidEncodingResponse.Body.String(), invalidEncodingResponse.Header())
	}
	if !strings.Contains(logs.String(), "public project metadata compatibility failure") {
		t.Fatalf("invalid encoding omitted exception log: %s", logs.String())
	}
	if throttle.calls != 4 {
		t.Fatalf("session/throttle calls=%d want=4", throttle.calls)
	}
}

func TestPublicProjectMetadataReadsHTTPExpiresInvalidDjangoSessionCookie(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	throttle := &publicProjectMetadataThrottleStub{allowed: true, expire: true}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx,
		system.NewEndpoints(systemapi.NewSystem()),
		false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: throttle,
			publicProjectMetadataCookie:   djangoSessionCookieConfig{Name: "custom-session", Domain: ".example.test"},
		},
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", nil))
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies=%v headers=%v", cookies, response.Header())
	}
	cookie := cookies[0]
	if cookie.Name != "custom-session" || cookie.Path != "/" ||
		cookie.Domain != ".example.test" || cookie.MaxAge != -1 || !cookie.Expires.Equal(time.Unix(0, 0).UTC()) ||
		cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expired cookie=%#v", cookie)
	}
	if got := response.Header().Get("Set-Cookie"); got != `custom-session=""; Domain=.example.test; expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=0; Path=/; SameSite=Lax` {
		t.Fatalf("Set-Cookie=%q", got)
	}
}

func TestPublicProjectMetadataReadsHTTPProcessesSessionsOnNegotiationErrors(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	for _, testCase := range []struct {
		name       string
		path       string
		accept     string
		status     int
		expire     bool
		refresh    bool
		wantCookie string
	}{
		{
			name: "invalid format expires missing session", path: "/api/public/anchor/example/labels/?format=xml",
			status: http.StatusNotFound, expire: true, wantCookie: `session-id=""; expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=0; Path=/; SameSite=Lax`,
		},
		{
			name: "unacceptable media type refreshes session", path: "/api/public/anchor/example/labels/",
			accept: "application/xml", status: http.StatusNotAcceptable, refresh: true,
			wantCookie: "session-id=valid-session-key; expires=Wed, 02 Sep 2026 03:04:05 GMT; HttpOnly; Max-Age=604800; Path=/; SameSite=Lax",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			throttle := &publicProjectMetadataThrottleStub{
				allowed: true, expire: testCase.expire, refresh: testCase.refresh,
				result: djangosession.SessionRefresh{
					SessionKey: "valid-session-key", ExpiresAt: time.Date(2026, time.September, 2, 3, 4, 5, 0, time.UTC), MaxAgeSeconds: 604800,
				},
			}
			handler := newHTTPHandlerWithPendingReplacements(
				ctx, system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{
					publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
					publicProjectMetadataThrottle: throttle,
				},
			)
			request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
			request.Header.Set("Accept", testCase.accept)
			request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "valid-session-key"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != testCase.status || response.Header().Get("Set-Cookie") != testCase.wantCookie || throttle.calls != 1 {
				t.Fatalf("response=%d Set-Cookie=%q session calls=%d", response.Code, response.Header().Get("Set-Cookie"), throttle.calls)
			}
		})
	}

	t.Run("malformed format escape remains an unsupported format", func(t *testing.T) {
		handler := newHTTPHandlerWithPendingReplacements(
			ctx, system.NewEndpoints(systemapi.NewSystem()), false,
			nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
			pendingReplacementEndpoints{
				publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
				publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{allowed: true},
			},
		)
		request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", nil)
		request.URL.RawQuery = "format=%ZZ"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("response=%d %q", response.Code, response.Body.String())
		}
	})

	t.Run("too many query fields fail before anonymous throttling", func(t *testing.T) {
		var logs bytes.Buffer
		throttle := &publicProjectMetadataThrottleStub{allowed: true}
		handler := newHTTPHandlerWithPendingReplacements(
			newLogContext(context.Background(), &logs, false), system.NewEndpoints(systemapi.NewSystem()), false,
			nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
			pendingReplacementEndpoints{
				publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
				publicProjectMetadataThrottle: throttle,
			},
		)
		request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", nil)
		request.URL.RawQuery = strings.Repeat("x=&", 1000)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusInternalServerError || throttle.calls != 1 ||
			!strings.Contains(logs.String(), "public project metadata compatibility failure") {
			t.Fatalf("response=%d %q session inspections=%d", response.Code, response.Body.String(), throttle.calls)
		}
	})
}

func TestPublicProjectMetadataReadsHTTPDeletionCookiesUseSecurePrefixes(t *testing.T) {
	for _, name := range []string{"__Secure-session-id", "__Host-session-id"} {
		t.Run(name, func(t *testing.T) {
			headers := make(http.Header)
			if err := expireDjangoSessionCookie(headers, djangoSessionCookieConfig{Name: name}); err != nil {
				t.Fatal(err)
			}
			if got := headers.Get("Set-Cookie"); !strings.HasSuffix(got, "; Secure") {
				t.Fatalf("Set-Cookie=%q", got)
			}
		})
	}
}

func TestPublicProjectMetadataReadsHTTPEmptyCookieNameFailsWhenSessionMustChange(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	for _, testCase := range []struct {
		name     string
		throttle *publicProjectMetadataThrottleStub
	}{
		{name: "delete", throttle: &publicProjectMetadataThrottleStub{allowed: true, expire: true}},
		{name: "refresh", throttle: &publicProjectMetadataThrottleStub{
			allowed: true, refresh: true, result: djangosession.SessionRefresh{SessionKey: "refreshed-session-key"},
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			handler := newHTTPHandlerWithPendingReplacements(
				ctx, system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{
					publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
					publicProjectMetadataThrottle: testCase.throttle,
					publicProjectMetadataCookie:   djangoSessionCookieConfig{Configured: true},
				},
			)
			request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", nil)
			request.Header.Set("Cookie", "=valid-session-key")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusInternalServerError || response.Body.String() != string(djangoServerErrorBody) || response.Header().Get("Set-Cookie") != "" {
				t.Fatalf("response=%d %q headers=%v", response.Code, response.Body.String(), response.Header())
			}
		})
	}
}

func TestPublicProjectMetadataReadsHTTPDoesNotChangeAdminCookieForAnchorText(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		path      string
		status    int
		refresh   bool
		wantValue string
	}{
		{name: "expire public session", path: "/api/public/anchor/contains-instances-value/labels/", status: http.StatusOK},
		{name: "refresh public session", path: "/api/public/anchor/instances/labels/", status: http.StatusOK, refresh: true, wantValue: "refreshed-public-session"},
		{name: "slashless route", path: "/api/public/anchor/instances/labels", status: http.StatusMovedPermanently},
		{name: "slashless refresh public session", path: "/api/public/anchor/instances/labels", status: http.StatusMovedPermanently, refresh: true, wantValue: "refreshed-public-session"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			throttle := &publicProjectMetadataThrottleStub{
				allowed: true, expire: !scenario.refresh, refresh: scenario.refresh,
				result: djangosession.SessionRefresh{SessionKey: scenario.wantValue, MaxAgeSeconds: 604800},
			}
			handler := newHTTPHandlerWithPendingReplacements(
				newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{
					publicProjectMetadataReads:       publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
					publicProjectMetadataThrottle:    throttle,
					publicProjectMetadataCookie:      djangoSessionCookieConfig{Name: "session-id"},
					publicProjectMetadataAdminCookie: djangoSessionCookieConfig{Name: "admin-session-id"},
				},
			)
			request := httptest.NewRequest(http.MethodGet, scenario.path, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "normal-session-key"})
			request.AddCookie(&http.Cookie{Name: "admin-session-id", Value: "admin-session-key"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			cookies := response.Result().Cookies()
			if response.Code != scenario.status || len(cookies) != 1 || cookies[0].Name != "session-id" || cookies[0].Value != scenario.wantValue {
				t.Fatalf("response=%d cookies=%v", response.Code, cookies)
			}
		})
	}
}

func TestPublicProjectMetadataReadsHTTPRefreshesDjangoSessionAfterSuccess(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	expiresAt := time.Date(2026, time.September, 2, 3, 4, 5, 0, time.UTC)
	throttle := &publicProjectMetadataThrottleStub{
		allowed: true,
		refresh: true,
		result: djangosession.SessionRefresh{
			SessionKey: "valid-session-key", ExpiresAt: expiresAt, MaxAgeSeconds: 604800,
		},
	}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx, system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: throttle,
			publicProjectMetadataCookie:   djangoSessionCookieConfig{Name: "custom-session", Domain: ".example.test", Secure: true},
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", nil)
	request.AddCookie(&http.Cookie{Name: "custom-session", Value: "valid-session-key"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	want := "custom-session=valid-session-key; Domain=.example.test; expires=Wed, 02 Sep 2026 03:04:05 GMT; HttpOnly; Max-Age=604800; Path=/; SameSite=Lax; Secure"
	if response.Code != http.StatusOK || response.Header().Get("Set-Cookie") != want {
		t.Fatalf("response=%d Set-Cookie=%q", response.Code, response.Header().Get("Set-Cookie"))
	}
}

func TestPublicProjectMetadataReadsHTTPSessionRefreshFailureReplacesBufferedResponse(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	throttle := &publicProjectMetadataThrottleStub{allowed: true, refresh: true, refreshErr: errors.New("session disappeared")}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx, system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: throttle,
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "valid-session-key"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	want := string(djangoServerErrorBody)
	if response.Code != http.StatusInternalServerError || response.Body.String() != want || response.Header().Get("Set-Cookie") != "" ||
		response.Header().Get("X-Frame-Options") != "" || response.Header().Get("Content-Length") != "" ||
		response.Header().Get("Content-Encoding") != "" || response.Header().Get("Vary") != "origin" {
		t.Fatalf("response=%d %q headers=%v", response.Code, response.Body.String(), response.Header())
	}
}

func TestPublicProjectMetadataReadsHTTPRenderQueryFailureUsesDjangoHTML500(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	service, err := systemapi.NewPublicProjectMetadataReads(publicProjectMetadataReadsRenderErrorRepositoryStub{})
	if err != nil {
		t.Fatal(err)
	}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx, system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(service),
			publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{allowed: true},
		},
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", nil))
	if response.Code != http.StatusInternalServerError || response.Body.String() != string(djangoServerErrorBody) ||
		response.Header().Get("Content-Type") != "text/html; charset=utf-8" || response.Header().Get("Allow") != "" {
		t.Fatalf("response=%d %q headers=%v", response.Code, response.Body.String(), response.Header())
	}
}

func TestPublicProjectAnchorDuplicateQueryUsesDjangoJSON500(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	service, err := systemapi.NewPublicProjectMetadataReads(publicProjectMetadataReadsQueryErrorRepositoryStub{})
	if err != nil {
		t.Fatal(err)
	}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx, system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(service),
			publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{allowed: true},
		},
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/public/workspaces/demo/projects/00000000-0000-0000-0000-000000000000/anchor/", nil))
	if response.Code != http.StatusInternalServerError || response.Body.String() != "{\"error\":\"Something went wrong please try again later\"}" ||
		response.Header().Get("Content-Type") != "application/json" || response.Header().Get("Allow") != "GET, HEAD, OPTIONS" {
		t.Fatalf("response=%d %q headers=%v", response.Code, response.Body.String(), response.Header())
	}
}

func TestPublicProjectMetadataReadsHTTPConcurrentSessionDeletionReturnsDjangoBadRequest(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	for _, refreshErr := range []error{djangosession.ErrSessionNotFound, djangosession.ErrSessionUpdate} {
		throttle := &publicProjectMetadataThrottleStub{allowed: true, refresh: true, refreshErr: refreshErr}
		handler := newHTTPHandlerWithPendingReplacements(
			ctx, system.NewEndpoints(systemapi.NewSystem()), false,
			nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
			pendingReplacementEndpoints{
				publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
				publicProjectMetadataThrottle: throttle,
			},
		)
		request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", nil)
		request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "deleted-session-key"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || response.Body.String() != string(djangoBadRequestBody) ||
			response.Header().Get("Content-Type") != "text/html; charset=utf-8" ||
			response.Header().Get("X-Frame-Options") != "" || response.Header().Get("Content-Length") != "" ||
			strings.Contains(response.Header().Get("Vary"), "Cookie") {
			t.Fatalf("refresh error=%v response=%d %q headers=%v", refreshErr, response.Code, response.Body.String(), response.Header())
		}
	}
}

func TestRefreshDjangoSessionCookieOmitsPersistenceForBrowserClose(t *testing.T) {
	headers := make(http.Header)
	if err := refreshDjangoSessionCookie(headers, djangoSessionCookieConfig{}, djangosession.SessionRefresh{
		SessionKey: "valid-session-key", BrowserClose: true,
	}); err != nil {
		t.Fatal(err)
	}
	if got := headers.Get("Set-Cookie"); got != "session-id=valid-session-key; HttpOnly; Path=/; SameSite=Lax" {
		t.Fatalf("Set-Cookie=%q", got)
	}
}

func TestRefreshDjangoSessionCookieUsesSeparateCookieExpiry(t *testing.T) {
	headers := make(http.Header)
	if err := refreshDjangoSessionCookie(headers, djangoSessionCookieConfig{}, djangosession.SessionRefresh{
		SessionKey: "valid-session-key", ExpiresAt: time.Date(2026, time.September, 2, 3, 4, 5, 0, time.UTC),
		CookieExpiresAt: time.Date(2026, time.September, 2, 3, 4, 4, 0, time.UTC), MaxAgeSeconds: 61,
	}); err != nil {
		t.Fatal(err)
	}
	if got := headers.Get("Set-Cookie"); !strings.Contains(got, "expires=Wed, 02 Sep 2026 03:04:04 GMT") {
		t.Fatalf("Set-Cookie=%q", got)
	}
}

func TestDjangoSessionCookieHelpersRejectExplicitEmptyName(t *testing.T) {
	config := djangoSessionCookieConfig{Configured: true}
	if err := expireDjangoSessionCookie(make(http.Header), config); err == nil {
		t.Fatal("expireDjangoSessionCookie() accepted an empty cookie name")
	}
	if err := refreshDjangoSessionCookie(make(http.Header), config, djangosession.SessionRefresh{SessionKey: "valid-session-key"}); err == nil {
		t.Fatal("refreshDjangoSessionCookie() accepted an empty cookie name")
	}
}

func TestLastDjangoSessionCookieMatchesDjangoDuplicateCookieParsing(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Cookie", "session-id=first-session; session-id=last-session")
	cookie := lastDjangoSessionCookie(request, "session-id")
	if cookie == nil || cookie.Value != "last-session" {
		t.Fatalf("cookie=%#v", cookie)
	}

	request.Header["Cookie"] = []string{"session-id=first-session", "session-id=second-session"}
	cookie = lastDjangoSessionCookie(request, "session-id")
	if cookie == nil || cookie.Value != "first-session,session-id=second-session" {
		t.Fatalf("repeated-header cookie=%#v", cookie)
	}

	request.Header.Set("Cookie", `session-id="quoted\054value\x"`)
	cookie = lastDjangoSessionCookie(request, "session-id")
	if cookie == nil || cookie.Value != "quoted,valuex" {
		t.Fatalf("quoted cookie=%#v", cookie)
	}

	request.Header.Set("Cookie", `session-id="\303\251abcdef"`)
	cookie = lastDjangoSessionCookie(request, "session-id")
	if cookie == nil || cookie.Value != "\u00c3\u00a9abcdef" {
		t.Fatalf("octal-unquoted cookie=%#v", cookie)
	}

	request.Header.Set("Cookie", `session-id="éabcdef"`)
	cookie = lastDjangoSessionCookie(request, "session-id")
	if cookie == nil || cookie.Value != "\u00c3\u00a9abcdef" {
		t.Fatalf("Latin-1 raw-byte cookie=%#v", cookie)
	}

	request.Header.Set("Cookie", "session-id=\""+string([]byte{0xff})+"abcdefg\"")
	cookie = lastDjangoSessionCookie(request, "session-id")
	if cookie == nil || cookie.Value != "\u00ffabcdefg" {
		t.Fatalf("isolated Latin-1 cookie=%#v", cookie)
	}

	request.Header.Set("Cookie", "; ;")
	if cookie := lastDjangoSessionCookie(request, ""); cookie != nil {
		t.Fatalf("empty cookie chunks produced cookie=%#v", cookie)
	}
}

func TestPublicProjectMetadataReadsHTTPMatchesDjangoCORS(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	throttle := &publicProjectMetadataThrottleStub{allowed: true}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx,
		system.NewEndpoints(systemapi.NewSystem()),
		false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: throttle,
			publicProjectMetadataCORS:     []string{"https://app.example.test"},
		},
	)
	path := "/api/public/anchor/example/labels/"
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Origin", "https://app.example.test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK ||
		response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" ||
		response.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("CORS GET=%d headers=%v", response.Code, response.Header())
	}

	preflight := httptest.NewRequest(http.MethodOptions, path, nil)
	preflight.Header.Set("Origin", "https://app.example.test")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflight.Header.Set("Access-Control-Request-Headers", "x-api-key")
	preflightResponse := httptest.NewRecorder()
	handler.ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusOK || preflightResponse.Body.Len() != 0 ||
		preflightResponse.Header().Get("Content-Length") != "0" ||
		preflightResponse.Header().Get("Content-Type") != "text/html; charset=utf-8" ||
		preflightResponse.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" ||
		preflightResponse.Header().Get("Access-Control-Allow-Credentials") != "true" ||
		preflightResponse.Header().Get("Access-Control-Allow-Headers") != "accept, authorization, content-type, user-agent, x-csrftoken, x-requested-with, X-API-Key" ||
		preflightResponse.Header().Get("Access-Control-Allow-Methods") != "DELETE, GET, OPTIONS, PATCH, POST, PUT" ||
		preflightResponse.Header().Get("Access-Control-Max-Age") != "86400" {
		t.Fatalf("CORS preflight=%d %q headers=%v", preflightResponse.Code, preflightResponse.Body.String(), preflightResponse.Header())
	}
	if throttle.calls != 1 {
		t.Fatalf("preflight consumed throttle capacity: calls=%d", throttle.calls)
	}

	denied := httptest.NewRequest(http.MethodOptions, path, nil)
	denied.Header.Set("Origin", "https://denied.example.test")
	denied.Header.Set("Access-Control-Request-Method", http.MethodGet)
	deniedResponse := httptest.NewRecorder()
	handler.ServeHTTP(deniedResponse, denied)
	if deniedResponse.Code != http.StatusOK || deniedResponse.Body.Len() != 0 || deniedResponse.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("denied CORS preflight=%d %q headers=%v", deniedResponse.Code, deniedResponse.Body.String(), deniedResponse.Header())
	}
	if throttle.calls != 1 {
		t.Fatalf("denied preflight consumed throttle capacity: calls=%d", throttle.calls)
	}

	emptyMethod := httptest.NewRequest(http.MethodOptions, path, nil)
	emptyMethod.Header.Set("Origin", "https://app.example.test")
	emptyMethod.Header["Access-Control-Request-Method"] = []string{""}
	emptyMethodResponse := httptest.NewRecorder()
	handler.ServeHTTP(emptyMethodResponse, emptyMethod)
	if emptyMethodResponse.Code != http.StatusOK || emptyMethodResponse.Body.Len() != 0 ||
		emptyMethodResponse.Header().Get("Content-Length") != "0" ||
		emptyMethodResponse.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" {
		t.Fatalf("empty-method preflight=%d %q headers=%v", emptyMethodResponse.Code, emptyMethodResponse.Body.String(), emptyMethodResponse.Header())
	}
	if throttle.calls != 1 {
		t.Fatalf("empty-method preflight consumed throttle capacity: calls=%d", throttle.calls)
	}

	repeatedOrigin := httptest.NewRequest(http.MethodGet, path, nil)
	repeatedOrigin.Header["Origin"] = []string{"https://app.example.test", "https://denied.example.test"}
	repeatedOriginResponse := httptest.NewRecorder()
	handler.ServeHTTP(repeatedOriginResponse, repeatedOrigin)
	if repeatedOriginResponse.Code != http.StatusOK || repeatedOriginResponse.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("repeated-origin response=%d headers=%v", repeatedOriginResponse.Code, repeatedOriginResponse.Header())
	}

	repeatedMethod := httptest.NewRequest(http.MethodOptions, path, nil)
	repeatedMethod.Header.Set("Origin", "https://app.example.test")
	repeatedMethod.Header["Access-Control-Request-Method"] = []string{"", http.MethodGet}
	repeatedMethodResponse := httptest.NewRecorder()
	handler.ServeHTTP(repeatedMethodResponse, repeatedMethod)
	if repeatedMethodResponse.Code != http.StatusOK || repeatedMethodResponse.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" {
		t.Fatalf("repeated-method preflight=%d headers=%v", repeatedMethodResponse.Code, repeatedMethodResponse.Header())
	}
	if throttle.calls != 2 {
		t.Fatalf("repeated-method preflight consumed throttle capacity: calls=%d", throttle.calls)
	}
}

func TestPublicProjectMetadataReadsHTTPCORSRejectsMalformedOrigins(t *testing.T) {
	for _, origin := range []string{
		"https://[broken", "https://[V1.foo]",
	} {
		t.Run(origin, func(t *testing.T) {
			ctx := newLogContext(context.Background(), io.Discard, false)
			throttle := &publicProjectMetadataThrottleStub{allowed: true}
			handler := newHTTPHandlerWithPendingReplacements(
				ctx, system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{
					publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
					publicProjectMetadataThrottle: throttle,
				},
			)
			request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", nil)
			request.Header.Set("Origin", origin)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK || response.Header().Get("Access-Control-Allow-Origin") != "" ||
				response.Header().Get("Access-Control-Allow-Credentials") != "" {
				t.Fatalf("response=%d headers=%v", response.Code, response.Header())
			}

			preflight := httptest.NewRequest(http.MethodOptions, "/api/public/anchor/example/labels/", nil)
			preflight.Header.Set("Origin", origin)
			preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
			preflightResponse := httptest.NewRecorder()
			handler.ServeHTTP(preflightResponse, preflight)
			if preflightResponse.Code != http.StatusOK || preflightResponse.Header().Get("Content-Length") != "0" ||
				preflightResponse.Header().Get("Access-Control-Allow-Origin") != "" || throttle.calls != 1 {
				t.Fatalf("preflight=%d headers=%v throttle calls=%d", preflightResponse.Code, preflightResponse.Header(), throttle.calls)
			}
		})
	}
	for _, origin := range []string{
		"null", "://", "https://example.com/%", "https://[v1.foo]",
		"https://foo[::1]", "https://[::1]foo", "https://[::1][::2]", "http://[fe80::1%25eth0]",
	} {
		if !djangoCORSOriginValid(origin) {
			t.Errorf("urlsplit-compatible origin %q rejected", origin)
		}
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", "https://app.example.test/ignored-origin-path")
	headers := make(http.Header)
	if !publicProjectMetadataReadsCORSHeaders(headers, request, []string{"HTTPS://app.example.test/different-config-path"}) ||
		headers.Get("Access-Control-Allow-Origin") != "https://app.example.test/ignored-origin-path" {
		t.Fatalf("scheme/netloc allowlist comparison headers=%v", headers)
	}
}

func TestPublicProjectMetadataReadsHTTPCORSDecodesOriginAsASGILatin1(t *testing.T) {
	handler := newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{allowed: true},
		},
	)
	origin := "https://exa／mple.com"
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", nil)
	request.Header.Set("Origin", origin)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	want := djangoLatin1String([]byte(origin))
	if response.Code != http.StatusOK || response.Header().Get("Access-Control-Allow-Origin") != want {
		t.Fatalf("response=%d headers=%v want origin=%q", response.Code, response.Header(), want)
	}
}

func TestPublicProjectMetadataReadsHTTPRejectsOversizedBodiesBeforeAuditOrDispatch(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	for _, testCase := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "trailing GET", method: http.MethodGet, path: "/api/public/anchor/example/labels/"},
		{name: "trailing unsupported method", method: http.MethodPost, path: "/api/public/anchor/example/states/"},
		{name: "slashless redirect", method: http.MethodGet, path: "/api/public/anchor/example/labels"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			throttle := &publicProjectMetadataThrottleStub{allowed: true, earlyExpire: true}
			activity := &publicProjectMetadataActivityStub{records: make(chan publicprojectmetadatareadsstore.APIActivityLog, 1)}
			handler := newHTTPHandlerWithPendingReplacements(
				ctx, system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{
					publicProjectMetadataReads:     publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
					publicProjectMetadataThrottle:  throttle,
					publicProjectMetadataActivity:  activity,
					publicProjectMetadataSecret:    "audit-secret",
					publicProjectMetadataBodyLimit: 8,
					publicProjectMetadataBodySet:   true,
				},
			)
			request := httptest.NewRequest(testCase.method, testCase.path, strings.NewReader("123456789"))
			request.Header.Set("X-Api-Key", "sensitive-token")
			request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "short"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusRequestEntityTooLarge || response.Body.String() != string(djangoRequestTooLargeBody) ||
				response.Header().Get("Content-Type") != "application/json" ||
				response.Header().Get("Content-Length") != strconv.Itoa(len(djangoRequestTooLargeBody)) ||
				response.Header().Get("Allow") != "" || response.Header().Get("X-Frame-Options") != "DENY" ||
				response.Header().Get("Set-Cookie") == "" {
				t.Fatalf("response=%d %q headers=%v", response.Code, response.Body.String(), response.Header())
			}
			if throttle.calls != 0 {
				t.Fatalf("oversized request reached session/throttle: calls=%d", throttle.calls)
			}
			select {
			case record := <-activity.records:
				t.Fatalf("oversized request was audited: %#v", record)
			default:
			}
		})
	}
}

func TestPublicProjectMetadataReadsHTTPHonorsNonPositiveBodyLimits(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		limit int64
		body  io.Reader
	}{
		{name: "zero rejects nonempty", limit: 0, body: strings.NewReader("x")},
		{name: "negative rejects empty", limit: -1},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			throttle := &publicProjectMetadataThrottleStub{allowed: true}
			handler := newHTTPHandlerWithPendingReplacements(
				newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{
					publicProjectMetadataReads:     publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
					publicProjectMetadataThrottle:  throttle,
					publicProjectMetadataBodyLimit: testCase.limit,
					publicProjectMetadataBodySet:   true,
				},
			)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", testCase.body))
			if response.Code != http.StatusRequestEntityTooLarge || throttle.earlyCalls != 1 || throttle.calls != 0 {
				t.Fatalf("response=%d early=%d throttle=%d", response.Code, throttle.earlyCalls, throttle.calls)
			}
		})
	}
	throttle := &publicProjectMetadataThrottleStub{allowed: true}
	handler := newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:     publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle:  throttle,
			publicProjectMetadataBodyLimit: 0,
			publicProjectMetadataBodySet:   true,
		},
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", nil))
	if response.Code != http.StatusOK || throttle.calls != 1 {
		t.Fatalf("empty zero-limit response=%d throttle=%d", response.Code, throttle.calls)
	}
}

func TestPublicProjectMetadataReadsHTTPAllowsUnknownLengthBodyLikeDjango(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	throttle := &publicProjectMetadataThrottleStub{allowed: true}
	activity := &publicProjectMetadataActivityStub{records: make(chan publicprojectmetadatareadsstore.APIActivityLog, 1)}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx, system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:     publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle:  throttle,
			publicProjectMetadataActivity:  activity,
			publicProjectMetadataSecret:    "audit-secret",
			publicProjectMetadataBodyLimit: 8,
			publicProjectMetadataBodySet:   true,
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", strings.NewReader("123456789"))
	request.ContentLength = -1
	request.Header.Set("X-Api-Key", "sensitive-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || throttle.earlyCalls != 0 || throttle.calls != 1 {
		t.Fatalf("response=%d headers=%v throttle calls=%d", response.Code, response.Header(), throttle.calls)
	}
	select {
	case record := <-activity.records:
		if record.ResponseCode != http.StatusOK || record.Body != "123456789" {
			t.Fatalf("audit record=%#v", record)
		}
	case <-time.After(time.Second):
		t.Fatal("unknown-length request was not audited")
	}
}

func TestPublicProjectMetadataReadsHTTPRejectsUnknownLengthBodyWithNegativeLimit(t *testing.T) {
	throttle := &publicProjectMetadataThrottleStub{allowed: true}
	handler := newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:     publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle:  throttle,
			publicProjectMetadataBodyLimit: -1,
			publicProjectMetadataBodySet:   true,
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", strings.NewReader("body"))
	request.ContentLength = -1
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge || throttle.earlyCalls != 1 || throttle.calls != 0 {
		t.Fatalf("response=%d headers=%v early=%d throttle=%d", response.Code, response.Header(), throttle.earlyCalls, throttle.calls)
	}
}

func TestPublicProjectMetadataReadsHTTPEarlyBodyFailureExpiresOnlyInvalidCookie(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	for _, testCase := range []struct {
		name       string
		cookie     string
		wantCookie bool
	}{
		{name: "no cookie"},
		{name: "valid-shaped cookie", cookie: "valid-session-key"},
		{name: "invalid cookie", cookie: "short", wantCookie: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			throttle := &publicProjectMetadataThrottleStub{allowed: true, earlyExpire: testCase.wantCookie}
			handler := newHTTPHandlerWithPendingReplacements(
				ctx, system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{
					publicProjectMetadataReads:     publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
					publicProjectMetadataThrottle:  throttle,
					publicProjectMetadataBodyLimit: 8,
					publicProjectMetadataBodySet:   true,
				},
			)
			request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", strings.NewReader("123456789"))
			if testCase.cookie != "" {
				request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: testCase.cookie})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			hasCookieVary := strings.Contains(response.Header().Get("Vary"), "Cookie")
			wantVary := "origin"
			if testCase.wantCookie {
				wantVary = "Cookie, origin"
			}
			if response.Code != http.StatusRequestEntityTooLarge || hasCookieVary != testCase.wantCookie ||
				(response.Header().Get("Set-Cookie") != "") != testCase.wantCookie || response.Header().Get("Vary") != wantVary {
				t.Fatalf("response=%d headers=%v", response.Code, response.Header())
			}
		})
	}
}

func TestPublicProjectMetadataReadsHTTPRefreshesUnaccessedSessionAfter413(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		path       string
		cookieName string
		wantCookie string
	}{
		{name: "standard", path: "/api/public/anchor/example/labels/", cookieName: "session-id", wantCookie: "session-id=refreshed-session"},
		{name: "instances anchor", path: "/api/public/anchor/instances-example/labels/", cookieName: "session-id", wantCookie: "session-id=refreshed-session"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			refresh := &djangosession.SessionRefresh{SessionKey: "refreshed-session", MaxAgeSeconds: 3600, ExpiresAt: time.Date(2026, time.September, 2, 3, 4, 5, 0, time.UTC)}
			throttle := &publicProjectMetadataThrottleStub{allowed: true, earlyRefresh: refresh}
			handler := newHTTPHandlerWithPendingReplacements(
				newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{
					publicProjectMetadataReads:       publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
					publicProjectMetadataThrottle:    throttle,
					publicProjectMetadataCookie:      djangoSessionCookieConfig{Name: "session-id"},
					publicProjectMetadataAdminCookie: djangoSessionCookieConfig{Name: "admin-session-id"},
					publicProjectMetadataBodyLimit:   8,
					publicProjectMetadataBodySet:     true,
				},
			)
			request := httptest.NewRequest(http.MethodGet, testCase.path, strings.NewReader("123456789"))
			request.AddCookie(&http.Cookie{Name: testCase.cookieName, Value: "valid-session-key"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusRequestEntityTooLarge || !strings.HasPrefix(response.Header().Get("Set-Cookie"), testCase.wantCookie) ||
				response.Header().Get("Vary") != "origin" || throttle.earlyCalls != 1 || throttle.calls != 0 {
				t.Fatalf("response=%d headers=%v early=%d throttle=%d", response.Code, response.Header(), throttle.earlyCalls, throttle.calls)
			}
		})
	}
}

func TestPublicProjectMetadataReadsHTTPReplaces413OnUnaccessedSessionFailure(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{name: "session store", err: fmt.Errorf("database unavailable: %w", djangosession.ErrSessionStore), status: http.StatusInternalServerError, body: string(djangoServerErrorBody)},
		{name: "signed non-object session", err: djangosession.ErrInvalidSessionShape, status: http.StatusInternalServerError, body: string(djangoServerErrorBody)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var logs bytes.Buffer
			throttle := &publicProjectMetadataThrottleStub{earlyErr: testCase.err}
			handler := newHTTPHandlerWithPendingReplacements(
				newLogContext(context.Background(), &logs, false), system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{
					publicProjectMetadataReads:     publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
					publicProjectMetadataThrottle:  throttle,
					publicProjectMetadataBodyLimit: 8,
					publicProjectMetadataBodySet:   true,
				},
			)
			request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", strings.NewReader("123456789"))
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session-key"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != testCase.status || response.Body.String() != testCase.body ||
				response.Header().Get("Vary") != "origin" || throttle.earlyCalls != 1 ||
				!strings.Contains(logs.String(), "public project metadata compatibility failure") {
				t.Fatalf("response=%d %q early=%d logs=%s", response.Code, response.Body.String(), throttle.earlyCalls, logs.String())
			}
		})
	}
}

func TestPublicProjectMetadataReadsHTTPBodyReadFailureRetainsValidUnaccessedSessionCookie(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	throttle := &publicProjectMetadataThrottleStub{allowed: true}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx, system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:     publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle:  throttle,
			publicProjectMetadataBodyLimit: 8,
			publicProjectMetadataBodySet:   true,
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", nil)
	request.Body = failingRequestBody{}
	request.ContentLength = -1
	request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "valid-session-key"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || response.Body.String() != string(djangoServerErrorBody) ||
		response.Header().Get("Set-Cookie") != "" || strings.Contains(response.Header().Get("Vary"), "Cookie") || throttle.calls != 0 {
		t.Fatalf("response=%d %q headers=%v throttle calls=%d", response.Code, response.Body.String(), response.Header(), throttle.calls)
	}
}

func TestPublicProjectMetadataReadsHTTPSlashlessRequestsReachMiddlewareBeforeRedirect(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	throttle := &publicProjectMetadataThrottleStub{expire: true}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx, system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: throttle,
			publicProjectMetadataCORS:     []string{"https://app.example.test"},
		},
	)
	request := httptest.NewRequest(http.MethodPost, "/api/public/anchor/example/labels?search=bug", nil)
	request.Header.Set("Origin", "https://app.example.test")
	request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "missing-session-key"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusMovedPermanently || response.Header().Get("Location") != "/api/public/anchor/example/labels/?search=bug" ||
		response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" || response.Header().Get("Set-Cookie") == "" ||
		response.Header().Get("Content-Length") != "" || response.Header().Get("X-Frame-Options") != "" || throttle.calls != 1 {
		t.Fatalf("slashless redirect=%d headers=%v session calls=%d", response.Code, response.Header(), throttle.calls)
	}

	preflight := httptest.NewRequest("oPtIoNs", "/api/public/anchor/example/labels", nil)
	preflight.Header.Set("Origin", "https://app.example.test")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflightResponse := httptest.NewRecorder()
	handler.ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusOK || preflightResponse.Header().Get("Location") != "" ||
		preflightResponse.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" || throttle.calls != 1 {
		t.Fatalf("slashless preflight=%d headers=%v session calls=%d", preflightResponse.Code, preflightResponse.Header(), throttle.calls)
	}

	for _, escapedCase := range []struct {
		path string
		want string
	}{
		{path: "/api/public/anchor/foo%3Fbar/labels?search=bug", want: "/api/public/anchor/foo%3Fbar/labels/?search=bug"},
		{path: "/api/public/anchor/foo%7ebar/labels", want: "/api/public/anchor/foo~bar/labels/"},
		{path: "/api/public/anchor/foo%41bar/labels", want: "/api/public/anchor/fooAbar/labels/"},
		{path: "/api/public/anchor/example/labels?q=é", want: "/api/public/anchor/example/labels/?q=%C3%A9"},
	} {
		escaped := httptest.NewRequest(http.MethodGet, escapedCase.path, nil)
		escapedResponse := httptest.NewRecorder()
		handler.ServeHTTP(escapedResponse, escaped)
		if escapedResponse.Code != http.StatusMovedPermanently || escapedResponse.Header().Get("Location") != escapedCase.want {
			t.Fatalf("escaped slashless redirect for %q=%d location=%q want=%q", escapedCase.path, escapedResponse.Code, escapedResponse.Header().Get("Location"), escapedCase.want)
		}
	}
}

func TestPublicProjectMetadataReadsHTTPSlashlessUnsafeMethodFailsWhenDebugEnabled(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	throttle := &publicProjectMetadataThrottleStub{allowed: true}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx, system.NewEndpoints(systemapi.NewSystem()), true,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: throttle,
		},
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/public/anchor/example/labels?q=é", nil))
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "RuntimeError at /api/public/anchor/example/labels") ||
		!strings.Contains(response.Body.String(), "Django can&#39;t redirect") ||
		!strings.Contains(response.Body.String(), "/api/public/anchor/example/labels/?q=%C3%A9") ||
		response.Header().Get("Location") != "" || response.Header().Get("Vary") != "Cookie, origin" ||
		response.Header().Get("X-Frame-Options") != "" || response.Header().Get("Content-Length") != "" || throttle.calls != 1 {
		t.Fatalf("response=%d %q headers=%v throttle calls=%d", response.Code, response.Body.String(), response.Header(), throttle.calls)
	}
}

func TestPublicProjectMetadataReadsHTTPCatchesCaddyCleanedMalformedPaths(t *testing.T) {
	activity := &publicProjectMetadataActivityStub{records: make(chan publicprojectmetadatareadsstore.APIActivityLog, 1)}
	throttle := &publicProjectMetadataThrottleStub{allowed: true}
	handler := newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: throttle,
			publicProjectMetadataActivity: activity,
			publicProjectMetadataSecret:   "audit-secret",
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/x//labels/", nil)
	request.Header.Set("X-Api-Key", "token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || response.Body.String() != string(djangoNotFoundBody) ||
		response.Header().Get("X-Frame-Options") != "DENY" || response.Header().Get("Location") != "" || throttle.calls != 1 {
		t.Fatalf("response=%d %q headers=%v throttle=%d", response.Code, response.Body.String(), response.Header(), throttle.calls)
	}
	select {
	case record := <-activity.records:
		if record.Path != "/api/public/anchor/x//labels/" || record.ResponseCode != http.StatusNotFound || record.ResponseBody != string(djangoNotFoundBody) {
			t.Fatalf("activity record=%#v", record)
		}
	default:
		t.Fatal("missing API activity record")
	}
}

func TestPublicProjectMetadataReadsHTTPSlashlessDebugErrorBypassesInnerResponseMiddleware(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	throttle := &publicProjectMetadataThrottleStub{allowed: true}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx, system.NewEndpoints(systemapi.NewSystem()), true,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: throttle,
		},
	)
	request := httptest.NewRequest(http.MethodPost, "/api/public/anchor/example/labels", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || response.Header().Get("Content-Encoding") != "" ||
		response.Header().Get("Vary") != "Cookie, origin" || response.Header().Get("X-Frame-Options") != "" ||
		response.Header().Get("Content-Length") != "" || throttle.calls != 1 {
		t.Fatalf("response=%d headers=%v throttle calls=%d", response.Code, response.Header(), throttle.calls)
	}
	if !strings.Contains(response.Body.String(), "RuntimeError at /api/public/anchor/example/labels") {
		t.Fatalf("body=%q", response.Body.String())
	}
}

func TestPublicProjectMetadataReadsHTTPRequestLogUsesInnerDjangoResponse(t *testing.T) {
	var logs bytes.Buffer
	ctx := newLogContext(context.Background(), &logs, false)
	handler := newHTTPHandlerWithPendingReplacements(
		ctx, system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{allowed: true, userID: "user-id"},
		},
	)
	path := "/api/public/anchor/example/labels"
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, nil)
	request.Header.Set("User-Agent", "request-log-test")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusMovedPermanently {
		t.Fatalf("client response=%d", response.Code)
	}
	decoder := json.NewDecoder(bytes.NewReader(logs.Bytes()))
	var statuses []int
	for {
		var entry map[string]any
		if err := decoder.Decode(&entry); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if entry["path"] == path && entry[log.MessageKey] == "POST "+path+" 404" {
			if entry["user_id"] != "user-id" || entry["user_agent"] != "request-log-test" {
				t.Fatalf("request log fields=%v", entry)
			}
			statuses = append(statuses, int(entry["status_code"].(float64)))
		}
	}
	if !slices.Equal(statuses, []int{http.StatusNotFound}) {
		t.Fatalf("request log statuses=%v logs=%s", statuses, logs.String())
	}
}

func TestPublicProjectMetadataReadsHTTPMalformedPathUsesOnlyDjangoRequestLog(t *testing.T) {
	var logs bytes.Buffer
	handler := newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), &logs, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{allowed: true, userID: "user-id"},
		},
	)
	requestPath := "/api/public/anchor/example//labels/"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, requestPath, nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("client response=%d", response.Code)
	}
	decoder := json.NewDecoder(bytes.NewReader(logs.Bytes()))
	requestLogs := 0
	for {
		var entry map[string]any
		if err := decoder.Decode(&entry); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if entry[log.HTTPURLKey] == requestPath {
			t.Fatalf("malformed path entered ordinary request logger: %v", entry)
		}
		if entry["path"] == requestPath && entry["status_code"] != nil {
			requestLogs++
			if entry[log.MessageKey] != "GET "+requestPath+" 404" || entry["user_id"] != "user-id" {
				t.Fatalf("Django request log fields=%v", entry)
			}
		}
	}
	if requestLogs != 1 {
		t.Fatalf("Django request log count=%d logs=%s", requestLogs, logs.String())
	}
}

func TestPublicProjectMetadataReadsHTTPRequestLogCanonicalizesDjangoFullPath(t *testing.T) {
	var logs bytes.Buffer
	handler := newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), &logs, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{allowed: true},
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/foo%7ebar/labels/?q=é", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("response=%d", response.Code)
	}
	decoder := json.NewDecoder(bytes.NewReader(logs.Bytes()))
	for {
		var entry map[string]any
		if err := decoder.Decode(&entry); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if entry[log.MessageKey] == "GET /api/public/anchor/foo~bar/labels/?q=%C3%A9 200" {
			if entry["path"] != "/api/public/anchor/foo~bar/labels/" {
				t.Fatalf("request log=%v", entry)
			}
			return
		}
	}
	t.Fatalf("missing canonical request log: %s", logs.String())
}

func TestPublicProjectMetadataReadsHTTPRequestLogSkipsLazyAuthenticationFailure(t *testing.T) {
	var logs bytes.Buffer
	ctx := newLogContext(context.Background(), &logs, false)
	handler := newHTTPHandlerWithPendingReplacements(
		ctx, system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads: publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{
				err: fmt.Errorf("database unavailable: %w", djangosession.ErrSessionStore),
			},
		},
	)
	path := "/api/public/anchor/example/labels"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("client response=%d", response.Code)
	}
	decoder := json.NewDecoder(bytes.NewReader(logs.Bytes()))
	foundExceptionLog := false
	for {
		var entry map[string]any
		if err := decoder.Decode(&entry); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if entry[log.MessageKey] == "public project metadata compatibility failure" {
			foundExceptionLog = true
		}
		if entry["path"] == path && entry["status_code"] != nil {
			t.Fatalf("lazy authentication failure emitted request log: %v", entry)
		}
	}
	if !foundExceptionLog {
		t.Fatalf("lazy authentication failure omitted exception log: %s", logs.String())
	}
}

func TestPublicProjectMetadataReadsHTTPRequestLogIncludesSessionInvalidationFailure(t *testing.T) {
	var logs bytes.Buffer
	ctx := newLogContext(context.Background(), &logs, false)
	handler := newHTTPHandlerWithPendingReplacements(
		ctx, system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads: publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{
				err: fmt.Errorf("delete failed: %w", publicprojectmetadatareadsstore.ErrSessionInvalidation),
			},
		},
	)
	path := "/api/public/anchor/example/labels/"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != http.StatusInternalServerError || response.Body.String() != `{"error":"Something went wrong please try again later"}` ||
		response.Header().Get("Allow") != "GET, HEAD, OPTIONS" {
		t.Fatalf("client response=%d %q headers=%v", response.Code, response.Body.String(), response.Header())
	}
	decoder := json.NewDecoder(bytes.NewReader(logs.Bytes()))
	for {
		var entry map[string]any
		if err := decoder.Decode(&entry); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if entry["path"] == path {
			if entry["status_code"] != float64(http.StatusInternalServerError) || entry["user_id"] != nil {
				t.Fatalf("request log fields=%v", entry)
			}
			return
		}
	}
	t.Fatalf("missing request log: %s", logs.String())
}

func TestPublicProjectMetadataReadsHTTPPersistsRedactedXAPIKeyAuditLog(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	throttle := &publicProjectMetadataThrottleStub{allowed: true}
	activity := &publicProjectMetadataActivityStub{records: make(chan publicprojectmetadatareadsstore.APIActivityLog, 1)}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx, system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: throttle,
			publicProjectMetadataActivity: activity,
			publicProjectMetadataSecret:   "audit-secret",
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/?search=bug", nil)
	request.RemoteAddr = "192.0.2.30:1234"
	request.Header["X-Api-Key"] = []string{"sensitive-token", "second-token-part"}
	request.Header["User-Agent"] = []string{"audit-test", "second-agent-part"}
	request.Header["X_Underscored"] = []string{"discarded"}
	request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "session-key"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	select {
	case record := <-activity.records:
		if response.Code != http.StatusOK || record.TokenIdentifier != apiActivityTokenIdentifier("audit-secret", "sensitive-token,second-token-part") ||
			record.Path != "/api/public/anchor/example/labels/" || record.Method != http.MethodGet || record.QueryParams != "search=bug" ||
			record.ResponseCode != http.StatusOK || record.IPAddress != "192.0.2.30" || record.UserAgent != "audit-test,second-agent-part" ||
			strings.Contains(record.Headers.(string), "sensitive-token") || !strings.Contains(record.Headers.(string), "[REDACTED]") ||
			strings.Contains(record.Headers.(string), "Underscored") || strings.Contains(record.Headers.(string), "discarded") {
			t.Fatalf("response=%d record=%#v", response.Code, record)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for API activity log")
	}
}

func TestAPIActivityIPAddressTrustsOnlyConfiguredProxies(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		forwarded  string
		want       string
	}{
		{name: "trusted proxy", remoteAddr: "172.18.0.2:1234", forwarded: "198.51.100.1, 203.0.113.2", want: "203.0.113.2"},
		{name: "untrusted peer", remoteAddr: "198.51.100.9:1234", forwarded: "203.0.113.8", want: "198.51.100.9"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ""
			handler := loomhttp.RequestMetadataMiddleware(requestMetadataPolicy)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
				got = apiActivityIPAddress(request)
			}))
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.RemoteAddr = test.remoteAddr
			request.Header.Set("X-Forwarded-For", test.forwarded)
			handler.ServeHTTP(httptest.NewRecorder(), request)
			if got != test.want {
				t.Fatalf("client address = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPublicProjectMetadataReadsHTTPDecodesAuditHeadersAsASGILatin1(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	activity := &publicProjectMetadataActivityStub{records: make(chan publicprojectmetadatareadsstore.APIActivityLog, 1)}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx, system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{allowed: true},
			publicProjectMetadataActivity: activity,
			publicProjectMetadataSecret:   "audit-secret",
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", nil)
	request.Header["X-Api-Key"] = []string{string([]byte{0xff})}
	request.Header["User-Agent"] = []string{string([]byte{0xff})}
	request.Header["X-Visible"] = []string{string([]byte{0xff})}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	select {
	case record := <-activity.records:
		if response.Code != http.StatusOK || record.TokenIdentifier != apiActivityTokenIdentifier("audit-secret", "ÿ") ||
			record.UserAgent != "ÿ" || !strings.Contains(record.Headers.(string), "ÿ") {
			t.Fatalf("response=%d record=%#v", response.Code, record)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for API activity log")
	}
}

func TestPublicProjectMetadataReadsHTTPAuditMatchesDjangoMiddlewarePosition(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	for _, testCase := range []struct {
		name             string
		method           string
		path             string
		wantClientStatus int
		wantClientBody   string
		wantAuditStatus  int
		wantAuditBody    string
	}{
		{
			name: "HEAD records rendered GET body before the server suppresses it", method: http.MethodHead,
			path: "/api/public/anchor/example/labels/", wantClientStatus: http.StatusOK,
			wantAuditStatus: http.StatusOK, wantAuditBody: `"name":"Bug"`,
		},
		{
			name: "slashless records inner 404 before CommonMiddleware redirects", method: http.MethodPost,
			path: "/api/public/anchor/example/labels", wantClientStatus: http.StatusMovedPermanently,
			wantAuditStatus: http.StatusNotFound, wantAuditBody: "The requested resource was not found on this server.",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			activity := &publicProjectMetadataActivityStub{records: make(chan publicprojectmetadatareadsstore.APIActivityLog, 1)}
			handler := newHTTPHandlerWithPendingReplacements(
				ctx, system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{
					publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
					publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{allowed: true},
					publicProjectMetadataActivity: activity,
					publicProjectMetadataSecret:   "audit-secret",
				},
			)
			request := httptest.NewRequest(testCase.method, testCase.path, nil)
			request.Header.Set("X-Api-Key", "sensitive-token")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != testCase.wantClientStatus || response.Body.String() != testCase.wantClientBody {
				t.Fatalf("client response=%d %q", response.Code, response.Body.String())
			}
			select {
			case record := <-activity.records:
				if record.ResponseCode != testCase.wantAuditStatus || !strings.Contains(record.ResponseBody.(string), testCase.wantAuditBody) {
					t.Fatalf("audit record=%#v", record)
				}
			default:
				t.Fatal("request returned before its API activity record was submitted")
			}
		})
	}
}

func TestPublicProjectMetadataReadsHTTPSlashlessSessionFailureAuditsHTML500(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	activity := &publicProjectMetadataActivityStub{records: make(chan publicprojectmetadatareadsstore.APIActivityLog, 1)}
	handler := newHTTPHandlerWithPendingReplacements(
		ctx, system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads: publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{
				err: fmt.Errorf("database unavailable: %w", djangosession.ErrSessionStore),
			},
			publicProjectMetadataActivity: activity,
			publicProjectMetadataSecret:   "audit-secret",
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels", nil)
	request.Header.Set("X-Api-Key", "sensitive-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || response.Body.String() != string(djangoServerErrorBody) {
		t.Fatalf("client response=%d %q", response.Code, response.Body.String())
	}
	select {
	case record := <-activity.records:
		if record.ResponseCode != http.StatusInternalServerError || record.ResponseBody != string(djangoServerErrorBody) {
			t.Fatalf("audit record=%#v", record)
		}
	default:
		t.Fatal("missing API activity record")
	}
}

func TestPublicProjectMetadataReadsHTTPNegotiationSessionFailureAuditsHTML500(t *testing.T) {
	for _, sessionErr := range []error{
		fmt.Errorf("database unavailable: %w", djangosession.ErrSessionStore),
		djangosession.ErrInvalidSessionShape,
		fmt.Errorf("delete failed: %w", publicprojectmetadatareadsstore.ErrSessionInvalidation),
	} {
		t.Run(sessionErr.Error(), func(t *testing.T) {
			var logs bytes.Buffer
			ctx := newLogContext(context.Background(), &logs, false)
			activity := &publicProjectMetadataActivityStub{records: make(chan publicprojectmetadatareadsstore.APIActivityLog, 1)}
			handler := newHTTPHandlerWithPendingReplacements(
				ctx, system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{
					publicProjectMetadataReads: publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
					publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{
						err: sessionErr,
					},
					publicProjectMetadataActivity: activity,
					publicProjectMetadataSecret:   "audit-secret",
				},
			)
			request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/?format=xml", nil)
			request.Header.Set("X-Api-Key", "sensitive-token")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			want := string(djangoServerErrorBody)
			if response.Code != http.StatusInternalServerError || response.Body.String() != want {
				t.Fatalf("client response=%d %q", response.Code, response.Body.String())
			}
			select {
			case record := <-activity.records:
				if record.ResponseCode != http.StatusInternalServerError || record.ResponseBody != want {
					t.Fatalf("audit record=%#v", record)
				}
			default:
				t.Fatal("missing API activity record")
			}
			if strings.Contains(logs.String(), "GET /api/public/anchor/example/labels/?format=xml 500") {
				t.Fatalf("lazy authentication failure emitted request log=%s", logs.String())
			}
		})
	}
}

func TestPublicProjectMetadataReadsHTTPDistinguishesSessionAndThrottleFailures(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	for _, testCase := range []struct {
		name        string
		err         error
		status      int
		contentType string
		body        string
		allow       string
	}{
		{
			name: "session store", err: fmt.Errorf("database unavailable: %w", djangosession.ErrSessionStore),
			contentType: "text/html; charset=utf-8", body: string(djangoServerErrorBody),
		},
		{
			name: "user store", err: fmt.Errorf("database unavailable: %w", djangosession.ErrUserStore),
			contentType: "text/html; charset=utf-8", body: string(djangoServerErrorBody),
		},
		{
			name: "malformed user identifier", err: djangosession.ErrInvalidUserID,
			contentType: "text/html; charset=utf-8", body: string(djangoServerErrorBody),
		},
		{
			name: "signed non-object session", err: djangosession.ErrInvalidSessionShape,
			contentType: "text/html; charset=utf-8", body: string(djangoServerErrorBody),
		},
		{
			name: "missing user timezone", err: djangosession.ErrTimezoneNotFound, status: http.StatusBadRequest,
			contentType: "application/json", body: `{"error":"The required key does not exist."}`, allow: "GET, HEAD, OPTIONS",
		},
		{
			name: "invalid user timezone", err: djangosession.ErrInvalidTimezone,
			contentType: "application/json", body: `{"error":"Something went wrong please try again later"}`, allow: "GET, HEAD, OPTIONS",
		},
		{
			name: "anonymous throttle cache", err: errors.New("Redis unavailable"),
			contentType: "application/json", body: `{"error":"Something went wrong please try again later"}`, allow: "GET, HEAD, OPTIONS",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			wantStatus := testCase.status
			if wantStatus == 0 {
				wantStatus = http.StatusInternalServerError
			}
			handler := newHTTPHandlerWithPendingReplacements(
				ctx, system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{
					publicProjectMetadataReads: publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
					publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{
						err: testCase.err,
					},
				},
			)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/", nil))
			if response.Code != wantStatus || response.Header().Get("Content-Type") != testCase.contentType ||
				response.Header().Get("Allow") != testCase.allow || response.Body.String() != testCase.body {
				t.Fatalf("response=%d %q headers=%v", response.Code, response.Body.String(), response.Header())
			}
		})
	}
}

func TestPublicProjectMetadataReadsHTTPTimezoneFailurePrecedesEndpointMethods(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodOptions} {
		for _, path := range []string{"/api/public/anchor/example/labels/", "/api/public/anchor/example/states/"} {
			handler := newHTTPHandlerWithPendingReplacements(
				newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{
					publicProjectMetadataReads: publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
					publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{
						err: djangosession.ErrTimezoneNotFound,
					},
				},
			)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(method, path, nil))
			if response.Code != http.StatusBadRequest || response.Body.String() != `{"error":"The required key does not exist."}` {
				t.Fatalf("%s %s response=%d %q", method, path, response.Code, response.Body.String())
			}
		}
	}
}

func TestSafeDecodeAPIActivityBodyMatchesDjango(t *testing.T) {
	for _, testCase := range []struct {
		name string
		body []byte
		want any
	}{
		{name: "empty", want: nil},
		{name: "text", body: []byte("hello"), want: "hello"},
		{name: "PNG", body: []byte("\x89PNG\r\n"), want: "[Binary Content]"},
		{name: "JPEG", body: []byte("\xff\xd8\xff\xe0"), want: "[Binary Content]"},
		{name: "PDF", body: []byte("%PDF-1.7"), want: "[Binary Content]"},
		{name: "invalid UTF-8", body: []byte{0xff, 0xfe}, want: "[Could not decode content]"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := safeDecodeAPIActivityBody(testCase.body); got != testCase.want {
				t.Fatalf("safe decode=%#v want=%#v", got, testCase.want)
			}
		})
	}
}

func TestDjangoRequestHeadersRestoresNetHTTPTransportFields(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/public/anchor/example/labels/", strings.NewReader("payload"))
	request.TransferEncoding = []string{"chunked"}
	request.Close = true
	request.Header["X-Quote"] = []string{"a'b"}
	request.Header["X-Control"] = []string{"\x00"}
	headers := djangoRequestHeaders(request)
	for _, want := range []string{
		"'Host': 'example.com'",
		"'Content-Length': '7'",
		"'Transfer-Encoding': 'chunked'",
		"'Connection': 'close'",
		`'X-Quote': "a'b"`,
		`'X-Control': '\x00'`,
	} {
		if !strings.Contains(headers, want) {
			t.Fatalf("headers=%q missing %q", headers, want)
		}
	}
}

func TestPublicProjectMetadataReadsResponseWriterUsesCapturedRepositoryJSON(t *testing.T) {
	response := httptest.NewRecorder()
	raw := json.RawMessage(`[{"sequence":100.0}]`)
	writer := &publicProjectMetadataReadsResponseWriter{
		ResponseWriter: response,
		rawResponse:    &systemapi.PublicProjectMetadataReadsResponseCapture{Data: raw},
	}
	if _, err := writer.Write([]byte(`[{"sequence":100}]`)); err != nil {
		t.Fatal(err)
	}
	writer.finish()
	if response.Body.String() != string(raw) {
		t.Fatalf("body=%q want=%q", response.Body.String(), raw)
	}
}

func TestBufferedResponseWriterPreservesDjangoJSONEscaping(t *testing.T) {
	raw := json.RawMessage("{\"display_name\":\"R&D <ops>\",\"separator\":\"\u2028\"}")
	response := httptest.NewRecorder()
	writer := &publicProjectMetadataReadsResponseWriter{
		ResponseWriter:   response,
		capturedResponse: func() json.RawMessage { return raw },
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(`{"display_name":"R\u0026D \u003cops\u003e"}`))
	writer.finish()
	const want = `{"display_name":"R&D <ops>","separator":"\u2028"}`
	if response.Body.String() != want {
		t.Fatalf("body=%q want=%q", response.Body.String(), want)
	}
}

func TestPublicProjectMetadataReadsResponseWriterMatchesDjangoGZipThreshold(t *testing.T) {
	for _, testCase := range []struct {
		name string
		raw  json.RawMessage
		want bool
	}{
		{name: "small", raw: json.RawMessage(`[]`), want: false},
		{name: "large", raw: json.RawMessage(`{"value":"` + strings.Repeat("x", 200) + `"}`), want: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			response.Header().Set("Vary", "Cookie, origin")
			writer := &publicProjectMetadataReadsResponseWriter{
				ResponseWriter: response,
				rawResponse:    &systemapi.PublicProjectMetadataReadsResponseCapture{Data: testCase.raw},
			}
			if _, err := writer.Write([]byte(`[]`)); err != nil {
				t.Fatal(err)
			}
			writer.finish()
			got := strings.Contains(response.Header().Get("Vary"), "Accept-Encoding")
			if got != testCase.want {
				t.Fatalf("Vary=%q, want Accept-Encoding=%v", response.Header().Get("Vary"), testCase.want)
			}
			wantVary := "Cookie, origin"
			if testCase.want {
				wantVary = "Accept-Encoding, Cookie, origin"
			}
			if response.Header().Get("Vary") != wantVary {
				t.Fatalf("Vary=%q want %q", response.Header().Get("Vary"), wantVary)
			}
			if response.Header().Get("Content-Length") != strconv.Itoa(response.Body.Len()) {
				t.Fatalf("Content-Length=%q body=%d", response.Header().Get("Content-Length"), response.Body.Len())
			}
		})
	}
}

func TestPublicProjectMetadataReadsResponseWriterGzipsLikeDjango(t *testing.T) {
	raw := json.RawMessage(`{"value":"` + strings.Repeat("compressible", 40) + `"}`)
	response := httptest.NewRecorder()
	writer := &publicProjectMetadataReadsResponseWriter{
		ResponseWriter: response,
		rawResponse:    &systemapi.PublicProjectMetadataReadsResponseCapture{Data: raw},
		acceptsGzip:    true,
		gzipPadding:    func() (int, error) { return 17, nil },
	}
	if _, err := writer.Write([]byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	writer.finish()
	if response.Header().Get("Content-Encoding") != "gzip" ||
		!strings.Contains(response.Header().Get("Vary"), "Accept-Encoding") ||
		response.Header().Get("Content-Length") != strconv.Itoa(response.Body.Len()) {
		t.Fatalf("gzip headers=%v body=%d", response.Header(), response.Body.Len())
	}
	reader, err := gzip.NewReader(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if reader.Name != strings.Repeat("a", 17) {
		t.Fatalf("gzip padding name=%q", reader.Name)
	}
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decompressed, raw) {
		t.Fatalf("decompressed body=%q want=%q", decompressed, raw)
	}
	headResponse := httptest.NewRecorder()
	headWriter := &publicProjectMetadataReadsResponseWriter{
		ResponseWriter: workspaceSlugHeadWriter{headResponse},
		rawResponse:    &systemapi.PublicProjectMetadataReadsResponseCapture{Data: raw},
		acceptsGzip:    true,
		gzipPadding:    func() (int, error) { return 17, nil },
	}
	if _, err := headWriter.Write([]byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	headWriter.finish()
	if headResponse.Body.Len() != 0 ||
		headResponse.Header().Get("Content-Encoding") != "gzip" ||
		headResponse.Header().Get("Content-Length") != response.Header().Get("Content-Length") {
		t.Fatalf("HEAD gzip headers=%v body=%d", headResponse.Header(), headResponse.Body.Len())
	}
}

func TestPublicProjectMetadataReadsResponseWriterCompactsDjangoJSON(t *testing.T) {
	response := httptest.NewRecorder()
	raw := json.RawMessage("[{\"name\" : \"<&>\u2028\u2029\", \"sequence\" : 100.0}]\n")
	writer := &publicProjectMetadataReadsResponseWriter{
		ResponseWriter: response,
		rawResponse:    &systemapi.PublicProjectMetadataReadsResponseCapture{Data: raw},
	}
	if _, err := writer.Write([]byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	writer.finish()
	want := `[{"name":"<&>\u2028\u2029","sequence":100.0}]`
	if response.Body.String() != want || response.Header().Get("Content-Length") != strconv.Itoa(len(want)) {
		t.Fatalf("body=%q headers=%v", response.Body.String(), response.Header())
	}
}

func TestPublicProjectMetadataReadsResponseWriterMatchesDjangoIndentSeparators(t *testing.T) {
	response := httptest.NewRecorder()
	raw := json.RawMessage(`{"name":"a:b","sequence":100.0}`)
	writer := &publicProjectMetadataReadsResponseWriter{
		ResponseWriter: response,
		rawResponse:    &systemapi.PublicProjectMetadataReadsResponseCapture{Data: raw},
		jsonIndent:     4,
	}
	if _, err := writer.Write([]byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	writer.finish()
	want := "{\n    \"name\": \"a:b\",\n    \"sequence\": 100.0\n}"
	if response.Body.String() != want || response.Header().Get("Content-Length") != strconv.Itoa(len(want)) {
		t.Fatalf("body=%q headers=%v", response.Body.String(), response.Header())
	}
}

func TestAcceptsGzipEncodingMatchesDjangoPattern(t *testing.T) {
	for _, testCase := range []struct {
		header string
		want   bool
	}{
		{header: "gzip", want: true},
		{header: "br, gzip;q=0", want: true},
		{header: "x-gzip", want: true},
		{header: "GZIP", want: false},
		{header: "xgzip", want: false},
		{header: djangoLatin1String([]byte{'\xe9', 'g', 'z', 'i', 'p', '\xe9'}), want: false},
		{header: "©gzip©", want: true},
	} {
		if got := acceptsGzipEncoding(testCase.header); got != testCase.want {
			t.Errorf("acceptsGzipEncoding(%q)=%v want %v", testCase.header, got, testCase.want)
		}
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header["Accept-Encoding"] = []string{"br", "gzip"}
	if joined := djangoJoinedHeader(request, "Accept-Encoding"); joined != "br,gzip" || !acceptsGzipEncoding(joined) {
		t.Fatalf("repeated Accept-Encoding joined=%q", joined)
	}
}

func TestProjectMembershipReadsResponseWriterUsesCapturedRepositoryJSON(t *testing.T) {
	response := httptest.NewRecorder()
	raw := json.RawMessage(`{"preferences":{"integer":9007199254740993,"decimal":0.1234567890123456789}}`)
	writer := &projectMembershipReadsResponseWriter{
		ResponseWriter: response,
		requestPath:    "/api/workspaces/acme/members/",
		requestMethod:  http.MethodGet,
		rawResponse:    &systemapi.ProjectMembershipReadsResponseCapture{Data: raw},
	}
	encoded := []byte(`{"preferences":{"integer":9007199254740992}}` + "\n")
	encoder := loomhttp.ResponseEncoder(context.Background(), writer)
	if err := encoder.Encode(map[string]any{"preferences": map[string]any{"integer": 9007199254740992}}); err != nil {
		t.Fatal(err)
	}
	if response.Body.String() != string(raw) {
		t.Fatalf("response changed arbitrary JSON: got %s want %s", response.Body.String(), raw)
	}
	if len(encoded) == len(raw) {
		t.Fatal("test fixture does not exercise replacement write length")
	}
}

func TestProjectMembershipReadsResponseWriterIndentsCapturedRepositoryJSONWithoutLosingPrecision(t *testing.T) {
	response := httptest.NewRecorder()
	raw := json.RawMessage(`{"preferences":{"integer":9007199254740993,"decimal":0.1234567890123456789}}`)
	writer := &projectMembershipReadsResponseWriter{
		ResponseWriter: response,
		requestPath:    "/api/workspaces/acme/members/",
		requestMethod:  http.MethodGet,
		rawResponse:    &systemapi.ProjectMembershipReadsResponseCapture{Data: raw},
		jsonIndent:     2,
	}
	encoder := loomhttp.ResponseEncoder(context.Background(), writer)
	if err := encoder.Encode(map[string]any{"preferences": map[string]any{"integer": 9007199254740992}}); err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"preferences\": {\n    \"integer\": 9007199254740993,\n    \"decimal\": 0.1234567890123456789\n  }\n}"
	if response.Body.String() != want {
		t.Fatalf("indented response changed arbitrary JSON: got %q want %q", response.Body.String(), want)
	}
}

func TestProjectMembershipReadsJSONNegotiation(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		accept     string
		present    bool
		accepted   bool
		wantIndent int
		wantError  bool
	}{
		{name: "missing", accepted: true},
		{name: "explicit empty", present: true},
		{name: "wildcard", accept: "*/*; indent=2", present: true, accepted: true, wantIndent: 2},
		{name: "application wildcard", accept: "application/*; indent=3", present: true, accepted: true, wantIndent: 3},
		{name: "JSON", accept: "application/json; indent=2", present: true, accepted: true, wantIndent: 2},
		{name: "whitespace indent", accept: `application/json; indent=" 4 "`, present: true, accepted: true, wantIndent: 4},
		{name: "arbitrary precision indent", accept: "application/json; indent=999999999999999999999999999999", present: true, accepted: true, wantIndent: 8},
		{name: "underscored indent", accept: "application/json; indent=1_0", present: true, accepted: true, wantIndent: 8},
		{name: "malformed parameter", accept: "application/json; indent", present: true, accepted: true},
		{name: "extended parameter", accept: "application/json; indent*=UTF-8''4", present: true, accepted: true, wantIndent: 4},
		{name: "Python charset alias", accept: "application/json; indent*=cp1252''%34", present: true, accepted: true, wantIndent: 4},
		{name: "Python UTF-8 code page alias", accept: "application/json; indent*=cp65001''%34", present: true, accepted: true, wantIndent: 4},
		{name: "Python short UTF-8 alias and Unicode digit", accept: "application/json; indent*=u8''%D9%A4", present: true, accepted: true, wantIndent: 4},
		{name: "Python UTF-8 signature codec", accept: "application/json; indent*=utf-8-sig''%EF%BB%BF%34", present: true, accepted: true, wantIndent: 4},
		{name: "Python unicode escape codec", accept: "application/json; indent*=unicode_escape''%5Cu0034", present: true, accepted: true, wantIndent: 4},
		{name: "Python raw unicode escape codec", accept: "application/json; indent*=raw_unicode_escape''%5CU00000034", present: true, accepted: true, wantIndent: 4},
		{name: "post-Python-3.12 Unicode digit", accept: "application/json; indent=𑛑", present: true, accepted: true},
		{name: "unknown extended encoding", accept: "application/json; indent*=bogus''%34", present: true, wantError: true},
		{name: "large indent", accept: "application/json; indent=20", present: true, accepted: true, wantIndent: 8},
		{name: "negative indent", accept: "application/json; indent=-1", present: true, accepted: true},
		{name: "invalid indent", accept: "application/json; indent=invalid", present: true, accepted: true},
		{name: "specific parameter", accept: "application/json, application/json; indent=4", present: true, accepted: true, wantIndent: 4},
		{name: "plain JSON precedence", accept: "application/json, application/*; indent=4", present: true, accepted: true},
		{name: "unsupported", accept: "application/xml", present: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			accepted, indent, err := projectMembershipReadsJSONNegotiation(testCase.accept, testCase.present)
			if (err != nil) != testCase.wantError || accepted != testCase.accepted || indent != testCase.wantIndent {
				t.Errorf("Accept %q present=%v = (%v, %d, %v), want (%v, %d, error=%v)", testCase.accept, testCase.present, accepted, indent, err, testCase.accepted, testCase.wantIndent, testCase.wantError)
			}
		})
	}
}

func TestWorkspaceFavoritesHTTPMountsAndDjangoProtocol(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	systemEndpoints := system.NewEndpoints(systemapi.NewSystem())
	handler := newHTTPHandlerWithWorkspaceReplacements(
		ctx, systemEndpoints, false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil,
		workspacefavoritesapi.NewEndpoints(workspaceFavoritesHTTPStub{}),
	)
	id := "00000000-0000-4000-8000-000000000011"
	collection := "/api/workspaces/acme/user-favorites/"
	detail := collection + id + "/"
	group := detail + "group/"
	for _, testCase := range []struct {
		method string
		path   string
		body   string
		status int
		allow  string
	}{
		{http.MethodGet, collection, "", http.StatusOK, "GET, POST, PATCH, DELETE, HEAD, OPTIONS"},
		{http.MethodPost, collection, `{"entity_type":"folder"}`, http.StatusOK, "GET, POST, PATCH, DELETE, HEAD, OPTIONS"},
		{http.MethodPatch, collection, `{}`, http.StatusInternalServerError, "GET, POST, PATCH, DELETE, HEAD, OPTIONS"},
		{http.MethodDelete, collection, "", http.StatusInternalServerError, "GET, POST, PATCH, DELETE, HEAD, OPTIONS"},
		{http.MethodGet, detail, "", http.StatusInternalServerError, "GET, POST, PATCH, DELETE, HEAD, OPTIONS"},
		{http.MethodPost, detail, `{}`, http.StatusInternalServerError, "GET, POST, PATCH, DELETE, HEAD, OPTIONS"},
		{http.MethodPatch, detail, `{"name":"Renamed"}`, http.StatusOK, "GET, POST, PATCH, DELETE, HEAD, OPTIONS"},
		{http.MethodDelete, detail, "", http.StatusNoContent, "GET, POST, PATCH, DELETE, HEAD, OPTIONS"},
		{http.MethodGet, group, "", http.StatusOK, "GET, HEAD, OPTIONS"},
	} {
		request := httptest.NewRequest(testCase.method, testCase.path, strings.NewReader(testCase.body))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != testCase.status || response.Header().Get("Allow") != testCase.allow {
			t.Errorf("%s %s = %d %q Allow=%q", testCase.method, testCase.path, response.Code, response.Body.String(), response.Header().Get("Allow"))
		}
	}

	for path, status := range map[string]int{collection: http.StatusOK, detail: http.StatusInternalServerError, group: http.StatusOK} {
		request := httptest.NewRequest(http.MethodHead, path, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != status || response.Body.Len() != 0 {
			t.Errorf("HEAD %s = %d %q", path, response.Code, response.Body.String())
		}
	}
	for path, allow := range map[string]string{collection: "GET, POST, PATCH, DELETE, HEAD, OPTIONS", detail: "GET, POST, PATCH, DELETE, HEAD, OPTIONS", group: "GET, HEAD, OPTIONS"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodOptions, path, nil))
		if response.Code != http.StatusOK || response.Header().Get("Allow") != allow {
			t.Errorf("OPTIONS %s = %d %q Allow=%q", path, response.Code, response.Body.String(), response.Header().Get("Allow"))
		}
	}
}

func TestLastVisitedWorkspaceHTTPMountsAndDjangoProtocol(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	systemEndpoints := system.NewEndpoints(systemapi.NewSystem())
	build := func(stub lastVisitedWorkspaceHTTPStub) http.Handler {
		return newHTTPHandlerWithWorkspaceReplacementsAndLastVisited(
			ctx, systemEndpoints, false,
			nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
			lastvisitedworkspaceapi.NewEndpoints(stub),
		)
	}
	handler := build(lastVisitedWorkspaceHTTPStub{})
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		request := httptest.NewRequest(method, "/api/users/last-visited-workspace/", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s last visited workspace = %d %q", method, response.Code, response.Body.String())
		}
		if response.Header().Get("Allow") != "GET, HEAD, OPTIONS" {
			t.Fatalf("%s Allow = %q", method, response.Header().Get("Allow"))
		}
		if !strings.Contains(response.Header().Get("Content-Type"), "application/json") {
			t.Fatalf("%s Content-Type = %q", method, response.Header().Get("Content-Type"))
		}
		if method == http.MethodHead && response.Body.Len() != 0 {
			t.Fatalf("HEAD body = %q", response.Body.String())
		}
		if method == http.MethodGet && strings.Contains(response.Header().Get("Vary"), "Cookie") {
			t.Fatalf("successful GET Vary = %q", response.Header().Get("Vary"))
		}
	}
	response := httptest.NewRecorder()
	build(lastVisitedWorkspaceHTTPStub{unauthenticated: true}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/users/last-visited-workspace/", nil))
	if response.Code != http.StatusUnauthorized || response.Body.String() != "{\"detail\":\"Authentication credentials were not provided.\"}\n" || !strings.Contains(response.Header().Get("Vary"), "Cookie") {
		t.Fatalf("anonymous GET = %d %q Vary=%q", response.Code, response.Body.String(), response.Header().Get("Vary"))
	}
}

func TestLivenessHTTP(t *testing.T) {
	var logs bytes.Buffer
	ctx := newLogContext(context.Background(), &logs, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	endpoints.Use(log.Endpoint)
	handler := newHTTPHandler(ctx, endpoints, false)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	serverCtx, stop := context.WithCancel(ctx)
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- serveHTTP(serverCtx, listener, handler, time.Second)
	}()

	requestID := "01989c80-222e-7d61-8c36-b6d00b74f8dd"
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/health/live", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set(platform.RequestIDHeader, requestID)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET liveness endpoint: %v", err)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if err := res.Body.Close(); err != nil {
		t.Fatalf("close response: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if got := res.Header.Get(platform.RequestIDHeader); got != requestID {
		t.Fatalf("request ID = %q, want %q", got, requestID)
	}
	if got, want := string(body), "{\"status\":\"ok\"}\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}

	stop()
	select {
	case err := <-serverErr:
		if err != nil {
			t.Fatalf("serveHTTP() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down")
	}

	assertJSONRequestLog(t, logs.Bytes(), requestID)
}

func TestCSRFTokenHTTPCompatibility(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	handler := newHTTPHandler(ctx, endpoints, false)

	request := httptest.NewRequest(http.MethodGet, "/auth/get-csrf-token/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body = %q", response.Code, response.Body.String())
	}
	if err := loomhttp.ValidateResponseContract(response.Result(), csrftokensvr.CsrfTokenResponseContractCases()[0]); err != nil {
		t.Fatalf("GET response contract: %v", err)
	}
	if got := response.Header().Get("Content-Length"); got != "81" {
		t.Errorf("Content-Length = %q, want 81", got)
	}
	for name, want := range map[string]string{
		"Allow":                      "GET, HEAD, OPTIONS",
		"Content-Type":               "application/json",
		"Cross-Origin-Opener-Policy": "same-origin",
		"Referrer-Policy":            "same-origin",
		"Vary":                       "Cookie, origin",
		"X-Content-Type-Options":     "nosniff",
		"X-Frame-Options":            "DENY",
	} {
		if got := response.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.CSRFToken) != 64 {
		t.Fatalf("token length = %d, want 64", len(body.CSRFToken))
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %#v, want one CSRF cookie", cookies)
	}
	cookie := cookies[0]
	if cookie.Name != defaultCSRFCookieName || len(cookie.Value) != 32 || !cookie.HttpOnly || cookie.Path != "/" || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge != 31_449_600 {
		t.Fatalf("CSRF cookie = %#v", cookie)
	}
	tokenSecret, err := systemapi.NormalizeCSRFSecret(body.CSRFToken)
	if err != nil {
		t.Fatalf("normalize response token: %v", err)
	}
	if tokenSecret != cookie.Value {
		t.Fatalf("response token secret = %q, cookie = %q", tokenSecret, cookie.Value)
	}

	repeatRequest := httptest.NewRequest(http.MethodGet, "/auth/get-csrf-token/", nil)
	repeatRequest.AddCookie(cookie)
	repeatResponse := httptest.NewRecorder()
	handler.ServeHTTP(repeatResponse, repeatRequest)
	var repeatBody struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(repeatResponse.Body.Bytes(), &repeatBody); err != nil {
		t.Fatalf("decode repeated body: %v", err)
	}
	repeatedCookies := repeatResponse.Result().Cookies()
	if len(repeatedCookies) != 1 || repeatedCookies[0].Value != cookie.Value {
		t.Fatalf("repeated cookies = %#v, want unchanged secret", repeatedCookies)
	}
	if repeatBody.CSRFToken == body.CSRFToken {
		t.Fatal("repeated GET reused the response mask")
	}

	invalidRequest := httptest.NewRequest(http.MethodGet, "/auth/get-csrf-token/", nil)
	invalidRequest.AddCookie(&http.Cookie{Name: defaultCSRFCookieName, Value: "short"})
	invalidResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidResponse, invalidRequest)
	invalidCookies := invalidResponse.Result().Cookies()
	if len(invalidCookies) != 1 || invalidCookies[0].Value == "short" || len(invalidCookies[0].Value) != 32 {
		t.Fatalf("invalid-cookie response = %#v", invalidCookies)
	}

	firstSecret := strings.Repeat("a", 32)
	lastSecret := strings.Repeat("b", 32)
	duplicateRequest := httptest.NewRequest(http.MethodGet, "/auth/get-csrf-token/", nil)
	duplicateRequest.Header.Add("Cookie", defaultCSRFCookieName+"="+firstSecret+"; "+defaultCSRFCookieName+"="+lastSecret)
	duplicateResponse := httptest.NewRecorder()
	handler.ServeHTTP(duplicateResponse, duplicateRequest)
	duplicateCookies := duplicateResponse.Result().Cookies()
	if len(duplicateCookies) != 1 || duplicateCookies[0].Value != lastSecret {
		t.Fatalf("duplicate-cookie response = %#v, want Django's last value %q", duplicateCookies, lastSecret)
	}
}

func TestCSRFTokenHTTPProtocolMethods(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	handler := newHTTPHandler(ctx, endpoints, false)

	headRequest := httptest.NewRequest(http.MethodHead, "/auth/get-csrf-token/", nil)
	headResponse := httptest.NewRecorder()
	handler.ServeHTTP(headResponse, headRequest)
	if headResponse.Code != http.StatusOK || headResponse.Body.Len() != 0 || len(headResponse.Result().Cookies()) != 1 {
		t.Fatalf("HEAD = %d body=%q cookies=%#v", headResponse.Code, headResponse.Body.String(), headResponse.Result().Cookies())
	}
	if err := loomhttp.ValidateResponseContract(headResponse.Result(), csrftokensvr.CsrfTokenResponseContractCases()[0]); err != nil {
		t.Fatalf("HEAD response contract: %v", err)
	}
	if got := headResponse.Header().Get("Content-Length"); got != "81" {
		t.Errorf("HEAD Content-Length = %q, want 81", got)
	}
	indentGETRequest := httptest.NewRequest(http.MethodGet, "/auth/get-csrf-token/", nil)
	indentGETRequest.Header.Set("Accept", "application/json; indent=2")
	indentGETResponse := httptest.NewRecorder()
	handler.ServeHTTP(indentGETResponse, indentGETRequest)
	indentHEADRequest := httptest.NewRequest(http.MethodHead, "/auth/get-csrf-token/", nil)
	indentHEADRequest.Header.Set("Accept", "application/json; indent=2")
	indentHEADResponse := httptest.NewRecorder()
	handler.ServeHTTP(indentHEADResponse, indentHEADRequest)
	if indentHEADResponse.Code != http.StatusOK || indentHEADResponse.Body.Len() != 0 ||
		indentHEADResponse.Header().Get("Content-Length") != strconv.Itoa(indentGETResponse.Body.Len()) {
		t.Fatalf("indented HEAD = %d body=%q headers=%#v, GET length=%d", indentHEADResponse.Code, indentHEADResponse.Body.String(), indentHEADResponse.Header(), indentGETResponse.Body.Len())
	}

	optionsRequest := httptest.NewRequest(http.MethodOptions, "/auth/get-csrf-token/", nil)
	optionsResponse := httptest.NewRecorder()
	handler.ServeHTTP(optionsResponse, optionsRequest)
	if optionsResponse.Code != http.StatusOK || len(optionsResponse.Result().Cookies()) != 0 {
		t.Fatalf("OPTIONS = %d cookies=%#v", optionsResponse.Code, optionsResponse.Result().Cookies())
	}
	if err := loomhttp.ValidateResponseContract(optionsResponse.Result(), csrftokensvr.CsrfTokenOptionsResponseContractCases()[0]); err != nil {
		t.Fatalf("OPTIONS response contract: %v", err)
	}
	wantBody := "{\"name\":\"Csrf Token Endpoint\",\"description\":\"\",\"renders\":[\"application/json\"],\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"]}"
	if got := optionsResponse.Body.String(); got != wantBody {
		t.Fatalf("OPTIONS body = %q, want %q", got, wantBody)
	}
	if got := optionsResponse.Header().Get("Content-Length"); got != "166" {
		t.Errorf("OPTIONS Content-Length = %q, want 166", got)
	}

	corsOptionsRequest := httptest.NewRequest(http.MethodOptions, "/auth/get-csrf-token/", nil)
	corsOptionsRequest.Header.Set("Origin", "https://app.example.test")
	corsOptionsResponse := httptest.NewRecorder()
	handler.ServeHTTP(corsOptionsResponse, corsOptionsRequest)
	if corsOptionsResponse.Code != http.StatusOK || corsOptionsResponse.Body.String() != wantBody ||
		corsOptionsResponse.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" ||
		corsOptionsResponse.Header().Get("Access-Control-Allow-Credentials") != "true" ||
		corsOptionsResponse.Header().Get("Access-Control-Allow-Headers") != "accept, authorization, content-type, user-agent, x-csrftoken, x-requested-with, X-API-Key" ||
		corsOptionsResponse.Header().Get("Access-Control-Allow-Methods") != "DELETE, GET, OPTIONS, PATCH, POST, PUT" ||
		corsOptionsResponse.Header().Get("Access-Control-Max-Age") != "86400" ||
		corsOptionsResponse.Header().Get("Vary") != "Cookie, origin" {
		t.Fatalf("CORS metadata OPTIONS = %d body=%q headers=%#v", corsOptionsResponse.Code, corsOptionsResponse.Body.String(), corsOptionsResponse.Header())
	}

	preflightRequest := httptest.NewRequest(http.MethodOptions, "/auth/get-csrf-token/", nil)
	preflightRequest.Header.Set("Origin", "https://app.example.test")
	preflightRequest.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflightResponse := httptest.NewRecorder()
	handler.ServeHTTP(preflightResponse, preflightRequest)
	if preflightResponse.Code != http.StatusOK || preflightResponse.Body.Len() != 0 || preflightResponse.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" {
		t.Fatalf("preflight = %d body=%q headers=%#v", preflightResponse.Code, preflightResponse.Body.String(), preflightResponse.Header())
	}

	unsupportedPreflightRequest := httptest.NewRequest(http.MethodOptions, "/auth/get-csrf-token/", nil)
	unsupportedPreflightRequest.Header.Set("Origin", "https://app.example.test")
	unsupportedPreflightRequest.Header.Set("Access-Control-Request-Method", http.MethodTrace)
	unsupportedPreflightResponse := httptest.NewRecorder()
	handler.ServeHTTP(unsupportedPreflightResponse, unsupportedPreflightRequest)
	if unsupportedPreflightResponse.Code != http.StatusOK || unsupportedPreflightResponse.Body.Len() != 0 ||
		unsupportedPreflightResponse.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" ||
		unsupportedPreflightResponse.Header().Get("Access-Control-Allow-Credentials") != "true" ||
		unsupportedPreflightResponse.Header().Get("Vary") != "Origin" {
		t.Fatalf("unsupported-method preflight = %d body=%q headers=%#v", unsupportedPreflightResponse.Code, unsupportedPreflightResponse.Body.String(), unsupportedPreflightResponse.Header())
	}

	configuredHandler := newHTTPHandlerWithPendingReplacements(
		ctx, endpoints, false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{csrfTokenCORS: []string{"HTTPS://app.example.test/config-path"}},
	)
	equivalentOrigin := "https://app.example.test/ignored-origin-path"
	equivalentRequest := httptest.NewRequest(http.MethodGet, "/auth/get-csrf-token/", nil)
	equivalentRequest.Header.Set("Origin", equivalentOrigin)
	equivalentResponse := httptest.NewRecorder()
	configuredHandler.ServeHTTP(equivalentResponse, equivalentRequest)
	if equivalentResponse.Code != http.StatusOK || equivalentResponse.Header().Get("Access-Control-Allow-Origin") != equivalentOrigin ||
		equivalentResponse.Header().Get("Vary") != "Cookie, origin" {
		t.Fatalf("equivalent-origin response = %d headers=%#v", equivalentResponse.Code, equivalentResponse.Header())
	}
	equivalentHeadRequest := httptest.NewRequest(http.MethodHead, "/auth/get-csrf-token/", nil)
	equivalentHeadRequest.Header.Set("Origin", equivalentOrigin)
	equivalentHeadResponse := httptest.NewRecorder()
	configuredHandler.ServeHTTP(equivalentHeadResponse, equivalentHeadRequest)
	if equivalentHeadResponse.Code != http.StatusOK || equivalentHeadResponse.Header().Get("Access-Control-Allow-Origin") != equivalentOrigin ||
		equivalentHeadResponse.Header().Get("Vary") != "Cookie, origin" {
		t.Fatalf("equivalent-origin HEAD = %d headers=%#v", equivalentHeadResponse.Code, equivalentHeadResponse.Header())
	}
	equivalentPreflightRequest := httptest.NewRequest(http.MethodOptions, "/auth/get-csrf-token/", nil)
	equivalentPreflightRequest.Header.Set("Origin", equivalentOrigin)
	equivalentPreflightRequest.Header.Set("Access-Control-Request-Method", http.MethodTrace)
	equivalentPreflightResponse := httptest.NewRecorder()
	configuredHandler.ServeHTTP(equivalentPreflightResponse, equivalentPreflightRequest)
	if equivalentPreflightResponse.Code != http.StatusOK || equivalentPreflightResponse.Header().Get("Access-Control-Allow-Origin") != equivalentOrigin {
		t.Fatalf("equivalent-origin preflight = %d headers=%#v", equivalentPreflightResponse.Code, equivalentPreflightResponse.Header())
	}

	repeatedOriginRequest := httptest.NewRequest(http.MethodGet, "/auth/get-csrf-token/", nil)
	repeatedOriginRequest.Header["Origin"] = []string{"https://app.example.test/ignored-origin-path", "https://denied.example.test"}
	repeatedOriginResponse := httptest.NewRecorder()
	configuredHandler.ServeHTTP(repeatedOriginResponse, repeatedOriginRequest)
	if repeatedOriginResponse.Code != http.StatusOK || repeatedOriginResponse.Header().Get("Access-Control-Allow-Origin") != "" ||
		repeatedOriginResponse.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatalf("repeated-origin response = %d headers=%#v", repeatedOriginResponse.Code, repeatedOriginResponse.Header())
	}
	repeatedOriginPreflightRequest := httptest.NewRequest(http.MethodOptions, "/auth/get-csrf-token/", nil)
	repeatedOriginPreflightRequest.Header["Origin"] = []string{"https://app.example.test/ignored-origin-path", "https://denied.example.test"}
	repeatedOriginPreflightRequest.Header.Set("Access-Control-Request-Method", http.MethodGet)
	repeatedOriginPreflightResponse := httptest.NewRecorder()
	configuredHandler.ServeHTTP(repeatedOriginPreflightResponse, repeatedOriginPreflightRequest)
	if repeatedOriginPreflightResponse.Code != http.StatusOK || repeatedOriginPreflightResponse.Header().Get("Access-Control-Allow-Origin") != "" ||
		repeatedOriginPreflightResponse.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatalf("repeated-origin preflight = %d headers=%#v", repeatedOriginPreflightResponse.Code, repeatedOriginPreflightResponse.Header())
	}
}

func TestDjangoCSRFCookieConfiguration(t *testing.T) {
	config := djangoCSRFCookieConfig{Domain: ".example.test", Secure: true}
	cookie := &http.Cookie{
		Name:     defaultCSRFCookieName,
		Value:    strings.Repeat("a", 32),
		Path:     "/",
		MaxAge:   60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	before := time.Now().Add(time.Minute)
	if err := djangoCSRFResponseCookiePolicy(config)(context.Background(), cookie); err != nil {
		t.Fatalf("Django CSRF response-cookie policy error = %v", err)
	}
	if cookie.Domain != config.Domain || !cookie.Secure || cookie.Expires.Before(before) || cookie.Expires.After(time.Now().Add(time.Minute+time.Second)) {
		t.Fatalf("runtime CSRF cookie attributes = %#v", cookie)
	}
	if cookie.Path != "/" || cookie.MaxAge != 60 || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("policy changed Loom-owned CSRF cookie attributes: %#v", cookie)
	}
}

func TestDjangoRuntimeCORSPolicyCompatibility(t *testing.T) {
	policy, err := djangoRuntimeCORSPolicy([]string{"HTTPS://app.example.test/config-path"})
	if err != nil {
		t.Fatalf("runtime CORS policy: %v", err)
	}
	handler := djangoCORSRequestCompatibilityMiddleware(policy.Handler(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/timezones/", nil)
	request.Header.Set("Origin", "https://app.example.test/ignored-origin-path")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent ||
		response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test/ignored-origin-path" ||
		response.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("runtime CORS response = %d headers=%v", response.Code, response.Header())
	}

	repeatedOrigin := httptest.NewRequest(http.MethodGet, "/api/timezones/", nil)
	repeatedOrigin.Header["Origin"] = []string{"https://app.example.test", "https://denied.example.test"}
	repeatedResponse := httptest.NewRecorder()
	handler.ServeHTTP(repeatedResponse, repeatedOrigin)
	if repeatedResponse.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("repeated Origin CORS headers=%v", repeatedResponse.Header())
	}

	allowAll, err := djangoRuntimeCORSPolicy(nil)
	if err != nil {
		t.Fatalf("allow-all runtime CORS policy: %v", err)
	}
	allowAllHandler := djangoCORSRequestCompatibilityMiddleware(allowAll.Handler(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	}))
	malformedOrigin := httptest.NewRequest(http.MethodGet, "/api/timezones/", nil)
	malformedOrigin.Header.Set("Origin", "https://[")
	malformedResponse := httptest.NewRecorder()
	allowAllHandler.ServeHTTP(malformedResponse, malformedOrigin)
	if malformedResponse.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("malformed Origin CORS headers=%v", malformedResponse.Header())
	}

	if _, err := djangoRuntimeCORSPolicy([]string{""}); err == nil {
		t.Fatal("empty configured CORS origin was accepted")
	}
}

func TestCSRFTokenCORSPreflightCompatibility(t *testing.T) {
	testCases := []struct {
		name           string
		allowedOrigins []string
		origin         string
		wantOrigin     string
	}{
		{name: "open origin", origin: "https://app.example.test", wantOrigin: "https://app.example.test"},
		{name: "malformed origin", origin: "https://[invalid"},
		{name: "missing origin"},
		{name: "disallowed configured origin", allowedOrigins: []string{"https://allowed.example.test"}, origin: "https://other.example.test"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			policy, err := djangoCSRFTokenRuntimeCORSPolicy(testCase.allowedOrigins)
			if err != nil {
				t.Fatalf("runtime CORS policy: %v", err)
			}
			generatedOptions := policy.OptionsHandler(func(http.ResponseWriter, *http.Request) {
				t.Fatal("preflight reached the ordinary OPTIONS handler")
			}, []string{http.MethodGet, http.MethodOptions})
			handler := djangoCORSRequestCompatibilityMiddleware(csrfTokenCORSPreflightMiddleware(generatedOptions))
			request := httptest.NewRequest(http.MethodOptions, "/auth/get-csrf-token/", nil)
			if testCase.origin != "" {
				request.Header.Set("Origin", testCase.origin)
			}
			request.Header.Set("Access-Control-Request-Method", http.MethodGet)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != http.StatusOK || response.Body.Len() != 0 || response.Header().Get("Content-Length") != "0" ||
				response.Header().Get("Content-Type") != "text/html; charset=utf-8" {
				t.Fatalf("preflight = %d body=%q headers=%v", response.Code, response.Body.String(), response.Header())
			}
			if got := response.Header().Get("Vary"); got != "Origin" {
				t.Errorf("Vary = %q, want Origin", got)
			}
			if got := response.Header().Get("Access-Control-Allow-Origin"); got != testCase.wantOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, testCase.wantOrigin)
			}
			if testCase.wantOrigin == "" {
				for _, name := range []string{
					"Access-Control-Allow-Credentials",
					"Access-Control-Allow-Headers",
					"Access-Control-Allow-Methods",
					"Access-Control-Max-Age",
				} {
					if got := response.Header().Get(name); got != "" {
						t.Errorf("%s = %q, want absent", name, got)
					}
				}
			}
		})
	}
}

func TestCSRFTokenHTTPSessionLifecycle(t *testing.T) {
	sessionConfig := djangoSessionCookieConfig{Name: "session-id", Domain: ".example.test", Secure: true, Configured: true}
	newRoute := func(method string, throttle publicProjectMetadataThrottle) http.Handler {
		endpoints := csrftoken.NewEndpoints(systemapi.NewCSRFToken())
		mux := loomhttp.NewMuxer()
		corsPolicy, err := djangoCSRFTokenRuntimeCORSPolicy(nil)
		if err != nil {
			t.Fatalf("runtime CORS policy: %v", err)
		}
		server := csrftokensvr.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, corsPolicy)
		var route http.Handler = server.CsrfToken
		switch method {
		case http.MethodHead:
			route = loomhttp.DerivedHeadHandler(route)
		case http.MethodOptions:
			route = server.CsrfTokenOptions
		}
		if method == http.MethodGet || method == http.MethodHead {
			route = djangoCSRFResponseCookiePolicy(djangoCSRFCookieConfig{}).Handler(route)
		}
		route = publicProjectMetadataReadsCompatibilityMiddleware(
			throttle, nil, sessionConfig, nil, "", defaultFileSizeLimit, false,
			false,
		)(route)
		route = csrfTokenResponseCompatibilityMiddleware(route)
		if method == http.MethodOptions {
			route = corsPolicy.OptionsHandler(loomhttp.AsHandlerFunc(route), []string{http.MethodGet, http.MethodOptions})
		} else {
			route = corsPolicy.Handler(loomhttp.AsHandlerFunc(route))
		}
		route = csrfTokenCORSPreflightMiddleware(route)
		route = djangoCORSRequestCompatibilityMiddleware(route)
		return route
	}

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		t.Run("expire invalid session on "+method, func(t *testing.T) {
			sessions := &publicProjectMetadataThrottleStub{allowed: true, expire: true}
			request := httptest.NewRequest(method, "/auth/get-csrf-token/", nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "expired-session"})
			response := httptest.NewRecorder()
			newRoute(method, sessions).ServeHTTP(response, request)
			if response.Code != http.StatusOK || sessions.calls != 1 {
				t.Fatalf("%s status=%d session calls=%d body=%q", method, response.Code, sessions.calls, response.Body.String())
			}
			foundExpiredSession := false
			for _, cookie := range response.Header().Values("Set-Cookie") {
				if strings.HasPrefix(cookie, `session-id="";`) && strings.Contains(cookie, "Max-Age=0") {
					foundExpiredSession = true
				}
			}
			if !foundExpiredSession {
				t.Fatalf("%s Set-Cookie=%q, want expired session cookie", method, response.Header().Values("Set-Cookie"))
			}
		})
	}

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		t.Run("ignore invalid timezone on "+method, func(t *testing.T) {
			sessions := &publicProjectMetadataThrottleStub{
				allowed:     true,
				userID:      "00000000-0000-4000-8000-000000000b01",
				timezoneErr: djangosession.ErrInvalidTimezone,
			}
			request := httptest.NewRequest(method, "/auth/get-csrf-token/", nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "authenticated-session"})
			response := httptest.NewRecorder()
			newRoute(method, sessions).ServeHTTP(response, request)
			if response.Code != http.StatusOK || sessions.calls != 1 {
				t.Fatalf("%s status=%d session calls=%d body=%q", method, response.Code, sessions.calls, response.Body.String())
			}
		})
	}

	now := time.Date(2026, time.September, 2, 16, 0, 0, 0, time.UTC)
	prepared := &djangosession.SessionRefresh{SessionKey: "existing-session"}
	sessions := &publicProjectMetadataThrottleStub{
		allowed: true,
		rotated: prepared,
		result: djangosession.SessionRefresh{
			SessionKey:      "refreshed-session",
			ExpiresAt:       now.Add(time.Hour),
			CookieExpiresAt: now.Add(time.Hour),
			MaxAgeSeconds:   3600,
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/auth/get-csrf-token/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: prepared.SessionKey})
	response := httptest.NewRecorder()
	newRoute(http.MethodGet, sessions).ServeHTTP(response, request)
	if response.Code != http.StatusOK || sessions.calls != 1 || sessions.refreshCalls != 1 {
		t.Fatalf("refreshed GET status=%d inspect calls=%d refresh calls=%d body=%q", response.Code, sessions.calls, sessions.refreshCalls, response.Body.String())
	}
	wantCookie := "session-id=refreshed-session; Domain=.example.test; expires=Wed, 02 Sep 2026 17:00:00 GMT; HttpOnly; Max-Age=3600; Path=/; SameSite=Lax; Secure"
	if !slices.Contains(response.Header().Values("Set-Cookie"), wantCookie) {
		t.Fatalf("refreshed GET Set-Cookie=%q, want %q", response.Header().Values("Set-Cookie"), wantCookie)
	}

	preflightSessions := &publicProjectMetadataThrottleStub{expire: true}
	preflightRequest := httptest.NewRequest(http.MethodOptions, "/auth/get-csrf-token/", nil)
	preflightRequest.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflightResponse := httptest.NewRecorder()
	newRoute(http.MethodOptions, preflightSessions).ServeHTTP(preflightResponse, preflightRequest)
	if preflightResponse.Code != http.StatusOK || preflightSessions.calls != 0 || len(preflightResponse.Header().Values("Set-Cookie")) != 0 {
		t.Fatalf("preflight status=%d session calls=%d cookies=%q", preflightResponse.Code, preflightSessions.calls, preflightResponse.Header().Values("Set-Cookie"))
	}
	for _, header := range []string{"Allow", "Cross-Origin-Opener-Policy", "Referrer-Policy", "X-Content-Type-Options", "X-Frame-Options"} {
		if got := preflightResponse.Header().Get(header); got != "" {
			t.Errorf("preflight %s=%q, want absent", header, got)
		}
	}
	if strings.Contains(preflightResponse.Header().Get("Vary"), "Cookie") {
		t.Errorf("preflight Vary=%q, must not include Cookie", preflightResponse.Header().Get("Vary"))
	}

	assertNoCSRFCookie := func(t *testing.T, response *httptest.ResponseRecorder) {
		t.Helper()
		for _, cookie := range response.Result().Cookies() {
			if cookie.Name == defaultCSRFCookieName {
				t.Fatalf("unexpected CSRF cookie on status %d: %#v", response.Code, cookie)
			}
		}
	}

	throttled := &publicProjectMetadataThrottleStub{wait: 1500 * time.Millisecond}
	throttledResponse := httptest.NewRecorder()
	newRoute(http.MethodGet, throttled).ServeHTTP(throttledResponse, httptest.NewRequest(http.MethodGet, "/auth/get-csrf-token/", nil))
	if throttledResponse.Code != http.StatusTooManyRequests || throttledResponse.Header().Get("Retry-After") != "2" ||
		throttledResponse.Body.String() != `{"error_code":5900,"error_message":"RATE_LIMIT_EXCEEDED"}` {
		t.Fatalf("throttled GET status=%d body=%q headers=%v", throttledResponse.Code, throttledResponse.Body.String(), throttledResponse.Header())
	}
	assertNoCSRFCookie(t, throttledResponse)

	for name, request := range map[string]*http.Request{
		"unsupported format": httptest.NewRequest(http.MethodGet, "/auth/get-csrf-token/?format=xml", nil),
		"unacceptable media": func() *http.Request {
			request := httptest.NewRequest(http.MethodGet, "/auth/get-csrf-token/", nil)
			request.Header.Set("Accept", "text/plain")
			return request
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			newRoute(http.MethodGet, &publicProjectMetadataThrottleStub{allowed: true}).ServeHTTP(response, request)
			wantStatus := http.StatusNotFound
			if name == "unacceptable media" {
				wantStatus = http.StatusNotAcceptable
			}
			if response.Code != wantStatus {
				t.Fatalf("status=%d body=%q headers=%v", response.Code, response.Body.String(), response.Header())
			}
			assertNoCSRFCookie(t, response)
		})
	}

	indentRequest := httptest.NewRequest(http.MethodGet, "/auth/get-csrf-token/", nil)
	indentRequest.Header.Set("Accept", "application/json; indent=2")
	indentResponse := httptest.NewRecorder()
	newRoute(http.MethodGet, &publicProjectMetadataThrottleStub{allowed: true}).ServeHTTP(indentResponse, indentRequest)
	if indentResponse.Code != http.StatusOK || !strings.Contains(indentResponse.Body.String(), "\n  \"csrf_token\": ") {
		t.Fatalf("indented GET status=%d body=%q", indentResponse.Code, indentResponse.Body.String())
	}

	oversizedRequest := httptest.NewRequest(http.MethodGet, "/auth/get-csrf-token/", strings.NewReader("x"))
	oversizedRequest.ContentLength = defaultFileSizeLimit + 1
	oversizedResponse := httptest.NewRecorder()
	newRoute(http.MethodGet, &publicProjectMetadataThrottleStub{allowed: true}).ServeHTTP(oversizedResponse, oversizedRequest)
	if oversizedResponse.Code != http.StatusRequestEntityTooLarge || oversizedResponse.Body.String() != string(djangoRequestTooLargeBody) {
		t.Fatalf("oversized GET status=%d body=%q headers=%v", oversizedResponse.Code, oversizedResponse.Body.String(), oversizedResponse.Header())
	}
	assertNoCSRFCookie(t, oversizedResponse)

	assembledHandler := newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			csrfTokenThrottle:  &publicProjectMetadataThrottleStub{allowed: true},
			csrfTokenBodyLimit: defaultFileSizeLimit,
			csrfTokenBodySet:   true,
		},
	)
	assembledOversizedRequest := httptest.NewRequest(http.MethodGet, "/auth/get-csrf-token/", strings.NewReader("x"))
	assembledOversizedRequest.ContentLength = defaultFileSizeLimit + 1
	assembledOversizedResponse := httptest.NewRecorder()
	assembledHandler.ServeHTTP(assembledOversizedResponse, assembledOversizedRequest)
	if assembledOversizedResponse.Code != http.StatusRequestEntityTooLarge || assembledOversizedResponse.Header().Get("Allow") != "" ||
		assembledOversizedResponse.Header().Get("Vary") != "origin" || strings.Contains(assembledOversizedResponse.Header().Get("Vary"), "Cookie") {
		t.Fatalf("assembled oversized GET status=%d body=%q headers=%v", assembledOversizedResponse.Code, assembledOversizedResponse.Body.String(), assembledOversizedResponse.Header())
	}

	for name, refreshErr := range map[string]error{
		"missing session": fmt.Errorf("refresh: %w", djangosession.ErrSessionNotFound),
		"update race":     fmt.Errorf("refresh: %w", djangosession.ErrSessionUpdate),
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			newRoute(http.MethodGet, &publicProjectMetadataThrottleStub{
				allowed:    true,
				rotated:    &djangosession.SessionRefresh{SessionKey: "existing-session"},
				refreshErr: refreshErr,
			}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/auth/get-csrf-token/", nil))
			if response.Code != http.StatusBadRequest || response.Body.String() != string(djangoBadRequestBody) ||
				response.Header().Get("Content-Length") != "" || response.Header().Get("Allow") != "" || response.Header().Get("X-Frame-Options") != "" {
				t.Fatalf("session interruption status=%d body=%q headers=%v", response.Code, response.Body.String(), response.Header())
			}
			assertNoCSRFCookie(t, response)
		})
	}

	storeErrorResponse := httptest.NewRecorder()
	newRoute(http.MethodGet, &publicProjectMetadataThrottleStub{
		allowed: true,
		err:     fmt.Errorf("inspect: %w", djangosession.ErrStore),
	}).ServeHTTP(storeErrorResponse, httptest.NewRequest(http.MethodGet, "/auth/get-csrf-token/", nil))
	if storeErrorResponse.Code != http.StatusInternalServerError || storeErrorResponse.Body.String() != string(djangoServerErrorBody) ||
		storeErrorResponse.Header().Get("Cross-Origin-Opener-Policy") != "same-origin" {
		t.Fatalf("session store error status=%d body=%q headers=%v", storeErrorResponse.Code, storeErrorResponse.Body.String(), storeErrorResponse.Header())
	}
	assertNoCSRFCookie(t, storeErrorResponse)
}

func TestTimezoneHTTPCompatibilitySurface(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	endpoints := system.NewEndpoints(systemapi.NewSystem())
	handler := newHTTPHandler(ctx, endpoints, false)
	request := httptest.NewRequest(http.MethodGet, "/api/timezones/", nil)
	response := httptest.NewRecorder()
	before := time.Now().UTC().Add(timezoneCacheDuration)

	handler.ServeHTTP(response, request)

	after := time.Now().UTC().Add(timezoneCacheDuration)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for name, want := range map[string]string{
		"Allow":                      "GET, HEAD, OPTIONS",
		"Cache-Control":              "max-age=7200",
		"Content-Type":               "application/json",
		"Cross-Origin-Opener-Policy": "same-origin",
		"Referrer-Policy":            "same-origin",
		"Vary":                       "Accept-Encoding, Cookie, origin",
		"X-Content-Type-Options":     "nosniff",
		"X-Frame-Options":            "DENY",
	} {
		if got := response.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	expires, err := time.Parse(http.TimeFormat, response.Header().Get("Expires"))
	if err != nil {
		t.Fatalf("parse Expires header: %v", err)
	}
	if expires.Before(before.Truncate(time.Second)) || expires.After(after.Truncate(time.Second)) {
		t.Fatalf("Expires = %s, want between %s and %s", expires, before, after)
	}
	var body struct {
		Timezones []json.RawMessage `json:"timezones"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Timezones) != 120 {
		t.Fatalf("timezone count = %d, want 120", len(body.Timezones))
	}
}

func TestTimezoneProtocolCompanionsAreMounted(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	handler := newHTTPHandler(ctx, system.NewEndpoints(systemapi.NewSystem()), false)

	head := httptest.NewRecorder()
	handler.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/api/timezones/", nil))
	if head.Code != http.StatusOK || head.Body.Len() != 0 || head.Header().Get("Allow") != "GET, HEAD, OPTIONS" {
		t.Fatalf("HEAD response = %d %q %v", head.Code, head.Body.String(), head.Header())
	}
	unsupportedHeadRequest := httptest.NewRequest(http.MethodHead, "/api/timezones/", nil)
	unsupportedHeadRequest.Header.Set("Accept", "application/xml")
	unsupportedHead := httptest.NewRecorder()
	handler.ServeHTTP(unsupportedHead, unsupportedHeadRequest)
	if unsupportedHead.Code != http.StatusNotAcceptable || unsupportedHead.Body.Len() != 0 ||
		unsupportedHead.Header().Get("Content-Type") != "application/json" || unsupportedHead.Header().Get("Content-Length") != "58" ||
		unsupportedHead.Header().Get("Cache-Control") != "" || unsupportedHead.Header().Get("Expires") != "" ||
		!strings.Contains(unsupportedHead.Header().Get("Vary"), "Accept") {
		t.Fatalf("unsupported HEAD Accept response = %d %q %v", unsupportedHead.Code, unsupportedHead.Body.String(), unsupportedHead.Header())
	}

	options := httptest.NewRecorder()
	handler.ServeHTTP(options, httptest.NewRequest(http.MethodOptions, "/api/timezones/", nil))
	if options.Code != http.StatusOK || options.Header().Get("Allow") != "GET, HEAD, OPTIONS" ||
		options.Header().Get("Cache-Control") != "" || options.Header().Get("Expires") != "" {
		t.Fatalf("OPTIONS response = %d %q %v", options.Code, options.Body.String(), options.Header())
	}
	var body map[string]any
	if err := json.Unmarshal(options.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode OPTIONS response: %v", err)
	}
	if body["name"] != "Timezone Endpoint" {
		t.Fatalf("OPTIONS name = %v, want Timezone Endpoint", body["name"])
	}
	if _, exists := body["actions"]; exists {
		t.Fatalf("OPTIONS actions = %v, want omitted", body["actions"])
	}
	unsupportedAccept := httptest.NewRequest(http.MethodOptions, "/api/timezones/", nil)
	unsupportedAccept.Header.Set("Accept", "application/xml")
	notAcceptable := httptest.NewRecorder()
	handler.ServeHTTP(notAcceptable, unsupportedAccept)
	if notAcceptable.Code != http.StatusNotAcceptable ||
		notAcceptable.Body.String() != "{\"detail\":\"Could not satisfy the request Accept header.\"}\n" ||
		notAcceptable.Header().Get("Cache-Control") != "" || notAcceptable.Header().Get("Expires") != "" ||
		!strings.Contains(notAcceptable.Header().Get("Vary"), "Accept") {
		t.Fatalf("unsupported Accept response = %d %q", notAcceptable.Code, notAcceptable.Body.String())
	}
	preflightRequest := httptest.NewRequest(http.MethodOptions, "/api/timezones/", nil)
	preflightRequest.Header.Set("Origin", "https://app.example.test")
	preflightRequest.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflight := httptest.NewRecorder()
	handler.ServeHTTP(preflight, preflightRequest)
	if preflight.Code != http.StatusOK || preflight.Body.Len() != 0 ||
		preflight.Header().Get("Content-Length") != "0" ||
		preflight.Header().Get("Content-Type") != "text/html; charset=utf-8" ||
		preflight.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" ||
		preflight.Header().Get("Vary") != "origin" {
		t.Fatalf("preflight response = %d %q %v", preflight.Code, preflight.Body.String(), preflight.Header())
	}
}

type stickyHTTPStub struct{ success bool }

type stickyMediaCaptureStub struct {
	stickyHTTPStub
	created *sticky.CreatePayload
	updated *sticky.UpdatePayload
}

func (stub *stickyMediaCaptureStub) Create(_ context.Context, payload *sticky.CreatePayload) (*sticky.StickyResponse, error) {
	stub.created = payload
	return stickyHTTPSuccessResponse(), nil
}

func (stub *stickyMediaCaptureStub) Update(_ context.Context, payload *sticky.UpdatePayload) (*sticky.StickyResponse, error) {
	stub.updated = payload
	return stickyHTTPSuccessResponse(), nil
}

func (stickyHTTPStub) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	if key == "valid-session" {
		return ctx, nil
	}
	return ctx, &sticky.StickyUnauthenticatedError{Detail: "Authentication credentials were not provided."}
}
func (stub stickyHTTPStub) List(context.Context, *sticky.ListPayload) (*sticky.StickyPage, error) {
	results := []*sticky.StickyResponse{}
	if stub.success {
		results = append(results, stickyHTTPSuccessResponse())
	}
	return &sticky.StickyPage{NextCursor: "20:1:0", PrevCursor: "20:-1:1", Results: results}, nil
}
func (stub stickyHTTPStub) Create(context.Context, *sticky.CreatePayload) (*sticky.StickyResponse, error) {
	if stub.success {
		return stickyHTTPSuccessResponse(), nil
	}
	return nil, errors.New("unexpected create")
}
func (stub stickyHTTPStub) Retrieve(context.Context, *sticky.RetrievePayload) (*sticky.StickyResponse, error) {
	if stub.success {
		return stickyHTTPSuccessResponse(), nil
	}
	return nil, &sticky.StickyNotFoundError{Detail: "No Sticky matches the given query."}
}
func (stub stickyHTTPStub) Update(context.Context, *sticky.UpdatePayload) (*sticky.StickyResponse, error) {
	if stub.success {
		return stickyHTTPSuccessResponse(), nil
	}
	return nil, &sticky.StickyPermissionError{Message: "You don't have the required permissions."}
}
func (stub stickyHTTPStub) Delete(context.Context, *sticky.DeletePayload) error {
	if stub.success {
		return nil
	}
	return &sticky.StickyNotFoundError{Detail: "No Sticky matches the given query."}
}

func TestStickyHTTPCompatibilitySurface(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	systemEndpoints := system.NewEndpoints(systemapi.NewSystem())
	stickyEndpoints := sticky.NewEndpoints(stickyHTTPStub{})
	stub := stickyHTTPStub{}
	handler := newHTTPHandlerWithStickies(ctx, systemEndpoints, false, stickyEndpoints, stub.APIKeyAuth)

	request := httptest.NewRequest(http.MethodGet, "/api/workspaces/test/stickies/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || response.Body.String() != "{\"detail\":\"Authentication credentials were not provided.\"}\n" {
		t.Fatalf("unauthenticated response = %d %q", response.Code, response.Body.String())
	}
	if response.Header().Get("loom-error") != "" || response.Header().Get("X-Frame-Options") != "DENY" || response.Header().Get("Vary") != "Cookie, origin" {
		t.Fatalf("compatibility headers = %v", response.Header())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/workspaces/test/stickies/?cursor=bad", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || response.Body.String() != "{\"detail\":\"Invalid cursor parameter.\"}\n" {
		t.Fatalf("invalid cursor response = %d %q", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/workspaces/test/stickies/?per_page=1001", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || response.Body.String() != "{\"detail\":\"Invalid per_page value. Cannot exceed 1000.\"}\n" {
		t.Fatalf("invalid per_page response = %d %q", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/workspaces/test/stickies/?per_page=not-a-number", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || response.Body.String() != "{\"detail\":\"Invalid per_page parameter.\"}\n" {
		t.Fatalf("non-integer per_page response = %d %q", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPatch, "/api/workspaces/test/stickies/0198a28d-d227-7752-82ac-bc4c1c290703/", strings.NewReader(`{"sort_order":"bad"}`))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || response.Body.String() != "{\"sort_order\":[\"A valid number is required.\"]}\n" {
		t.Fatalf("invalid sort response = %d %q", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPatch, "/api/workspaces/test/stickies/0198a28d-d227-7752-82ac-bc4c1c290703/", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || response.Body.String() != "{\"error\":\"You don't have the required permissions.\"}\n" {
		t.Fatalf("permission response = %d %q", response.Code, response.Body.String())
	}

	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		request = httptest.NewRequest(method, "/api/workspaces/test/stickies/0198a28d-d227-7752-82ac-bc4c1c290703/", nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound || response.Body.String() != "{\"detail\":\"No Sticky matches the given query.\"}\n" {
			t.Fatalf("%s not-found response = %d %q", method, response.Code, response.Body.String())
		}
	}

	request = httptest.NewRequest(http.MethodPost, "/api/workspaces/test/stickies/", strings.NewReader(strings.Repeat("x", stickyRequestLimit+1)))
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge || response.Body.Len() != 0 || response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("oversized response = %d %q %v", response.Code, response.Body.String(), response.Header())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/test/stickies/", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("API-v1 status = %d, want Django-only route absent", response.Code)
	}
}

func TestStickyProtocolCompanionsAreMounted(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	systemEndpoints := system.NewEndpoints(systemapi.NewSystem())
	stub := stickyHTTPStub{success: true}
	handler := newHTTPHandlerWithStickies(ctx, systemEndpoints, false, sticky.NewEndpoints(stub), stub.APIKeyAuth)
	id := "0198a28d-d227-7752-82ac-bc4c1c290703"

	for _, testCase := range []struct {
		path  string
		allow string
	}{
		{"/api/workspaces/test/stickies/", "GET, POST, HEAD, OPTIONS"},
		{"/api/workspaces/test/stickies/" + id + "/", "GET, PATCH, DELETE, HEAD, OPTIONS"},
	} {
		headRequest := httptest.NewRequest(http.MethodHead, testCase.path, nil)
		headRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		head := httptest.NewRecorder()
		handler.ServeHTTP(head, headRequest)
		if head.Code != http.StatusOK || head.Body.Len() != 0 || head.Header().Get("Allow") != testCase.allow {
			t.Errorf("HEAD %s response = %d %q %v", testCase.path, head.Code, head.Body.String(), head.Header())
		}
	}

	invalidHeadRequest := httptest.NewRequest(http.MethodHead, "/api/workspaces/test/stickies/?per_page=invalid", nil)
	invalidHeadRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	invalidHead := httptest.NewRecorder()
	handler.ServeHTTP(invalidHead, invalidHeadRequest)
	if invalidHead.Code != http.StatusBadRequest || invalidHead.Body.Len() != 0 || invalidHead.Header().Get("Content-Type") != "application/json" || invalidHead.Header().Get("Content-Length") != "41" {
		t.Fatalf("invalid pagination HEAD response = %d %q %v", invalidHead.Code, invalidHead.Body.String(), invalidHead.Header())
	}
	unsupportedHeadRequest := httptest.NewRequest(http.MethodHead, "/api/workspaces/test/stickies/", nil)
	unsupportedHeadRequest.Header.Set("Accept", "application/xml")
	unsupportedHead := httptest.NewRecorder()
	handler.ServeHTTP(unsupportedHead, unsupportedHeadRequest)
	if unsupportedHead.Code != http.StatusNotAcceptable || unsupportedHead.Body.Len() != 0 ||
		unsupportedHead.Header().Get("Content-Type") != "application/json" || unsupportedHead.Header().Get("Content-Length") != "58" {
		t.Fatalf("unsupported HEAD Accept response = %d %q %v", unsupportedHead.Code, unsupportedHead.Body.String(), unsupportedHead.Header())
	}
	invalidUnsupportedHeadRequest := httptest.NewRequest(http.MethodHead, "/api/workspaces/test/stickies/?per_page=invalid", nil)
	invalidUnsupportedHeadRequest.Header.Set("Accept", "application/xml")
	invalidUnsupportedHead := httptest.NewRecorder()
	handler.ServeHTTP(invalidUnsupportedHead, invalidUnsupportedHeadRequest)
	if invalidUnsupportedHead.Code != http.StatusNotAcceptable || invalidUnsupportedHead.Body.Len() != 0 ||
		invalidUnsupportedHead.Header().Get("Content-Length") != "58" {
		t.Fatalf("unsupported HEAD Accept with invalid pagination response = %d %q %v", invalidUnsupportedHead.Code, invalidUnsupportedHead.Body.String(), invalidUnsupportedHead.Header())
	}
	listRequest := httptest.NewRequest(http.MethodOptions, "/api/workspaces/test/stickies/", nil)
	listRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	listOptions := httptest.NewRecorder()
	handler.ServeHTTP(listOptions, listRequest)
	if listOptions.Code != http.StatusOK || listOptions.Header().Get("Allow") != "GET, POST, HEAD, OPTIONS" {
		t.Fatalf("list OPTIONS response = %d %q %v", listOptions.Code, listOptions.Body.String(), listOptions.Header())
	}
	var listBody map[string]any
	if err := json.Unmarshal(listOptions.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode list OPTIONS response: %v", err)
	}
	actions, ok := listBody["actions"].(map[string]any)
	if !ok {
		t.Fatalf("list OPTIONS actions = %T, want object", listBody["actions"])
	}
	post, ok := actions[http.MethodPost].(map[string]any)
	if !ok {
		t.Fatalf("list OPTIONS POST metadata = %T, want object", actions[http.MethodPost])
	}
	idField, idOK := post["id"].(map[string]any)
	colorField, colorOK := post["color"].(map[string]any)
	if len(post) != 17 || !idOK || idField["read_only"] != true || !colorOK || colorField["max_length"] != float64(255) {
		t.Fatalf("list OPTIONS POST metadata = %#v", post)
	}

	detailRequest := httptest.NewRequest(http.MethodOptions, "/api/workspaces/test/stickies/"+id+"/", nil)
	detailRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	detailOptions := httptest.NewRecorder()
	handler.ServeHTTP(detailOptions, detailRequest)
	if detailOptions.Code != http.StatusOK || detailOptions.Header().Get("Allow") != "GET, PATCH, DELETE, HEAD, OPTIONS" {
		t.Fatalf("detail OPTIONS response = %d %q %v", detailOptions.Code, detailOptions.Body.String(), detailOptions.Header())
	}
	var detailBody map[string]any
	if err := json.Unmarshal(detailOptions.Body.Bytes(), &detailBody); err != nil {
		t.Fatalf("decode detail OPTIONS response: %v", err)
	}
	if detailBody["name"] != "Sticky Instance" {
		t.Fatalf("detail OPTIONS name = %v", detailBody["name"])
	}
	if _, exists := detailBody["actions"]; exists {
		t.Fatalf("detail OPTIONS actions = %v, want omitted", detailBody["actions"])
	}

	unauthenticated := httptest.NewRecorder()
	handler.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodOptions, "/api/workspaces/test/stickies/", nil))
	var detail map[string]string
	if err := json.Unmarshal(unauthenticated.Body.Bytes(), &detail); err != nil ||
		unauthenticated.Code != http.StatusUnauthorized || detail["detail"] != "Authentication credentials were not provided." ||
		unauthenticated.Header().Get("Allow") != "GET, POST, HEAD, OPTIONS" {
		t.Fatalf("unauthenticated OPTIONS response = %d %q", unauthenticated.Code, unauthenticated.Body.String())
	}

	unsupportedAccept := httptest.NewRequest(http.MethodOptions, "/api/workspaces/test/stickies/", nil)
	unsupportedAccept.Header.Set("Accept", "application/xml")
	notAcceptable := httptest.NewRecorder()
	handler.ServeHTTP(notAcceptable, unsupportedAccept)
	if notAcceptable.Code != http.StatusNotAcceptable ||
		notAcceptable.Body.String() != "{\"detail\":\"Could not satisfy the request Accept header.\"}\n" {
		t.Fatalf("unsupported Accept response = %d %q", notAcceptable.Code, notAcceptable.Body.String())
	}
	malformed := httptest.NewRecorder()
	handler.ServeHTTP(malformed, httptest.NewRequest(http.MethodOptions, "/api/workspaces/test/stickies/not-a-uuid/", nil))
	if malformed.Code != http.StatusNotFound {
		t.Fatalf("malformed detail OPTIONS response = %d %q", malformed.Code, malformed.Body.String())
	}

	for _, requestedMethod := range []string{http.MethodGet, http.MethodTrace, ""} {
		preflightRequest := httptest.NewRequest(http.MethodOptions, "/api/workspaces/test/stickies/", nil)
		preflightRequest.Header.Set("Origin", "https://app.example.test")
		preflightRequest.Header.Set("Accept", "application/xml")
		preflightRequest.Header["Access-Control-Request-Method"] = []string{requestedMethod}
		preflight := httptest.NewRecorder()
		handler.ServeHTTP(preflight, preflightRequest)
		if preflight.Code != http.StatusOK || preflight.Body.Len() != 0 ||
			preflight.Header().Get("Content-Length") != "0" ||
			preflight.Header().Get("Content-Type") != "text/html; charset=utf-8" ||
			preflight.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" ||
			preflight.Header().Get("Vary") != "origin" {
			t.Errorf("preflight %q response = %d %q %v", requestedMethod, preflight.Code, preflight.Body.String(), preflight.Header())
		}
	}
}

type stickyWorkspaceAuthorizerStub struct {
	err  error
	slug string
}

func (stub *stickyWorkspaceAuthorizerStub) AuthorizeStickyWorkspace(_ context.Context, slug string) error {
	stub.slug = slug
	return stub.err
}

func TestStickyOptionsRequireWorkspaceMembership(t *testing.T) {
	mux := loomhttp.NewMuxer()
	authorizer := &stickyWorkspaceAuthorizerStub{err: &sticky.StickyPermissionError{Message: "You don't have the required permissions."}}
	handler := stickyWorkspaceAuthorizationMiddleware(mux, authorizer, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("workspace-denied OPTIONS reached metadata handler")
	}))
	handler = stickyProtocolHeadersMiddleware(handler)
	mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/stickies/", handler)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodOptions, "/api/workspaces/other/stickies/", nil))
	if response.Code != http.StatusForbidden || response.Body.String() != "{\"error\":\"You don't have the required permissions.\"}\n" ||
		response.Header().Get("Allow") != "GET, POST, HEAD, OPTIONS" {
		t.Fatalf("cross-workspace OPTIONS response = %d %q", response.Code, response.Body.String())
	}
	if authorizer.slug != "other" {
		t.Fatalf("authorized workspace slug = %q, want other", authorizer.slug)
	}

	authorizer.err = errors.New("workspace lookup unavailable")
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodOptions, "/api/workspaces/other/stickies/", nil))
	if response.Code != http.StatusInternalServerError || response.Body.String() != "{\"error\":\"Something went wrong please try again later\"}\n" ||
		response.Header().Get("Allow") != "GET, POST, HEAD, OPTIONS" {
		t.Fatalf("workspace-error OPTIONS response = %d %q", response.Code, response.Body.String())
	}
}

func TestStickyHeadPaginationRequiresWorkspaceMembership(t *testing.T) {
	mux := loomhttp.NewMuxer()
	authorizer := &stickyWorkspaceAuthorizerStub{err: &sticky.StickyPermissionError{Message: "You don't have the required permissions."}}
	stub := stickyHTTPStub{}
	handler := stickyHeadPaginationAuthorizationMiddleware(mux, stub.APIKeyAuth, authorizer, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("workspace-denied HEAD reached list handler")
	}))
	handler = djangoProtocolOptionsJSONNegotiationMiddleware(handler)
	handler = stickyProtocolHeadersMiddleware(handler)
	loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/stickies/", handler)

	request := httptest.NewRequest(http.MethodHead, "/api/workspaces/other/stickies/?per_page=invalid", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || response.Body.Len() != 0 ||
		response.Header().Get("Allow") != "GET, POST, HEAD, OPTIONS" {
		t.Fatalf("cross-workspace pagination HEAD response = %d %q", response.Code, response.Body.String())
	}
}
func TestStickyHeadersMergeVaryTokens(t *testing.T) {
	header := http.Header{"Vary": {"Accept-Encoding, cookie", "Origin"}}
	setStickyHeaders(header)
	if got, want := header.Get("Vary"), "Accept-Encoding, cookie, Origin"; got != want {
		t.Fatalf("Vary = %q, want %q", got, want)
	}
}

func TestAllFiveStickyHTTPRoutesReturnDeclaredSuccessStatuses(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	systemEndpoints := system.NewEndpoints(systemapi.NewSystem())
	stub := stickyHTTPStub{success: true}
	handler := newHTTPHandlerWithStickies(ctx, systemEndpoints, false, sticky.NewEndpoints(stub), stub.APIKeyAuth)
	id := "0198a28d-d227-7752-82ac-bc4c1c290703"
	cases := []struct {
		method string
		path   string
		body   string
		status int
	}{
		{http.MethodGet, "/api/workspaces/test/stickies/", "", http.StatusOK},
		{http.MethodPost, "/api/workspaces/test/stickies/", `{}`, http.StatusCreated},
		{http.MethodGet, "/api/workspaces/test/stickies/" + id + "/", "", http.StatusOK},
		{http.MethodPatch, "/api/workspaces/test/stickies/" + id + "/", `{}`, http.StatusOK},
		{http.MethodDelete, "/api/workspaces/test/stickies/" + id + "/", "", http.StatusNoContent},
	}
	for _, testCase := range cases {
		request := httptest.NewRequest(testCase.method, testCase.path, strings.NewReader(testCase.body))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != testCase.status {
			t.Errorf("%s %s status = %d, want %d; body %q", testCase.method, testCase.path, response.Code, testCase.status, response.Body.String())
		}
		if testCase.status == http.StatusNoContent && response.Body.Len() != 0 {
			t.Errorf("DELETE body = %q, want empty", response.Body.String())
		}
		isList := testCase.method == http.MethodGet && testCase.path == "/api/workspaces/test/stickies/"
		if testCase.status != http.StatusNoContent && !isList && !strings.Contains(response.Body.String(), `"deleted_at":null`) {
			t.Errorf("%s %s body omits the recorded nullable fields: %q", testCase.method, testCase.path, response.Body.String())
		}
	}
}

func TestStickyFrontendDetailPathsWithoutTrailingSlash(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	systemEndpoints := system.NewEndpoints(systemapi.NewSystem())
	stub := stickyHTTPStub{success: true}
	handler := newHTTPHandlerWithStickies(ctx, systemEndpoints, false, sticky.NewEndpoints(stub), stub.APIKeyAuth)
	path := "/api/workspaces/test/stickies/0198a28d-d227-7752-82ac-bc4c1c290703"

	for _, testCase := range []struct {
		method string
		status int
	}{
		{method: http.MethodGet, status: http.StatusOK},
		{method: http.MethodDelete, status: http.StatusNoContent},
	} {
		request := httptest.NewRequest(testCase.method, path, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != testCase.status {
			t.Errorf("%s %s status = %d, want %d; body %q", testCase.method, path, response.Code, testCase.status, response.Body.String())
		}
	}
}

func TestStickyWriteMediaTypesReachEquivalentServicePayloads(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPatch} {
		var baseline string
		for _, mediaType := range []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"} {
			t.Run(method+"/"+mediaType, func(t *testing.T) {
				body, contentType := stickyMediaBody(t, mediaType)
				stub := &stickyMediaCaptureStub{stickyHTTPStub: stickyHTTPStub{success: true}}
				handler := newHTTPHandlerWithStickies(
					newLogContext(context.Background(), io.Discard, false),
					system.NewEndpoints(systemapi.NewSystem()), false,
					sticky.NewEndpoints(stub), stub.APIKeyAuth,
				)
				path := "/api/workspaces/test/stickies/"
				if method == http.MethodPatch {
					path += "0198a28d-d227-7752-82ac-bc4c1c290703/"
				}
				request := httptest.NewRequest(method, path, body)
				request.Header.Set("Content-Type", contentType)
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != map[string]int{http.MethodPost: http.StatusCreated, http.MethodPatch: http.StatusOK}[method] {
					t.Fatalf("response = %d %q", response.Code, response.Body.String())
				}
				got := capturedStickyFields(t, stub, method)
				if baseline == "" {
					baseline = got
				} else if got != baseline {
					t.Fatalf("service payload = %s, want %s", got, baseline)
				}
			})
		}
	}
}

func TestStickyPatchPreservesNullableAuditFieldPresence(t *testing.T) {
	createdBy := "0198a28d-d227-7752-82ac-bc4c1c290704"
	updatedBy := "0198a28d-d227-7752-82ac-bc4c1c290705"
	deletedAt := "2026-08-13T02:03:04Z"
	tests := []struct {
		name string
		body string
		want func(*testing.T, loom.Nullable[string], loom.Nullable[string])
	}{
		{name: "omitted", body: `{}`, want: func(t *testing.T, created, deleted loom.Nullable[string]) {
			if created.Present() || deleted.Present() {
				t.Fatalf("nullable fields = created %#v deleted %#v, want absent", created, deleted)
			}
		}},
		{name: "null", body: `{"background_color":null,"color":null,"created_by":null,"deleted_at":null,"description_stripped":null,"name":null,"updated_by":null}`, want: func(t *testing.T, created, deleted loom.Nullable[string]) {
			if !created.IsNull() || !deleted.IsNull() {
				t.Fatalf("nullable fields = created %#v deleted %#v, want null", created, deleted)
			}
		}},
		{name: "value", body: `{"background_color":"white","color":"blue","created_by":"` + createdBy + `","deleted_at":"` + deletedAt + `","description_stripped":"derived","name":"note","updated_by":"` + updatedBy + `"}`, want: func(t *testing.T, created, deleted loom.Nullable[string]) {
			gotCreated, createdOK := created.Value()
			gotDeleted, deletedOK := deleted.Value()
			if !createdOK || !deletedOK || gotCreated != createdBy || gotDeleted != deletedAt {
				t.Fatalf("nullable fields = created %#v deleted %#v", created, deleted)
			}
		}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			stub := &stickyMediaCaptureStub{stickyHTTPStub: stickyHTTPStub{success: true}}
			handler := newHTTPHandlerWithStickies(
				newLogContext(context.Background(), io.Discard, false),
				system.NewEndpoints(systemapi.NewSystem()), false,
				sticky.NewEndpoints(stub), stub.APIKeyAuth,
			)
			request := httptest.NewRequest(http.MethodPatch, "/api/workspaces/test/stickies/0198a28d-d227-7752-82ac-bc4c1c290703/", strings.NewReader(testCase.body))
			request.Header.Set("Content-Type", "application/json")
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK || stub.updated == nil {
				t.Fatalf("response = %d %q; payload = %#v", response.Code, response.Body.String(), stub.updated)
			}
			testCase.want(t, stub.updated.CreatedBy, stub.updated.DeletedAt)
			assertNullableWriteFields(t, testCase.name, map[string]loom.Nullable[string]{
				"background_color":     stub.updated.BackgroundColor,
				"color":                stub.updated.Color,
				"description_stripped": stub.updated.DescriptionStripped,
				"name":                 stub.updated.Name,
				"updated_by":           stub.updated.UpdatedBy,
			}, map[string]string{
				"background_color": "white", "color": "blue", "description_stripped": "derived", "name": "note", "updated_by": updatedBy,
			})
		})
	}
}

func TestStickyPostAcceptsNullableRequestStrings(t *testing.T) {
	updatedBy := "0198a28d-d227-7752-82ac-bc4c1c290705"
	values := map[string]string{
		"background_color": "white", "color": "blue", "created_by": "0198a28d-d227-7752-82ac-bc4c1c290704",
		"deleted_at": "2026-08-13T02:03:04Z", "description_stripped": "derived", "name": "note", "updated_by": updatedBy,
	}
	for _, testCase := range []struct{ name, body string }{
		{name: "omitted", body: `{}`},
		{name: "null", body: `{"background_color":null,"color":null,"created_by":null,"deleted_at":null,"description_stripped":null,"name":null,"updated_by":null}`},
		{name: "value", body: `{"background_color":"white","color":"blue","created_by":"0198a28d-d227-7752-82ac-bc4c1c290704","deleted_at":"2026-08-13T02:03:04Z","description_stripped":"derived","name":"note","updated_by":"` + updatedBy + `"}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			stub := &stickyMediaCaptureStub{stickyHTTPStub: stickyHTTPStub{success: true}}
			handler := newHTTPHandlerWithStickies(
				newLogContext(context.Background(), io.Discard, false),
				system.NewEndpoints(systemapi.NewSystem()), false,
				sticky.NewEndpoints(stub), stub.APIKeyAuth,
			)
			request := httptest.NewRequest(http.MethodPost, "/api/workspaces/test/stickies/", strings.NewReader(testCase.body))
			request.Header.Set("Content-Type", "application/json")
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusCreated || stub.created == nil {
				t.Fatalf("response = %d %q; payload = %#v", response.Code, response.Body.String(), stub.created)
			}
			assertNullableWriteFields(t, testCase.name, map[string]loom.Nullable[string]{
				"background_color": stub.created.BackgroundColor, "color": stub.created.Color,
				"created_by": stub.created.CreatedBy, "deleted_at": stub.created.DeletedAt,
				"description_stripped": stub.created.DescriptionStripped, "name": stub.created.Name, "updated_by": stub.created.UpdatedBy,
			}, values)
		})
	}
}

func assertNullableWriteFields(t *testing.T, state string, fields map[string]loom.Nullable[string], values map[string]string) {
	t.Helper()
	for name, field := range fields {
		switch state {
		case "omitted":
			if field.Present() {
				t.Fatalf("%s = %#v, want absent", name, field)
			}
		case "null":
			if !field.IsNull() {
				t.Fatalf("%s = %#v, want null", name, field)
			}
		case "value":
			value, ok := field.Value()
			if !ok || value != values[name] {
				t.Fatalf("%s = %#v, want %q", name, field, values[name])
			}
		}
	}
}

func TestStickyListEncodesNullDescriptionBinary(t *testing.T) {
	stub := stickyHTTPStub{success: true}
	handler := newHTTPHandlerWithStickies(
		newLogContext(context.Background(), io.Discard, false),
		system.NewEndpoints(systemapi.NewSystem()), false,
		sticky.NewEndpoints(stub), stub.APIKeyAuth,
	)
	request := httptest.NewRequest(http.MethodGet, "/api/workspaces/test/stickies/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
	var body struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Results) != 1 {
		t.Fatalf("results = %#v, want one Sticky", body.Results)
	}
	value, present := body.Results[0]["description_binary"]
	if !present || value != nil {
		t.Fatalf("description_binary = %#v, present %t, want explicit null", value, present)
	}
}

func stickyMediaBody(t *testing.T, mediaType string) (io.Reader, string) {
	t.Helper()
	values := url.Values{
		"name": {"Migration fixture"}, "description": {`{"content":[],"type":"doc"}`},
		"description_html": {"<p>Migration fixture</p>"}, "logo_props": {`{}`}, "sort_order": {"65535"},
	}
	switch mediaType {
	case "application/json":
		return strings.NewReader(`{"description":{"content":[],"type":"doc"},"description_html":"<p>Migration fixture</p>","logo_props":{},"name":"Migration fixture","sort_order":65535}`), mediaType
	case "application/x-www-form-urlencoded":
		return strings.NewReader(values.Encode()), mediaType
	default:
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		for name, items := range values {
			for _, value := range items {
				if err := writer.WriteField(name, value); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		return &body, writer.FormDataContentType()
	}
}

func capturedStickyFields(t *testing.T, stub *stickyMediaCaptureStub, method string) string {
	t.Helper()
	var value any
	if method == http.MethodPost {
		value = stub.created
	} else {
		value = stub.updated
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	delete(payload, "session_id")
	delete(payload, "slug")
	delete(payload, "pk")
	canonical, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return string(canonical)
}

func stickyHTTPSuccessResponse() *sticky.StickyResponse {
	return &sticky.StickyResponse{
		ID: "0198a28d-d227-7752-82ac-bc4c1c290703", Workspace: "0198a28d-d227-7752-82ac-bc4c1c290702", Owner: "0198a28d-d227-7752-82ac-bc4c1c290701",
		CreatedAt: "2026-08-13T01:02:03Z", UpdatedAt: "2026-08-13T01:02:03Z", Description: loom.JSONValue(`{}`), DescriptionHTML: "<p></p>", DescriptionBinary: loom.NullValue[[]byte](), LogoProps: loom.JSONValue(`{}`), SortOrder: 65535,
	}
}

func assertJSONRequestLog(t *testing.T, data []byte, requestID string) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	foundRequestLog := false
	for {
		var entry map[string]any
		if err := decoder.Decode(&entry); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("decode structured log: %v\nlogs:\n%s", err, data)
		}
		if entry[log.RequestIDKey] == requestID {
			foundRequestLog = true
		}
	}
	if !foundRequestLog {
		t.Fatalf("no structured log contains request ID %q\nlogs:\n%s", requestID, data)
	}
}

func TestParseConfigRejectsInvalidValues(t *testing.T) {
	for _, args := range [][]string{
		{"--http-address="},
		{"--shutdown-timeout=0s"},
		{"unexpected"},
	} {
		if _, err := parseConfig(args); err == nil {
			t.Fatalf("parseConfig(%q) returned nil error", args)
		}
	}
}

func TestParseConfigRequiresBackgroundTasksForNoWorkersMode(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-secret")
	if _, err := parseConfig([]string{"--go-background-tasks-no-workers"}); err == nil || !strings.Contains(err.Error(), "requires go-background-tasks") {
		t.Fatalf("parseConfig(no-workers without runtime) error = %v", err)
	}
	cfg, err := parseConfig([]string{"--go-background-tasks", "--go-background-tasks-no-workers"})
	if err != nil {
		t.Fatalf("parseConfig(no-workers runtime mode): %v", err)
	}
	if !cfg.goBackgroundTasks || !cfg.goBackgroundTasksNoWorkers {
		t.Fatalf("runtime flags = %t/%t, want true/true", cfg.goBackgroundTasks, cfg.goBackgroundTasksNoWorkers)
	}
}

func TestParseConfigUsesDjangoReadReplicaSettings(t *testing.T) {
	t.Run("URL", func(t *testing.T) {
		t.Setenv("SECRET_KEY", "test-secret")
		t.Setenv("ENABLE_READ_REPLICA", "1")
		t.Setenv("DATABASE_READ_REPLICA_URL", "postgresql://reader:secret@replica:5432/plane")
		cfg, err := parseConfig(nil)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.readReplicaDatabaseURL != "postgresql://reader:secret@replica:5432/plane" {
			t.Fatalf("readReplicaDatabaseURL = %q", cfg.readReplicaDatabaseURL)
		}
	})

	t.Run("components", func(t *testing.T) {
		t.Setenv("SECRET_KEY", "test-secret")
		t.Setenv("ENABLE_READ_REPLICA", "1")
		t.Setenv("DATABASE_READ_REPLICA_URL", "")
		t.Setenv("POSTGRES_READ_REPLICA_DB", "plane_read")
		t.Setenv("POSTGRES_READ_REPLICA_USER", "reader")
		t.Setenv("POSTGRES_READ_REPLICA_PASSWORD", "p@ss")
		t.Setenv("POSTGRES_READ_REPLICA_HOST", "replica-db")
		t.Setenv("POSTGRES_READ_REPLICA_PORT", "5433")
		cfg, err := parseConfig(nil)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.readReplicaDatabaseURL != "postgresql://reader:p%40ss@replica-db:5433/plane_read" {
			t.Fatalf("readReplicaDatabaseURL = %q", cfg.readReplicaDatabaseURL)
		}
	})

	t.Run("flag overrides incomplete components", func(t *testing.T) {
		t.Setenv("SECRET_KEY", "test-secret")
		t.Setenv("ENABLE_READ_REPLICA", "1")
		t.Setenv("DATABASE_READ_REPLICA_URL", "")
		t.Setenv("POSTGRES_READ_REPLICA_DB", "")
		t.Setenv("POSTGRES_READ_REPLICA_USER", "")
		t.Setenv("POSTGRES_READ_REPLICA_HOST", "")
		for _, override := range []string{"postgresql://reader:secret@replica:5432/plane", ""} {
			cfg, err := parseConfig([]string{"--read-replica-database-url=" + override})
			if err != nil {
				t.Fatalf("parseConfig(%q): %v", override, err)
			}
			if cfg.readReplicaDatabaseURL != override {
				t.Fatalf("readReplicaDatabaseURL = %q, want %q", cfg.readReplicaDatabaseURL, override)
			}
		}
	})

	t.Run("incomplete components", func(t *testing.T) {
		t.Setenv("ENABLE_READ_REPLICA", "1")
		t.Setenv("DATABASE_READ_REPLICA_URL", "")
		t.Setenv("POSTGRES_READ_REPLICA_DB", "")
		t.Setenv("POSTGRES_READ_REPLICA_USER", "")
		t.Setenv("POSTGRES_READ_REPLICA_HOST", "")
		if _, err := parseConfig(nil); err == nil || !strings.Contains(err.Error(), "read replica requires") {
			t.Fatalf("parseConfig() error = %v", err)
		}
	})
}

func TestRunRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SECRET_KEY", "test-secret")
	if err := run(context.Background(), nil, io.Discard); err == nil || err.Error() != "database-url is required" {
		t.Fatalf("run() error = %v, want missing database URL error", err)
	}
}

func TestParseConfigRequiresDjangoSecretKey(t *testing.T) {
	t.Setenv("SECRET_KEY", "")
	if _, err := parseConfig(nil); err == nil || err.Error() != "django-secret-key is required" {
		t.Fatalf("parseConfig() error = %v", err)
	}
}

func TestParseConfigUsesDjangoSessionCookieName(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-secret")
	t.Setenv("SESSION_COOKIE_NAME", "custom-session")
	t.Setenv("COOKIE_DOMAIN", ".example.test")
	t.Setenv("SESSION_COOKIE_AGE", "3600")
	t.Setenv("ADMIN_SESSION_COOKIE_AGE", "1800")
	t.Setenv("SESSION_SAVE_EVERY_REQUEST", "1")
	t.Setenv("SECRET_KEY_FALLBACKS", " old-one,old-two, ")
	cfg, err := parseConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.sessionCookie != "custom-session" {
		t.Fatalf("sessionCookie=%q", cfg.sessionCookie)
	}
	if cfg.cookieDomain != ".example.test" {
		t.Fatalf("cookieDomain=%q", cfg.cookieDomain)
	}
	if cfg.sessionCookieAge != 3600 || !cfg.saveEveryRequest {
		t.Fatalf("sessionCookieAge=%v saveEveryRequest=%v", cfg.sessionCookieAge, cfg.saveEveryRequest)
	}
	if cfg.adminSessionCookie != "admin-session-id" || cfg.adminSessionCookieAge != 1800 {
		t.Fatalf("admin session cookie=%q age=%v", cfg.adminSessionCookie, cfg.adminSessionCookieAge)
	}
	if !slices.Equal(cfg.fallbackKeys, []string{"old-one", "old-two"}) {
		t.Fatalf("fallbackKeys=%v", cfg.fallbackKeys)
	}
	t.Setenv("SESSION_COOKIE_NAME", "")
	cfg, err = parseConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.sessionCookie != "" {
		t.Fatalf("explicit empty sessionCookie=%q", cfg.sessionCookie)
	}
}

func TestParseConfigUsesDjangoCORSAllowedOrigins(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-secret")
	previousCSRFOrigins, hadCSRFOrigins := os.LookupEnv("CSRF_TRUSTED_ORIGINS")
	if err := os.Unsetenv("CSRF_TRUSTED_ORIGINS"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if hadCSRFOrigins {
			_ = os.Setenv("CSRF_TRUSTED_ORIGINS", previousCSRFOrigins)
		} else {
			_ = os.Unsetenv("CSRF_TRUSTED_ORIGINS")
		}
	})
	t.Setenv("CORS_ALLOWED_ORIGINS", " https://one.example.test,https://two.example.test, ")
	cfg, err := parseConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://one.example.test", "https://two.example.test"}
	if !slices.Equal(cfg.corsOrigins, want) {
		t.Fatalf("corsOrigins=%v want=%v", cfg.corsOrigins, want)
	}
	if !slices.Equal(cfg.csrfTrustedOrigins, want) {
		t.Fatalf("default csrfTrustedOrigins=%v want CORS origins %v", cfg.csrfTrustedOrigins, want)
	}
	if !cfg.sessionCookieSecure {
		t.Fatal("HTTPS-only CORS origins must enable secure session cookies")
	}
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://one.example.test,http://localhost:3000")
	cfg, err = parseConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.sessionCookieSecure {
		t.Fatal("an HTTP CORS origin must disable secure session cookies")
	}
	t.Setenv("CSRF_TRUSTED_ORIGINS", " https://csrf.example.test, ")
	cfg, err = parseConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cfg.corsOrigins, []string{"https://one.example.test", "http://localhost:3000"}) {
		t.Fatalf("explicit CSRF trust changed CORS origins: %v", cfg.corsOrigins)
	}
	if !slices.Equal(cfg.csrfTrustedOrigins, []string{"https://csrf.example.test"}) {
		t.Fatalf("explicit csrfTrustedOrigins=%v", cfg.csrfTrustedOrigins)
	}
	cfg, err = parseConfig([]string{
		"--cors-allowed-origins=https://cors-cli.example.test",
		"--csrf-trusted-origins=https://csrf-cli.example.test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cfg.corsOrigins, []string{"https://cors-cli.example.test"}) {
		t.Fatalf("explicit CSRF CLI setting changed CORS origins: %v", cfg.corsOrigins)
	}
	if !slices.Equal(cfg.csrfTrustedOrigins, []string{"https://csrf-cli.example.test"}) {
		t.Fatalf("explicit csrfTrustedOrigins CLI value=%v", cfg.csrfTrustedOrigins)
	}
}

func TestParseConfigUsesDjangoDebugEnvironment(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-secret")
	for _, testCase := range []struct {
		value string
		want  bool
	}{
		{value: "0"},
		{value: "1", want: true},
		{value: " 1 ", want: true},
		{value: "١", want: true},
		{value: "2"},
		{value: "-1"},
		{value: "1_0"},
		{value: "999999999999999999999999999999999999999999"},
	} {
		t.Run(testCase.value, func(t *testing.T) {
			t.Setenv("DEBUG", testCase.value)
			cfg, err := parseConfig(nil)
			if err != nil || cfg.debug != testCase.want {
				t.Fatalf("DEBUG=%q debug=%v err=%v", testCase.value, cfg.debug, err)
			}
		})
	}
	for _, value := range []string{"", "not-an-integer", "1__0", "𑛑"} {
		t.Setenv("DEBUG", value)
		if _, err := parseConfig(nil); err == nil {
			t.Fatalf("DEBUG=%q was accepted", value)
		}
	}
}

func TestParseConfigEnforcesPythonDecimalIntegerLimit(t *testing.T) {
	for _, name := range []string{"DEBUG", "FILE_SIZE_LIMIT", "SESSION_COOKIE_AGE", "ADMIN_SESSION_COOKIE_AGE"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("SECRET_KEY", "test-secret")
			t.Setenv(name, strings.Repeat("9", 4301))
			if _, err := parseConfig(nil); err == nil {
				t.Fatalf("%s accepted a 4301-digit Python integer", name)
			}
		})
	}
}

func TestParseConfigParsesDjangoSessionCookieAge(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-secret")
	t.Setenv("SESSION_COOKIE_AGE", " 3_600 ")
	if cfg, err := parseConfig(nil); err != nil || cfg.sessionCookieAge != 3600 {
		t.Fatalf("Python-compatible SESSION_COOKIE_AGE=%v err=%v", cfg.sessionCookieAge, err)
	}
	t.Setenv("SESSION_COOKIE_AGE", "10_000_000_000")
	if cfg, err := parseConfig(nil); err != nil || cfg.sessionCookieAge != 10_000_000_000 {
		t.Fatalf("large SESSION_COOKIE_AGE=%v err=%v", cfg.sessionCookieAge, err)
	}
	t.Setenv("SESSION_COOKIE_AGE", "999999999999999999999999999999")
	if cfg, err := parseConfig(nil); err != nil || cfg.sessionCookieAge != math.MaxInt64 {
		t.Fatalf("arbitrary-sized SESSION_COOKIE_AGE=%v err=%v", cfg.sessionCookieAge, err)
	}
	for _, testCase := range []struct {
		age  string
		want int64
	}{{age: "0"}, {age: "-1", want: -1}} {
		t.Run(testCase.age, func(t *testing.T) {
			t.Setenv("SESSION_COOKIE_AGE", testCase.age)
			cfg, err := parseConfig(nil)
			if err != nil || cfg.sessionCookieAge != testCase.want {
				t.Fatalf("SESSION_COOKIE_AGE=%q age=%v err=%v", testCase.age, cfg.sessionCookieAge, err)
			}
		})
	}
	for _, age := range []string{"not-a-number"} {
		t.Run(age, func(t *testing.T) {
			t.Setenv("SESSION_COOKIE_AGE", age)
			if _, err := parseConfig(nil); err == nil {
				t.Fatalf("SESSION_COOKIE_AGE=%q was accepted", age)
			}
		})
	}
	t.Setenv("SESSION_COOKIE_AGE", "3600")
	for _, testCase := range []struct {
		age  string
		want int64
	}{{age: "0"}, {age: "-1", want: -1}} {
		t.Run("admin "+testCase.age, func(t *testing.T) {
			t.Setenv("ADMIN_SESSION_COOKIE_AGE", testCase.age)
			cfg, err := parseConfig(nil)
			if err != nil || cfg.adminSessionCookieAge != testCase.want {
				t.Fatalf("ADMIN_SESSION_COOKIE_AGE=%q age=%v err=%v", testCase.age, cfg.adminSessionCookieAge, err)
			}
		})
	}
	for _, age := range []string{"not-a-number"} {
		t.Run("admin "+age, func(t *testing.T) {
			t.Setenv("ADMIN_SESSION_COOKIE_AGE", age)
			if _, err := parseConfig(nil); err == nil {
				t.Fatalf("ADMIN_SESSION_COOKIE_AGE=%q was accepted", age)
			}
		})
	}
}

func TestParseConfigUsesDjangoFileSizeLimit(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-secret")
	t.Setenv("FILE_SIZE_LIMIT", " ١٢_٣٤٥ ")
	cfg, err := parseConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.fileSizeLimit != 12345 {
		t.Fatalf("fileSizeLimit=%d", cfg.fileSizeLimit)
	}
	for _, testCase := range []struct {
		limit string
		want  int64
	}{{limit: "0"}, {limit: "-1", want: -1}} {
		t.Run(testCase.limit, func(t *testing.T) {
			t.Setenv("FILE_SIZE_LIMIT", testCase.limit)
			cfg, err := parseConfig(nil)
			if err != nil || cfg.fileSizeLimit != testCase.want {
				t.Fatalf("FILE_SIZE_LIMIT=%q limit=%d err=%v", testCase.limit, cfg.fileSizeLimit, err)
			}
		})
	}
	for _, limit := range []string{"not-a-number"} {
		t.Run(limit, func(t *testing.T) {
			t.Setenv("FILE_SIZE_LIMIT", limit)
			if _, err := parseConfig(nil); err == nil {
				t.Fatalf("FILE_SIZE_LIMIT=%q was accepted", limit)
			}
		})
	}
}
