//go:build integration

package workitems

import (
	"context"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestGroupedLabelSubgroupUniverseExcludesPrivateProjectLabels(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer func() { _ = connection.Close(context.WithoutCancel(ctx)) }()
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL, deleted_at timestamptz);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, deleted_at timestamptz, archived_at timestamptz);
CREATE TABLE project_members (project_id uuid, workspace_id uuid, member_id uuid, is_active boolean, deleted_at timestamptz, created_at timestamptz);
CREATE TABLE labels (id uuid PRIMARY KEY, project_id uuid, workspace_id uuid, deleted_at timestamptz, created_at timestamptz);
CREATE TABLE issues (id uuid PRIMARY KEY, project_id uuid, workspace_id uuid, priority text, created_at timestamptz, created_by_id uuid);
CREATE TABLE issue_labels (issue_id uuid, label_id uuid, deleted_at timestamptz);
INSERT INTO workspaces (id, slug) VALUES ('00000000-0000-4000-8000-000000000001', 'demo');
INSERT INTO projects (id, workspace_id) VALUES
 ('00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000001');
INSERT INTO projects (id, workspace_id, archived_at) VALUES
 ('00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000001', now());
INSERT INTO project_members (project_id, workspace_id, member_id, is_active) VALUES
 ('00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000004', true),
 ('00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000004', true);
INSERT INTO projects (id, workspace_id) VALUES
 ('00000000-0000-4000-8000-000000000009', '00000000-0000-4000-8000-000000000001');
INSERT INTO labels (id, project_id, workspace_id, created_at) VALUES
 ('00000000-0000-4000-8000-000000000005', '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000001', now()),
 ('00000000-0000-4000-8000-000000000006', '00000000-0000-4000-8000-000000000009', '00000000-0000-4000-8000-000000000001', now()),
 ('00000000-0000-4000-8000-000000000008', '00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000001', now());
INSERT INTO issues (id, project_id, workspace_id, priority, created_at, created_by_id) VALUES
 ('00000000-0000-4000-8000-000000000007', '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000001', 'high', now(), '00000000-0000-4000-8000-000000000004');
INSERT INTO issue_labels (issue_id, label_id) VALUES
 ('00000000-0000-4000-8000-000000000007', '00000000-0000-4000-8000-000000000005');
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
	repository, err := NewRepository(connection, modelPublisher{}, "")
	if err != nil {
		t.Fatal(err)
	}

	visibleIssues, err := repository.rows(ctx, `SELECT jsonb_build_object(
      'id', issue.id,
      'label_ids', COALESCE((SELECT jsonb_agg(DISTINCT relation.label_id) FROM issue_labels relation
        WHERE relation.issue_id=issue.id AND relation.deleted_at IS NULL), '[]'::jsonb),
      'priority', issue.priority,
      'created_at', issue.created_at
    ) FROM issues issue
    JOIN projects project ON project.id=issue.project_id AND project.workspace_id=issue.workspace_id
    JOIN workspaces workspace ON workspace.id=project.workspace_id
    WHERE issue.id=$1 AND workspace.slug=$2 AND EXISTS (
      SELECT 1 FROM project_members member WHERE member.project_id=project.id
        AND member.workspace_id=workspace.id AND member.member_id=$3
        AND member.is_active=true AND member.deleted_at IS NULL
    )`, "00000000-0000-4000-8000-000000000007", "demo", "00000000-0000-4000-8000-000000000004")
	if err != nil {
		t.Fatal(err)
	}
	if len(visibleIssues) != 1 {
		t.Fatalf("visible issue rows = %#v; want one active project member issue", visibleIssues)
	}
	visibleIssue := visibleIssues[0]
	query := Request{
		Slug:    "demo",
		ActorID: "00000000-0000-4000-8000-000000000004",
		Params:  map[string]string{"group_by": "labels__id", "sub_group_by": "priority"},
	}
	grouped, err := repository.groupedPageWithVisibleRowCounts(ctx, query, []any{visibleIssue}, visibleIssues, 10, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	body, ok := grouped.Body.(map[string]any)
	if !ok {
		t.Fatalf("group response = %#v; want object", grouped.Body)
	}
	results, ok := body["results"].(map[string]any)
	if !ok {
		t.Fatalf("group results = %#v; want group map", body["results"])
	}
	if _, exists := results["00000000-0000-4000-8000-000000000006"]; exists {
		t.Fatalf("grouped response exposed a label from a project with no active actor membership: %#v", results)
	}
	if _, exists := results["00000000-0000-4000-8000-000000000005"]; !exists {
		t.Fatalf("grouped response omitted the visible project's label: %#v", results)
	}
	if _, exists := results["00000000-0000-4000-8000-000000000008"]; !exists {
		t.Fatalf("workspace grouping omitted a label from another project the actor can access: %#v", results)
	}
	query.ProjectID = "00000000-0000-4000-8000-000000000002"
	projectGroups, err := repository.groupUniverse(ctx, query, "labels__id", nil)
	if err != nil {
		t.Fatal(err)
	}
	projectGroupSet := map[string]bool{}
	for _, group := range projectGroups {
		projectGroupSet[group.(string)] = true
	}
	if len(projectGroupSet) != 2 || !projectGroupSet["00000000-0000-4000-8000-000000000005"] || !projectGroupSet["None"] {
		t.Fatalf("project-scoped label universe = %#v; want only that project's live label plus None", projectGroups)
	}
	archivedProjectQuery := query
	archivedProjectQuery.ProjectID = "00000000-0000-4000-8000-000000000003"
	archivedProjectQuery.Operation = "project_archived_issues"
	archivedGroups, err := repository.groupUniverse(ctx, archivedProjectQuery, "labels__id", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(archivedGroups) != 2 || archivedGroups[0] != "00000000-0000-4000-8000-000000000008" {
		t.Fatalf("archived-project label universe = %#v; want its live label plus None", archivedGroups)
	}
	publicProjectQuery := query
	publicProjectQuery.ProjectID = "00000000-0000-4000-8000-000000000002"
	publicProjectQuery.ActorID = ""
	publicProjectQuery.Operation = "public_feedback_issue_list"
	publicGroups, err := repository.groupUniverse(ctx, publicProjectQuery, "labels__id", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(publicGroups) != 2 || publicGroups[0] != "00000000-0000-4000-8000-000000000005" {
		t.Fatalf("actorless public label universe = %#v; want the publication-scoped label plus None", publicGroups)
	}
}
