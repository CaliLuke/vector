//go:build integration

package workitemtypes_test

import (
	"context"
	"strings"
	"testing"
	"time"
	"uuid"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

// Hold a schema change open while an intake edit starts. The edit must read
// after that change commits, including when its request only changes the title.
func TestPublicIntakeEditCannotUndoPropertyPreservation(t *testing.T) {
	f, _, item, field, intake, entry := seedIntakeWriter(t)
	connection, err := pgx.ConnectConfig(f.ctx, f.db.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.CloseContext(t, connection, context.WithoutCancel(f.ctx)) }()
	writer, err := workitems.NewRepository(connection, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.Rollback(t, tx, f.ctx) }()
	if _, err = tx.Exec(f.ctx, `SELECT work_item_schema_lock($1::uuid)`, f.workspace); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, failure := writer.PublicIntakeUpdate(f.ctx, workitems.PublicFeedbackScope{WorkspaceID: f.workspace, WorkspaceSlug: "types", ProjectID: f.project, IntakeID: &intake}, intake, entry, f.member, strings.NewReader(`{"issue":{"name":"After"}}`), "application/json")
		done <- failure
	}()
	waitForSchemaWriter(t, f, tx, int(connection.PgConn().PID()), done)
	// Model the description/value writes performed by the schema migration.
	preserved := "<p>Existing</p><h3>Risk removed</h3><p>7</p>"
	if _, err = tx.Exec(f.ctx, `UPDATE issues SET description_html=$2,custom_properties=custom_properties-$3 WHERE id=$1`, item, preserved, field); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-f.ctx.Done():
		t.Fatal(f.ctx.Err())
	}
	var name, description string
	if err = f.db.QueryRow(f.ctx, `SELECT name,description_html FROM issues WHERE id=$1`, item).Scan(&name, &description); err != nil {
		t.Fatal(err)
	}
	if name != "After" || description != preserved || len(draftValues(t, f, "issues", item)) != 0 {
		t.Fatalf("edit reverted preservation: %q %q", name, description)
	}
}

func waitForSchemaWriter(t *testing.T, f projectTypeFixture, tx pgx.Tx, pid int, done <-chan error) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for {
		var waiting bool
		if err := tx.QueryRow(f.ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE pid=$1 AND locktype='advisory' AND NOT granted)`, pid).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		select {
		case err := <-done:
			t.Fatalf("writer returned before schema commit: %v", err)
		case <-deadline.C:
			t.Fatal("writer did not reach schema lock")
		case <-ticker.C:
		}
	}
}

func seedIntakeWriter(t *testing.T) (projectTypeFixture, *workitems.Repository, string, string, string, string) {
	t.Helper()
	f := newProjectTypeFixture(t)
	field := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: field, Title: "Risk", Kind: property.Number, Configuration: property.Configuration{Active: true}})
	repo, _, _ := seedPublicationDraft(t, f)
	created, err := writeValueItem(f, repo, "POST", "", `{"name":"Before","custom_properties":{"`+field+`":7}}`)
	if err != nil {
		t.Fatal(err)
	}
	item := resultIssueID(t, created)
	intake, entry := uuid.New().String(), uuid.New().String()
	f.exec(`INSERT INTO intakes(id,name,description,is_default,view_props,logo_props,project_id,workspace_id,created_at,updated_at) VALUES($1,'Intake','',false,'{}','{}',$2,$3,now(),now())`, intake, f.project, f.workspace)
	f.exec(`INSERT INTO intake_issues(id,intake_id,issue_id,status,source,extra,project_id,workspace_id,created_at,updated_at,created_by_id) VALUES($1,$2,$3,-2,'IN_APP','{}',$4,$5,now(),now(),$6)`, entry, intake, item, f.project, f.workspace, f.member)
	return f, repo, item, field, intake, entry
}

func TestIntakeAcceptanceRetainsEditsFromSameRequest(t *testing.T) {
	f, repo, item, _, intake, _ := seedIntakeWriter(t)
	triage, backlog := uuid.New().String(), uuid.New().String()
	f.exec(`INSERT INTO states(id,name,description,color,slug,sequence,"group",is_triage,"default",project_id,workspace_id,created_at,updated_at) VALUES($1,'Triage','','#000000','triage',0,'triage',true,false,$3,$4,now(),now()),($2,'Backlog','','#000000','backlog',1,'backlog',false,true,$3,$4,now(),now())`, triage, backlog, f.project, f.workspace)
	f.exec(`UPDATE issues SET state_id=$2 WHERE id=$1`, item, triage)
	_, err := repo.Execute(f.ctx, workitems.Request{Operation: "api_workspaces_by_slug_projects_by_project_id_intake_issues", Method: "PATCH", ActorID: f.owner, Slug: "types", ProjectID: f.project, IssueID: item, Timezone: "UTC", Params: map[string]string{"intake_id": intake}}, strings.NewReader(`{"issue":{"name":"Accepted edit","description_html":"<p>Edited description</p>"},"status":1}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	var name, description, state string
	if err = f.db.QueryRow(f.ctx, `SELECT name,description_html,state_id::text FROM issues WHERE id=$1`, item).Scan(&name, &description, &state); err != nil {
		t.Fatal(err)
	}
	if name != "Accepted edit" || description != "<p>Edited description</p>" || state != backlog {
		t.Fatalf("acceptance lost edit: %q %q %s", name, description, state)
	}
}

func TestIntakeDeletionAdmitsCleanupOnlyWithCommit(t *testing.T) {
	f, repo, _, _, intake, entry := seedIntakeWriter(t)
	scope := workitems.PublicFeedbackScope{WorkspaceID: f.workspace, WorkspaceSlug: "types", ProjectID: f.project, IntakeID: &intake}
	f.exec(`CREATE FUNCTION reject_intake_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.deleted_at IS NOT NULL THEN RAISE EXCEPTION 'reject intake commit'; END IF; RETURN NEW; END $$; CREATE CONSTRAINT TRIGGER reject_intake_commit AFTER UPDATE ON intake_issues DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_intake_commit()`)
	_, err := repo.PublicIntakeDelete(f.ctx, scope, intake, entry, f.member)
	if err == nil {
		t.Fatal("forced commit failure succeeded")
	}
	var deleted bool
	var effects int
	if err = f.db.QueryRow(f.ctx, `SELECT deleted_at IS NOT NULL,(SELECT count(*) FROM work_item_effects WHERE kind='soft_delete') FROM intake_issues WHERE id=$1`, entry).Scan(&deleted, &effects); err != nil {
		t.Fatal(err)
	}
	if deleted || effects != 0 {
		t.Fatalf("rollback left deletion/effect: %v %d", deleted, effects)
	}
	f.exec(`DROP TRIGGER reject_intake_commit ON intake_issues`)
	if _, err = repo.PublicIntakeDelete(f.ctx, scope, intake, entry, f.member); err != nil {
		t.Fatal(err)
	}
	if err = f.db.QueryRow(f.ctx, `SELECT deleted_at IS NOT NULL,(SELECT count(*) FROM work_item_effects WHERE kind='soft_delete') FROM intake_issues WHERE id=$1`, entry).Scan(&deleted, &effects); err != nil {
		t.Fatal(err)
	}
	if !deleted || effects != 1 {
		t.Fatalf("commit omitted deletion/effect: %v %d", deleted, effects)
	}
}

func TestCrossProjectSubissueHistoryRetainsChildScopeAndPreviousParent(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, _, _ := seedPublicationDraft(t, f)
	created, err := writeValueItem(f, repo, "POST", "", `{"name":"Parent"}`)
	if err != nil {
		t.Fatal(err)
	}
	parent := resultIssueID(t, created)
	created, err = writeValueItem(f, repo, "POST", "", `{"name":"Former parent"}`)
	if err != nil {
		t.Fatal(err)
	}
	previous := resultIssueID(t, created)
	project := uuid.New().String()
	f.addProject(project)
	f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,20,true,now(),now(),'{}','{}','{}',0),(gen_random_uuid(),$1,$2,$4,15,true,now(),now(),'{}','{}','{}',0)`, f.workspace, project, f.owner, f.member)
	childFixture := f
	childFixture.project = project
	created, err = writeValueItem(childFixture, repo, "POST", "", `{"name":"Child"}`)
	if err != nil {
		t.Fatal(err)
	}
	child := resultIssueID(t, created)
	f.exec(`UPDATE issues SET parent_id=$2 WHERE id=$1`, child, previous)
	_, err = repo.Execute(f.ctx, workitems.Request{Operation: "api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_sub_issues", Method: "POST", ActorID: f.owner, Slug: "types", ProjectID: f.project, IssueID: parent, Timezone: "UTC"}, strings.NewReader(`{"sub_issue_ids":["`+child+`"]}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	var activityProject, current, requested string
	if err = f.db.QueryRow(f.ctx, `SELECT payload->>'project_id',payload->>'current_instance',payload->>'requested_data' FROM work_item_effects WHERE kind='activity' AND payload->>'issue_id'=$1 AND payload->>'type'='issue.activity.updated'`, child).Scan(&activityProject, &current, &requested); err != nil {
		t.Fatal(err)
	}
	if activityProject != project || !strings.Contains(current, previous) || !strings.Contains(requested, parent) {
		t.Fatalf("reparenting history has wrong scope/parent: %s %s %s", activityProject, current, requested)
	}
}
