//go:build integration

package projectlabelreads

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"uuid"
)

func TestRepositoryMatchesProjectLabelReadQueries(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, archived_at timestamptz);
CREATE TABLE workspace_members (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, member_id uuid NOT NULL,
  is_active boolean NOT NULL, deleted_at timestamptz
);
CREATE TABLE project_members (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid NOT NULL,
  member_id uuid NOT NULL, is_active boolean NOT NULL, deleted_at timestamptz
);
CREATE TABLE labels (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid,
  parent_id uuid, name text NOT NULL, color text NOT NULL,
  sort_order double precision NOT NULL, deleted_at timestamptz
);
INSERT INTO workspaces (id, slug) VALUES
  ('00000000-0000-4000-8000-000000000001', 'demo');
INSERT INTO projects (id, workspace_id) VALUES
  ('00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000001');
INSERT INTO workspace_members (id, workspace_id, member_id, is_active) VALUES
  ('00000000-0000-4000-8000-000000000004', '00000000-0000-4000-8000-000000000001',
   '00000000-0000-4000-8000-000000000003', true);
INSERT INTO project_members (id, workspace_id, project_id, member_id, is_active) VALUES
  ('00000000-0000-4000-8000-000000000005', '00000000-0000-4000-8000-000000000001',
   '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000003', true);
INSERT INTO labels (id, workspace_id, project_id, parent_id, name, color, sort_order) VALUES
  ('00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000001',
   '00000000-0000-4000-8000-000000000002', NULL, 'Parent', '#FF0000', 200),
  ('00000000-0000-4000-8000-000000000011', '00000000-0000-4000-8000-000000000001',
   '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000010', 'Child', '#00FF00', 100);
`)
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
	repository, _ := NewRepository(connection)
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	projectID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	childID := uuid.MustParse("00000000-0000-4000-8000-000000000011")

	data, err := repository.List(ctx, userID, "demo", projectID)
	if err != nil {
		t.Fatal(err)
	}
	var labels []struct {
		Name   string     `json:"name"`
		Parent *uuid.UUID `json:"parent"`
	}
	if err := json.Unmarshal(data, &labels); err != nil {
		t.Fatal(err)
	}
	if len(labels) != 2 || labels[0].Name != "Child" || labels[0].Parent == nil || labels[1].Name != "Parent" {
		t.Fatalf("labels=%s", data)
	}
	if _, err := repository.Detail(ctx, userID, "demo", projectID, childID); err != nil {
		t.Fatal(err)
	}

	if _, err := connection.Exec(ctx, "UPDATE project_members SET is_active=false, deleted_at=NOW() WHERE member_id=$1", userID); err != nil {
		t.Fatal(err)
	}
	data, err = repository.List(ctx, userID, "demo", projectID)
	if err != nil || len(data) == 0 || string(data) == "[]" {
		t.Fatalf("inactive soft-deleted project membership list=%s err=%v", data, err)
	}
	if _, err := repository.Detail(ctx, userID, "demo", projectID, childID); err != nil {
		t.Fatalf("inactive soft-deleted project membership detail err=%v", err)
	}

	if _, err := connection.Exec(ctx, "UPDATE workspace_members SET is_active=false WHERE member_id=$1", userID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.List(ctx, userID, "demo", projectID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("inactive workspace membership err=%v", err)
	}
	if err := repository.AuthorizeWorkspace(ctx, uuid.New(), "demo"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-member authorization err=%v", err)
	}
}
