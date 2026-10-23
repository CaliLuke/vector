//go:build integration

package deployboardreads

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"uuid"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

const integrationUnsavedDeployBoardDefaults = `{"created_by":null,"deleted_at":null,"entity_identifier":null,"entity_name":"","intake":null,"is_activity_enabled":false,"is_comments_enabled":false,"is_disabled":false,"is_reactions_enabled":false,"is_votes_enabled":false,"updated_by":null,"view_props":null}`

func TestRepositoryPreservesCollectionShapeAndPreventsCrossProjectReads(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE users (id uuid PRIMARY KEY, user_timezone text);
CREATE TABLE file_assets (
  id uuid PRIMARY KEY, entity_type text NOT NULL, workspace_id uuid,
  project_id uuid, issue_id uuid
);
CREATE TABLE workspaces (
  id uuid PRIMARY KEY, name text NOT NULL, slug text NOT NULL,
  logo text, logo_asset_id uuid, deleted_at timestamptz
);
CREATE TABLE projects (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, identifier text NOT NULL,
  name text NOT NULL, cover_image text, cover_image_asset_id uuid,
  logo_props jsonb NOT NULL, description text NOT NULL, deleted_at timestamptz
);
CREATE TABLE project_members (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid NOT NULL,
  member_id uuid NOT NULL, is_active boolean NOT NULL, deleted_at timestamptz
);
CREATE TABLE deploy_boards (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid,
  entity_identifier uuid, entity_name text, anchor text NOT NULL,
  is_comments_enabled boolean NOT NULL, is_reactions_enabled boolean NOT NULL,
  intake_id uuid, is_votes_enabled boolean NOT NULL, view_props jsonb NOT NULL,
  is_activity_enabled boolean NOT NULL, is_disabled boolean NOT NULL,
  created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
  created_by_id uuid, updated_by_id uuid, deleted_at timestamptz
);
INSERT INTO users (id, user_timezone) VALUES
  ('00000000-0000-4000-8000-000000000001', 'UTC');
INSERT INTO file_assets (id, entity_type, workspace_id, project_id) VALUES
  ('00000000-0000-4000-8000-000000000020', 'WORKSPACE_LOGO', '00000000-0000-4000-8000-000000000010', NULL),
  ('00000000-0000-4000-8000-000000000021', 'PROJECT_COVER', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000100');
INSERT INTO workspaces (id, name, slug, logo, logo_asset_id) VALUES
  ('00000000-0000-4000-8000-000000000010', 'Demo Workspace', 'demo', 'https://unused/logo.png', '00000000-0000-4000-8000-000000000020'),
  ('00000000-0000-4000-8000-000000000011', 'Other Workspace', 'other', NULL, NULL);
INSERT INTO projects (id, workspace_id, identifier, name, cover_image, cover_image_asset_id, logo_props, description) VALUES
  ('00000000-0000-4000-8000-000000000100', '00000000-0000-4000-8000-000000000010', 'DEMO', 'Demo Project', 'https://unused/cover.png', '00000000-0000-4000-8000-000000000021', '{"emoji":"rocket"}', 'Primary project'),
  ('00000000-0000-4000-8000-000000000101', '00000000-0000-4000-8000-000000000010', 'EMPTY', 'Empty Project', NULL, NULL, '{}', ''),
  ('00000000-0000-4000-8000-000000000102', '00000000-0000-4000-8000-000000000010', 'SAME', 'Same Tenant Project', NULL, NULL, '{}', ''),
  ('00000000-0000-4000-8000-000000000103', '00000000-0000-4000-8000-000000000011', 'OTHER', 'Other Tenant Project', NULL, NULL, '{}', '');
INSERT INTO project_members (id, workspace_id, project_id, member_id, is_active) VALUES
  ('00000000-0000-4000-8000-000000000200', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000100', '00000000-0000-4000-8000-000000000001', true),
  ('00000000-0000-4000-8000-000000000201', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000101', '00000000-0000-4000-8000-000000000001', true),
  ('00000000-0000-4000-8000-000000000202', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000102', '00000000-0000-4000-8000-000000000001', true),
  ('00000000-0000-4000-8000-000000000203', '00000000-0000-4000-8000-000000000011', '00000000-0000-4000-8000-000000000103', '00000000-0000-4000-8000-000000000001', true);
INSERT INTO deploy_boards (
  id, workspace_id, project_id, entity_identifier, entity_name, anchor,
  is_comments_enabled, is_reactions_enabled, is_votes_enabled, view_props,
  is_activity_enabled, is_disabled, created_at, updated_at, deleted_at
) VALUES
  ('00000000-0000-4000-8000-000000000300', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000100', '00000000-0000-4000-8000-000000000100', 'project', 'demo-anchor', true, false, true, '{"views":["list"]}', true, false, '2026-01-02 03:04:05+00', '2026-01-02 03:04:06+00', NULL),
  ('00000000-0000-4000-8000-000000000301', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000100', '00000000-0000-4000-8000-000000000101', 'project', 'deleted-anchor', false, false, false, '{}', true, false, '2026-01-01+00', '2026-01-01+00', '2026-01-03+00'),
  ('00000000-0000-4000-8000-000000000302', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000102', '00000000-0000-4000-8000-000000000102', 'project', 'same-tenant-anchor', false, false, false, '{}', true, false, '2026-01-01+00', '2026-01-01+00', NULL),
  ('00000000-0000-4000-8000-000000000303', '00000000-0000-4000-8000-000000000011', '00000000-0000-4000-8000-000000000103', '00000000-0000-4000-8000-000000000103', 'project', 'other-tenant-anchor', false, false, false, '{}', true, false, '2026-01-01+00', '2026-01-01+00', NULL),
  ('00000000-0000-4000-8000-000000000304', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000100', '00000000-0000-4000-8000-000000000100', 'module', 'same-project-module-anchor', false, false, false, '{}', true, false, '2026-01-01+00', '2026-01-01+00', NULL),
  ('00000000-0000-4000-8000-000000000305', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000100', '00000000-0000-4000-8000-000000000101', 'project', 'mismatched-project-anchor', false, false, false, '{}', true, false, '2026-01-01+00', '2026-01-01+00', NULL);
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
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	projectID := uuid.MustParse("00000000-0000-4000-8000-000000000100")
	emptyProjectID := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	boardID := uuid.MustParse("00000000-0000-4000-8000-000000000300")
	deletedBoardID := uuid.MustParse("00000000-0000-4000-8000-000000000301")
	sameTenantBoardID := uuid.MustParse("00000000-0000-4000-8000-000000000302")
	otherTenantProjectID := uuid.MustParse("00000000-0000-4000-8000-000000000103")
	otherTenantBoardID := uuid.MustParse("00000000-0000-4000-8000-000000000303")
	sameProjectModuleBoardID := uuid.MustParse("00000000-0000-4000-8000-000000000304")

	body, err := repository.Collection(ctx, userID, "demo", projectID)
	if err != nil {
		t.Fatal(err)
	}
	var board map[string]any
	if err := json.Unmarshal(body, &board); err != nil {
		t.Fatal(err)
	}
	if board["id"] != boardID.String() || board["anchor"] != "demo-anchor" {
		t.Fatalf("collection board = %s", body)
	}
	projectDetails := board["project_details"].(map[string]any)
	workspaceDetails := board["workspace_detail"].(map[string]any)
	if got := projectDetails["cover_image_url"]; got != "/api/assets/v2/static/00000000-0000-4000-8000-000000000021/" {
		t.Errorf("cover_image_url = %v", got)
	}
	if got := workspaceDetails["logo_url"]; got != "/api/assets/v2/static/00000000-0000-4000-8000-000000000020/" {
		t.Errorf("logo_url = %v", got)
	}

	// Board 305 is live and has this entity identifier, but its project FK
	// points at project 100. The collection must not expose that project.
	unsavedDefaults, err := repository.Collection(ctx, userID, "demo", emptyProjectID)
	if err != nil {
		t.Fatal(err)
	}
	var gotDefaults, wantDefaults map[string]any
	if err := json.Unmarshal(unsavedDefaults, &gotDefaults); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(integrationUnsavedDeployBoardDefaults), &wantDefaults); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotDefaults, wantDefaults) {
		t.Fatalf("active project-FK mismatch returned %s, want serializer defaults %s", unsavedDefaults, integrationUnsavedDeployBoardDefaults)
	}
	if _, err := repository.Detail(ctx, userID, "demo", projectID, boardID); err != nil {
		t.Fatalf("valid detail: %v", err)
	}
	if _, err := repository.Detail(ctx, userID, "demo", projectID, sameProjectModuleBoardID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("same-project non-project deploy board err=%v", err)
	}
	if _, err := repository.Detail(ctx, userID, "demo", projectID, deletedBoardID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("soft-deleted detail err=%v", err)
	}
	if _, err := repository.Detail(ctx, userID, "demo", projectID, sameTenantBoardID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-project detail err=%v", err)
	}
	if _, err := repository.Detail(ctx, userID, "demo", projectID, otherTenantBoardID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-workspace detail err=%v", err)
	}
	if _, err := repository.Detail(ctx, userID, "demo", otherTenantProjectID, otherTenantBoardID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("mismatched workspace/project authorization err=%v", err)
	}
	if _, err := connection.Exec(ctx, "UPDATE projects SET deleted_at=NOW() WHERE id=$1", projectID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Collection(ctx, userID, "demo", projectID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("soft-deleted project authorization err=%v", err)
	}
	if _, err := connection.Exec(ctx, "UPDATE projects SET deleted_at=NULL WHERE id=$1", projectID); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, "UPDATE workspaces SET deleted_at=NOW() WHERE slug='demo'"); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Collection(ctx, userID, "demo", projectID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("soft-deleted workspace authorization err=%v", err)
	}
	if _, err := connection.Exec(ctx, "UPDATE workspaces SET deleted_at=NULL WHERE slug='demo'"); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, "UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2", projectID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Collection(ctx, userID, "demo", projectID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("inactive project membership err=%v", err)
	}
}
