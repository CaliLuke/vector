//go:build integration

package workspaceuserinsights

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

const (
	insightsWorkspaceID = "00000000-0000-4000-8000-000000000010"
	insightsProject1ID  = "00000000-0000-4000-8000-000000000011"
	insightsProject2ID  = "00000000-0000-4000-8000-000000000012"
	insightsRequesterID = "00000000-0000-4000-8000-000000000001"
	insightsTargetID    = "00000000-0000-4000-8000-000000000002"
	insightsIssue1ID    = "00000000-0000-4000-8000-000000000021"
	insightsIssue2ID    = "00000000-0000-4000-8000-000000000022"
	insightsActivity1ID = "00000000-0000-4000-8000-000000000031"
	insightsActivity2ID = "00000000-0000-4000-8000-000000000032"
)

func startWorkspaceUserInsightsDB(t *testing.T) (*pgx.Conn, context.Context) {
	t.Helper()
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer testsupport.CloseContext(t, connection, context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (
  id uuid PRIMARY KEY, slug text NOT NULL, name text NOT NULL,
  logo_asset_id uuid, logo text, deleted_at timestamptz
);
CREATE TABLE workspace_members (
  workspace_id uuid NOT NULL, member_id uuid NOT NULL, role integer NOT NULL,
  is_active boolean NOT NULL, deleted_at timestamptz
);
CREATE TABLE projects (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, identifier text NOT NULL,
  name text NOT NULL, cover_image text, cover_image_asset_id uuid,
  description text, logo_props jsonb NOT NULL DEFAULT '{}'::jsonb,
  archived_at timestamptz, deleted_at timestamptz, guest_view_all_features boolean NOT NULL DEFAULT false
);
CREATE TABLE project_members (
  project_id uuid NOT NULL, member_id uuid NOT NULL, is_active boolean NOT NULL, workspace_id uuid, role integer NOT NULL DEFAULT 15, deleted_at timestamptz
);
CREATE TABLE users (
  id uuid PRIMARY KEY, is_active boolean NOT NULL DEFAULT true, email text, first_name text, last_name text, avatar text,
  avatar_asset_id uuid, is_bot boolean NOT NULL DEFAULT false, display_name text,
  cover_image text, cover_image_asset_id uuid, user_timezone text, date_joined timestamptz
);
CREATE TABLE issues (
  id uuid PRIMARY KEY, workspace_id uuid, project_id uuid, state_id uuid, created_by_id uuid, completed_at timestamptz,
  deleted_at timestamptz, archived_at timestamptz, name text, description_json jsonb, description_html text,
  priority text, start_date date, target_date date, sequence_id integer,
  sort_order numeric, is_draft boolean
);
CREATE TABLE states (id uuid PRIMARY KEY, "group" text);
CREATE TABLE issue_assignees (issue_id uuid NOT NULL, assignee_id uuid NOT NULL, deleted_at timestamptz);
CREATE TABLE file_assets (
  id uuid PRIMARY KEY, entity_type text NOT NULL, workspace_id uuid,
  project_id uuid, issue_id uuid
);
CREATE TABLE issue_activities (
  id uuid PRIMARY KEY, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
  created_by_id uuid, updated_by_id uuid, deleted_at timestamptz,
  workspace_id uuid NOT NULL, project_id uuid NOT NULL, issue_id uuid,
  verb text, field text, old_value text, new_value text, comment text,
  attachments jsonb, issue_comment_id uuid, actor_id uuid,
  old_identifier text, new_identifier text, epoch bigint
);
INSERT INTO workspaces(id,slug,name,logo) VALUES
  ('`+insightsWorkspaceID+`','demo','Demo','');
INSERT INTO workspace_members(workspace_id,member_id,role,is_active) VALUES
  ('`+insightsWorkspaceID+`','`+insightsRequesterID+`',5,true),
  ('`+insightsWorkspaceID+`','`+insightsTargetID+`',20,true);
INSERT INTO projects(id,workspace_id,identifier,name) VALUES
  ('`+insightsProject1ID+`','`+insightsWorkspaceID+`','ONE','First'),
  ('`+insightsProject2ID+`','`+insightsWorkspaceID+`','TWO','Second');
INSERT INTO project_members(project_id,member_id,is_active) VALUES
  ('`+insightsProject1ID+`','`+insightsRequesterID+`',true),
  ('`+insightsProject2ID+`','`+insightsRequesterID+`',true);
UPDATE project_members SET workspace_id='`+insightsWorkspaceID+`';
INSERT INTO users(id,email,first_name,last_name,avatar,is_bot,display_name,cover_image,user_timezone,date_joined) VALUES
  ('`+insightsTargetID+`','target@example.test','Target','User','',false,'Target User','', 'UTC','2026-01-01T00:00:00Z');
INSERT INTO issues(id,project_id,name,description_json,description_html,priority,sequence_id,sort_order,is_draft) VALUES
  ('`+insightsIssue1ID+`','`+insightsProject1ID+`','First issue','{}','', 'high',1,1,false),
  ('`+insightsIssue2ID+`','`+insightsProject2ID+`','Second issue','{}','', 'low',2,2,false);
UPDATE issues SET workspace_id='`+insightsWorkspaceID+`';
INSERT INTO issue_activities(id,created_at,updated_at,workspace_id,project_id,issue_id,verb,field,actor_id,attachments,epoch) VALUES
  ('`+insightsActivity1ID+`','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z','`+insightsWorkspaceID+`','`+insightsProject1ID+`','`+insightsIssue1ID+`','updated','state','`+insightsTargetID+`','[]',1),
  ('`+insightsActivity2ID+`','2026-01-02T00:00:00Z','2026-01-02T00:00:00Z','`+insightsWorkspaceID+`','`+insightsProject2ID+`','`+insightsIssue2ID+`','updated','priority','`+insightsTargetID+`','[]',2);
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Errorf("close PostgreSQL testcontainer: %v", closeErr)
		}
	})
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.CloseContext(t, connection, context.WithoutCancel(ctx)) })
	return connection, ctx
}

func TestWorkspaceAuthorizationAndActivityQueryUseSourceScopeAndCTEAliases(t *testing.T) {
	connection, ctx := startWorkspaceUserInsightsDB(t)
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	if authorizeWorkspaceErr := repository.AuthorizeWorkspace(ctx, insightsRequesterID, "demo", false); authorizeWorkspaceErr != nil {
		t.Fatalf("safe workspace member permission: %v", authorizeWorkspaceErr)
	}
	if err := repository.AuthorizeWorkspace(ctx, insightsRequesterID, "demo", true); err != ErrForbidden {
		t.Fatalf("viewer unsafe permission = %v, want ErrForbidden", err)
	}
	if authorizeWorkspaceErr := repository.AuthorizeWorkspace(ctx, insightsTargetID, "demo", true); authorizeWorkspaceErr != nil {
		t.Fatalf("Admin/Member unsafe permission: %v", authorizeWorkspaceErr)
	}
	allowed, err := repository.CanExportActivity(ctx, insightsTargetID, "demo")
	if err != nil || !allowed {
		t.Fatalf("export permission = (%v, %v), want allowed", allowed, err)
	}

	query := djangoapi.Query{
		Params:  map[string]string{"cursor": "10:0:0", "per_page": "10", "order_by": "-created_at"},
		Present: map[string]bool{"cursor": true, "per_page": true, "order_by": true},
	}
	body, err := repository.Activity(ctx, insightsRequesterID, "demo", insightsTargetID, []string{insightsProject2ID}, query)
	if err != nil {
		t.Fatalf("activity query: %v", err)
	}
	var response struct {
		Results []map[string]any `json:"results"`
	}
	if unmarshalErr := json.Unmarshal(body, &response); unmarshalErr != nil {
		t.Fatalf("decode activity JSON: %v", unmarshalErr)
	}
	if len(response.Results) != 1 || response.Results[0]["id"] != insightsActivity2ID {
		t.Fatalf("project-filtered activity result = %s", body)
	}
	if stamp, ok := response.Results[0]["created_at"].(string); !ok || stamp != "2026-01-02T00:00:00Z" {
		t.Fatalf("activity created_at = %#v", response.Results[0]["created_at"])
	}
	for _, field := range []string{"actor", "issue", "project", "workspace", "issue_comment"} {
		if _, ok := response.Results[0][field]; !ok {
			t.Errorf("activity omitted Django model relation field %q: %s", field, body)
		}
	}
	for _, field := range []string{"actor_id", "issue_id", "project_id", "workspace_id", "issue_comment_id"} {
		if _, ok := response.Results[0][field]; ok {
			t.Errorf("activity exposed database column field %q instead of Django relation field: %s", field, body)
		}
	}
	if !strings.Contains(string(body), `"sort_order" : 2.0`) {
		t.Errorf("activity issue sort_order should preserve the serializer's floating numeric representation: %s", body)
	}

	profile, err := repository.Profile(ctx, insightsRequesterID, "demo", insightsTargetID)
	if err != nil {
		t.Fatalf("profile query: %v", err)
	}
	var profileBody struct {
		UserData struct {
			DateJoined string `json:"date_joined"`
		} `json:"user_data"`
	}
	if unmarshalErr := json.Unmarshal(profile, &profileBody); unmarshalErr != nil {
		t.Fatalf("decode profile JSON: %v", unmarshalErr)
	}
	if profileBody.UserData.DateJoined != "2026-01-01T00:00:00Z" {
		t.Errorf("profile date_joined = %q, want DRF UTC Z format", profileBody.UserData.DateJoined)
	}
	invalidPage := djangoapi.Query{Params: map[string]string{"per_page": "0"}, Present: map[string]bool{"per_page": true}}
	_, err = repository.Activity(ctx, insightsRequesterID, "demo", insightsTargetID, nil, invalidPage)
	var httpError *djangoapi.HTTPError
	if !errors.As(err, &httpError) || httpError.Status != 400 {
		t.Fatalf("per_page=0 error = %#v, want source-approved HTTP 400", err)
	}
}
