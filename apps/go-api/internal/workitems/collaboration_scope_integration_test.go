//go:build integration

package workitems

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBoundCollaborationReadsRejectScopeEscapes(t *testing.T) {
	ctx, db := postgresfixture.OpenSQL(t, childVisibilitySchema+`
 CREATE TABLE users(id uuid PRIMARY KEY,is_active boolean);
 INSERT INTO users VALUES('10000000-0000-4000-8000-000000000001',true);
 INSERT INTO workspaces(id,slug) VALUES('20000000-0000-4000-8000-000000000001','demo');
 INSERT INTO projects(id,workspace_id) SELECT '30000000-0000-4000-8000-000000000001',id FROM workspaces;
 INSERT INTO workspace_members(id,workspace_id,member_id,role) SELECT gen_random_uuid(),w.id,u.id,15 FROM workspaces w,users u;
 INSERT INTO project_members(id,workspace_id,project_id,member_id,role) SELECT gen_random_uuid(),p.workspace_id,p.id,u.id,15 FROM projects p,users u;
 INSERT INTO issues(id,workspace_id,project_id) SELECT '80000000-0000-4000-8000-000000000001',workspace_id,id FROM projects;
 INSERT INTO issue_comments(id,project_id,workspace_id,issue_id,comment_html) SELECT '90000000-0000-4000-8000-000000000001',project_id,workspace_id,id,'comment' FROM issues;
 INSERT INTO issue_links(id,project_id,workspace_id,issue_id,url) SELECT '90000000-0000-4000-8000-000000000001',project_id,workspace_id,id,'https://example.test' FROM issues;
 `)
	publisher := &childVisibilityPublisher{}
	repo, err := NewRepository(db, publisher, "https://example.test")
	require.NoError(t, err)
	workspace := uuid.MustParse("20000000-0000-4000-8000-000000000001")
	q := Request{ExpectedWorkspaceID: &workspace, API: true, Method: "GET", Operation: "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments",
		Slug: "demo", ActorID: "10000000-0000-4000-8000-000000000001", ProjectID: "30000000-0000-4000-8000-000000000001", IssueID: "80000000-0000-4000-8000-000000000001", Timezone: "UTC", Params: map[string]string{}}
	result, err := repo.ExecuteCollaboration(ctx, q, nil, "")
	require.NoError(t, err)
	require.Len(t, result.Body.(map[string]any)["results"], 1)
	for _, change := range []func(*Request){
		func(q *Request) { q.API = false }, func(q *Request) { q.Method = "POST" }, func(q *Request) {
			q.Operation = "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_reactions"
		},
		func(q *Request) { q.Params = map[string]string{"fields": "actor"} }, func(q *Request) { q.Params = map[string]string{"expand": "actor"} },
	} {
		invalid := q
		change(&invalid)
		_, err = repo.ExecuteCollaboration(ctx, invalid, nil, "")
		var failure *djangoapi.HTTPError
		require.ErrorAs(t, err, &failure)
		require.Equal(t, 403, failure.Status)
	}
	wrong := uuid.New()
	q.ExpectedWorkspaceID = &wrong
	_, err = repo.ExecuteCollaboration(ctx, q, nil, "")
	requireChildHidden(t, err)
	q.ExpectedWorkspaceID = &workspace
	_, err = db.Exec(ctx, `UPDATE project_members SET is_active=false`)
	require.NoError(t, err)
	_, err = repo.ExecuteCollaboration(ctx, q, nil, "")
	requireChildHidden(t, err)
	require.Zero(t, publisher.activities)
	require.Zero(t, publisher.cleanup)
}
