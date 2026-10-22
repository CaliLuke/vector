//go:build integration

package estimatereads

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"uuid"
)

func TestRepositoryPreservesEstimateReadSemantics(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, seedEstimateReads)
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
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	workspaceID := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	projectID := uuid.MustParse("00000000-0000-4000-8000-000000000020")
	selectedEstimateID := uuid.MustParse("00000000-0000-4000-8000-000000000030")
	unselectedEstimateID := uuid.MustParse("00000000-0000-4000-8000-000000000031")

	workspaceData, err := repository.WorkspaceSelected(ctx, userID, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	workspaceEstimates := decodeEstimates(t, workspaceData)
	if len(workspaceEstimates) != 2 || workspaceEstimates[0].Name != "Beta" || workspaceEstimates[1].Name != "Zulu" {
		t.Fatalf("selected workspace estimates=%s", workspaceData)
	}
	for _, estimate := range workspaceEstimates {
		if estimate.Name == "Alpha" {
			t.Fatalf("workspace response included unselected estimate: %s", workspaceData)
		}
	}

	projectData, err := repository.ProjectList(ctx, userID, "alpha", projectID)
	if err != nil {
		t.Fatal(err)
	}
	projectEstimates := decodeEstimates(t, projectData)
	if len(projectEstimates) != 2 || projectEstimates[0].Name != "Alpha" || projectEstimates[1].Name != "Zulu" {
		t.Fatalf("project estimates=%s", projectData)
	}
	zulu := projectEstimates[1]
	if zulu.ID != selectedEstimateID || zulu.Workspace != workspaceID || len(zulu.Points) != 2 {
		t.Fatalf("nested estimate=%+v data=%s", zulu, projectData)
	}
	if zulu.CreatedBy != nil || zulu.UpdatedBy != nil || zulu.DeletedAt != nil {
		t.Fatalf("nullable estimate audit fields=%+v", zulu)
	}
	if zulu.Points[0].Value != "1" || zulu.Points[1].Value != "2" {
		t.Fatalf("point ordering=%+v", zulu.Points)
	}
	for _, point := range zulu.Points {
		if point.CreatedBy != nil || point.UpdatedBy != nil || point.DeletedAt != nil {
			t.Fatalf("nullable audit fields=%+v", point)
		}
	}

	detailData, err := repository.ProjectDetail(ctx, userID, "alpha", projectID, unselectedEstimateID)
	if err != nil {
		t.Fatal(err)
	}
	detail := decodeEstimate(t, detailData)
	if detail.ID != unselectedEstimateID || detail.Name != "Alpha" {
		t.Fatalf("detail=%s", detailData)
	}

	secondProjectID := uuid.MustParse("00000000-0000-4000-8000-000000000021")
	if _, err := repository.ProjectList(ctx, userID, "alpha", secondProjectID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("project permission err=%v", err)
	}
	if _, err := repository.ProjectList(ctx, userID, "beta", projectID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("cross-tenant project err=%v", err)
	}
	if _, err := repository.WorkspaceSelected(ctx, userID, "beta"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("cross-tenant workspace err=%v", err)
	}
	if _, err := repository.ProjectDetail(ctx, userID, "alpha", projectID, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing detail err=%v", err)
	}
	secondEstimateID := uuid.MustParse("00000000-0000-4000-8000-000000000032")
	if _, err := repository.ProjectDetail(ctx, userID, "alpha", projectID, secondEstimateID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-project detail err=%v", err)
	}
}

type estimateWire struct {
	ID        uuid.UUID       `json:"id"`
	Workspace uuid.UUID       `json:"workspace"`
	Name      string          `json:"name"`
	CreatedBy *uuid.UUID      `json:"created_by"`
	UpdatedBy *uuid.UUID      `json:"updated_by"`
	DeletedAt *string         `json:"deleted_at"`
	Points    []estimatePoint `json:"points"`
}

type estimatePoint struct {
	Value     string     `json:"value"`
	CreatedBy *uuid.UUID `json:"created_by"`
	UpdatedBy *uuid.UUID `json:"updated_by"`
	DeletedAt *string    `json:"deleted_at"`
}

func decodeEstimates(t *testing.T, data json.RawMessage) []estimateWire {
	t.Helper()
	var estimates []estimateWire
	if err := json.Unmarshal(data, &estimates); err != nil {
		t.Fatalf("decode estimates %s: %v", data, err)
	}
	return estimates
}

func decodeEstimate(t *testing.T, data json.RawMessage) estimateWire {
	t.Helper()
	var estimate estimateWire
	if err := json.Unmarshal(data, &estimate); err != nil {
		t.Fatalf("decode estimate %s: %v", data, err)
	}
	return estimate
}

func seedEstimateReads(ctx context.Context, databaseURL string) error {
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer connection.Close(context.WithoutCancel(ctx))
	_, err = connection.Exec(ctx, `
CREATE TABLE users (id uuid PRIMARY KEY, user_timezone text NOT NULL);
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL, deleted_at timestamptz);
CREATE TABLE projects (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, estimate_id uuid, deleted_at timestamptz
);
CREATE TABLE workspace_members (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, member_id uuid NOT NULL,
  is_active boolean NOT NULL, deleted_at timestamptz
);
CREATE TABLE project_members (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid NOT NULL,
  member_id uuid NOT NULL, is_active boolean NOT NULL, deleted_at timestamptz
);
CREATE TABLE estimates (
  id uuid PRIMARY KEY, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
  created_by_id uuid, updated_by_id uuid, deleted_at timestamptz,
  workspace_id uuid NOT NULL, project_id uuid NOT NULL,
  name text NOT NULL, description text NOT NULL, type text NOT NULL, last_used boolean NOT NULL
);
CREATE TABLE estimate_points (
  id uuid PRIMARY KEY, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
  created_by_id uuid, updated_by_id uuid, deleted_at timestamptz,
  estimate_id uuid NOT NULL, workspace_id uuid NOT NULL, project_id uuid NOT NULL,
  key integer NOT NULL, value text NOT NULL, description text NOT NULL
);
INSERT INTO users (id, user_timezone) VALUES
  ('00000000-0000-4000-8000-000000000001', 'UTC');
INSERT INTO workspaces (id, slug) VALUES
  ('00000000-0000-4000-8000-000000000010', 'alpha'),
  ('00000000-0000-4000-8000-000000000011', 'beta');
INSERT INTO workspace_members (id, workspace_id, member_id, is_active) VALUES
  ('00000000-0000-4000-8000-000000000012', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000001', true);
INSERT INTO projects (id, workspace_id, estimate_id) VALUES
  ('00000000-0000-4000-8000-000000000020', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000030'),
  ('00000000-0000-4000-8000-000000000021', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000032'),
  ('00000000-0000-4000-8000-000000000022', '00000000-0000-4000-8000-000000000011', '00000000-0000-4000-8000-000000000033');
INSERT INTO project_members (id, workspace_id, project_id, member_id, is_active) VALUES
  ('00000000-0000-4000-8000-000000000023', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000020', '00000000-0000-4000-8000-000000000001', true),
  ('00000000-0000-4000-8000-000000000024', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000021', '00000000-0000-4000-8000-000000000001', false);
INSERT INTO estimates (
  id, created_at, updated_at, workspace_id, project_id, name, description, type, last_used
) VALUES
  ('00000000-0000-4000-8000-000000000030', '2026-01-02 03:04:05+00', '2026-01-02 03:04:05+00', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000020', 'Zulu', '', 'points', false),
  ('00000000-0000-4000-8000-000000000031', '2026-01-02 03:04:05+00', '2026-01-02 03:04:05+00', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000020', 'Alpha', '', 'categories', false),
  ('00000000-0000-4000-8000-000000000032', '2026-01-02 03:04:05+00', '2026-01-02 03:04:05+00', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000021', 'Beta', '', 'points', true),
  ('00000000-0000-4000-8000-000000000033', '2026-01-02 03:04:05+00', '2026-01-02 03:04:05+00', '00000000-0000-4000-8000-000000000011', '00000000-0000-4000-8000-000000000022', 'Other tenant', '', 'points', false);
INSERT INTO estimates (
  id, created_at, updated_at, workspace_id, project_id, name, description, type, last_used, deleted_at
) VALUES (
  '00000000-0000-4000-8000-000000000034', '2026-01-02 03:04:05+00', '2026-01-02 03:04:05+00',
  '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000020',
  'Deleted', '', 'points', false, NOW()
);
INSERT INTO estimate_points (
  id, created_at, updated_at, estimate_id, workspace_id, project_id, key, value, description, deleted_at
) VALUES
  ('00000000-0000-4000-8000-000000000040', '2026-01-02 03:04:05+00', '2026-01-02 03:04:05+00', '00000000-0000-4000-8000-000000000030', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000020', 2, '2', '', NULL),
  ('00000000-0000-4000-8000-000000000041', '2026-01-02 03:04:05+00', '2026-01-02 03:04:05+00', '00000000-0000-4000-8000-000000000030', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000020', 1, '1', '', NULL),
  ('00000000-0000-4000-8000-000000000042', '2026-01-02 03:04:05+00', '2026-01-02 03:04:05+00', '00000000-0000-4000-8000-000000000030', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000020', 0, '0', '', NOW());
`)
	return err
}
