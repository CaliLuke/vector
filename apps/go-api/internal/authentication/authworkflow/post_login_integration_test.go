//go:build integration

package authworkflow

import (
	"context"
	"testing"
	"time"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

type cacheCall struct {
	path, user string
	multiple   bool
}

type cacheSpy struct{ calls []cacheCall }

func (spy *cacheSpy) Invalidate(_ context.Context, path, user string, multiple bool) error {
	spy.calls = append(spy.calls, cacheCall{path: path, user: user, multiple: multiple})
	return nil
}

type taskCall struct {
	name   string
	args   []any
	kwargs map[string]any
}

type taskSpy struct{ calls []taskCall }

func (spy *taskSpy) Task(_ context.Context, name string, args []any, kwargs map[string]any) error {
	spy.calls = append(spy.calls, taskCall{name: name, args: args, kwargs: kwargs})
	return nil
}

func TestPostUserAuthWorkflowJoinsAcceptedInvitationsAndPublishesWorkspaceEvents(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pair, err := postgresfixture.Start(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL);
CREATE TABLE workspace_member_invites (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, email text NOT NULL, accepted boolean NOT NULL, role smallint NOT NULL, created_at timestamptz NOT NULL, deleted_at timestamptz);
CREATE TABLE project_member_invites (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid NOT NULL, email text NOT NULL, accepted boolean NOT NULL, role smallint NOT NULL, created_by_id uuid, created_at timestamptz NOT NULL, deleted_at timestamptz);
CREATE TABLE workspace_members (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, member_id uuid NOT NULL, role smallint NOT NULL, created_by_id uuid, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL, deleted_at timestamptz);
CREATE UNIQUE INDEX workspace_member_active_unique ON workspace_members(workspace_id,member_id) WHERE deleted_at IS NULL;
CREATE TABLE project_members (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid NOT NULL, member_id uuid NOT NULL, role smallint NOT NULL, created_by_id uuid, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL, deleted_at timestamptz);
CREATE UNIQUE INDEX project_member_active_unique ON project_members(project_id,member_id) WHERE deleted_at IS NULL;
`)
		return err
	})
	if err != nil {
		t.Fatalf("start PostgreSQL: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Errorf("close PostgreSQL: %v", closeErr)
		}
	})
	connection, err := pgx.Connect(ctx, pair.SourceURL)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(context.WithoutCancel(ctx))
	const (
		userID      = "11111111-1111-1111-1111-111111111111"
		creatorID   = "22222222-2222-2222-2222-222222222222"
		workspaceID = "33333333-3333-3333-3333-333333333333"
		projectID   = "44444444-4444-4444-4444-444444444444"
	)
	seedStatements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO workspaces VALUES ($1,'workspace-one')`, []any{workspaceID}},
		{`INSERT INTO workspace_member_invites VALUES ('55555555-5555-5555-5555-555555555555',$1,'luca@example.test',true,5,CURRENT_TIMESTAMP,NULL)`, []any{workspaceID}},
		{`INSERT INTO workspace_member_invites VALUES ('66666666-6666-6666-6666-666666666666',$1,'luca@example.test',false,15,CURRENT_TIMESTAMP,NULL)`, []any{workspaceID}},
		{`INSERT INTO project_member_invites VALUES ('77777777-7777-7777-7777-777777777777',$1,$2,'luca@example.test',true,20,$3,CURRENT_TIMESTAMP,NULL)`, []any{workspaceID, projectID, creatorID}},
		{`INSERT INTO workspace_members VALUES ('88888888-8888-8888-8888-888888888888',$1,$2,5,NULL,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,NULL)`, []any{workspaceID, userID}},
	}
	for _, seed := range seedStatements {
		if _, execErr := connection.Exec(ctx, seed.query, seed.args...); execErr != nil {
			t.Fatalf("seed invitations: %v", execErr)
		}
	}
	cache, publisher := &cacheSpy{}, &taskSpy{}
	workflow, err := NewPostUserAuthWorkflow(connection, cache, publisher)
	if err != nil {
		t.Fatal(err)
	}
	if err := workflow.Run(ctx, userID, "luca@example.test"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	var workspaceMemberships, projectMemberships int
	if err := connection.QueryRow(ctx, `SELECT count(*) FROM workspace_members WHERE member_id=$1 AND deleted_at IS NULL`, userID).Scan(&workspaceMemberships); err != nil {
		t.Fatal(err)
	}
	if err := connection.QueryRow(ctx, `SELECT count(*) FROM project_members WHERE member_id=$1 AND deleted_at IS NULL`, userID).Scan(&projectMemberships); err != nil {
		t.Fatal(err)
	}
	if workspaceMemberships != 1 || projectMemberships != 1 {
		t.Fatalf("membership counts workspace=%d project=%d, want 1 each", workspaceMemberships, projectMemberships)
	}
	var workspaceRole, projectRole int
	if err := connection.QueryRow(ctx, `SELECT role FROM workspace_members WHERE member_id=$1`, userID).Scan(&workspaceRole); err != nil {
		t.Fatal(err)
	}
	if err := connection.QueryRow(ctx, `SELECT role FROM project_members WHERE member_id=$1`, userID).Scan(&projectRole); err != nil {
		t.Fatal(err)
	}
	if workspaceRole != 5 || projectRole != 15 {
		t.Fatalf("project invite roles workspace=%d project=%d, want existing workspace role 5 and coerced project role 15", workspaceRole, projectRole)
	}
	var workspaceInviteCount, projectInviteCount int
	if err := connection.QueryRow(ctx, `SELECT count(*) FROM workspace_member_invites WHERE deleted_at IS NULL AND accepted=true`).Scan(&workspaceInviteCount); err != nil {
		t.Fatal(err)
	}
	if err := connection.QueryRow(ctx, `SELECT count(*) FROM project_member_invites WHERE deleted_at IS NULL AND accepted=true`).Scan(&projectInviteCount); err != nil {
		t.Fatal(err)
	}
	if workspaceInviteCount != 0 || projectInviteCount != 0 {
		t.Fatalf("active invite counts workspace=%d project=%d, want 0", workspaceInviteCount, projectInviteCount)
	}
	if len(cache.calls) != 1 || cache.calls[0] != (cacheCall{path: "/api/workspaces/workspace-one/members/", multiple: true}) {
		t.Fatalf("cache invalidations = %#v", cache.calls)
	}
	if len(publisher.calls) != 1 || publisher.calls[0].name != joinedWorkspaceTask {
		t.Fatalf("published task calls = %#v", publisher.calls)
	}
	kwargs := publisher.calls[0].kwargs
	if kwargs["user_id"] != userID || kwargs["event_name"] != "user_joined_workspace" || kwargs["slug"] != "workspace-one" {
		t.Fatalf("tracking task kwargs = %#v", kwargs)
	}
	properties, ok := kwargs["event_properties"].(map[string]any)
	if !ok || properties["user_id"] != userID || properties["workspace_id"] != workspaceID || properties["role"] != 5 {
		t.Fatalf("tracking event properties = %#v", kwargs["event_properties"])
	}
}
