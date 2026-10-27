//go:build integration

package activityreads

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const scopedActivitySchema = `
CREATE TABLE users(id uuid PRIMARY KEY,is_active boolean);
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text,deleted_at timestamptz);
CREATE TABLE workspace_members(workspace_id uuid,member_id uuid,role int,is_active boolean,deleted_at timestamptz);
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,deleted_at timestamptz,archived_at timestamptz,guest_view_all_features boolean);
CREATE TABLE project_members(project_id uuid,workspace_id uuid,member_id uuid,role int,is_active boolean,deleted_at timestamptz);
CREATE TABLE issues(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,created_by_id uuid,deleted_at timestamptz,is_draft boolean);
CREATE TABLE issue_comments(id uuid PRIMARY KEY,issue_id uuid,project_id uuid,workspace_id uuid);
CREATE TABLE issue_activities(id uuid PRIMARY KEY,created_at timestamptz,updated_at timestamptz,deleted_at timestamptz,workspace_id uuid,project_id uuid,issue_id uuid,
 verb text,field text,old_value text,new_value text,comment text,attachments jsonb,issue_comment_id uuid,actor_id uuid,old_identifier uuid,new_identifier uuid,epoch double precision);
INSERT INTO users VALUES('10000000-0000-4000-8000-000000000001',true);
INSERT INTO workspaces VALUES('20000000-0000-4000-8000-000000000001','demo',NULL);
INSERT INTO projects VALUES('30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001',NULL,NULL,false);
INSERT INTO workspace_members SELECT w.id,u.id,15,true,NULL FROM workspaces w,users u;
INSERT INTO project_members SELECT p.id,p.workspace_id,u.id,15,true,NULL FROM projects p,users u;
INSERT INTO issues SELECT '80000000-0000-4000-8000-000000000001',p.id,p.workspace_id,NULL,NULL,false FROM projects p;
INSERT INTO issue_activities(id,created_at,project_id,workspace_id,issue_id,verb,field,epoch)
 SELECT '90000000-0000-4000-8000-000000000001','2024-01-02',i.project_id,i.workspace_id,i.id,'updated','name',1704153600 FROM issues i;
`

func TestScopedActivityReadsPreserveVisibilityAndDistinguishMissing(t *testing.T) {
	ctx, db := postgresfixture.OpenSQL(t, scopedActivitySchema)
	repo, err := NewRepository(db)
	require.NoError(t, err)
	workspace := uuid.MustParse("20000000-0000-4000-8000-000000000001")
	principal := Principal{UserID: uuid.MustParse("10000000-0000-4000-8000-000000000001"), Authenticated: true, Timezone: "UTC"}
	q := Request{Operation: "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_activities", ExpectedWorkspaceID: &workspace, Params: map[string]string{
		"slug": "demo", "project_id": "30000000-0000-4000-8000-000000000001", "issue_id": "80000000-0000-4000-8000-000000000001",
	}, Present: map[string]bool{}}
	result, err := repo.Read(ctx, principal, q)
	require.NoError(t, err)
	require.Equal(t, 1, result.Body.(map[string]any)["total_count"])
	detail := q
	detail.Operation += "_by_pk"
	detail.Params["pk"] = "90000000-0000-4000-8000-000000000001"
	result, err = repo.Read(ctx, principal, detail)
	require.NoError(t, err)
	require.Equal(t, "updated", result.Body.(map[string]any)["verb"])
	detail.Params["pk"] = "90000000-0000-4000-8000-000000000002"
	_, err = repo.Read(ctx, principal, detail)
	require.ErrorIs(t, err, ErrNotFound)
	detail.Params["pk"] = "invalid"
	_, err = repo.Read(ctx, principal, detail)
	require.ErrorIs(t, err, ErrNotFound)
	q.Params["cursor"] = "invalid"
	q.Present["cursor"] = true
	_, err = repo.Read(ctx, principal, q)
	var failure *djangoapi.HTTPError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, 400, failure.Status)
	delete(q.Params, "cursor")
	delete(q.Present, "cursor")
	for _, field := range []string{"fields", "expand"} {
		q.Params[field] = "actor"
		_, err = repo.Read(ctx, principal, q)
		require.ErrorIs(t, err, ErrForbidden)
		delete(q.Params, field)
	}
	_, err = db.Exec(ctx, `UPDATE issue_activities SET deleted_at=now()`)
	require.NoError(t, err)
	result, err = repo.Read(ctx, principal, q)
	require.NoError(t, err)
	require.Empty(t, result.Body.(map[string]any)["results"])
	_, err = db.Exec(ctx, `UPDATE issues SET is_draft=true`)
	require.NoError(t, err)
	_, err = repo.Read(ctx, principal, q)
	require.ErrorIs(t, err, ErrNotFound)
	_, err = db.Exec(ctx, `UPDATE workspace_members SET is_active=false`)
	require.NoError(t, err)
	_, err = repo.Read(ctx, principal, q)
	require.ErrorIs(t, err, ErrForbidden)
	_, err = db.Exec(ctx, `DROP TABLE issue_activities`)
	require.NoError(t, err)
	_, err = repo.Read(ctx, principal, q)
	require.Error(t, err)
}
