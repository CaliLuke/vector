//go:build integration

package workitems

import (
	"context"
	"encoding/json"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestIssueProjectionOmitsDeletedRelationTargets(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer func() { _ = connection.Close(context.WithoutCancel(ctx)) }()
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, deleted_at timestamptz);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid, deleted_at timestamptz, archived_at timestamptz, guest_view_all_features boolean DEFAULT false);
CREATE TABLE project_members (project_id uuid, workspace_id uuid, member_id uuid, role integer, is_active boolean, deleted_at timestamptz);
CREATE TABLE workspace_members (workspace_id uuid, member_id uuid, is_active boolean, deleted_at timestamptz);
CREATE TABLE issues (id uuid PRIMARY KEY, workspace_id uuid, project_id uuid, created_by_id uuid, parent_id uuid, state_id uuid, archived_at timestamptz, deleted_at timestamptz, is_draft boolean DEFAULT false);
CREATE TABLE states (id uuid, "group" text);
CREATE TABLE cycles (id uuid PRIMARY KEY, project_id uuid, workspace_id uuid, deleted_at timestamptz);
CREATE TABLE cycle_issues (issue_id uuid, cycle_id uuid, deleted_at timestamptz, created_at timestamptz);
CREATE TABLE module_issues (issue_id uuid, module_id uuid, deleted_at timestamptz);
CREATE TABLE modules (id uuid, deleted_at timestamptz, archived_at timestamptz);
CREATE TABLE issue_labels (issue_id uuid, label_id uuid, deleted_at timestamptz);
CREATE TABLE issue_assignees (issue_id uuid, assignee_id uuid, deleted_at timestamptz);
CREATE TABLE intake_issues (issue_id uuid, status integer, project_id uuid, workspace_id uuid, deleted_at timestamptz);
CREATE TABLE file_assets (issue_id uuid, entity_type text, deleted_at timestamptz);
CREATE TABLE issue_links (issue_id uuid, deleted_at timestamptz);
CREATE TABLE issue_subscribers (issue_id uuid, project_id uuid, workspace_id uuid, subscriber_id uuid, deleted_at timestamptz);
CREATE TABLE labels (id uuid PRIMARY KEY, project_id uuid, workspace_id uuid, deleted_at timestamptz);
INSERT INTO workspaces (id) VALUES ('00000000-0000-4000-8000-000000000001');
INSERT INTO projects (id, workspace_id) VALUES
 ('00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000001'),
 ('00000000-0000-4000-8000-000000000007', '00000000-0000-4000-8000-000000000001');
INSERT INTO issues (id, workspace_id, project_id, created_by_id) VALUES
 ('00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000001',
  '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000004');
INSERT INTO labels (id, project_id, workspace_id, deleted_at) VALUES
 ('00000000-0000-4000-8000-000000000005', '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000001', now()),
 ('00000000-0000-4000-8000-000000000006', '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000001', NULL);
INSERT INTO issue_labels (issue_id, label_id) VALUES
 ('00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000005'),
 ('00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000006');
INSERT INTO cycles (id, project_id, workspace_id, deleted_at) VALUES
 ('00000000-0000-4000-8000-000000000008', '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000001', NULL),
 ('00000000-0000-4000-8000-000000000009', '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000001', now()),
 ('00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000007', '00000000-0000-4000-8000-000000000001', NULL);
INSERT INTO cycle_issues (issue_id, cycle_id, created_at) VALUES
 ('00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000008', now()),
 ('00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000009', now()+interval '1 second'),
 ('00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000010', now()+interval '2 seconds');
INSERT INTO modules (id, deleted_at, archived_at) VALUES
 ('00000000-0000-4000-8000-000000000011', NULL, NULL),
 ('00000000-0000-4000-8000-000000000012', NULL, now()),
 ('00000000-0000-4000-8000-000000000013', now(), NULL);
INSERT INTO module_issues (issue_id, module_id, deleted_at) VALUES
 ('00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000011', NULL),
 ('00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000012', NULL),
 ('00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000013', NULL);
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pair.Close() })

	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(ctx)) })

	var raw []byte
	err = connection.QueryRow(
		ctx,
		`SELECT `+issueAnnotationsSQL("$2", true)+` FROM issues i WHERE i.id=$1`,
		"00000000-0000-4000-8000-000000000003",
		"00000000-0000-4000-8000-000000000004",
	).Scan(&raw)
	if err != nil {
		t.Fatal(err)
	}
	var projection map[string]any
	err = json.Unmarshal(raw, &projection)
	if err != nil {
		t.Fatal(err)
	}
	labelIDs, ok := projection["label_ids"].([]any)
	if !ok {
		t.Fatalf("projected label_ids = %#v; want the live-label list", projection["label_ids"])
	}
	if len(labelIDs) != 1 || labelIDs[0] != "00000000-0000-4000-8000-000000000006" {
		t.Fatalf("projected label_ids = %#v; want the live parent label only", labelIDs)
	}
	if projection["cycle_id"] != "00000000-0000-4000-8000-000000000008" {
		t.Fatalf("projected cycle_id = %v; want live same-project cycle", projection["cycle_id"])
	}
	moduleIDs, ok := projection["module_ids"].([]any)
	if !ok || len(moduleIDs) != 1 || moduleIDs[0] != "00000000-0000-4000-8000-000000000011" {
		t.Fatalf("projected module_ids = %#v; want active module only", projection["module_ids"])
	}
	var collectionRaw []byte
	if err := connection.QueryRow(ctx, `SELECT `+issueCollectionAnnotationsSQL("NULL::uuid")+` FROM issues i WHERE i.id=$1`, "00000000-0000-4000-8000-000000000003").Scan(&collectionRaw); err != nil {
		t.Fatal(err)
	}
	var collection map[string]any
	if err := json.Unmarshal(collectionRaw, &collection); err != nil {
		t.Fatal(err)
	}
	collectionModuleIDs, ok := collection["module_ids"].([]any)
	if !ok || len(collectionModuleIDs) != 2 {
		t.Fatalf("public collection module_ids = %#v; want active and archived modules", collection["module_ids"])
	}
	for _, id := range collectionModuleIDs {
		if id == "00000000-0000-4000-8000-000000000013" {
			t.Fatalf("public collection leaked deleted module: %#v", collectionModuleIDs)
		}
	}
}
