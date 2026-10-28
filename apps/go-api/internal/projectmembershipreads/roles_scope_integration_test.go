//go:build integration

package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestRolesExcludeUnavailableProjectsAndDeletedWorkspaces(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pair, err := postgresfixture.StartUncloned(ctx, seedRoleScope)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.Close(t, pair) }()
	db, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.CloseContext(t, db, context.WithoutCancel(ctx)) }()
	repository, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	actor := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	raw, err := repository.Roles(ctx, actor, "current")
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]int
	if err = json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]int{
		"00000000-0000-4000-8000-000000000101": 15,
		"00000000-0000-4000-8000-000000000102": 20,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roles=%v, want only live and archived project roles %v", got, want)
	}
	if _, err = db.Exec(ctx, `UPDATE workspaces SET deleted_at=now() WHERE slug='current'`); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.Roles(ctx, actor, "current"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("deleted workspace error=%v, want forbidden", err)
	}
}

func seedRoleScope(ctx context.Context, url string) error {
	db, err := pgx.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close(context.WithoutCancel(ctx)) }()
	_, err = db.Exec(ctx, `
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text,deleted_at timestamptz);
CREATE TABLE workspace_members(workspace_id uuid,member_id uuid,is_active boolean,deleted_at timestamptz);
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,deleted_at timestamptz,archived_at timestamptz);
CREATE TABLE project_members(project_id uuid,workspace_id uuid,member_id uuid,role integer,is_active boolean,deleted_at timestamptz);
INSERT INTO workspaces VALUES
 ('00000000-0000-4000-8000-000000000010','current',NULL),
 ('00000000-0000-4000-8000-000000000020','other',NULL);
INSERT INTO workspace_members VALUES
 ('00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000001',true,NULL);
INSERT INTO projects VALUES
 ('00000000-0000-4000-8000-000000000101','00000000-0000-4000-8000-000000000010',NULL,NULL),
 ('00000000-0000-4000-8000-000000000102','00000000-0000-4000-8000-000000000010',NULL,now()),
 ('00000000-0000-4000-8000-000000000103','00000000-0000-4000-8000-000000000010',now(),NULL),
 ('00000000-0000-4000-8000-000000000104','00000000-0000-4000-8000-000000000020',NULL,NULL),
 ('00000000-0000-4000-8000-000000000106','00000000-0000-4000-8000-000000000010',NULL,NULL),
 ('00000000-0000-4000-8000-000000000107','00000000-0000-4000-8000-000000000010',NULL,NULL);
INSERT INTO project_members VALUES
 ('00000000-0000-4000-8000-000000000101','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000001',15,true,NULL),
 ('00000000-0000-4000-8000-000000000102','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000001',20,true,NULL),
 ('00000000-0000-4000-8000-000000000103','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000001',20,true,NULL),
 ('00000000-0000-4000-8000-000000000104','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000001',20,true,NULL),
 ('00000000-0000-4000-8000-000000000105','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000001',20,true,NULL),
 ('00000000-0000-4000-8000-000000000106','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000001',20,false,NULL),
 ('00000000-0000-4000-8000-000000000107','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000001',20,true,now());
`)
	return err
}
