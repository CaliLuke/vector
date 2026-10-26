//go:build integration

package workitemtypes_test

import (
	"os"
	"testing"
	"uuid"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
)

func TestCreationDefaultQueuedWriteRechecksProjectAdministrator(t *testing.T) {
	f := newProjectTypeFixture(t)
	before := f.mutate(t, "import", f.bug)
	connection := hierarchyConnection(t, f)
	repository := types.NewRepository(connection)
	lock := hierarchyTransaction(t, f)
	done := make(chan error, 1)
	go func() {
		_, err := repository.MutateProject(f.ctx, f.owner, "types", f.project, types.ProjectInput{Action: "set_default", Revision: before.Revision, TypeIDs: []string{f.bug}})
		done <- err
	}()
	waitForSchemaWriter(t, f, lock, int(connection.PgConn().PID()), done)
	if _, err := lock.Exec(f.ctx, `UPDATE project_members SET role=15 WHERE project_id=$1 AND member_id=$2`, f.project, f.owner); err != nil {
		t.Fatal(err)
	}
	if err := lock.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		requireTypeStatus(t, err, 403)
	case <-f.ctx.Done():
		t.Fatal("queued project default did not finish")
	}
	if f.catalog(t).DefaultTypeID != "" {
		t.Fatal("demoted admin changed preference")
	}
}
func TestCreationDefaultMigrationPreservesOnlyUnambiguousLegacyPreference(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	down, err := os.ReadFile("../bootstrap/schema_migrations/000011_work_item_creation_defaults.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	f.exec(string(down))
	// The original Task import and Bug are conflicting legacy project defaults.
	f.exec(`UPDATE project_issue_types SET is_default=true WHERE project_id=$1`, f.project)
	single := uuid.New().String()
	f.addProject(single)
	f.exec(`UPDATE project_issue_types SET is_default=false WHERE project_id=$1`, single)
	f.exec(`INSERT INTO project_issue_types(id,workspace_id,project_id,issue_type_id,level,is_default,is_active,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,0,true,true,now(),now())`, f.workspace, single, f.bug)
	up, err := os.ReadFile("../bootstrap/schema_migrations/000011_work_item_creation_defaults.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	f.exec(string(up))
	var conflicts int
	var imported string
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM work_item_project_defaults WHERE project_id=$1`, f.project).Scan(&conflicts); err != nil || conflicts != 0 {
		t.Fatalf("ambiguous legacy choice imported: %d %v", conflicts, err)
	}
	if err = f.db.QueryRow(f.ctx, `SELECT type_id::text FROM work_item_project_defaults WHERE project_id=$1`, single).Scan(&imported); err != nil || imported != f.bug {
		t.Fatalf("unambiguous legacy choice lost: %s %v", imported, err)
	}
}
func TestCreationDefaultTenantAndPublishedNullConstraints(t *testing.T) {
	f := newProjectTypeFixture(t)
	foreign := uuid.New().String()
	f.exec(`INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Foreign',$2,'foreign','UTC','',now(),now())`, foreign, f.owner)
	_, err := f.db.Exec(f.ctx, `INSERT INTO work_item_project_defaults(project_id,workspace_id,type_id) VALUES($1,$2,$3)`, f.project, foreign, f.task)
	requireHierarchyConstraint(t, err)
	item := addHierarchyItem(t, f, f.task, "")
	_, err = f.db.Exec(f.ctx, `UPDATE issues SET type_id=NULL WHERE id=$1`, item)
	requireHierarchyConstraint(t, err)
	_, draft, _ := seedPublicationDraft(t, f)
	f.exec(`UPDATE draft_issues SET type_id=NULL WHERE id=$1`, draft)
	_, err = f.db.Exec(f.ctx, `UPDATE draft_issues SET custom_properties='{"unexpected":"value"}' WHERE id=$1`, draft)
	requireHierarchyConstraint(t, err)
}
