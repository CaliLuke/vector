//go:build integration

package workspaceanalytics

import (
	"context"
	"encoding/json"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"uuid"
)

const (
	analyticsTestWorkspaceID = "00000000-0000-4000-8000-000000000101"
	analyticsTestProjectID   = "00000000-0000-4000-8000-000000000102"
	analyticsWorkspaceOnly   = "00000000-0000-4000-8000-000000000103"
	analyticsProjectMember   = "00000000-0000-4000-8000-000000000104"
	analyticsAdminFallback   = "00000000-0000-4000-8000-000000000105"
	analyticsProjectGuest    = "00000000-0000-4000-8000-000000000106"
)

func TestProjectAnalyticsAuthorizationMatchesPermissionBranches(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL);
CREATE TABLE workspace_members (
  workspace_id uuid NOT NULL, member_id uuid NOT NULL, role integer NOT NULL,
  is_active boolean NOT NULL, deleted_at timestamptz
);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid NOT NULL);
CREATE TABLE project_members (
  project_id uuid NOT NULL, member_id uuid NOT NULL, role integer NOT NULL,
  is_active boolean NOT NULL, deleted_at timestamptz
);
INSERT INTO workspaces(id,slug) VALUES ('`+analyticsTestWorkspaceID+`','demo');
INSERT INTO projects(id,workspace_id) VALUES ('`+analyticsTestProjectID+`','`+analyticsTestWorkspaceID+`');
-- A workspace member without project membership must not pass project auth.
INSERT INTO workspace_members(workspace_id,member_id,role,is_active) VALUES
  ('`+analyticsTestWorkspaceID+`','`+analyticsWorkspaceOnly+`',15,true),
  ('`+analyticsTestWorkspaceID+`','`+analyticsAdminFallback+`',20,true),
  ('`+analyticsTestWorkspaceID+`','`+analyticsProjectGuest+`',5,true);
-- Direct project membership is enough for ADMIN/MEMBER; workspace admin
-- fallback admits a project member of any role. A guest needs the chart route.
INSERT INTO project_members(project_id,member_id,role,is_active) VALUES
  ('`+analyticsTestProjectID+`','`+analyticsProjectMember+`',15,true),
  ('`+analyticsTestProjectID+`','`+analyticsAdminFallback+`',5,true),
  ('`+analyticsTestProjectID+`','`+analyticsProjectGuest+`',5,true);
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pair.Close(); err != nil {
			t.Errorf("close PostgreSQL testcontainer: %v", err)
		}
	})
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(ctx)) })
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	project := analyticsTestProjectID
	cases := []struct {
		name, user      string
		standard, chart bool
	}{
		{name: "workspace member without project membership", user: analyticsWorkspaceOnly},
		{name: "active project member role", user: analyticsProjectMember, standard: true, chart: true},
		{name: "workspace admin fallback for project member", user: analyticsAdminFallback, standard: true, chart: true},
		{name: "project guest is chart-only", user: analyticsProjectGuest, chart: true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			user := uuid.MustParse(test.user)
			allowed, err := repository.Authorized(ctx, user, "demo", project)
			if err != nil {
				t.Fatal(err)
			}
			if allowed != test.standard {
				t.Fatalf("standard project permission = %v, want %v", allowed, test.standard)
			}
			allowed, err = repository.AuthorizedProjectChart(ctx, user, "demo", project)
			if err != nil {
				t.Fatal(err)
			}
			if allowed != test.chart {
				t.Fatalf("project chart permission = %v, want %v", allowed, test.chart)
			}
		})
	}
	workspaceOnly := uuid.MustParse(analyticsWorkspaceOnly)
	allowed, err := repository.Authorized(ctx, workspaceOnly, "demo", "")
	if err != nil || !allowed {
		t.Fatalf("workspace ADMIN/MEMBER permission = (%v, %v), want allowed", allowed, err)
	}
}

func TestAnalyticsIssueManagerAndWorkspaceMemberScopes(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text NOT NULL,deleted_at timestamptz);
CREATE TABLE workspace_members(workspace_id uuid NOT NULL,member_id uuid NOT NULL,role integer NOT NULL,is_active boolean NOT NULL,deleted_at timestamptz,created_at timestamptz NOT NULL);
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid NOT NULL,deleted_at timestamptz,archived_at timestamptz,created_at timestamptz NOT NULL);
CREATE TABLE project_members(project_id uuid NOT NULL,member_id uuid NOT NULL,role integer NOT NULL,is_active boolean NOT NULL,deleted_at timestamptz,created_at timestamptz NOT NULL);
CREATE TABLE states(id uuid PRIMARY KEY,"group" text NOT NULL);
CREATE TABLE issues(id uuid PRIMARY KEY,workspace_id uuid NOT NULL,project_id uuid NOT NULL,deleted_at timestamptz,archived_at timestamptz,is_draft boolean,state_id uuid,created_at timestamptz NOT NULL);
CREATE TABLE users(id uuid PRIMARY KEY,is_bot boolean NOT NULL);
CREATE TABLE cycles(id uuid PRIMARY KEY,project_id uuid NOT NULL,deleted_at timestamptz,created_at timestamptz NOT NULL);
CREATE TABLE modules(id uuid PRIMARY KEY,project_id uuid NOT NULL,deleted_at timestamptz,created_at timestamptz NOT NULL);
CREATE TABLE intake_issues(issue_id uuid NOT NULL,status integer,deleted_at timestamptz,created_at timestamptz NOT NULL);
CREATE TABLE cycle_issues(issue_id uuid,cycle_id uuid,deleted_at timestamptz);
CREATE TABLE module_issues(issue_id uuid,module_id uuid,deleted_at timestamptz);
CREATE TABLE project_pages(project_id uuid NOT NULL,deleted_at timestamptz,created_at timestamptz NOT NULL);
CREATE TABLE issue_views(project_id uuid NOT NULL,deleted_at timestamptz,created_at timestamptz NOT NULL);
INSERT INTO users VALUES ('00000000-0000-4000-8000-000000000204',false),('00000000-0000-4000-8000-000000000205',false);
INSERT INTO workspaces(id,slug) VALUES ('00000000-0000-4000-8000-000000000201','demo');
INSERT INTO workspace_members(workspace_id,member_id,role,is_active,deleted_at,created_at) VALUES
 ('00000000-0000-4000-8000-000000000201','00000000-0000-4000-8000-000000000204',15,true,NULL,now()),
 ('00000000-0000-4000-8000-000000000201','00000000-0000-4000-8000-000000000205',15,true,NULL,now());
INSERT INTO projects(id,workspace_id,created_at) VALUES ('00000000-0000-4000-8000-000000000202','00000000-0000-4000-8000-000000000201',now());
INSERT INTO project_members(project_id,member_id,role,is_active,created_at) VALUES ('00000000-0000-4000-8000-000000000202','00000000-0000-4000-8000-000000000204',15,true,now());
INSERT INTO states(id,"group") VALUES
 ('00000000-0000-4000-8000-000000000206','started'),
 ('00000000-0000-4000-8000-000000000207','triage');
INSERT INTO issues(id,workspace_id,project_id,state_id,created_at) VALUES
 ('00000000-0000-4000-8000-000000000208','00000000-0000-4000-8000-000000000201','00000000-0000-4000-8000-000000000202','00000000-0000-4000-8000-000000000206',now()),
 ('00000000-0000-4000-8000-000000000209','00000000-0000-4000-8000-000000000201','00000000-0000-4000-8000-000000000202','00000000-0000-4000-8000-000000000207',now());
INSERT INTO intake_issues(issue_id,status,created_at) VALUES ('00000000-0000-4000-8000-000000000209',-2,now());
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pair.Close(); err != nil {
			t.Errorf("close PostgreSQL testcontainer: %v", err)
		}
	})
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(ctx)) })
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}

	user := uuid.MustParse("00000000-0000-4000-8000-000000000204")
	project := "00000000-0000-4000-8000-000000000202"
	stats, err := repository.WorkItemStats(ctx, user, "demo", project, "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	var counts map[string]struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(stats, &counts); err != nil {
		t.Fatal(err)
	}
	if got := counts["total_work_items"].Count; got != 1 {
		t.Errorf("total_work_items = %d, want only the non-triage issue", got)
	}

	overview, err := repository.Overview(ctx, user, "demo", project, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(overview, &counts); err != nil {
		t.Fatal(err)
	}
	if got := counts["total_intake"].Count; got != 1 {
		t.Errorf("total_intake = %d, want triage intake retained", got)
	}
	if got := counts["total_work_items"].Count; got != 1 {
		t.Errorf("overview total_work_items = %d, want triage issue excluded", got)
	}

	chart, err := repository.ProjectChart(ctx, user, "demo", project, "")
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Key   string `json:"key"`
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	var rows []struct {
		Key   string `json:"key"`
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	if err := json.Unmarshal(chart, &rows); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Key == "work_items" && row.Count != 1 {
			t.Errorf("work_items chart count = %d, want triage issue excluded", row.Count)
		}
		if row.Key == "members" {
			result = row
		}
	}
	if result.Key == "" || result.Count != 2 {
		t.Fatalf("members chart entry = %#v, want both active workspace members despite project_ids", result)
	}
}
