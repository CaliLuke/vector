//go:build integration

package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

const groupedListSeed = `
INSERT INTO labels(id,workspace_id,project_id,name)
SELECT '00000000-0000-4000-8000-000000000020',workspace_id,project_id,'Second live label' FROM issues WHERE sequence_id=1;
INSERT INTO issue_labels(issue_id,label_id)
SELECT id,'00000000-0000-4000-8000-000000000020' FROM issues WHERE sequence_id IN(1,2);
INSERT INTO issue_labels(issue_id,label_id)
SELECT id,'00000000-0000-4000-8000-000000000006' FROM issues WHERE sequence_id IN(1,3,5);
UPDATE issues SET priority='high' WHERE sequence_id=3;
ALTER TABLE issue_relations ADD COLUMN created_at timestamptz DEFAULT now();
INSERT INTO issue_relations(issue_id,related_issue_id,relation_type)
SELECT issue.id, related.id, 'relates_to' FROM issues issue CROSS JOIN issues related
WHERE issue.sequence_id IN(1,2,3) AND related.sequence_id=4;
`

func TestBrowserGroupedListWindowsPreserveMembershipCountsAndRelations(t *testing.T) {
	const labelB = "00000000-0000-4000-8000-000000000020"
	ctx, db := suiteDatabases.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed+pqlReadSeed+groupedListSeed)
	repo, err := NewRepository(db, modelPublisher{}, "")
	require.NoError(t, err)
	q := Request{Method: "GET", Operation: "get_api_workspaces_by_slug_projects_by_project_id_issues_detail", ActorID: testsupport.LabelActor, Slug: "demo", ProjectID: testsupport.LabelProject,
		Params:  map[string]string{"group_by": "labels__id", "sub_group_by": "priority", "order_by": "sequence_id", "per_page": "1", "expand": "issue_relation"},
		Present: map[string]bool{"per_page": true, "order_by": true}}

	read := func() map[string]any {
		t.Helper()
		result, err := repo.listIssues(ctx, q)
		require.NoError(t, err)
		require.Equal(t, 200, result.Status)
		body := result.Body.(map[string]any)
		require.Equal(t, 5, body["total_results"]) // One issue belongs to two buckets.
		require.Equal(t, 1, body["total_pages"])   // The visible intake issue does not count here.
		return body
	}
	bucket := func(body map[string]any, label, priority string) map[string]any {
		t.Helper()
		return body["results"].(map[string]any)[label].(map[string]any)["results"].(map[string]any)[priority].(map[string]any)
	}
	first := read()
	require.Equal(t, "1:1:0", first["next_cursor"])
	require.Equal(t, true, first["next_page_results"])
	require.Equal(t, 4, first["count"])
	highA := bucket(first, testsupport.LabelKept, "high")
	highB := bucket(first, labelB, "high")
	require.Equal(t, 1, highA["total_results"])
	require.Equal(t, 0, highB["total_results"])
	rows := highA["results"].([]map[string]any)
	require.Len(t, rows, 1)
	require.Equal(t, json.Number("1"), rows[0]["sequence_id"])
	require.Equal(t, testsupport.LabelKept, rows[0]["labels__id"])
	require.Equal(t, []any{testsupport.LabelKept, labelB}, rows[0]["label_ids"])
	expanded := rows[0]["issue_relation"].([]any)
	require.Len(t, expanded, 1)
	require.Equal(t, json.Number("4"), expanded[0].(map[string]any)["sequence_id"])
	require.NotContains(t, first["results"], testsupport.LabelDeleted)
	require.NotContains(t, first["results"], "00000000-0000-4000-8000-000000000009")

	q.Params["cursor"], q.Present["cursor"] = first["next_cursor"].(string), true
	second := read()
	require.False(t, second["next_page_results"].(bool))
	require.Equal(t, 1, second["count"])
	pageRows := bucket(second, testsupport.LabelKept, "high")["results"].([]map[string]any)
	require.Len(t, pageRows, 1)
	require.Equal(t, json.Number("3"), pageRows[0]["sequence_id"])
	require.Empty(t, bucket(second, labelB, "high")["results"])
	require.Contains(t, second["results"], "00000000-0000-4000-8000-000000000008") // An empty catalog group.

	delete(q.Params, "sub_group_by")
	q.Params["cursor"] = "1:7:0"
	pastEnd := read()
	require.Empty(t, pastEnd["results"])
	require.Equal(t, 0, pastEnd["count"])
}

type failingBrowserVisit struct {
	calls chan string
}

func (v failingBrowserVisit) AdmitRecentVisit(_ context.Context, _, _, entity, _, _, _ string) error {
	v.calls <- entity
	return errors.New("visit store unavailable")
}

func TestBrowserWorkItemReadsSucceedWhenVisitAdmissionFails(t *testing.T) {
	ctx, db := suiteDatabases.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed+pqlReadSeed)
	for _, tc := range []struct {
		name, entity string
		read         func(*Repository, context.Context, Request) (Result, error)
	}{
		{"project list", "project", (*Repository).listIssues},
		{"issue detail", "issue", (*Repository).readIssue},
		{"issue identifier", "issue", (*Repository).identifier},
	} {
		t.Run(tc.name, func(t *testing.T) {
			admitter := failingBrowserVisit{calls: make(chan string, 1)}
			repo, err := NewRepositoryWithNativeTasks(db, modelPublisher{}, "", nil, admitter)
			require.NoError(t, err)
			response, err := tc.read(repo, ctx, Request{
				Method: "GET", Operation: "get_api_workspaces_by_slug_projects_by_project_id_issues",
				Slug: "demo", ProjectID: testsupport.LabelProject, ActorID: testsupport.LabelActor, IssueID: testsupport.LabelIssue,
				Params: map[string]string{"identifier": "CUR-1"},
			})
			require.NoError(t, err)
			require.Equal(t, 200, response.Status)
			select {
			case entity := <-admitter.calls:
				require.Equal(t, tc.entity, entity)
			case <-time.After(recentVisitTimeout):
				t.Fatal("recent visit was not dispatched")
			}
		})
	}
}

func TestBrowserGroupedListSupportsBuiltInAxesAndRelationOrdering(t *testing.T) {
	const relID = "00000000-0000-4000-8000-000000000021"
	ctx, db := suiteDatabases.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed+pqlReadSeed+`
ALTER TABLE states ADD COLUMN created_at timestamptz DEFAULT now();
ALTER TABLE cycles ADD COLUMN created_at timestamptz DEFAULT now();
ALTER TABLE modules ADD COLUMN created_at timestamptz DEFAULT now();
INSERT INTO cycles(id,workspace_id,project_id)
SELECT '00000000-0000-4000-8000-000000000021',workspace_id,project_id FROM issues WHERE sequence_id=1;
INSERT INTO cycle_issues(issue_id,cycle_id)
SELECT id,'00000000-0000-4000-8000-000000000021' FROM issues WHERE sequence_id=1;
INSERT INTO modules(id,workspace_id,project_id)
SELECT '00000000-0000-4000-8000-000000000021',workspace_id,project_id FROM issues WHERE sequence_id=1;
INSERT INTO module_issues(issue_id,module_id)
SELECT id,'00000000-0000-4000-8000-000000000021' FROM issues WHERE sequence_id=1;
INSERT INTO issue_assignees(issue_id,assignee_id)
SELECT id,created_by_id FROM issues WHERE sequence_id=1;
`)
	repo, err := NewRepository(db, modelPublisher{}, "")
	require.NoError(t, err)
	base := Request{Method: "GET", Operation: "get_api_workspaces_by_slug_projects_by_project_id_issues", ActorID: testsupport.LabelActor, Slug: "demo", ProjectID: testsupport.LabelProject}
	for _, tc := range []struct{ axis, key string }{
		{"assignees__id", testsupport.LabelActor},
		{"created_by", testsupport.LabelActor},
		{"cycle_id", relID},
		{"issue_module__module_id", relID},
		{"labels__id", testsupport.LabelKept},
		{"priority", "high"},
		{"project_id", testsupport.LabelProject},
		{"start_date", "2026-10-20"},
		{"state_id", "00000000-0000-4000-8000-000000000055"},
		{"state__group", "started"},
		{"target_date", "2026-10-27"},
	} {
		t.Run(tc.axis, func(t *testing.T) {
			q := base
			q.Params = map[string]string{"group_by": tc.axis, "order_by": "labels__name", "labels": testsupport.LabelKept, "per_page": "1"}
			q.Present = map[string]bool{"order_by": true, "labels": true, "per_page": true}
			result, err := repo.listIssues(ctx, q)
			require.NoError(t, err)
			body := result.Body.(map[string]any)
			require.Equal(t, 1, body["total_results"])
			group := body["results"].(map[string]any)[tc.key].(map[string]any)
			require.Equal(t, 1, group["total_results"]) // Intake-only legacy sentinel.
			rows := group["results"].([]map[string]any)
			require.Len(t, rows, 1)
			require.Equal(t, testsupport.LabelIssue, rows[0]["id"])
		})
	}
}
