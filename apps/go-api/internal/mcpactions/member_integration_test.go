//go:build integration

package mcpactions_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/activityreads"
	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/go-plane/internal/workitems"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/go-plane/internal/mcpactions"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/modulereads"
	"github.com/CaliLuke/go-plane/internal/projectlabelreads"
	"github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/CaliLuke/go-plane/internal/useridentity"
	"github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const identitySchema = `
CREATE TABLE users (id uuid PRIMARY KEY, is_active boolean NOT NULL, user_timezone text,
 first_name text, last_name text, email text, avatar text, display_name text, avatar_asset_id uuid);
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL, deleted_at timestamptz);
CREATE TABLE workspace_members (workspace_id uuid NOT NULL, member_id uuid NOT NULL, role int NOT NULL, is_active boolean NOT NULL, deleted_at timestamptz);
CREATE TABLE api_tokens (token text PRIMARY KEY, user_id uuid NOT NULL, workspace_id uuid, is_active boolean NOT NULL, expired_at timestamptz, deleted_at timestamptz, last_used timestamptz);
CREATE TABLE file_assets (id uuid PRIMARY KEY, entity_type text, workspace_id uuid, project_id uuid, issue_id uuid);
INSERT INTO users VALUES ('10000000-0000-4000-8000-000000000001',true,'UTC','Zoë','First',NULL,'','first',NULL),
 ('10000000-0000-4000-8000-000000000002',true,'UTC','Second','User','second@example.test','','second',NULL);
INSERT INTO workspaces VALUES ('20000000-0000-4000-8000-000000000001','workspace-one',NULL);
INSERT INTO workspace_members SELECT '20000000-0000-4000-8000-000000000001',id,15,true,NULL FROM users;
INSERT INTO api_tokens VALUES ('test-one','10000000-0000-4000-8000-000000000001',NULL,true,NULL,NULL,NULL),
 ('test-two','10000000-0000-4000-8000-000000000002',NULL,true,NULL,NULL,NULL);
ALTER TABLE users ADD COLUMN created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP;
CREATE TABLE projects(id uuid PRIMARY KEY, workspace_id uuid NOT NULL, deleted_at timestamptz);
CREATE TABLE project_members(project_id uuid NOT NULL, workspace_id uuid NOT NULL, member_id uuid NOT NULL,
 role int NOT NULL, is_active boolean NOT NULL, deleted_at timestamptz);
INSERT INTO workspaces VALUES ('20000000-0000-4000-8000-000000000002','workspace-two',NULL);
INSERT INTO projects VALUES
 ('30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001',NULL),
 ('30000000-0000-4000-8000-000000000002','20000000-0000-4000-8000-000000000001',NULL),
 ('30000000-0000-4000-8000-000000000003','20000000-0000-4000-8000-000000000002',NULL);
INSERT INTO project_members VALUES
 ('30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000001',15,true,NULL),
 ('30000000-0000-4000-8000-000000000002','20000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000002',15,true,NULL),
 ('30000000-0000-4000-8000-000000000003','20000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000002',15,true,NULL);
ALTER TABLE projects ADD COLUMN archived_at timestamptz, ADD COLUMN guest_view_all_features boolean;
ALTER TABLE users ADD COLUMN is_bot boolean NOT NULL DEFAULT false;
ALTER TABLE workspaces ADD COLUMN owner_id uuid NOT NULL DEFAULT '10000000-0000-4000-8000-000000000001';
ALTER TABLE workspace_members ADD COLUMN id uuid NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE workspace_members ADD COLUMN created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE workspace_members ADD COLUMN updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP;
`

func databaseFixture(t *testing.T) (fixture, *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	pair, err := postgresfixture.Start(ctx, func(ctx context.Context, url string) error {
		conn, err := pgx.Connect(ctx, url)
		if err != nil {
			return err
		}
		defer func() { require.NoError(t, conn.Close(context.WithoutCancel(ctx))) }()
		_, err = conn.Exec(ctx, identitySchema)
		return err
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pair.Close()) })
	pool, err := pgxpool.New(ctx, pair.SourceURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	service := databaseService(t, pool, nil)
	keys, err := apikey.NewWithReaderWriter(pool, pool)
	require.NoError(t, err)
	members, err := mcpauth.NewWorkspaceReader(pool)
	require.NoError(t, err)
	auth, err := mcpauth.NewPersonalTokens(keys, members, throttle{})
	require.NoError(t, err)
	return fixture{service: service, auth: auth}, pool
}

type rosterPublisher struct{}

func (rosterPublisher) Task(context.Context, string, []any, map[string]any) error {
	return nil
}

func TestMemberReadsRealIdentityThroughBothProtocolRevisions(t *testing.T) {
	f, pool := databaseFixture(t)
	h := f.handler(t, nil)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		t.Run(version, func(t *testing.T) {
			result := rpcVersion(t, h, "test-one", "tools/call", memberCall(`{"action":"me"}`), version)
			require.False(t, result.IsError, result.Content)
			assert.JSONEq(t, `{"id":"10000000-0000-4000-8000-000000000001","first_name":"Zoë","last_name":"First","email":null,"avatar":"","avatar_url":null,"display_name":"first"}`, string(result.StructuredContent))
			second := rpcVersion(t, h, "test-two", "tools/call", memberCall(`{"action":"me"}`), version)
			require.False(t, second.IsError, second.Content)
			assert.Contains(t, string(second.StructuredContent), "second@example.test")
		})
	}
	_, err := pool.Exec(t.Context(), `UPDATE users SET first_name='Renamed' WHERE id='10000000-0000-4000-8000-000000000001'`)
	require.NoError(t, err)
	updated := rpc(t, h, "test-one", "tools/call", memberCall(`{"action":"me"}`))
	require.False(t, updated.IsError, updated.Content)
	assert.Contains(t, string(updated.StructuredContent), "Renamed")
}

func TestMemberRechecksPrimaryDatabaseAfterRequestAuthentication(t *testing.T) {
	for _, tc := range []struct{ name, mutation, code string }{
		{"token revoked", `UPDATE api_tokens SET deleted_at=CURRENT_TIMESTAMP WHERE token='test-one'`, "unauthorized"},
		{"membership removed", `DELETE FROM workspace_members WHERE member_id='10000000-0000-4000-8000-000000000001'`, "forbidden"},
		{"user deactivated", `UPDATE users SET is_active=false WHERE id='10000000-0000-4000-8000-000000000001'`, "unauthorized"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, pool := databaseFixture(t)
			h := f.handler(t, func(ctx context.Context, request *http.Request) (context.Context, error) {
				verified, err := f.auth.Authenticate(ctx, request)
				require.NoError(t, err)
				_, mutationErr := pool.Exec(ctx, tc.mutation)
				require.NoError(t, mutationErr)
				return verified, nil
			})
			result := rpc(t, h, "test-one", "tools/call", memberCall(`{"action":"me"}`))
			require.True(t, result.IsError)
			require.NotEmpty(t, result.Content)
			assert.Contains(t, result.Content[0].Text, "["+tc.code+"]")
			assert.Empty(t, result.StructuredContent)
		})
	}
}

func databaseService(t *testing.T, pool *pgxpool.Pool, changeCycle func()) *mcpactions.Service {
	t.Helper()
	apps := databaseApplications(t, pool)
	apps.Cycles = changingCycleReader{apps.Cycles.(*cyclereads.Repository), changeCycle}
	service, err := mcpactions.New(apps)
	require.NoError(t, err)
	return service
}

func databaseApplications(t *testing.T, pool *pgxpool.Pool) mcpactions.Applications {
	t.Helper()
	repository, err := useridentity.NewRepository(pool)
	require.NoError(t, err)
	reader, err := useridentity.NewIdentityReader(repository)
	require.NoError(t, err)
	projectRepository, err := projectmembershipreads.NewLifecycleRepository(pool, rosterPublisher{}, "https://vector.example")
	require.NoError(t, err)
	projectReader, err := projectmembershipreads.NewMemberReader(projectRepository)
	require.NoError(t, err)
	directory, err := projectmembershipreads.NewDirectoryReader(pool)
	require.NoError(t, err)
	projectReads, err := workspaceprojectreads.NewRepository(pool)
	require.NoError(t, err)
	cycleReads, err := cyclereads.NewRepository(pool)
	require.NoError(t, err)
	moduleReads, err := modulereads.NewRepository(pool)
	require.NoError(t, err)
	labelReads, err := projectlabelreads.NewRepository(pool)
	require.NoError(t, err)
	collaborationReads, err := workitems.NewRepository(pool, rosterPublisher{}, "https://vector.example")
	require.NoError(t, err)
	activityReads, err := activityreads.NewRepository(pool)
	require.NoError(t, err)
	milestoneRepository, err := milestones.NewRepository(pool)
	require.NoError(t, err)
	return mcpactions.Applications{Identity: reader, ProjectMembers: projectReader, MemberDirectory: directory, Projects: projectReads, ProjectWrites: projectLifecycle(t, pool), Cycles: cycleReads, Modules: moduleReads, Labels: labelReads, Collaboration: collaborationReads, Activities: activityReads, Milestones: milestoneRepository}
}
