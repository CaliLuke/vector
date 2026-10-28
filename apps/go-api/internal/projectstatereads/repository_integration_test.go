//go:build integration

package projectstatereads

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"uuid"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestRepositoryMatchesProjectStateReadQueries(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer testsupport.CloseContext(t, connection, context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (
  id uuid PRIMARY KEY,
  slug text NOT NULL
);
CREATE TABLE projects (
  id uuid PRIMARY KEY,
  workspace_id uuid NOT NULL,
  archived_at timestamptz
);
CREATE TABLE project_members (
  id uuid PRIMARY KEY,
  workspace_id uuid NOT NULL,
  project_id uuid NOT NULL,
  member_id uuid NOT NULL,
  role integer NOT NULL,
  is_active boolean NOT NULL,
  deleted_at timestamptz
);
CREATE TABLE workspace_members (
  id uuid PRIMARY KEY,
  workspace_id uuid NOT NULL,
  member_id uuid NOT NULL,
  role integer NOT NULL,
  is_active boolean NOT NULL,
  deleted_at timestamptz
);
CREATE TABLE states (
  id uuid PRIMARY KEY,
  workspace_id uuid NOT NULL,
  project_id uuid NOT NULL,
  name text NOT NULL,
  color text NOT NULL,
  "group" text NOT NULL,
  "default" boolean NOT NULL,
  description text NOT NULL,
  sequence double precision NOT NULL,
  is_triage boolean NOT NULL,
  deleted_at timestamptz
);
INSERT INTO workspaces (id, slug) VALUES
  ('00000000-0000-4000-8000-000000000001', 'demo');
INSERT INTO projects (id, workspace_id) VALUES
  ('00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000001');
INSERT INTO project_members (id, workspace_id, project_id, member_id, role, is_active) VALUES
  ('00000000-0000-4000-8000-000000000004', '00000000-0000-4000-8000-000000000001',
   '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000003', 15, true);
INSERT INTO states (
  id, workspace_id, project_id, name, color, "group", "default", description, sequence, is_triage
) VALUES
  ('00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000001',
   '00000000-0000-4000-8000-000000000002', 'Backlog A', '#60646C', 'backlog', true, '', 100, false),
  ('00000000-0000-4000-8000-000000000011', '00000000-0000-4000-8000-000000000001',
   '00000000-0000-4000-8000-000000000002', 'Backlog B', '#60646C', 'backlog', false, '', 200, false),
  ('00000000-0000-4000-8000-000000000012', '00000000-0000-4000-8000-000000000001',
   '00000000-0000-4000-8000-000000000002', 'Triage', '#4E5355', 'triage', false, '', 300, true);
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, pair)

	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.CloseContext(t, connection, context.WithoutCancel(ctx))
	repository, _ := NewRepository(connection)
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	projectID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	stateID := uuid.MustParse("00000000-0000-4000-8000-000000000010")

	data, err := repository.List(ctx, userID, "demo", projectID)
	if err != nil {
		t.Fatal(err)
	}
	var states []struct {
		Name  string  `json:"name"`
		Order float64 `json:"order"`
	}
	if unmarshalErr := json.Unmarshal(data, &states); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	if len(states) != 2 || states[0].Name != "Backlog A" || states[0].Order != 0.5 || states[1].Order != 1 {
		t.Fatalf("list=%s", data)
	}
	if _, detailErr := repository.Detail(ctx, userID, "demo", projectID, stateID); detailErr != nil {
		t.Fatal(detailErr)
	}
	intake, err := repository.Intake(ctx, userID, "demo", projectID)
	if err != nil || !json.Valid(intake) {
		t.Fatalf("intake=%s err=%v", intake, err)
	}

	if _, execErr := connection.Exec(ctx, "UPDATE project_members SET deleted_at=NOW() WHERE member_id=$1", userID); execErr != nil {
		t.Fatal(execErr)
	}
	if _, detailErr := repository.Detail(ctx, userID, "demo", projectID, stateID); detailErr != nil {
		t.Fatalf("active soft-deleted membership detail err=%v", detailErr)
	}
	if _, listErr := repository.List(ctx, userID, "demo", projectID); !errors.Is(listErr, ErrForbidden) {
		t.Fatalf("active soft-deleted membership list err=%v", listErr)
	}
	if _, execErr := connection.Exec(ctx, "UPDATE project_members SET deleted_at=NULL WHERE member_id=$1", userID); execErr != nil {
		t.Fatal(execErr)
	}

	if _, execErr := connection.Exec(ctx, "UPDATE projects SET archived_at=NOW() WHERE id=$1", projectID); execErr != nil {
		t.Fatal(execErr)
	}
	data, err = repository.List(ctx, userID, "demo", projectID)
	if err != nil || string(data) != "[]" {
		t.Fatalf("archived list=%s err=%v", data, err)
	}
	if _, err := repository.Detail(ctx, userID, "demo", projectID, stateID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("archived detail err=%v", err)
	}
	if _, err := repository.Intake(ctx, userID, "demo", projectID); err != nil {
		t.Fatalf("archived intake err=%v", err)
	}
	if err := repository.AuthorizeProject(ctx, uuid.New(), "demo", projectID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-member authorization err=%v", err)
	}
}
