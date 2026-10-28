//go:build integration

package cyclereads

import (
	"context"
	"strings"
	"testing"
	"uuid"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestCycleIssueQueriesBindRequestScopeWithoutDateFilters(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer func() { testsupport.CloseContext(t, connection, context.WithoutCancel(ctx)) }()
		_, err = connection.Exec(ctx, cycleIssueQuerySchema)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.Close(t, pair) })
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.CloseContext(t, connection, context.WithoutCancel(ctx)) })
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	projectID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	cycleID := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	deletedCycleID := uuid.MustParse("00000000-0000-4000-8000-000000000007")
	deletedModuleID := uuid.MustParse("00000000-0000-4000-8000-000000000008")
	activeModuleID := uuid.MustParse("00000000-0000-4000-8000-000000000009")
	deletedBridgeModuleID := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	if _, execErr := connection.Exec(ctx, `INSERT INTO cycles(id,project_id,workspace_id) VALUES($1,$2,'00000000-0000-4000-8000-000000000004')`, deletedCycleID, projectID); execErr != nil {
		t.Fatal(execErr)
	}
	if _, execErr := connection.Exec(ctx, `UPDATE cycles SET deleted_at=now() WHERE id=$1`, deletedCycleID); execErr != nil {
		t.Fatal(execErr)
	}
	if _, execErr := connection.Exec(ctx, `INSERT INTO cycle_issues(issue_id,cycle_id,created_at) VALUES('00000000-0000-4000-8000-000000000005',$1,now()+interval '1 second')`, deletedCycleID); execErr != nil {
		t.Fatal(execErr)
	}
	if _, execErr := connection.Exec(ctx, `INSERT INTO modules(id,project_id) VALUES($1,$2),($3,$2),($4,$2)`, activeModuleID, projectID, deletedModuleID, deletedBridgeModuleID); execErr != nil {
		t.Fatal(execErr)
	}
	if _, execErr := connection.Exec(ctx, `UPDATE modules SET deleted_at=now() WHERE id=$1`, deletedModuleID); execErr != nil {
		t.Fatal(execErr)
	}
	if _, execErr := connection.Exec(ctx, `INSERT INTO module_issues(issue_id,module_id,deleted_at) VALUES('00000000-0000-4000-8000-000000000005',$1,NULL),('00000000-0000-4000-8000-000000000005',$2,NULL),('00000000-0000-4000-8000-000000000005',$3,now())`, activeModuleID, deletedModuleID, deletedBridgeModuleID); execErr != nil {
		t.Fatal(execErr)
	}
	options := IssueListOptions{Filters: map[string]string{"priority": "urgent,high"}}
	result, err := repository.queryIssues(ctx, userID, "demo", projectID, cycleID, options, 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.total != 1 || len(result.rows) != 1 || result.rows[0]["name"] != "Urgent work" {
		t.Fatalf("filtered cycle issues = %#v", result)
	}
	if result.rows[0]["cycle_id"] != cycleID.String() {
		t.Errorf("cycle projection=%v, want live in-scope cycle %s", result.rows[0]["cycle_id"], cycleID)
	}
	moduleIDs, ok := result.rows[0]["module_ids"].([]any)
	if !ok || len(moduleIDs) != 1 || moduleIDs[0] != activeModuleID.String() {
		t.Errorf("module projection=%#v, want only live module %s", result.rows[0]["module_ids"], activeModuleID)
	}
	otherWorkspace, err := repository.queryIssues(ctx, userID, "other", projectID, cycleID, options, 30, 0)
	if err != nil || otherWorkspace.total != 0 || len(otherWorkspace.rows) != 0 {
		t.Fatalf("cross-workspace query = %#v, %v", otherWorkspace, err)
	}
	options.GroupBy = "priority"
	grouped, err := repository.groupedIssues(ctx, userID, "demo", projectID, cycleID, options, 30, 30, 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(grouped), `"Urgent work"`) || strings.Contains(string(grouped), `"Low priority work"`) {
		t.Fatalf("grouped cycle issues = %s", grouped)
	}
	if !strings.Contains(string(grouped), cycleID.String()) || strings.Contains(string(grouped), deletedCycleID.String()) || !strings.Contains(string(grouped), activeModuleID.String()) || strings.Contains(string(grouped), deletedModuleID.String()) || strings.Contains(string(grouped), deletedBridgeModuleID.String()) {
		t.Fatalf("grouped issue projection includes deleted/foreign relation targets: %s", grouped)
	}
}

const cycleIssueQuerySchema = `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text, deleted_at timestamptz);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid, archived_at timestamptz, deleted_at timestamptz, guest_view_all_features boolean DEFAULT false);
CREATE TABLE project_members (project_id uuid, workspace_id uuid, member_id uuid, role int, is_active boolean, deleted_at timestamptz);
CREATE TABLE workspace_members (workspace_id uuid, member_id uuid, is_active boolean, deleted_at timestamptz);
CREATE TABLE states (id uuid PRIMARY KEY, "group" text, project_id uuid, sequence int, is_triage boolean DEFAULT false, deleted_at timestamptz);
CREATE TABLE issues (
  type_id uuid,
 id uuid PRIMARY KEY, name text, state_id uuid, workspace_id uuid, project_id uuid, parent_id uuid,
 estimate_point_id uuid, priority text, sort_order double precision, sequence_id int,
 start_date date, target_date date, completed_at timestamptz, created_at timestamptz, updated_at timestamptz,
 created_by_id uuid, updated_by_id uuid, archived_at timestamptz, deleted_at timestamptz, is_draft boolean DEFAULT false
);
CREATE TABLE cycle_issues (issue_id uuid, cycle_id uuid, created_at timestamptz, deleted_at timestamptz);
CREATE TABLE intake_issues (issue_id uuid, status int);
CREATE TABLE file_assets (issue_id uuid, entity_type text, deleted_at timestamptz);
CREATE TABLE issue_links (issue_id uuid, deleted_at timestamptz);
CREATE TABLE issue_assignees (issue_id uuid, assignee_id uuid, deleted_at timestamptz);
CREATE TABLE labels(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,name text,created_at timestamptz DEFAULT now(),deleted_at timestamptz);
CREATE TABLE cycles(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,deleted_at timestamptz);
INSERT INTO cycles(id,project_id,workspace_id) VALUES('00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000004');
CREATE TABLE issue_labels (issue_id uuid, label_id uuid, deleted_at timestamptz);
CREATE TABLE module_issues (issue_id uuid, module_id uuid, deleted_at timestamptz);
CREATE TABLE modules (id uuid, project_id uuid, archived_at timestamptz, deleted_at timestamptz);
INSERT INTO workspaces(id,slug) VALUES ('00000000-0000-4000-8000-000000000004','demo');
INSERT INTO projects(id,workspace_id) VALUES ('00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000004');
INSERT INTO issues(id,name,workspace_id,project_id,priority,created_at,updated_at) VALUES
 ('00000000-0000-4000-8000-000000000005','Urgent work','00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000002','urgent',now(),now()),
 ('00000000-0000-4000-8000-000000000006','Low priority work','00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000002','low',now(),now());
INSERT INTO cycle_issues(issue_id,cycle_id,created_at) SELECT id,'00000000-0000-4000-8000-000000000003',now() FROM issues;
`
