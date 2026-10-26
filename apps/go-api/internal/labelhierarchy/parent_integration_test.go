//go:build integration

package labelhierarchy

import (
	"context"
	"errors"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

const (
	labelWorkspaceID     = "00000000-0000-4000-8000-000000000001"
	labelProjectID       = "10000000-0000-4000-8000-000000000001"
	labelActorID         = "20000000-0000-4000-8000-000000000001"
	labelRootID          = "30000000-0000-4000-8000-000000000001"
	labelChildID         = "30000000-0000-4000-8000-000000000002"
	labelGrandchildID    = "30000000-0000-4000-8000-000000000003"
	labelOtherRootID     = "30000000-0000-4000-8000-000000000004"
	labelForeignChildID  = "30000000-0000-4000-8000-000000000005"
	labelDeletedParentID = "30000000-0000-4000-8000-000000000006"
	labelMissingParentID = "30000000-0000-4000-8000-000000000099"
)

const labelHierarchySchema = `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid NOT NULL);
CREATE TABLE project_members (project_id uuid NOT NULL, member_id uuid NOT NULL);
CREATE TABLE labels (
 id uuid PRIMARY KEY,
 parent_id uuid,
 project_id uuid NOT NULL,
 workspace_id uuid NOT NULL,
 deleted_at timestamptz,
 updated_at timestamptz,
 updated_by_id uuid
);
CREATE TABLE work_item_effects (id uuid PRIMARY KEY, kind text NOT NULL, payload jsonb NOT NULL);
INSERT INTO workspaces VALUES
 ('00000000-0000-4000-8000-000000000001', 'demo'),
 ('00000000-0000-4000-8000-000000000002', 'other');
INSERT INTO projects VALUES
 ('10000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000001'),
 ('10000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000002');
INSERT INTO project_members VALUES
 ('10000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000001');
INSERT INTO labels (id,parent_id,project_id,workspace_id,deleted_at) VALUES
 ('30000000-0000-4000-8000-000000000001',NULL,'10000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000001',NULL),
 ('30000000-0000-4000-8000-000000000002','30000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000001',NULL),
 ('30000000-0000-4000-8000-000000000003','30000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000001',NULL),
 ('30000000-0000-4000-8000-000000000004',NULL,'10000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000001',NULL),
 ('30000000-0000-4000-8000-000000000005',NULL,'10000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000002',NULL),
 ('30000000-0000-4000-8000-000000000006',NULL,'10000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000001',now());
`

func labelHierarchyFixture(t *testing.T) (context.Context, *pgx.Conn) {
	t.Helper()
	return postgresfixture.OpenSQL(t, labelHierarchySchema)
}

func TestLockProjectAndValidateParentEnforceScopeAndAncestry(t *testing.T) {
	ctx, db := labelHierarchyFixture(t)
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	workspaceID, err := LockProject(ctx, tx, labelProjectID, "demo")
	if err != nil {
		t.Fatalf("lock project: %v", err)
	}
	if workspaceID != labelWorkspaceID {
		t.Fatalf("locked workspace = %q, want %q", workspaceID, labelWorkspaceID)
	}
	if _, err = LockProject(ctx, tx, labelProjectID, "other"); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("lock project with mismatched workspace slug = %v, want no rows", err)
	}

	for _, test := range []struct {
		name     string
		labelID  string
		parentID string
		want     bool
	}{
		{name: "same-scope parent", labelID: labelRootID, parentID: labelOtherRootID, want: true},
		{name: "descendant parent", labelID: labelRootID, parentID: labelGrandchildID},
		{name: "self parent", labelID: labelRootID, parentID: labelRootID},
		{name: "cross-project parent", labelID: labelRootID, parentID: labelForeignChildID},
		{name: "deleted parent", labelID: labelRootID, parentID: labelDeletedParentID},
		{name: "missing parent", labelID: labelRootID, parentID: labelMissingParentID},
	} {
		t.Run(test.name, func(t *testing.T) {
			valid, validateErr := ValidateParent(ctx, tx, test.labelID, test.parentID, labelProjectID, workspaceID)
			if validateErr != nil {
				t.Fatalf("validate parent: %v", validateErr)
			}
			if valid != test.want {
				t.Fatalf("valid parent = %v, want %v", valid, test.want)
			}
		})
	}
}

func TestSoftDeleteTreeDeletesScopedDescendantsAndQueuesOneEffectPerLabel(t *testing.T) {
	ctx, db := labelHierarchyFixture(t)
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	if _, err = LockProject(ctx, tx, labelProjectID, "demo"); err != nil {
		t.Fatal(err)
	}
	deleted, err := SoftDeleteTree(ctx, tx, labelRootID, labelActorID, labelProjectID, labelWorkspaceID)
	if err != nil {
		t.Fatalf("soft delete tree: %v", err)
	}
	if !deleted {
		t.Fatal("soft delete tree reported no change")
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	var deletedCount, effectCount int
	if err = db.QueryRow(ctx, `SELECT count(*) FROM labels WHERE id=ANY($1::uuid[]) AND deleted_at IS NOT NULL`, []string{labelRootID, labelChildID, labelGrandchildID}).Scan(&deletedCount); err != nil {
		t.Fatal(err)
	}
	if deletedCount != 3 {
		t.Fatalf("deleted descendants = %d, want 3", deletedCount)
	}
	if err = db.QueryRow(ctx, `SELECT count(*) FROM work_item_effects WHERE kind='soft_delete'`).Scan(&effectCount); err != nil {
		t.Fatal(err)
	}
	if effectCount != 3 {
		t.Fatalf("durable cleanup effects = %d, want 3", effectCount)
	}
	for _, labelID := range []string{labelRootID, labelChildID, labelGrandchildID} {
		var effects int
		if err = db.QueryRow(ctx, `SELECT count(*) FROM work_item_effects
 WHERE kind='soft_delete' AND payload->>'app'='db' AND payload->>'model'='label' AND payload->>'id'=$1`, labelID).Scan(&effects); err != nil {
			t.Fatal(err)
		}
		if effects != 1 {
			t.Fatalf("cleanup effects for label %s = %d, want 1", labelID, effects)
		}
	}
}

func TestSoftDeleteTreeRejectsOutOfScopeChildWithoutChangingRows(t *testing.T) {
	ctx, db := labelHierarchyFixture(t)
	if _, err := db.Exec(ctx, `UPDATE labels SET parent_id=$1 WHERE id=$2`, labelRootID, labelForeignChildID); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	if _, err = LockProject(ctx, tx, labelProjectID, "demo"); err != nil {
		t.Fatal(err)
	}
	deleted, err := SoftDeleteTree(ctx, tx, labelRootID, labelActorID, labelProjectID, labelWorkspaceID)
	if !errors.Is(err, ErrOutOfScopeChild) {
		t.Fatalf("soft delete error = %v, want ErrOutOfScopeChild", err)
	}
	if deleted {
		t.Fatal("rejected tree deletion reported a change")
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	var deletedRows, effects int
	if err = db.QueryRow(ctx, `SELECT count(*) FROM labels WHERE id=ANY($1::uuid[]) AND deleted_at IS NOT NULL`, []string{labelRootID, labelChildID, labelGrandchildID}).Scan(&deletedRows); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(ctx, `SELECT count(*) FROM work_item_effects`).Scan(&effects); err != nil {
		t.Fatal(err)
	}
	if deletedRows != 0 || effects != 0 {
		t.Fatalf("rejected delete changed rows=%d effects=%d; want both zero", deletedRows, effects)
	}
}

func TestSoftDeleteTreeRequiresMembershipAndRollsBackWithCallerTransaction(t *testing.T) {
	t.Run("non-member", func(t *testing.T) {
		ctx, db := labelHierarchyFixture(t)
		if _, err := db.Exec(ctx, `DELETE FROM project_members WHERE project_id=$1 AND member_id=$2`, labelProjectID, labelActorID); err != nil {
			t.Fatal(err)
		}
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
		if _, err = LockProject(ctx, tx, labelProjectID, "demo"); err != nil {
			t.Fatal(err)
		}
		deleted, deleteErr := SoftDeleteTree(ctx, tx, labelRootID, labelActorID, labelProjectID, labelWorkspaceID)
		if deleteErr != nil {
			t.Fatal(deleteErr)
		}
		if deleted {
			t.Fatal("non-member deleted the label tree")
		}
		if err = tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		var effects int
		if err = db.QueryRow(ctx, `SELECT count(*) FROM work_item_effects`).Scan(&effects); err != nil {
			t.Fatal(err)
		}
		if effects != 0 {
			t.Fatalf("non-member queued %d effects, want zero", effects)
		}
	})

	t.Run("caller rollback", func(t *testing.T) {
		ctx, db := labelHierarchyFixture(t)
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
		if _, err = LockProject(ctx, tx, labelProjectID, "demo"); err != nil {
			t.Fatal(err)
		}
		deleted, deleteErr := SoftDeleteTree(ctx, tx, labelRootID, labelActorID, labelProjectID, labelWorkspaceID)
		if deleteErr != nil || !deleted {
			t.Fatalf("soft delete before rollback = (%v, %v), want (true, nil)", deleted, deleteErr)
		}
		if err = tx.Rollback(ctx); err != nil {
			t.Fatal(err)
		}
		var deletedRows, effects int
		if err = db.QueryRow(ctx, `SELECT count(*) FROM labels WHERE id=ANY($1::uuid[]) AND deleted_at IS NOT NULL`, []string{labelRootID, labelChildID, labelGrandchildID}).Scan(&deletedRows); err != nil {
			t.Fatal(err)
		}
		if err = db.QueryRow(ctx, `SELECT count(*) FROM work_item_effects`).Scan(&effects); err != nil {
			t.Fatal(err)
		}
		if deletedRows != 0 || effects != 0 {
			t.Fatalf("caller rollback left deleted rows=%d effects=%d; want both zero", deletedRows, effects)
		}
	})
}
