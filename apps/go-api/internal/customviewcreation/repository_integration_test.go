//go:build integration

package customviewcreation

import (
	"context"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestRepositoryUpdatesMemberViewSettingsOnPostgreSQL(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text UNIQUE,deleted_at timestamptz);
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,deleted_at timestamptz);
CREATE TABLE project_members(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,member_id uuid,is_active boolean,view_props jsonb NOT NULL,default_props jsonb NOT NULL,preferences jsonb NOT NULL,sort_order double precision NOT NULL,updated_at timestamptz,updated_by_id uuid,deleted_at timestamptz);
CREATE TABLE workspace_members(id uuid PRIMARY KEY,workspace_id uuid,member_id uuid,is_active boolean,view_props jsonb NOT NULL,updated_at timestamptz,updated_by_id uuid,deleted_at timestamptz);
INSERT INTO workspaces VALUES ('00000000-0000-4000-8000-000000000010','demo',NULL);
INSERT INTO projects VALUES ('00000000-0000-4000-8000-000000000100','00000000-0000-4000-8000-000000000010',NULL);
INSERT INTO project_members VALUES ('00000000-0000-4000-8000-000000000200','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000100','00000000-0000-4000-8000-000000000001',true,'{}','{}','{}',0,NULL,NULL,NULL);
INSERT INTO workspace_members VALUES ('00000000-0000-4000-8000-000000000300','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000001',true,'{}',NULL,NULL,NULL);`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer pair.Close()
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(context.WithoutCancel(ctx))
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	user, project := "00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-000000000100"
	if err := repository.ProjectTarget(ctx, user, "demo", project); err != nil {
		t.Fatalf("project target: %v", err)
	}
	changes := ProjectViewChanges{ViewProps: map[string]any{"layout": "kanban"}, HasViewProps: true, SortOrder: 12.5, HasSortOrder: true}
	if err := repository.UpdateProject(ctx, user, "demo", project, changes); err != nil {
		t.Fatalf("update project member: %v", err)
	}
	if err := repository.WorkspaceTarget(ctx, user, "demo"); err != nil {
		t.Fatalf("workspace target: %v", err)
	}
	if err := repository.UpdateWorkspace(ctx, user, "demo", map[string]any{"layout": "list"}); err != nil {
		t.Fatalf("update workspace member: %v", err)
	}
	var projectProps, workspaceProps []byte
	var order float64
	if err := connection.QueryRow(ctx, `SELECT view_props,sort_order FROM project_members WHERE member_id=$1::uuid`, user).Scan(&projectProps, &order); err != nil {
		t.Fatal(err)
	}
	if string(projectProps) != `{"layout": "kanban"}` || order != 12.5 {
		t.Fatalf("project settings = %s / %v", projectProps, order)
	}
	if err := connection.QueryRow(ctx, `SELECT view_props FROM workspace_members WHERE member_id=$1::uuid`, user).Scan(&workspaceProps); err != nil {
		t.Fatal(err)
	}
	if string(workspaceProps) != `{"layout": "list"}` {
		t.Fatalf("workspace settings = %s", workspaceProps)
	}
}
