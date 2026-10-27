//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/go-plane/internal/mcpactions"
	membership "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	"github.com/CaliLuke/go-plane/internal/useridentity"
	projects "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	googleuuid "github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type changingMemberReader struct {
	*membership.MemberReader
	change func()
}

func (r changingMemberReader) ListProjectMembersInWorkspace(ctx context.Context, actor uuid.UUID, slug string, id, workspace uuid.UUID) (json.RawMessage, error) {
	r.change()
	return r.MemberReader.ListProjectMembersInWorkspace(ctx, actor, slug, id, workspace)
}

type changingDirectoryReader struct {
	*membership.DirectoryReader
	change func()
}

func (r changingDirectoryReader) ListWorkspaceMembers(ctx context.Context, actor uuid.UUID, slug string, q membership.WorkspaceMembersQuery) (membership.WorkspaceMembersPage, error) {
	r.change()
	return r.DirectoryReader.ListWorkspaceMembers(ctx, actor, slug, q)
}
func (r changingDirectoryReader) ListRolesInWorkspace(ctx context.Context, actor uuid.UUID, slug, namespace string, limit, page int, workspace uuid.UUID) ([]membership.RoleDefinition, int, error) {
	r.change()
	return r.DirectoryReader.ListRolesInWorkspace(ctx, actor, slug, namespace, limit, page, workspace)
}
func (r changingDirectoryReader) RetrieveRoleInWorkspace(ctx context.Context, actor uuid.UUID, slug string, id, workspace uuid.UUID) (membership.RoleDefinition, error) {
	r.change()
	return r.DirectoryReader.RetrieveRoleInWorkspace(ctx, actor, slug, id, workspace)
}

func memberFixtureWithScopeChange(t *testing.T, f fixture, pool *pgxpool.Pool, mutation string) fixture {
	t.Helper()
	repo, err := useridentity.NewRepository(pool)
	require.NoError(t, err)
	identity, err := useridentity.NewIdentityReader(repo)
	require.NoError(t, err)
	members, err := membership.NewLifecycleRepository(pool, rosterPublisher{}, "https://vector.example")
	require.NoError(t, err)
	memberReader, err := membership.NewMemberReader(members)
	require.NoError(t, err)
	directory, err := membership.NewDirectoryReader(pool)
	require.NoError(t, err)
	projectReader, err := projects.NewRepository(pool)
	require.NoError(t, err)
	change := func() { _, changeErr := pool.Exec(t.Context(), mutation); require.NoError(t, changeErr) }
	cycleReads, err := cyclereads.NewRepository(pool)
	require.NoError(t, err)
	f.service, err = mcpactions.New(identity, changingMemberReader{memberReader, change}, changingDirectoryReader{directory, change}, projectReader, projectLifecycle(t, pool), cycleReads)
	require.NoError(t, err)
	return f
}

func TestMemberDataCannotFollowReassignedWorkspaceSlug(t *testing.T) {
	otherRole := googleuuid.NewSHA1(googleuuid.MustParse("20000000-0000-4000-8000-000000000002"), []byte("vector.role/v1/workspace/member"))
	for _, input := range []string{
		`{"action":"list_workspace"}`, `{"action":"list_roles"}`,
		`{"action":"retrieve_role","role_id":"` + otherRole.String() + `"}`,
		`{"action":"list_project","project_id":"30000000-0000-4000-8000-000000000003"}`,
	} {
		t.Run(input, func(t *testing.T) {
			f, pool := databaseFixture(t)
			_, err := pool.Exec(t.Context(), `
UPDATE api_tokens SET workspace_id='20000000-0000-4000-8000-000000000001' WHERE token='test-one';
INSERT INTO workspace_members(workspace_id,member_id,role,is_active) VALUES('20000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000001',15,true);
INSERT INTO project_members(project_id,workspace_id,member_id,role,is_active) VALUES('30000000-0000-4000-8000-000000000003','20000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000001',15,true);`)
			require.NoError(t, err)
			f = memberFixtureWithScopeChange(t, f, pool, `UPDATE workspaces SET slug='renamed' WHERE slug='workspace-one'; UPDATE workspaces SET slug='workspace-one' WHERE slug='workspace-two'`)
			result := rpc(t, f.handler(t, nil), "test-one", "tools/call", memberCall(input))
			require.True(t, result.IsError, result.Content)
			require.Empty(t, result.StructuredContent)
		})
	}
}

func TestProjectMemberReadRechecksWorkspaceMembershipAfterActionAdmission(t *testing.T) {
	f, pool := databaseFixture(t)
	f = memberFixtureWithScopeChange(t, f, pool, `UPDATE workspace_members SET is_active=false WHERE member_id='10000000-0000-4000-8000-000000000001'`)
	result := rpc(t, f.handler(t, nil), "test-one", "tools/call", memberCall(`{"action":"list_project","project_id":"30000000-0000-4000-8000-000000000001"}`))
	require.True(t, result.IsError, result.Content)
	require.Empty(t, result.StructuredContent)
}
