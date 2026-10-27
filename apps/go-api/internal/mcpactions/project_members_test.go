package mcpactions_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const projectOne = "30000000-0000-4000-8000-000000000001"
const projectTwo = "30000000-0000-4000-8000-000000000002"

type projectScope struct{ actor, workspace, project string }

var firstProjectScope = projectScope{"10000000-0000-4000-8000-000000000001", "workspace-one", projectOne}

type projectStore struct {
	rows                  map[projectScope]any
	authorizeErr, readErr error
	reads                 atomic.Int64
}

func newProjectStore() *projectStore {
	return &projectStore{rows: map[projectScope]any{
		firstProjectScope: []any{map[string]any{"id": firstProjectScope.actor, "first_name": "First project", "email": "first@example.test"}},
		{"10000000-0000-4000-8000-000000000002", "workspace-one", projectTwo}: []any{map[string]any{"id": "10000000-0000-4000-8000-000000000002", "email": "second@example.test"}},
	}}
}

func (s *projectStore) AuthorizeLifecycle(_ context.Context, command projectmembershipreads.LifecycleCommand) error {
	if s.authorizeErr != nil {
		return s.authorizeErr
	}
	if _, ok := s.rows[projectScope{command.UserID, command.Slug, command.ProjectID}]; !ok {
		return &djangoapi.HTTPError{Status: http.StatusForbidden, Body: map[string]any{"detail": "private permission details"}}
	}
	return nil
}

func (s *projectStore) ExecuteLifecycle(ctx context.Context, command projectmembershipreads.LifecycleCommand) (any, error) {
	if err := s.AuthorizeLifecycle(ctx, command); err != nil {
		return nil, err
	}
	s.reads.Add(1)
	return s.rows[projectScope{command.UserID, command.Slug, command.ProjectID}], s.readErr
}

func TestProjectMemberCallsPreserveSDKShapeAndActorScope(t *testing.T) {
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		t.Run(version, func(t *testing.T) {
			f := newFixture(t)
			h := f.handler(t, nil)
			first := rpcVersion(t, h, "test-one", "tools/call", memberCall(`{"action":"list_project","project_id":"`+projectOne+`"}`), version)
			require.False(t, first.IsError, first.Content)
			require.Len(t, first.Content, 1)
			assert.Empty(t, first.StructuredContent)
			assert.JSONEq(t, `[{"id":"10000000-0000-4000-8000-000000000001","first_name":"First project","last_name":null,"email":"first@example.test","avatar":null,"avatar_url":null,"display_name":null,"role":null,"role_slug":null,"is_active":null,"is_bot":null}]`, first.Content[0].Text)
			second := rpcVersion(t, h, "test-two", "tools/call", memberCall(`{"action":"list_project","project_id":"`+projectTwo+`","workspace_slug":"workspace-one"}`), version)
			require.False(t, second.IsError, second.Content)
			require.Len(t, second.Content, 1)
			assert.Contains(t, second.Content[0].Text, "second@example.test")
			assert.NotContains(t, second.Content[0].Text, "first@example.test")
			denied := rpcVersion(t, h, "test-two", "tools/call", memberCall(`{"action":"list_project","project_id":"`+projectOne+`"}`), version)
			require.True(t, denied.IsError)
			assert.Contains(t, denied.Content[0].Text, "[forbidden]")
			assert.NotContains(t, denied.Content[0].Text, "private permission")
			assert.EqualValues(t, 2, f.projects.reads.Load())
		})
	}
}

func TestEmptyProjectMemberResultHasNoContent(t *testing.T) {
	f := newFixture(t)
	f.projects.rows[firstProjectScope] = []any{}
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		result := rpcVersion(t, f.handler(t, nil), "test-one", "tools/call", memberCall(`{"action":"list_project","project_id":"`+projectOne+`"}`), version)
		require.False(t, result.IsError, result.Content)
		assert.Empty(t, result.Content)
		assert.Empty(t, result.StructuredContent)
	}
}

func TestProjectMemberRejectsOtherActionFieldsBeforeReading(t *testing.T) {
	f := newFixture(t)
	for _, suffix := range []string{`,"per_page":10`, `,"role_id":"` + projectTwo + `"`} {
		result := rpc(t, f.handler(t, nil), "test-one", "tools/call", memberCall(`{"action":"list_project","project_id":"`+projectOne+`"`+suffix+`}`))
		require.True(t, result.IsError)
		assert.Contains(t, result.Content[0].Text, "[invalid_params]")
		assert.Zero(t, f.projects.reads.Load())
	}
}

func TestConflictingWorkspaceSelectorsNeverReadApplicationData(t *testing.T) {
	f := newFixture(t)
	for _, input := range []string{
		`{"action":"me","workspace_slug":"workspace-two"}`,
		`{"action":"me","workspace_slug":""}`,
		`{"action":"list_project","project_id":"` + projectOne + `","workspace_slug":"workspace-two"}`,
	} {
		request := httptest.NewRequest(http.MethodPost, "https://vector.example/mcp/api-key", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"member","arguments":`+input+`}}`))
		request.Header.Set("Authorization", "Bearer test-one")
		request.Header.Set("X-Workspace-Slug", "workspace-one")
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json, text/event-stream")
		request.Header.Set("Mcp-Protocol-Version", "2025-11-25")
		response := httptest.NewRecorder()
		f.handler(t, nil).ServeHTTP(response, request)
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Equal(t, "invalid_request\n", response.Body.String())
		assert.Zero(t, f.store.reads.Load())
		assert.Zero(t, f.projects.reads.Load())
	}
}

func TestProjectMemberFailuresAreSafe(t *testing.T) {
	for _, tc := range []struct {
		name, code string
		failure    error
		body       any
	}{
		{"database error", "internal_error", errors.New("private database password"), []any{}},
		{"deleted workspace", "not_found", &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "private details"}}, []any{}},
		{"not found", "not_found", &djangoapi.HTTPError{Status: 404}, []any{}},
		{"domain failure", "internal_error", &djangoapi.HTTPError{Status: 500}, []any{}},
		{"null result", "internal_error", nil, nil},
		{"object result", "internal_error", nil, map[string]any{}},
		{"null row", "internal_error", nil, []any{nil}},
		{"invalid member field", "internal_error", nil, []any{map[string]any{"role": "admin"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.projects.readErr = tc.failure
			f.projects.rows[firstProjectScope] = tc.body
			result := rpc(t, f.handler(t, nil), "test-one", "tools/call", memberCall(`{"action":"list_project","project_id":"`+projectOne+`"}`))
			require.True(t, result.IsError)
			require.NotEmpty(t, result.Content)
			assert.Contains(t, result.Content[0].Text, "["+tc.code+"]")
			assert.NotContains(t, result.Content[0].Text, "private")
			assert.Empty(t, result.StructuredContent)
		})
	}
}

func TestDirectMemberCallsStillValidateGeneratedBranchAndProject(t *testing.T) {
	f := newFixture(t)
	request := httptest.NewRequest(http.MethodPost, "https://vector.example/mcp/api-key", nil)
	request.Header.Set("Authorization", "Bearer test-one")
	request.Header.Set("X-Workspace-Slug", "workspace-one")
	ctx, err := f.auth.Authenticate(t.Context(), request)
	require.NoError(t, err)
	for _, input := range []api.MemberListProjectOrMemberMe{
		{},
		api.NewMemberListProjectOrMemberMeMemberMe(&api.MemberMe{Action: "unknown"}),
		api.NewMemberListProjectOrMemberMeMemberMe(&api.MemberMe{Action: "me", WorkspaceSlug: new("workspace-two")}),
		api.NewMemberListProjectOrMemberMeMemberListProject(&api.MemberListProject{Action: "list_project", ProjectID: "invalid"}),
	} {
		_, err := f.service.Member(ctx, &input)
		require.Error(t, err)
	}
	assert.Zero(t, f.projects.reads.Load())
	assert.Zero(t, f.store.reads.Load())
}
