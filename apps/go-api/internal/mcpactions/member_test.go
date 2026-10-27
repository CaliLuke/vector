package mcpactions_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/mcpactions"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	"github.com/CaliLuke/go-plane/internal/useridentity"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	googleuuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type identityStore struct {
	users map[uuid.UUID]json.RawMessage
	reads atomic.Int64
	err   error
}

func (s *identityStore) APIV1Identity(_ context.Context, actor uuid.UUID) (json.RawMessage, error) {
	s.reads.Add(1)
	if s.err != nil {
		return nil, s.err
	}
	body, ok := s.users[actor]
	if !ok {
		return nil, useridentity.ErrNotFound
	}
	return body, nil
}

type tokens struct {
	owners map[string]apikey.Principal
	err    error
}

func (s *tokens) Validate(_ context.Context, token string) (apikey.Principal, error) {
	if s.err != nil {
		return apikey.Principal{}, s.err
	}
	owner, ok := s.owners[token]
	if !ok {
		return apikey.Principal{}, apikey.ErrInvalid
	}
	return owner, nil
}

type memberships struct{ err error }

func (m *memberships) Membership(context.Context, googleuuid.UUID, string) (mcpauth.Workspace, error) {
	return mcpauth.Workspace{ID: googleuuid.MustParse("20000000-0000-4000-8000-000000000001"), Role: 15}, m.err
}

type throttle struct{}

func (throttle) AllowCacheKey(context.Context, string, string) (apikey.Rate, error) {
	return apikey.Rate{Allowed: true}, nil
}

type fixture struct {
	service       *mcpactions.Service
	auth          *mcpauth.PersonalTokens
	store         *identityStore
	keys          *tokens
	members       *memberships
	projects      *projectStore
	directory     *directoryStub
	projectReads  *projectReadStub
	projectWrites *projectWriteStub
}

func newFixture(t *testing.T, options ...mcpactions.Option) fixture {
	t.Helper()
	one := googleuuid.MustParse("10000000-0000-4000-8000-000000000001")
	two := googleuuid.MustParse("10000000-0000-4000-8000-000000000002")
	store := &identityStore{users: map[uuid.UUID]json.RawMessage{
		uuid.UUID(one): json.RawMessage(`{"id":"10000000-0000-4000-8000-000000000001","first_name":"Zoë","email":null}`),
		uuid.UUID(two): json.RawMessage(`{"id":"10000000-0000-4000-8000-000000000002","first_name":"Second","email":"second@example.test"}`),
	}}
	reader, err := useridentity.NewIdentityReader(store)
	require.NoError(t, err)
	projects := newProjectStore()
	projectReader, err := projectmembershipreads.NewMemberReader(projects)
	require.NoError(t, err)
	directory := &directoryStub{capabilities: projectmembershipreads.ReadCapabilities{Workspace: true, WorkspaceMembers: true, ProjectMembers: true}}
	projectReads := newProjectReadStub()
	projectWrites := newProjectWriteStub()
	service, err := mcpactions.New(reader, projectReader, directory, projectReads, projectWrites, options...)
	require.NoError(t, err)
	keys := &tokens{owners: map[string]apikey.Principal{
		"test-one": {UserID: one, Timezone: "UTC"}, "test-two": {UserID: two, Timezone: "UTC"},
	}}
	members := &memberships{}
	auth, err := mcpauth.NewPersonalTokens(keys, members, throttle{})
	require.NoError(t, err)
	return fixture{service: service, auth: auth, store: store, keys: keys, members: members, projects: projects, directory: directory, projectReads: projectReads, projectWrites: projectWrites}
}

func (f fixture) handler(t *testing.T, authenticate mcptransport.Authenticate) http.Handler {
	t.Helper()
	cfg := mcptransport.DefaultConfig()
	cfg.Enabled, cfg.Mode, cfg.ResourceURL = true, "header", "https://vector.example/mcp"
	if authenticate == nil {
		authenticate = f.auth.Authenticate
	}
	h, err := mcptransport.New(cfg, mcptransport.Options{Service: f.service, HeaderAuth: authenticate}, http.NotFoundHandler())
	require.NoError(t, err)
	return h
}

type rpcResult struct {
	Content           []struct{ Text string }
	StructuredContent json.RawMessage
	IsError           bool
	Tools             []struct {
		Name        string
		Annotations map[string]bool
		InputSchema struct {
			Type                 string
			Properties           map[string]json.RawMessage
			OneOf                []json.RawMessage
			AdditionalProperties bool
		}
	}
}

func rpc(t *testing.T, handler http.Handler, token, method string, params any) rpcResult {
	t.Helper()
	return rpcVersion(t, handler, token, method, params, "2025-11-25")
}

func rpcVersion(t *testing.T, handler http.Handler, token, method string, params any, version string) rpcResult {
	t.Helper()
	if version == "2026-07-28" {
		params.(map[string]any)["_meta"] = map[string]any{
			"io.modelcontextprotocol/protocolVersion":    version,
			"io.modelcontextprotocol/clientInfo":         map[string]string{"name": "test", "version": "1"},
			"io.modelcontextprotocol/clientCapabilities": map[string]any{},
		}
	}
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	require.NoError(t, err)
	r := httptest.NewRequest(http.MethodPost, "https://vector.example/mcp/api-key", strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	r.Header.Set("Mcp-Protocol-Version", version)
	if version == "2026-07-28" {
		r.Header.Set("Mcp-Method", method)
		if method == "tools/call" {
			r.Header.Set("Mcp-Name", params.(map[string]any)["name"].(string))
		}
	}
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("X-Workspace-Slug", "workspace-one")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	var response struct {
		Result rpcResult
		Error  json.RawMessage
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Empty(t, response.Error, w.Body.String())
	return response.Result
}

func memberCall(input string) map[string]any {
	return map[string]any{"name": "member", "arguments": json.RawMessage(input)}
}

func TestMemberDiscoveryAndIdentityAreScopedToTheAuthenticatedUser(t *testing.T) {
	f := newFixture(t)
	h := f.handler(t, nil)
	discovery := rpc(t, h, "test-one", "tools/list", map[string]any{})
	require.Len(t, discovery.Tools, 2)
	assert.Equal(t, "member", discovery.Tools[0].Name)
	assert.Equal(t, map[string]bool{"readOnlyHint": true, "destructiveHint": false, "idempotentHint": false, "openWorldHint": false}, discovery.Tools[0].Annotations)
	assert.Equal(t, "object", discovery.Tools[0].InputSchema.Type)
	assert.JSONEq(t, `{"anyOf":[{"type":"string","enum":["me"]},{"type":"string","enum":["list_project"]},{"type":"string","enum":["list_workspace"]},{"type":"string","enum":["list_roles"]},{"type":"string","enum":["retrieve_role"]}]}`, string(discovery.Tools[0].InputSchema.Properties["action"]))
	require.Len(t, discovery.Tools[0].InputSchema.OneOf, 5)
	for _, branch := range discovery.Tools[0].InputSchema.OneOf {
		var schema struct{ AdditionalProperties *bool }
		require.NoError(t, json.Unmarshal(branch, &schema))
		require.NotNil(t, schema.AdditionalProperties)
		assert.False(t, *schema.AdditionalProperties)
	}
	first := rpc(t, h, "test-one", "tools/call", memberCall(`{"action":"me"}`))
	require.False(t, first.IsError, first.Content)
	assert.JSONEq(t, `{"id":"10000000-0000-4000-8000-000000000001","first_name":"Zoë","last_name":null,"email":null,"avatar":null,"avatar_url":null,"display_name":null}`, string(first.StructuredContent))
	second := rpc(t, h, "test-two", "tools/call", memberCall(`{"action":"me","workspace_slug":"workspace-one"}`))
	require.False(t, second.IsError, second.Content)
	assert.Contains(t, string(second.StructuredContent), "second@example.test")
	assert.NotContains(t, string(first.StructuredContent), "second@example.test")
}

func TestMemberRejectsUnknownActionsAndCrossActionFieldsBeforeApplicationRead(t *testing.T) {
	f := newFixture(t)
	h := f.handler(t, nil)
	for _, input := range []string{
		`{}`, `{"action":null}`, `{"action":"update"}`,
		`{"action":"list_workspace","namespace":"project"}`,
		`{"action":"list_workspace","is_active":"false"}`,
		`{"action":"list_roles","first_name":"example"}`,
		`{"action":"list_roles","namespace":"unknown"}`,
		`{"action":"retrieve_role"}`,
		`{"action":"retrieve_role","role_id":"invalid"}`,
		`{"action":"list_project"}`, `{"action":"list_project","project_id":null}`,
		`{"action":"list_project","project_id":"not-a-uuid"}`,
		`{"action":"me","project_id":"10000000-0000-4000-8000-000000000001"}`,
		`{"action":"me","user_id":"10000000-0000-4000-8000-000000000002"}`,
		`{"action":"me","workspace_slug":null}`,
	} {
		t.Run(input, func(t *testing.T) {
			result := rpc(t, h, "test-one", "tools/call", memberCall(input))
			require.True(t, result.IsError)
			require.NotEmpty(t, result.Content)
			assert.Contains(t, result.Content[0].Text, "[invalid_params]")
			assert.Zero(t, f.store.reads.Load())
		})
	}
}

func TestMemberRevalidatesBeforeApplicationRead(t *testing.T) {
	for _, tc := range []struct {
		name, code string
		revoke     func(fixture)
	}{
		{"token revoked", "unauthorized", func(f fixture) { f.keys.err = apikey.ErrInvalid }},
		{"membership revoked", "forbidden", func(f fixture) { f.members.err = mcptransport.ErrAccessDenied }},
		{"auth database unavailable", "internal_error", func(f fixture) { f.keys.err = errors.New("private database details") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			h := f.handler(t, func(ctx context.Context, r *http.Request) (context.Context, error) {
				verified, err := f.auth.Authenticate(ctx, r)
				require.NoError(t, err)
				tc.revoke(f)
				return verified, nil
			})
			result := rpc(t, h, "test-one", "tools/call", memberCall(`{"action":"me"}`))
			require.True(t, result.IsError)
			require.NotEmpty(t, result.Content)
			assert.Contains(t, result.Content[0].Text, "["+tc.code+"]")
			assert.NotContains(t, result.Content[0].Text, "private database")
			assert.Zero(t, f.store.reads.Load())
		})
	}
}

func TestMemberSanitizesApplicationFailures(t *testing.T) {
	for _, tc := range []struct {
		name    string
		failure error
		body    json.RawMessage
		code    string
	}{
		{"missing user", useridentity.ErrNotFound, nil, "not_found"},
		{"database failure", errors.New("private database details"), nil, "internal_error"},
		{"invalid result", nil, json.RawMessage(`{"email":123}`), "internal_error"},
		{"null result", nil, json.RawMessage(`null`), "internal_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.store.err = tc.failure
			if tc.body != nil {
				f.store.users[uuid.UUID(f.keys.owners["test-one"].UserID)] = tc.body
			}
			result := rpc(t, f.handler(t, nil), "test-one", "tools/call", memberCall(`{"action":"me"}`))
			require.True(t, result.IsError)
			require.NotEmpty(t, result.Content)
			assert.Contains(t, result.Content[0].Text, "["+tc.code+"]")
			assert.NotContains(t, result.Content[0].Text, "private database")
		})
	}
}

func TestMemberRequiresAuthenticatedContextAndValidAction(t *testing.T) {
	f := newFixture(t)
	input := api.NewMemberRequestMemberMe(&api.MemberMe{Action: "me"})
	_, err := f.service.Member(t.Context(), &input)
	require.Error(t, err)
	_, err = f.service.Member(t.Context(), nil)
	require.Error(t, err)
	assert.Zero(t, f.store.reads.Load())
	_, err = mcpactions.New(nil, nil, nil, nil, nil)
	require.Error(t, err)
	reader, err := useridentity.NewIdentityReader(f.store)
	require.NoError(t, err)
	_, err = mcpactions.New(reader, nil, &directoryStub{}, nil, nil)
	require.Error(t, err)
}

func TestMemberRejectsInvalidTimezoneBeforeReadingIdentity(t *testing.T) {
	f := newFixture(t)
	owner := f.keys.owners["test-one"]
	owner.Timezone = "Invalid/Timezone"
	f.keys.owners["test-one"] = owner
	result := rpc(t, f.handler(t, nil), "test-one", "tools/call", memberCall(`{"action":"me"}`))
	require.True(t, result.IsError)
	assert.Zero(t, f.store.reads.Load())
}
