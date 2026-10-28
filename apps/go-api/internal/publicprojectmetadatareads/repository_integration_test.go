//go:build integration

package publicprojectmetadatareads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"uuid"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestRepositoryMatchesPublicProjectMetadataQueries(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, name text NOT NULL, slug text NOT NULL, logo text, logo_asset_id uuid);
CREATE TABLE projects (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL,
  identifier text NOT NULL, name text NOT NULL, cover_image text,
  cover_image_asset_id uuid, logo_props jsonb, icon_prop jsonb, emoji text,
  description text NOT NULL, deleted_at timestamptz
);
CREATE TABLE deploy_boards (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid,
  entity_identifier uuid, entity_name text,
  anchor text NOT NULL, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
  deleted_at timestamptz, created_by_id uuid, updated_by_id uuid, intake_id uuid,
  is_comments_enabled boolean NOT NULL, is_reactions_enabled boolean NOT NULL,
  is_votes_enabled boolean NOT NULL, view_props jsonb NOT NULL,
  is_activity_enabled boolean NOT NULL, is_disabled boolean NOT NULL
);
CREATE TABLE file_assets (
  id uuid PRIMARY KEY, entity_type text, workspace_id uuid, project_id uuid, issue_id uuid
);
CREATE TABLE cycles (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid NOT NULL,
  name text NOT NULL, created_at timestamptz NOT NULL, deleted_at timestamptz
);
CREATE TABLE labels (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid,
  parent_id uuid, name text NOT NULL, color text NOT NULL,
  created_at timestamptz NOT NULL, deleted_at timestamptz
);
CREATE TABLE modules (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid NOT NULL,
  name text NOT NULL, created_at timestamptz NOT NULL, deleted_at timestamptz
);
CREATE TABLE users (
  id uuid PRIMARY KEY, display_name text NOT NULL, avatar text NOT NULL, user_timezone text NOT NULL
);
CREATE TABLE project_members (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid NOT NULL,
  member_id uuid, is_active boolean NOT NULL,
  created_at timestamptz NOT NULL, deleted_at timestamptz
);
CREATE TABLE states (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid NOT NULL,
  name text NOT NULL, "group" text NOT NULL, color text NOT NULL,
  sequence double precision NOT NULL, deleted_at timestamptz
);
CREATE TABLE sessions (session_key varchar(128) PRIMARY KEY);
INSERT INTO workspaces VALUES ('00000000-0000-4000-8000-000000000001', 'Demo Workspace', 'demo', 'https://example.com/logo.png', NULL);
INSERT INTO projects VALUES (
  '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000001',
  'DEMO', 'Public <Project>', 'https://example.com/cover.png', NULL, '{"name":"hexagon","color":"#654321"}',
  '{"name":"rocket","color":"#123456"}', '🚀', 'Metadata & details', NULL
);
INSERT INTO deploy_boards VALUES (
  '00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000001',
  '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000002', 'project',
  'anchor', '2026-01-02 03:04:05.123456+00', '2026-01-02 03:04:05.123456+00', NULL, NULL,
  '00000000-0000-4000-8000-000000000050', NULL, TRUE, TRUE, TRUE, '{"list":true,"kanban":false}', FALSE, TRUE
), (
  '00000000-0000-4000-8000-000000000004', '00000000-0000-4000-8000-000000000001',
  NULL, '00000000-0000-4000-8000-000000000001', 'workspace', 'workspace-anchor', NOW(), NOW(), NULL,
  NULL, NULL, NULL, FALSE, FALSE, FALSE, '{}', TRUE, FALSE
), (
  '00000000-0000-4000-8000-000000000005', '00000000-0000-4000-8000-000000000001',
  '00000000-0000-4000-8000-000000000099', '00000000-0000-4000-8000-000000000099', 'project',
  'orphan-anchor', NOW(), NOW(), NULL, NULL, NULL, NULL, FALSE, FALSE, FALSE, '{}', TRUE, FALSE
), (
  '00000000-0000-4000-8000-000000000006', '00000000-0000-4000-8000-000000000001',
  NULL, '00000000-0000-4000-8000-000000000098', 'project',
  'null-project-anchor', NOW(), NOW(), NULL, NULL, NULL, NULL, FALSE, FALSE, FALSE, '{}', TRUE, FALSE
);
INSERT INTO labels VALUES
  ('00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', NULL, 'Parent', '#FF0000', NOW() - INTERVAL '1 second', NULL),
  ('00000000-0000-4000-8000-000000000011', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000010', 'Child', '#00FF00', NOW(), NULL),
  ('00000000-0000-4000-8000-000000000012', '00000000-0000-4000-8000-000000000001', NULL, NULL, 'Workspace', '#0000FF', NOW(), NULL);
INSERT INTO cycles VALUES
  ('00000000-0000-4000-8000-000000000030', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Cycle One', NOW() - INTERVAL '1 second', NULL),
  ('00000000-0000-4000-8000-000000000031', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Cycle Two', NOW(), NULL),
  ('00000000-0000-4000-8000-000000000032', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Deleted Cycle', NOW() + INTERVAL '1 second', NOW());
INSERT INTO modules VALUES
  ('00000000-0000-4000-8000-000000000040', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Module One', NOW() - INTERVAL '1 second', NULL),
  ('00000000-0000-4000-8000-000000000041', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Module Two', NOW(), NULL),
  ('00000000-0000-4000-8000-000000000042', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Deleted Module', NOW() + INTERVAL '1 second', NOW());
INSERT INTO users VALUES
  ('00000000-0000-4000-8000-000000000050', 'Owner', 'https://example.com/owner.png', 'America/Los_Angeles'),
  ('00000000-0000-4000-8000-000000000051', 'Member', '', 'UTC');
INSERT INTO project_members VALUES
  ('00000000-0000-4000-8000-000000000060', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000050', TRUE, NOW() - INTERVAL '2 seconds', NULL),
  ('00000000-0000-4000-8000-000000000061', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000051', TRUE, NOW() - INTERVAL '1 second', NULL),
  ('00000000-0000-4000-8000-000000000062', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', NULL, TRUE, NOW(), NULL),
  ('00000000-0000-4000-8000-000000000063', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000051', FALSE, NOW() + INTERVAL '1 second', NULL),
  ('00000000-0000-4000-8000-000000000064', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000051', TRUE, NOW() + INTERVAL '2 seconds', NOW());
INSERT INTO states VALUES
  ('00000000-0000-4000-8000-000000000027', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Signed Zero', 'backlog', '#000000', '-0'::float8, NULL),
  ('00000000-0000-4000-8000-000000000020', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Backlog', 'backlog', '#60646C', 100, NULL),
  ('00000000-0000-4000-8000-000000000021', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Triage', 'backlog', '#4E5355', 200, NULL),
  ('00000000-0000-4000-8000-000000000022', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Inbox', 'triage', '#4E5355', 300, NULL),
  ('00000000-0000-4000-8000-000000000024', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Large Fixed', 'completed', '#00AA01', 1e15, NULL),
  ('00000000-0000-4000-8000-000000000025', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Large Fraction Fixed', 'completed', '#00AA02', 1.234e15, NULL),
  ('00000000-0000-4000-8000-000000000026', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Exponent Boundary', 'completed', '#00AA03', 1e16, NULL),
  ('00000000-0000-4000-8000-000000000023', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Far Future', 'completed', '#00AA00', 1e20, NULL);
INSERT INTO sessions VALUES ('invalid-auth-hash-session');
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
	metaData, err := repository.Meta(ctx, "anchor")
	if err != nil {
		t.Fatal(err)
	}
	var meta struct {
		Identifier  string         `json:"identifier"`
		Name        string         `json:"name"`
		CoverImage  string         `json:"cover_image"`
		IconProp    map[string]any `json:"icon_prop"`
		Emoji       string         `json:"emoji"`
		Description string         `json:"description"`
	}
	if unmarshalErr := json.Unmarshal(metaData, &meta); unmarshalErr != nil || meta.Identifier != "DEMO" || meta.Name != "Public <Project>" || meta.IconProp["name"] != "rocket" || meta.Emoji != "🚀" {
		t.Fatalf("meta=%s err=%v", metaData, unmarshalErr)
	}
	if _, metaErr := repository.Meta(ctx, "orphan-anchor"); !errors.Is(metaErr, ErrProjectNotPublished) {
		t.Fatalf("orphan meta err=%v", metaErr)
	}
	settingsData, err := repository.Settings(ctx, "anchor")
	if err != nil {
		t.Fatal(err)
	}
	wantSettings := `{"id":"00000000-0000-4000-8000-000000000003","project_details":{"id":"00000000-0000-4000-8000-000000000002","identifier":"DEMO","name":"Public <Project>","cover_image":"https://example.com/cover.png","cover_image_url":"https://example.com/cover.png","logo_props":{"name":"hexagon","color":"#654321"},"description":"Metadata & details"},"workspace_detail":{"name":"Demo Workspace","slug":"demo","id":"00000000-0000-4000-8000-000000000001","logo_url":"https://example.com/logo.png"},"created_at":"2026-01-02T03:04:05.123456Z","updated_at":"2026-01-02T03:04:05.123456Z","deleted_at":null,"entity_identifier":"00000000-0000-4000-8000-000000000002","entity_name":"project","anchor":"anchor","is_comments_enabled":true,"is_reactions_enabled":true,"is_votes_enabled":true,"view_props":{"list":true,"kanban":false},"is_activity_enabled":false,"is_disabled":true,"created_by":null,"updated_by":"00000000-0000-4000-8000-000000000050","workspace":"00000000-0000-4000-8000-000000000001","project":"00000000-0000-4000-8000-000000000002","intake":null}`
	if string(settingsData) != wantSettings {
		t.Fatalf("settings=%s\nwant=%s", settingsData, wantSettings)
	}
	authenticatedSettings, err := repository.Settings(
		WithUserID(ctx, "00000000-0000-4000-8000-000000000050"), "anchor",
	)
	if err != nil || !bytes.Contains(authenticatedSettings, []byte(`"created_at":"2026-01-01T19:04:05.123456-08:00"`)) ||
		!bytes.Contains(authenticatedSettings, []byte(`"updated_at":"2026-01-01T19:04:05.123456-08:00"`)) {
		t.Fatalf("authenticated settings=%s err=%v", authenticatedSettings, err)
	}
	nullProjectSettings, err := repository.Settings(ctx, "null-project-anchor")
	if err != nil || !bytes.Contains(nullProjectSettings, []byte(`"project_details":null`)) || !bytes.Contains(nullProjectSettings, []byte(`"project":null`)) {
		t.Fatalf("null project settings=%s err=%v", nullProjectSettings, err)
	}
	if _, settingsErr := repository.Settings(ctx, "missing"); !errors.Is(settingsErr, ErrRequiredObject) {
		t.Fatalf("missing settings err=%v", settingsErr)
	}
	projectID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	projectAnchor, err := repository.ProjectAnchor(ctx, "demo", projectID)
	if err != nil || string(projectAnchor) != wantSettings {
		t.Fatalf("project anchor=%s err=%v", projectAnchor, err)
	}
	authenticatedProjectAnchor, err := repository.ProjectAnchor(
		WithUserID(ctx, "00000000-0000-4000-8000-000000000050"), "demo", projectID,
	)
	if err != nil || !bytes.Contains(authenticatedProjectAnchor, []byte(`"created_at":"2026-01-01T19:04:05.123456-08:00"`)) ||
		!bytes.Contains(authenticatedProjectAnchor, []byte(`"updated_at":"2026-01-01T19:04:05.123456-08:00"`)) {
		t.Fatalf("authenticated project anchor=%s err=%v", authenticatedProjectAnchor, err)
	}
	if _, execErr := connection.Exec(ctx, `
INSERT INTO deploy_boards VALUES (
  '00000000-0000-4000-8000-000000000007', '00000000-0000-4000-8000-000000000001',
  '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000099', 'project',
  'duplicate-project-anchor', NOW(), NOW(), NULL, NULL, NULL, NULL,
  FALSE, FALSE, FALSE, '{}', TRUE, FALSE
)
`); execErr != nil {
		t.Fatal(execErr)
	}
	if _, projectAnchorErr := repository.ProjectAnchor(ctx, "demo", projectID); projectAnchorErr == nil || errors.Is(projectAnchorErr, ErrRender) {
		t.Fatalf("duplicate project anchor err=%v", projectAnchorErr)
	}
	if _, projectAnchorErr := repository.ProjectAnchor(ctx, "wrong", projectID); !errors.Is(projectAnchorErr, ErrRequiredObject) {
		t.Fatalf("missing project anchor err=%v", projectAnchorErr)
	}
	if _, projectAnchorErr := repository.ProjectAnchor(ctx, "demo", uuid.MustParse("00000000-0000-4000-8000-000000000097")); !errors.Is(projectAnchorErr, ErrRequiredObject) {
		t.Fatalf("wrong-project anchor err=%v", projectAnchorErr)
	}

	cyclesData, err := repository.Cycles(ctx, "anchor")
	if err != nil {
		t.Fatal(err)
	}
	var cycles []struct {
		Name string `json:"name"`
	}
	if unmarshalErr := json.Unmarshal(cyclesData, &cycles); unmarshalErr != nil || len(cycles) != 2 || cycles[0].Name != "Cycle Two" {
		t.Fatalf("cycles=%s err=%v", cyclesData, unmarshalErr)
	}

	labelsData, err := repository.Labels(ctx, "anchor")
	if err != nil {
		t.Fatal(err)
	}
	var labels []struct {
		Name string `json:"name"`
	}
	if unmarshalErr := json.Unmarshal(labelsData, &labels); unmarshalErr != nil || len(labels) != 2 || labels[0].Name != "Child" {
		t.Fatalf("labels=%s err=%v", labelsData, unmarshalErr)
	}
	modulesData, err := repository.Modules(ctx, "anchor")
	if err != nil {
		t.Fatal(err)
	}
	var modules []struct {
		Name string `json:"name"`
	}
	if unmarshalErr := json.Unmarshal(modulesData, &modules); unmarshalErr != nil || len(modules) != 2 || modules[0].Name != "Module Two" {
		t.Fatalf("modules=%s err=%v", modulesData, unmarshalErr)
	}
	membersData, err := repository.Members(ctx, "anchor")
	if err != nil {
		t.Fatal(err)
	}
	var members []struct {
		Member      *string `json:"member"`
		DisplayName *string `json:"member__display_name"`
		Avatar      *string `json:"member__avatar"`
	}
	if unmarshalErr := json.Unmarshal(membersData, &members); unmarshalErr != nil || len(members) != 3 || members[0].Member != nil || members[0].DisplayName != nil || members[0].Avatar != nil {
		t.Fatalf("members=%s err=%v", membersData, unmarshalErr)
	}
	if members[1].DisplayName == nil || *members[1].DisplayName != "Member" || members[1].Avatar == nil || *members[1].Avatar != "" {
		t.Fatalf("members=%s", membersData)
	}
	statesData, err := repository.States(ctx, "anchor")
	if err != nil {
		t.Fatal(err)
	}
	var states []struct {
		Name string `json:"name"`
	}
	if unmarshalErr := json.Unmarshal(statesData, &states); unmarshalErr != nil || len(states) != 6 || states[0].Name != "Signed Zero" || states[5].Name != "Far Future" {
		t.Fatalf("states=%s err=%v", statesData, unmarshalErr)
	}
	if !bytes.Contains(statesData, []byte(`"sequence":-0.0`)) {
		t.Fatalf("states do not preserve Django signed-zero JSON: %s", statesData)
	}
	if !bytes.Contains(statesData, []byte(`"sequence":100.0`)) {
		t.Fatalf("states do not preserve Django whole-float JSON: %s", statesData)
	}
	if !bytes.Contains(statesData, []byte(`"sequence":1000000000000000.0`)) ||
		!bytes.Contains(statesData, []byte(`"sequence":1234000000000000.0`)) {
		t.Fatalf("states do not preserve Django fixed-float boundary JSON: %s", statesData)
	}
	if !bytes.Contains(statesData, []byte(`"sequence":1e+16`)) ||
		!bytes.Contains(statesData, []byte(`"sequence":1e+20`)) {
		t.Fatalf("states do not preserve Django exponent-float JSON: %s", statesData)
	}
	workspaceLabelsData, err := repository.Labels(ctx, "workspace-anchor")
	if err != nil {
		t.Fatal(err)
	}
	var workspaceLabels []struct {
		Name string `json:"name"`
	}
	if unmarshalErr := json.Unmarshal(workspaceLabelsData, &workspaceLabels); unmarshalErr != nil || len(workspaceLabels) != 1 || workspaceLabels[0].Name != "Workspace" {
		t.Fatalf("workspace labels=%s err=%v", workspaceLabelsData, unmarshalErr)
	}
	if _, labelsErr := repository.Labels(ctx, "missing"); !errors.Is(labelsErr, ErrInvalidAnchor) {
		t.Fatalf("missing anchor err=%v", labelsErr)
	}
	invalidator, err := NewPostgreSQLSessionInvalidator(connection)
	if err != nil {
		t.Fatal(err)
	}
	if err := invalidator.InvalidateSession(ctx, "invalid-auth-hash-session"); err != nil {
		t.Fatal(err)
	}
	var sessions int
	if err := connection.QueryRow(ctx, "SELECT count(*) FROM sessions").Scan(&sessions); err != nil || sessions != 0 {
		t.Fatalf("sessions=%d err=%v", sessions, err)
	}
}
