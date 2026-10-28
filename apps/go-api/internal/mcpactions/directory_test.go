package mcpactions_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	membership "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
)

type directoryStub struct {
	capabilities    membership.ReadCapabilities
	capabilitiesErr error
	body            json.RawMessage
	total           int
	err             error
}

func (s *directoryStub) ListWorkspaceMembers(context.Context, uuid.UUID, string, membership.WorkspaceMembersQuery) (membership.WorkspaceMembersPage, error) {
	return membership.WorkspaceMembersPage{Results: s.body, Total: s.total}, s.err
}
func (s *directoryStub) ListRolesInWorkspace(context.Context, uuid.UUID, string, string, int, int, uuid.UUID) ([]membership.RoleDefinition, int, error) {
	return nil, s.total, s.err
}
func (s *directoryStub) RetrieveRoleInWorkspace(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (membership.RoleDefinition, error) {
	return membership.RoleDefinition{}, s.err
}

func TestDirectoryFailuresNeverExposeApplicationDetails(t *testing.T) {
	for _, tc := range []struct {
		input string
		err   error
		body  string
		total int
		code  string
	}{
		{`{"action":"list_workspace"}`, errors.New("private database details"), "", 0, "internal_error"},
		{`{"action":"list_roles"}`, errors.New("private database details"), "", 0, "internal_error"},
		{`{"action":"retrieve_role","role_id":"10000000-0000-4000-8000-000000000001"}`, errors.New("private database details"), "", 0, "internal_error"},
		{`{"action":"list_workspace"}`, nil, `{"private":"bad shape"}`, 0, "internal_error"},
		{`{"action":"list_workspace"}`, nil, `[]`, -1, "internal_error"},
		{`{"action":"list_roles"}`, nil, "", -1, "internal_error"},
		{`{"action":"list_workspace","cursor":"1:1:1"}`, nil, "", 0, "internal_error"},
		{`{"action":"list_roles","cursor":"bad"}`, nil, "", 0, "invalid_params"},
	} {
		t.Run(tc.input+tc.body+tc.code, func(t *testing.T) {
			f := newFixture(t)
			f.directory.err = tc.err
			f.directory.body = json.RawMessage(tc.body)
			f.directory.total = tc.total
			result := rpc(t, f.handler(t, nil), "test-one", "tools/call", memberCall(tc.input))
			require.True(t, result.IsError)
			assert.Contains(t, result.Content[0].Text, "["+tc.code+"]")
			assert.NotContains(t, result.Content[0].Text, "private")
		})
	}
}

func TestDirectDirectoryCallsValidatePageSizeAndRoleID(t *testing.T) {
	f := newFixture(t)
	request := httptest.NewRequest(http.MethodPost, "https://vector.example/mcp/api-key", nil)
	request.Header.Set("Authorization", "Bearer test-one")
	request.Header.Set("X-Workspace-Slug", "workspace-one")
	ctx, err := f.auth.Authenticate(t.Context(), request)
	require.NoError(t, err)
	for _, input := range []api.MemberRequest{
		api.NewMemberRequestMemberListWorkspace(&api.MemberListWorkspace{Action: "list_workspace", PerPage: new(-1)}),
		api.NewMemberRequestMemberRetrieveRole(&api.MemberRetrieveRole{Action: "retrieve_role", RoleID: "bad"}),
	} {
		_, err = f.service.Member(ctx, &input)
		require.Error(t, err)
	}
}

func (s *directoryStub) ReadCapabilities(context.Context, uuid.UUID, uuid.UUID, string) (membership.ReadCapabilities, error) {
	return s.capabilities, s.capabilitiesErr
}

func (s *directoryStub) WorkspaceIdentity(context.Context, uuid.UUID, uuid.UUID, string) (membership.WorkspaceIdentity, error) {
	return membership.WorkspaceIdentity{}, s.err
}
