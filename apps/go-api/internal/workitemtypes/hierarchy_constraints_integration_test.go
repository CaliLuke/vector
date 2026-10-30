//go:build integration

package workitemtypes_test

import (
	"errors"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func addHierarchyItem(t *testing.T, f projectTypeFixture, itemType, parent string) string {
	t.Helper()
	id := uuid.New().String()
	f.exec(`INSERT INTO issues(id,workspace_id,project_id,type_id,parent_id,name,description_json,description_html,priority,sort_order,sequence_id,is_draft,created_at,updated_at,created_by_id) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,'Hierarchy item','{}','','none',0,1,false,now(),now(),$6)`, id, f.workspace, f.project, itemType, parent, f.owner)
	return id
}
func requireHierarchyConstraint(t *testing.T, err error) {
	t.Helper()
	var failure *pgconn.PgError
	if !errors.As(err, &failure) || failure.Code != "23514" {
		t.Fatalf("expected hierarchy constraint rejection: %v", err)
	}
}
func hierarchyTransaction(t *testing.T, f projectTypeFixture) pgx.Tx {
	t.Helper()
	tx, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.Rollback(t, tx, f.ctx) })
	if _, err = tx.Exec(f.ctx, `SELECT work_item_schema_lock($1::uuid)`, f.workspace); err != nil {
		t.Fatal(err)
	}
	return tx
}
func configureHierarchy(t *testing.T, f projectTypeFixture) {
	t.Helper()
	input := hierarchyInput(t, f, 1)
	review, err := f.repo.PreviewHierarchy(f.ctx, f.owner, "types", input)
	if err != nil {
		t.Fatal(err)
	}
	if review.Links != 0 {
		t.Fatalf("fixture unexpectedly invalid: %+v", review)
	}
	input.ReviewID = review.ID
	if _, err = f.repo.CommitHierarchy(f.ctx, f.owner, "types", input); err != nil {
		t.Fatal(err)
	}
}
func TestHierarchyGuardsCyclesBeforeActivationAndDeletedItems(t *testing.T) {
	f := newProjectTypeFixture(t)
	first := addHierarchyItem(t, f, f.task, "")
	second := addHierarchyItem(t, f, f.task, first)
	third := addHierarchyItem(t, f, f.task, second)
	for _, parent := range []string{first, third} {
		_, err := f.db.Exec(f.ctx, `UPDATE issues SET parent_id=$2 WHERE id=$1`, first, parent)
		requireHierarchyConstraint(t, err)
	}
	f.exec(`UPDATE issues SET deleted_at=now() WHERE id=$1`, first)
	_, err := f.db.Exec(f.ctx, `UPDATE issues SET parent_id=$2 WHERE id=$1`, first, third)
	requireHierarchyConstraint(t, err)
	// Settings-only activation must not make existing same-level links invalid.
	_, err = f.db.Exec(f.ctx, `UPDATE work_item_type_settings SET hierarchy_enabled=true WHERE workspace_id=$1`, f.workspace)
	requireHierarchyConstraint(t, err)
}
func TestHierarchyDeferredValidationUsesFinalRows(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	parent := addHierarchyItem(t, f, f.bug, "")
	child := addHierarchyItem(t, f, f.task, parent)
	configureHierarchy(t, f)
	tx := hierarchyTransaction(t, f)
	if _, err := tx.Exec(f.ctx, `SET CONSTRAINTS work_item_hierarchy_guard,type_hierarchy_guard DEFERRED`); err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		sql  string
		args []any
	}{
		{`UPDATE issue_types SET level=1-level WHERE workspace_id=$1`, []any{f.workspace}},
		{`UPDATE issues SET parent_id=$2 WHERE id=$1`, []any{parent, child}},
		{`UPDATE issues SET parent_id=NULL WHERE id=$1`, []any{child}},
	} {
		if _, err := tx.Exec(f.ctx, step.sql, step.args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(f.ctx); err != nil {
		t.Fatalf("valid final graph rejected intermediate event: %v", err)
	}
	var parentID *string
	if err := f.db.QueryRow(f.ctx, `SELECT parent_id::text FROM issues WHERE id=$1`, parent).Scan(&parentID); err != nil || parentID == nil || *parentID != child {
		t.Fatalf("final graph lost: %v %v", parentID, err)
	}
}
func TestHierarchyDeferredUpdateThenDeleteDoesNotValidateOldEvent(t *testing.T) {
	f := newProjectTypeFixture(t)
	item := addHierarchyItem(t, f, f.task, "")
	tx := hierarchyTransaction(t, f)
	if _, err := tx.Exec(f.ctx, `SET CONSTRAINTS work_item_hierarchy_guard DEFERRED`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(f.ctx, `UPDATE issues SET parent_id=id WHERE id=$1`, item); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(f.ctx, `DELETE FROM issues WHERE id=$1`, item); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(f.ctx); err != nil {
		t.Fatalf("deleted deferred event rejected: %v", err)
	}
}
func TestHierarchyParentTypeChangeChecksPrivateDraftEvenInMigrationMode(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	_, draft, _ := seedPublicationDraft(t, f)
	parent := addHierarchyItem(t, f, f.bug, "")
	f.exec(`UPDATE draft_issues SET parent_id=$2,deleted_at=now() WHERE id=$1`, draft, parent)
	configureHierarchy(t, f)
	tx := hierarchyTransaction(t, f)
	if _, err := tx.Exec(f.ctx, `SET LOCAL vector.schema_migration='on'`); err != nil {
		t.Fatal(err)
	}
	_, err := tx.Exec(f.ctx, `UPDATE issues SET type_id=$2 WHERE id=$1`, parent, f.task)
	requireHierarchyConstraint(t, err)
}
func TestHierarchyDeletingUsedParentTypeCannotInvalidateChildren(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	parent := addHierarchyItem(t, f, f.bug, "")
	addHierarchyItem(t, f, f.task, parent)
	configureHierarchy(t, f)
	tx := hierarchyTransaction(t, f)
	_, err := tx.Exec(f.ctx, `DELETE FROM issue_types WHERE id=$1`, f.bug)
	requireHierarchyConstraint(t, err)
}
func TestHierarchyLateTypeInsertionRechecksTrashedItem(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	parent := addHierarchyItem(t, f, f.bug, "")
	configureHierarchy(t, f)
	tx := hierarchyTransaction(t, f)
	late := uuid.New().String()
	if _, err := tx.Exec(f.ctx, `CREATE TEMP TABLE late_type ON COMMIT DROP AS SELECT * FROM issue_types WHERE id=$1`, f.task); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(f.ctx, `UPDATE late_type SET id=$1,name='Late',is_default=false,level=1`, late); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(f.ctx, `INSERT INTO issues(id,workspace_id,project_id,type_id,parent_id,name,description_json,description_html,priority,sort_order,sequence_id,is_draft,created_at,updated_at,deleted_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,'Trash','{}','','none',0,1,false,now(),now(),now())`, f.workspace, f.project, late, parent); err != nil {
		t.Fatal(err)
	}
	_, err := tx.Exec(f.ctx, `INSERT INTO issue_types SELECT * FROM late_type`)
	requireHierarchyConstraint(t, err)
}

func TestHierarchyCannotDisableButWorkspaceCascadeDeletes(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	parent := addHierarchyItem(t, f, f.bug, "")
	addHierarchyItem(t, f, f.task, parent)
	configureHierarchy(t, f)
	for _, statement := range []string{`UPDATE work_item_type_settings SET hierarchy_enabled=false WHERE workspace_id=$1`, `DELETE FROM work_item_type_settings WHERE workspace_id=$1`} {
		_, err := f.db.Exec(f.ctx, statement, f.workspace)
		requireHierarchyConstraint(t, err)
	}
	if _, err := f.db.Exec(f.ctx, `DELETE FROM workspaces WHERE id=$1`, f.workspace); err != nil {
		t.Fatalf("configured workspace cascade failed: %v", err)
	}
	var count int
	if err := f.db.QueryRow(f.ctx, `SELECT count(*) FROM issues WHERE workspace_id=$1`, f.workspace).Scan(&count); err != nil || count != 0 {
		t.Fatalf("cascade retained items: %d %v", count, err)
	}
}
func TestHierarchyRejectsSnapshotWriteIsolation(t *testing.T) {
	f := newProjectTypeFixture(t)
	item := addHierarchyItem(t, f, f.task, "")
	tx, err := f.db.BeginTx(f.ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.Rollback(t, tx, f.ctx) }()
	_, err = tx.Exec(f.ctx, `UPDATE issues SET name='Snapshot write' WHERE id=$1`, item)
	var failure *pgconn.PgError
	if !errors.As(err, &failure) || failure.Code != "25000" {
		t.Fatalf("unsupported isolation accepted: %v", err)
	}
}
