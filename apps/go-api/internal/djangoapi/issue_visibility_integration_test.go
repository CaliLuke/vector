//go:build integration

package djangoapi

import (
	"context"
	"errors"
	"testing"
	"time"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// This small schema exercises the serializer's SQL contract independently of
// product defaults and graph mutation rules.
func visibilityDatabase(t *testing.T) (context.Context, *pgx.Conn, string, map[string]any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	pair, err := postgresfixture.StartUncloned(ctx, func(context.Context, string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pair.Close() })
	db, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close(context.WithoutCancel(ctx)) })
	_, err = db.Exec(ctx, `
 CREATE TABLE workspaces(id uuid PRIMARY KEY,deleted_at timestamptz);
 CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,deleted_at timestamptz,guest_view_all_features bool DEFAULT false);
 CREATE TABLE workspace_members(workspace_id uuid,member_id uuid,is_active bool,deleted_at timestamptz);
 CREATE TABLE project_members(project_id uuid,workspace_id uuid,member_id uuid,role int,is_active bool,deleted_at timestamptz);
 CREATE TABLE issues(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,created_by_id uuid,parent_id uuid,sequence_id int,name text,description_binary bytea,deleted_at timestamptz,archived_at timestamptz,state_id uuid,is_draft bool DEFAULT false);
 CREATE TABLE issue_assignees(issue_id uuid,assignee_id uuid,created_at timestamptz,deleted_at timestamptz);
 CREATE TABLE labels(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,deleted_at timestamptz);
 CREATE TABLE issue_labels(issue_id uuid,label_id uuid,created_at timestamptz,deleted_at timestamptz);
 CREATE TABLE states(id uuid,"group" text);`)
	if err != nil {
		t.Fatal(err)
	}
	workspace, project, actor, parent, child := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	if _, err = db.Exec(ctx, `INSERT INTO workspaces(id) VALUES($1);`, workspace); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(ctx, `INSERT INTO projects(id,workspace_id) VALUES($1,$2)`, project, workspace); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(ctx, `INSERT INTO workspace_members VALUES($1,$2,true,null)`, workspace, actor); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(ctx, `INSERT INTO project_members VALUES($1,$2,$3,15,true,null)`, project, workspace, actor); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(ctx, `INSERT INTO issues(id,workspace_id,project_id,created_by_id,parent_id,name,sequence_id) VALUES($1,$2,$3,$4,null,'Parent',1),($5,$2,$3,$4,$1,'Child',2)`, parent, workspace, project, actor, child); err != nil {
		t.Fatal(err)
	}
	return ctx, db, actor, map[string]any{"id": child, "workspace_id": workspace, "project_id": project, "parent_id": parent, "name": "Child"}
}

func TestVisibleSerializerPreservesSourceAndMasksRevokedRelationships(t *testing.T) {
	ctx, db, actor, row := visibilityDatabase(t)
	parent := row["parent_id"]
	shown, err := SerializeVisibleIssue(ctx, db, row, "name,parent", "parent", actor)
	if err != nil || shown["parent"].(map[string]any)["id"] != parent {
		t.Fatalf("visible parent: %#v %v", shown, err)
	}
	nested, err := SerializeVisible(ctx, db, map[string]any{"issue_id": row["id"]}, []string{"issue"}, "", "issue", false, actor)
	if err != nil || nested["issue"].(map[string]any)["parent"] != parent {
		t.Fatalf("nested parent: %#v %v", nested, err)
	}
	if _, err = db.Exec(ctx, `UPDATE project_members SET deleted_at=now()`); err != nil {
		t.Fatal(err)
	}
	masked, err := SerializeVisibleIssue(ctx, db, row, "name,parent", "", actor)
	if err != nil || masked["parent"] != nil || row["parent_id"] != parent || masked["name"] != "Child" {
		t.Fatalf("mask changed source or exposed parent: %#v %v", masked, err)
	}
	nested, err = SerializeVisible(ctx, db, map[string]any{"issue_id": row["id"]}, []string{"issue"}, "", "issue", false, actor)
	if err != nil || len(nested["issue"].(map[string]any)) != 0 {
		t.Fatalf("revoked nested issue: %#v %v", nested, err)
	}
	cancelled, stop := context.WithCancel(ctx)
	stop()
	if _, err = SerializeVisibleIssue(cancelled, db, row, "parent", "", actor); !errors.Is(err, context.Canceled) {
		t.Fatalf("failed visibility read was swallowed: %v", err)
	}
	if _, err = SerializeVisible(ctx, db, map[string]any{"issue_id": row["id"]}, []string{"issue"}, "", "issue", false, "not-a-uuid"); err == nil {
		t.Fatal("invalid visibility query succeeded")
	}
}

func TestVisibilityPredicatesRespectGuestAndPublicBoardScope(t *testing.T) {
	ctx, db, actor, row := visibilityDatabase(t)
	parent := row["parent_id"]
	if _, err := db.Exec(ctx, `UPDATE project_members SET role=5; UPDATE issues SET created_by_id=null WHERE parent_id IS NULL`); err != nil {
		t.Fatal(err)
	}
	masked, err := MaskIssueParent(ctx, db, row, actor)
	if err != nil || masked["parent_id"] != nil {
		t.Fatalf("guest saw another creator: %#v %v", masked, err)
	}
	if _, err = db.Exec(ctx, `UPDATE projects SET guest_view_all_features=true`); err != nil {
		t.Fatal(err)
	}
	masked, err = MaskIssueParent(ctx, db, row, actor)
	if err != nil || masked["parent_id"] != parent {
		t.Fatalf("guest-all parent missing: %#v %v", masked, err)
	}
	var visible, public *string
	query := `SELECT ` + VisibleParentSQL("i", "$2::uuid") + `::text,` + PublicParentSQL("i") + `::text FROM issues i WHERE i.id=$1`
	if err = db.QueryRow(ctx, query, row["id"], actor).Scan(&visible, &public); err != nil || visible == nil || public == nil {
		t.Fatalf("same-board parent: %v %v %v", visible, public, err)
	}
	if _, err = db.Exec(ctx, `UPDATE issues SET project_id=$2 WHERE id=$1`, parent, uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(ctx, query, row["id"], actor).Scan(&visible, &public); err != nil || visible != nil || public != nil {
		t.Fatalf("cross-board parent leaked: %v %v %v", visible, public, err)
	}
	withoutParent := map[string]any{"id": row["id"], "name": "Unlinked"}
	result, err := SerializeVisibleIssue(ctx, nil, withoutParent, "name", "", actor)
	if err != nil || result["name"] != "Unlinked" {
		t.Fatalf("unlinked item requires relationship I/O: %#v %v", result, err)
	}
}
