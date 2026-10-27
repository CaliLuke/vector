//go:build integration

package workitems

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/stretchr/testify/require"
)

const pqlMilestoneSeed = `
CREATE TABLE milestones(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,deleted_at timestamptz);
CREATE TABLE milestone_issues(id uuid,issue_id uuid,milestone_id uuid,project_id uuid,workspace_id uuid,deleted_at timestamptz);
INSERT INTO milestones(id,project_id,workspace_id,deleted_at)
SELECT ('30000000-0000-4000-8000-'||lpad(n::text,12,'0'))::uuid,
 CASE WHEN n=3 THEN '00000000-0000-4000-8000-000000000099' ELSE '00000000-0000-4000-8000-000000000002' END::uuid,
 CASE WHEN n=4 THEN '00000000-0000-4000-8000-000000000099' ELSE '00000000-0000-4000-8000-000000000001' END::uuid,
 CASE WHEN n=2 THEN now() END
FROM generate_series(1,4) n;
INSERT INTO milestone_issues(issue_id,milestone_id,project_id,workspace_id)
SELECT id,'30000000-0000-4000-8000-000000000001',project_id,workspace_id FROM issues WHERE sequence_id IN(1,2,5,6,7,8);
INSERT INTO milestone_issues SELECT * FROM milestone_issues WHERE issue_id='00000000-0000-4000-8000-000000000004';
INSERT INTO milestone_issues(issue_id,milestone_id,project_id,workspace_id)
SELECT CASE WHEN n=2 THEN '90000000-0000-4000-8000-000000000003' ELSE '90000000-0000-4000-8000-000000000004' END::uuid,
 ('30000000-0000-4000-8000-'||lpad(n::text,12,'0'))::uuid,'00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001'
FROM generate_series(2,5) n;
INSERT INTO milestone_issues(issue_id,milestone_id,project_id,workspace_id,deleted_at) VALUES
 ('90000000-0000-4000-8000-000000000004','30000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000099','00000000-0000-4000-8000-000000000001',NULL),
 ('90000000-0000-4000-8000-000000000004','30000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000099',NULL),
 ('90000000-0000-4000-8000-000000000004','30000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001',now());
`

func newPQLMilestoneFixture(t *testing.T) pqlIdentityFixture {
	t.Helper()
	ctx, db := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed+pqlReadSeed+pqlMilestoneSeed)
	repo, err := NewRepository(db, modelPublisher{}, "https://example.test")
	require.NoError(t, err)
	q := Request{API: true, Method: "GET", Operation: "get_api_v1_workspaces_by_slug_projects_by_project_id_issues", ActorID: testsupport.LabelActor, Slug: "demo", ProjectID: testsupport.LabelProject, Timezone: "UTC"}
	require.NoError(t, repo.AuthorizeInitial(ctx, q))
	return pqlIdentityFixture{ctx, db, repo, q}
}

func TestAPIPQLMilestoneFiltersOwnedVisibleRelationsBeforePages(t *testing.T) {
	f := newPQLMilestoneFixture(t)
	f.examples(t, map[string][]int{
		`milestone = "30000000-0000-4000-8000-000000000001"`:                                               {1, 2},
		`milestone != "30000000-0000-4000-8000-000000000001"`:                                              {3, 4},
		`milestone IN ("30000000-0000-4000-8000-000000000001","30000000-0000-4000-8000-000000000002")`:     {1, 2},
		`milestone NOT IN ("30000000-0000-4000-8000-000000000001","30000000-0000-4000-8000-000000000002")`: {3, 4},
		`milestone IS NULL`: {3, 4}, `milestone IS EMPTY`: {3, 4},
		`milestone IS NOT NULL`: {1, 2}, `milestone IS NOT EMPTY`: {1, 2},
		`NOT milestone = "30000000-0000-4000-8000-000000000001"`: {3, 4},
		`milestone IS EMPTY OR priority = "high"`:                {1, 3, 4},
		`milestone IN ("30000000-0000-4000-8000-000000000002","30000000-0000-4000-8000-000000000003","30000000-0000-4000-8000-000000000004","30000000-0000-4000-8000-000000000005")`: {},
	})
	assertPQLFirstTwoPages(t, f.ctx, f.repo, f.request, `milestone IS NOT EMPTY`, 2, "1", "2")
	q := f.request
	q.Operation = "get_api_workspaces_by_slug_projects_by_project_id_archived_issues"
	assertPQLSequences(t, f.ctx, f.repo, q, `milestone IS NOT EMPTY`, []int{6})
}

func TestAPIPQLMilestoneGuestCannotInferTargetsThroughNegation(t *testing.T) {
	f := newPQLMilestoneFixture(t)
	_, err := f.db.Exec(f.ctx, `UPDATE project_members SET role=5`)
	require.NoError(t, err)
	_, err = f.db.Exec(f.ctx, `UPDATE issues SET created_by_id=$1`, testsupport.LabelActor)
	require.NoError(t, err)
	f.examples(t, map[string][]int{
		`milestone IS NOT EMPTY`: {}, `milestone IS EMPTY`: {1, 2, 3, 4},
		`milestone != "30000000-0000-4000-8000-000000000001"`:       {1, 2, 3, 4},
		`milestone NOT IN ("30000000-0000-4000-8000-000000000002")`: {1, 2, 3, 4},
		`NOT milestone = "30000000-0000-4000-8000-000000000001"`:    {1, 2, 3, 4},
		`milestone IS NOT EMPTY OR priority = "high"`:               {1},
	})
	_, err = f.db.Exec(f.ctx, `UPDATE projects SET guest_view_all_features=true WHERE id=$1`, testsupport.LabelProject)
	require.NoError(t, err)
	f.examples(t, map[string][]int{`milestone IS NOT EMPTY`: {1, 2}, `milestone IS EMPTY`: {3, 4}})
	_, err = f.db.Exec(f.ctx, `UPDATE projects SET archived_at=now() WHERE id=$1`, testsupport.LabelProject)
	require.NoError(t, err)
	// The active-item route also excludes the archived project itself.
	f.examples(t, map[string][]int{`milestone IS NOT EMPTY`: {}, `milestone IS EMPTY`: {}})
	q := f.request
	q.Operation = "get_api_workspaces_by_slug_projects_by_project_id_archived_issues"
	assertPQLSequences(t, f.ctx, f.repo, q, `milestone IS EMPTY`, []int{6})
	assertPQLSequences(t, f.ctx, f.repo, q, `milestone IS NOT EMPTY`, []int{})
}
