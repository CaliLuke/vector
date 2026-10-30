package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	stduuid "uuid"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	store "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	invitationserver "github.com/CaliLuke/go-plane/invitationsmembership/gen/http/invitations_membership/server"
	loomhttp "github.com/CaliLuke/loom/http"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	api "github.com/CaliLuke/go-plane/invitationsmembership/gen/invitations_membership"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type invitationsMembershipKeyValidator struct {
	key   string
	err   error
	calls int
}

func (validator *invitationsMembershipKeyValidator) Validate(_ context.Context, key string) (apikey.Principal, error) {
	validator.calls++
	validator.key = key
	return apikey.Principal{UserID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Timezone: "UTC"}, validator.err
}

type invitationsMembershipSessionValidator struct{}

func (invitationsMembershipSessionValidator) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{}, errors.New("unexpected session validation")
}

func TestInvitationsMembershipAPIKeyAuthValidatesKeyAndRecordsAuthentication(t *testing.T) {
	validator := &invitationsMembershipKeyValidator{}
	service := &invitationsMembershipService{keys: validator, sessions: invitationsMembershipSessionValidator{}}
	effects := &apikey.Effects{}
	ctx, err := service.APIKeyAuth(apikey.WithEffects(context.Background(), effects), "test-key", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		t.Fatal(err)
	}
	principal, ok := ctx.Value(invitationsMembershipPrincipalKey{}).(invitationsMembershipPrincipal)
	if !ok || principal.UserID.String() != "00000000-0000-4000-8000-000000000001" || !principal.APIKey || principal.Key != "test-key" {
		t.Fatalf("principal=%+v present=%t", principal, ok)
	}
	if validator.calls != 1 || validator.key != "test-key" {
		t.Fatalf("validator calls=%d key=%q", validator.calls, validator.key)
	}
	if !effects.Authenticated {
		t.Fatal("successful API-key validation did not mark request authenticated")
	}
}

func TestInvitationsMembershipAPIKeyAuthRejectsMissingAndInvalidKeys(t *testing.T) {
	for _, test := range []struct {
		name       string
		key        string
		err        error
		wantStatus string
	}{
		{name: "missing", err: apikey.ErrInvalid, wantStatus: "status_401"},
		{name: "invalid", key: "bad-key", err: apikey.ErrInvalid, wantStatus: "status_403"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &invitationsMembershipService{keys: &invitationsMembershipKeyValidator{err: test.err}, sessions: invitationsMembershipSessionValidator{}}
			_, err := service.APIKeyAuth(context.Background(), test.key, &security.APIKeyScheme{Name: "ApiKeyAuth"})
			require.IsType(t, (*api.InvitationsMembershipAPIKeyError)(nil), err)
			var response *api.InvitationsMembershipAPIKeyError
			require.ErrorAs(t, err, &response)
			if response.Name != test.wantStatus {
				t.Fatalf("error=%T %v, want %s", err, err, test.wantStatus)
			}
		})
	}
}

type invitationCacheSpy struct{ events *[]string }

func (spy invitationCacheSpy) Invalidate(_ context.Context, path, user string, multiple bool) error {
	*spy.events = append(*spy.events, path)
	return nil
}

type invitationCommandSpy struct {
	*store.InvitationsRepository
	events     *[]string
	permission error
}

func (spy invitationCommandSpy) AuthorizeWorkspaceRole(context.Context, uuid.UUID, string, ...int) error {
	*spy.events = append(*spy.events, "permission")
	return spy.permission
}
func (spy invitationCommandSpy) Invitation(context.Context, store.InvitationCommand) (json.RawMessage, error) {
	*spy.events = append(*spy.events, "command")
	return nil, &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You do not have permission to join the workspace"}}
}
func TestWorkspaceJoinInvalidatesBeforeTokenFailure(t *testing.T) {
	events := []string{}
	service := &invitationsMembershipService{repository: invitationCommandSpy{events: &events}, sessions: invitationsMembershipSessionValidator{}, cache: invitationCacheSpy{&events}}
	_, err := service.executeInvitation(context.Background(), store.InvitationCommand{Operation: "workspace-join", Slug: "example"})
	var failure *api.InvitationsMembershipAPIKeyError
	if !errors.As(err, &failure) || failure.Name != "status_403" {
		t.Fatalf("error=%v", err)
	}
	want := []string{"/api/workspaces/", "/api/users/me/workspaces/", "/api/workspaces/example/members/", "/api/users/me/settings/", "command"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events=%v want=%v", events, want)
	}
}
func TestWorkspaceLeaveInvalidatesBeforeActionPermissionFailure(t *testing.T) {
	events := []string{}
	service := &invitationsMembershipService{repository: invitationCommandSpy{events: &events, permission: store.ErrForbidden}, cache: invitationCacheSpy{&events}}
	ctx := context.WithValue(context.Background(), invitationsMembershipPrincipalKey{}, invitationsMembershipPrincipal{UserID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Timezone: "UTC"})
	_, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "member-leave", Slug: "example"})
	var failure *api.InvitationsMembershipAPIKeyError
	if !errors.As(err, &failure) || failure.Name != "status_403" {
		t.Fatalf("error=%v", err)
	}
	want := []string{"/api/workspaces/example/members/", "/api/users/me/settings/", "api/users/me/workspaces/", "permission"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events=%v want=%v", events, want)
	}
}

type invitationRosterPageRepository struct{ *store.InvitationsRepository }

func (invitationRosterPageRepository) AuthorizeWorkspaceRole(context.Context, uuid.UUID, string, ...int) error {
	return nil
}
func (invitationRosterPageRepository) WorkspaceExists(context.Context, string) (bool, error) {
	return true, nil
}
func (invitationRosterPageRepository) ListWorkspaceMembers(_ context.Context, _ stduuid.UUID, _ string, query store.WorkspaceMembersQuery) (store.WorkspaceMembersPage, error) {
	limit, offset := query.Limit+1, query.Page*query.Limit
	if limit != 3 {
		return store.WorkspaceMembersPage{}, errors.New("wrong page limit")
	}
	if offset == 2 {
		return store.WorkspaceMembersPage{Results: json.RawMessage(`[{"id":"00000000-0000-4000-8000-000000000003","role_slug":null},{"id":"00000000-0000-4000-8000-000000000004","role_slug":null}]`), Total: 4}, nil
	}
	return store.WorkspaceMembersPage{Results: json.RawMessage(`[{"id":"00000000-0000-4000-8000-000000000002","role_slug":null},{"id":"00000000-0000-4000-8000-000000000003","role_slug":null},{"id":"00000000-0000-4000-8000-000000000004","role_slug":null}]`), Total: 4}, nil
}
func TestWorkspaceRosterCursorUsesPageIndexForMultirowPage(t *testing.T) {
	service := &invitationsMembershipService{repository: invitationRosterPageRepository{}}
	ctx := context.WithValue(context.Background(), invitationsMembershipPrincipalKey{}, invitationsMembershipPrincipal{UserID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Timezone: "UTC"})
	perPage, cursor := 2, "2:1:0"
	result, err := service.GetAPIV1WorkspacesBySlugMembersLite(ctx, &api.GetAPIV1WorkspacesBySlugMembersLitePayload{Slug: "workspace", PerPage: &perPage, Cursor: &cursor})
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(result)
	var body struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
		Next bool `json:"next_page_results"`
	}
	if err = json.Unmarshal(encoded, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Results) != 2 || body.Results[0].ID != "00000000-0000-4000-8000-000000000003" || body.Next {
		t.Fatalf("page=%s", encoded)
	}
}

type invitationTimezoneRepository struct{ *store.InvitationsRepository }

func (invitationTimezoneRepository) AuthorizeWorkspaceRole(context.Context, uuid.UUID, string, ...int) error {
	return nil
}
func (invitationTimezoneRepository) Invitation(context.Context, store.InvitationCommand) (json.RawMessage, error) {
	return json.RawMessage(`{"workspace":"00000000-0000-4000-8000-000000000001","created_at":"2026-01-01T01:02:03.1Z","default_props":{"created_at":"2026-01-01T01:02:03.1Z"}}`), nil
}
func TestInvitationSerializerUsesActorTimezoneAndPreservesJSONData(t *testing.T) {
	service := &invitationsMembershipService{repository: invitationTimezoneRepository{}}
	ctx := context.WithValue(context.Background(), invitationsMembershipPrincipalKey{}, invitationsMembershipPrincipal{UserID: uuid.MustParse("00000000-0000-4000-8000-000000000002"), Timezone: "Asia/Kolkata"})
	result, err := service.PatchAPIWorkspacesBySlugMembersByPk(ctx, &api.PatchAPIWorkspacesBySlugMembersByPkPayload{Slug: "workspace", Pk: "member"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(result)
	var body map[string]any
	_ = json.Unmarshal(encoded, &body)
	if body["created_at"] != "2026-01-01T06:32:03.100000+05:30" {
		t.Fatalf("created_at=%v", body["created_at"])
	}
	if body["default_props"].(map[string]any)["created_at"] != "2026-01-01T01:02:03.1Z" {
		t.Fatalf("JSON field rewritten: %s", encoded)
	}
}

type invitationEmptyStringRepository struct{ *store.InvitationsRepository }

func (invitationEmptyStringRepository) AuthorizeWorkspaceRole(context.Context, uuid.UUID, string, ...int) error {
	return nil
}
func (invitationEmptyStringRepository) WorkspaceExists(context.Context, string) (bool, error) {
	return true, nil
}
func (invitationEmptyStringRepository) ListWorkspaceMembers(context.Context, stduuid.UUID, string, store.WorkspaceMembersQuery) (store.WorkspaceMembersPage, error) {
	return store.WorkspaceMembersPage{Results: json.RawMessage(`[{"id":"00000000-0000-4000-8000-000000000001","first_name":"","last_name":"","display_name":"","email":"invitee@example.com","avatar":"","avatar_url":null,"is_bot":false,"is_active":false,"role":5,"role_slug":"guest"}]`), Total: 1}, nil
}
func TestInvitationGeneratedRosterRetainsPresentEmptyStrings(t *testing.T) {
	service := &invitationsMembershipService{repository: invitationEmptyStringRepository{}, keys: &invitationsMembershipKeyValidator{}, sessions: invitationsMembershipSessionValidator{}, throttle: &activityThrottleStub{rate: apikey.Rate{Allowed: true}}}
	mux := loomhttp.NewMuxer()
	transport := invitationserver.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	transport.Mount(mux)
	request := httptest.NewRequest("GET", "/api/v1/workspaces/workspace/members-lite/", nil)
	request.Header.Set("X-Api-Key", "key")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Results) != 1 {
		t.Fatalf("body=%s", response.Body.String())
	}
	row := body.Results[0]
	for _, key := range []string{"first_name", "last_name", "display_name", "avatar"} {
		if value, ok := row[key]; !ok || value != "" {
			t.Errorf("field %s=%v present=%t", key, value, ok)
		}
	}
	if row["is_bot"] != false || row["is_active"] != false {
		t.Fatalf("zero boolean lost: %v", row)
	}
}

func TestInvitationMalformedV1IdentifierAuthenticatesBeforeValidation(t *testing.T) {
	for _, test := range []struct {
		name, key     string
		authError     error
		status        int
		authenticated bool
	}{{"anonymous", "", apikey.ErrInvalid, 401, false}, {"authenticated", "valid", nil, 400, true}} {
		t.Run(test.name, func(t *testing.T) {
			validator := &invitationsMembershipKeyValidator{err: test.authError}
			service := &invitationsMembershipService{repository: invitationRosterPageRepository{}, keys: validator, sessions: invitationsMembershipSessionValidator{}, throttle: &activityThrottleStub{rate: apikey.Rate{Allowed: true}}}
			mux := loomhttp.NewMuxer()
			transport := invitationserver.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
			transport.Mount(mux)
			request := httptest.NewRequest("GET", "/api/v1/workspaces/workspace/invitations/not-a-uuid/", nil)
			if test.key != "" {
				request.Header.Set("X-Api-Key", test.key)
			}
			effects := &apikey.Effects{}
			request = request.WithContext(apikey.WithEffects(request.Context(), effects))
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != test.status || validator.calls != 1 || effects.Authenticated != test.authenticated {
				t.Fatalf("status=%d calls=%d authenticated=%t body=%s", response.Code, validator.calls, effects.Authenticated, response.Body.String())
			}
		})
	}
}
func TestInvitationRosterParsesOriginalInvalidPageSizeAfterAuthentication(t *testing.T) {
	service := &invitationsMembershipService{repository: invitationRosterPageRepository{}}
	ctx := context.WithValue(context.Background(), invitationsMembershipPrincipalKey{}, invitationsMembershipPrincipal{UserID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Timezone: "UTC"})
	ctx = WithReadServiceQueryPresence(ctx, url.Values{"per_page": {"1", "many"}})
	normalized := 1
	_, err := service.GetAPIV1WorkspacesBySlugMembersLite(ctx, &api.GetAPIV1WorkspacesBySlugMembersLitePayload{Slug: "workspace", PerPage: &normalized})
	var failure *api.InvitationsMembershipAPIKeyError
	if !errors.As(err, &failure) || failure.Name != "status_400" {
		t.Fatalf("error=%v", err)
	}
	encoded, _ := json.Marshal(failure)
	if !strings.Contains(string(encoded), "Invalid per_page parameter.") {
		t.Fatalf("error=%s", encoded)
	}
}

type invitationMissingRepository struct{ *store.InvitationsRepository }

func (invitationMissingRepository) AuthorizeWorkspaceRole(context.Context, uuid.UUID, string, ...int) error {
	return nil
}
func (invitationMissingRepository) CreateAPIWorkspaceInvite(context.Context, string, uuid.UUID, io.Reader, string) (json.RawMessage, error) {
	return nil, store.ErrNotFound
}
func (invitationMissingRepository) WorkspaceInvite(context.Context, string, uuid.UUID) (json.RawMessage, error) {
	return nil, store.ErrNotFound
}
func TestInvitationGeneratedMissingObjectsRemain404AfterAuthentication(t *testing.T) {
	for _, test := range []struct{ method, path, body string }{{"POST", "/api/v1/workspaces/missing/invitations/", `{"email":"invalid","role":false}`}, {"GET", "/api/v1/workspaces/workspace/invitations/00000000-0000-4000-8000-000000000001/", ""}} {
		t.Run(test.method, func(t *testing.T) {
			validator := &invitationsMembershipKeyValidator{}
			service := &invitationsMembershipService{repository: invitationMissingRepository{}, keys: validator, sessions: invitationsMembershipSessionValidator{}, throttle: &activityThrottleStub{rate: apikey.Rate{Allowed: true}}}
			mux := loomhttp.NewMuxer()
			transport := invitationserver.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
			transport.Mount(mux)
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			request.Header.Set("X-Api-Key", "valid")
			request.Header.Set("Content-Type", "application/json")
			effects := &apikey.Effects{}
			request = request.WithContext(apikey.WithEffects(request.Context(), effects))
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			var body map[string]any
			_ = json.Unmarshal(response.Body.Bytes(), &body)
			if response.Code != 404 || body["error"] != "The required object does not exist." || !effects.Authenticated || validator.calls != 1 {
				t.Fatalf("status=%d authenticated=%t calls=%d body=%s", response.Code, effects.Authenticated, validator.calls, response.Body.String())
			}
		})
	}
}

func TestInvitationGeneratedTransportPreservesUnsupportedMedia415(t *testing.T) {
	events := []string{}
	service := &invitationsMembershipService{repository: &store.InvitationsRepository{}, sessions: &activitySessionStub{principal: djangosession.Principal{UserID: uuid.NewString(), UserTimezone: "UTC"}}, cache: invitationCacheSpy{&events}}
	mux := loomhttp.NewMuxer()
	transport := invitationserver.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	transport.Mount(mux)
	request := httptest.NewRequest("POST", "/api/users/me/workspaces/invitations/", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "text/plain")
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid"})
	response := httptest.NewRecorder()
	policy, err := loomhttp.NewRequestMetadataPolicy([]string{"Content-Type", "Cookie"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	loomhttp.RequestMetadataMiddleware(policy)(mux).ServeHTTP(response, request)
	var body map[string]any
	_ = json.Unmarshal(response.Body.Bytes(), &body)
	if response.Code != 415 || body["detail"] != `Unsupported media type "text/plain" in request.` {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

type invitationEmptyBodyRepository struct {
	invitationMissingRepository
	called bool
}

func (repository *invitationEmptyBodyRepository) CreateAPIWorkspaceInvite(_ context.Context, _ string, _ uuid.UUID, body io.Reader, contentType string) (json.RawMessage, error) {
	repository.called = true
	if body != nil {
		data, err := io.ReadAll(body)
		if err != nil || len(data) != 0 {
			return nil, errors.New("expected empty body")
		}
	}
	return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"email": []string{"This field is required."}}}
}

func TestInvitationBodyReaderPreservesEmptyRequestsForSerializerValidation(t *testing.T) {
	for _, body := range []io.ReadCloser{nil, io.NopCloser(strings.NewReader(""))} {
		repository := &invitationEmptyBodyRepository{}
		service := &invitationsMembershipService{repository: repository}
		ctx := context.WithValue(context.Background(), invitationsMembershipPrincipalKey{}, invitationsMembershipPrincipal{UserID: uuid.New(), Timezone: "UTC"})
		_, err := service.PostAPIV1WorkspacesBySlugInvitations(ctx, &api.PostAPIV1WorkspacesBySlugInvitationsPayload{Slug: "workspace"}, body)
		encoded, _ := json.Marshal(err)
		if !repository.called || !strings.Contains(string(encoded), "This field is required.") {
			t.Fatalf("empty request did not reach serializer: called=%t error=%s", repository.called, encoded)
		}
	}
}

type invitationOptionsRepository struct {
	*store.InvitationsRepository
	events                  *[]string
	permission, objectError error
}

func (repository invitationOptionsRepository) AuthorizeWorkspaceRole(context.Context, uuid.UUID, string, ...int) error {
	*repository.events = append(*repository.events, "permission")
	return repository.permission
}
func (repository invitationOptionsRepository) WorkspaceInvite(context.Context, string, uuid.UUID) (json.RawMessage, error) {
	*repository.events = append(*repository.events, "object")
	return json.RawMessage(`{}`), repository.objectError
}

func TestInvitationDetailOptionsLooksUpObjectAfterAuthenticationAndPermission(t *testing.T) {
	for _, test := range []struct {
		name, method, id, timezone         string
		authError, permission, objectError error
		status                             int
		key, message                       string
		events                             []string
	}{
		{name: "existing", method: "OPTIONS", id: "00000000-0000-4000-8000-000000000001", events: []string{"permission", "object"}},
		{name: "missing", method: "OPTIONS", id: "00000000-0000-4000-8000-000000000001", objectError: store.ErrNotFound, status: 404, key: "error", message: "The required object does not exist.", events: []string{"permission", "object"}},
		{name: "malformed", method: "OPTIONS", id: "not-a-uuid", status: 400, key: "error", message: "Please provide valid detail", events: []string{"permission"}},
		{name: "anonymous-before-malformed", method: "OPTIONS", id: "not-a-uuid", authError: apikey.ErrInvalid, status: 401, key: "detail", message: "Authentication credentials were not provided."},
		{name: "permission-before-object", method: "OPTIONS", id: "not-a-uuid", permission: store.ErrForbidden, status: 403, key: "detail", message: "You do not have permission to perform this action.", events: []string{"permission"}},
		{name: "timezone-before-object", method: "OPTIONS", id: "not-a-uuid", timezone: "not-a-timezone", status: 400, key: "error", message: "The required key does not exist.", events: []string{"permission"}},
		{name: "head-no-metadata-lookup", method: "HEAD", id: "not-a-uuid", events: []string{"permission"}},
		{name: "collection-no-object", method: "OPTIONS", events: []string{"permission"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			events := []string{}
			zone := test.timezone
			if zone == "" {
				zone = "UTC"
			}
			keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: zone}, err: test.authError}
			service := &invitationsMembershipService{repository: invitationOptionsRepository{events: &events, permission: test.permission, objectError: test.objectError}, keys: keys}
			credential := "valid"
			if test.authError != nil {
				credential = ""
			}
			status, body := service.AuthorizeInvitationsMembershipProtocol(context.Background(), credential, SessionProtocolResource{Method: test.method, Pattern: "/api/v1/workspaces/{slug}/invitations/{pk}/", Slug: "workspace", ID: test.id}, true)
			if status != test.status || (test.key != "" && body[test.key] != test.message) || len(events) != len(test.events) || (len(events) > 0 && !reflect.DeepEqual(events, test.events)) || keys.calls != 1 {
				t.Fatalf("status=%d body=%v events=%v calls=%d", status, body, events, keys.calls)
			}
		})
	}
}
