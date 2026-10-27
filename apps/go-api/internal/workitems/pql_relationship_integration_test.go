//go:build integration

package workitems

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/stretchr/testify/require"
)

func TestAPIPQLRelationshipsUseOwnedLiveTargetsBeforeCounting(t *testing.T) {
	ctx, db := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed+pqlReadSeed+pqlRelationshipSeed)
	repo, err := NewRepository(db, modelPublisher{}, "https://example.test")
	require.NoError(t, err)
	repo.now = func() time.Time { return time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC) }
	base := Request{API: true, Method: "GET", Operation: "get_api_v1_workspaces_by_slug_projects_by_project_id_issues", ActorID: testsupport.LabelActor, Slug: "demo", ProjectID: testsupport.LabelProject, Timezone: "America/Los_Angeles"}
	require.NoError(t, repo.AuthorizeInitial(ctx, base))
	for _, tc := range []struct {
		query string
		want  []int
	}{
		{`state = "00000000-0000-4000-8000-000000000055"`, []int{1}},
		{`state != "00000000-0000-4000-8000-000000000055"`, []int{2, 9, 10, 11}},
		{`state IN ("00000000-0000-4000-8000-000000000055","00000000-0000-4000-8000-000000000057")`, []int{1}},
		{`state NOT IN ("00000000-0000-4000-8000-000000000055")`, []int{2, 9, 10, 11}},
		{`state IS NULL`, []int{3, 4}}, {`state IS NOT NULL`, []int{1, 2, 9, 10, 11}},
		{`NOT state = "00000000-0000-4000-8000-000000000057"`, []int{1, 2, 9, 10, 11}},
		{`stateGroup = "started"`, []int{1}}, {`stateGroup != "completed"`, []int{1, 9, 10, 11}},
		{`stateGroup IN openStates()`, []int{1, 9, 10}},
		{`stateGroup IN ("backlog","unstarted","started")`, []int{1, 9, 10}},
		{`stateGroup NOT IN openStates()`, []int{2, 11}},
		{`stateGroup IN closedStates()`, []int{2, 11}},
		{`stateGroup IN activeStates()`, []int{1, 10}},
		{`stateGroup IS NULL`, []int{3, 4}}, {`stateGroup IS NOT NULL`, []int{1, 2, 9, 10, 11}},
		{`type = "70000000-0000-4000-8000-000000000001"`, []int{1}},
		{`type != "70000000-0000-4000-8000-000000000001"`, []int{2, 11}},
		{`type IN ("70000000-0000-4000-8000-000000000001","70000000-0000-4000-8000-000000000003")`, []int{1}},
		{`type NOT IN ("70000000-0000-4000-8000-000000000001")`, []int{2, 11}},
		{`type IS NULL`, []int{3, 4, 9, 10}}, {`type IS NOT NULL`, []int{1, 2, 11}},
		{`NOT type = "70000000-0000-4000-8000-000000000003"`, []int{1, 2, 11}},
		{`isEpic()`, []int{1}}, {`NOT isEpic()`, []int{2, 11}},
		{`label = "00000000-0000-4000-8000-000000000006"`, []int{1}},
		{`label != "00000000-0000-4000-8000-000000000006"`, []int{2, 3, 4, 9, 10, 11}},
		{`label IN ("00000000-0000-4000-8000-000000000006","00000000-0000-4000-8000-000000000007")`, []int{1}},
		{`label NOT IN ("00000000-0000-4000-8000-000000000006")`, []int{2, 3, 4, 9, 10, 11}},
		{`NOT label = "00000000-0000-4000-8000-000000000006"`, []int{2, 3, 4, 9, 10, 11}},
		{`label IS EMPTY`, []int{2, 3, 4, 9, 10, 11}}, {`label IS NULL`, []int{2, 3, 4, 9, 10, 11}},
		{`label IS NOT EMPTY`, []int{1}}, {`label IS NOT NULL`, []int{1}},
		{`hasNoLabel()`, []int{2, 3, 4, 9, 10, 11}}, {`NOT hasNoLabel()`, []int{1}},
		{`cycle IS NOT EMPTY`, []int{1, 2, 9, 10, 11}}, {`cycle IS EMPTY`, []int{3, 4}},
		{`cycle IN activeCycle()`, []int{1, 2}},
		{`cycle NOT IN activeCycle()`, []int{3, 4, 9, 10, 11}},
		{`NOT cycle IN activeCycle()`, []int{3, 4, 9, 10, 11}},
		{`cycle IN completedCycles()`, []int{9, 11}},
		{`cycle IN upcomingCycles()`, []int{10, 11}},
		{`cycle IN ("60000000-0000-4000-8000-000000000001","60000000-0000-4000-8000-000000000006")`, []int{1, 2}},
		{`cycle != "60000000-0000-4000-8000-000000000006"`, []int{1, 3, 4, 9, 10, 11}},
		{`module = "50000000-0000-4000-8000-000000000001"`, []int{1}},
		{`module IN ("50000000-0000-4000-8000-000000000001","50000000-0000-4000-8000-000000000002")`, []int{1, 2}},
		{`module NOT IN ("50000000-0000-4000-8000-000000000002")`, []int{1, 3, 4, 9, 10, 11}},
		{`module IS NULL`, []int{3, 4, 9, 10, 11}}, {`module IS NOT NULL`, []int{1, 2}},
		{`isIntake()`, []int{1, 2}}, {`NOT isIntake()`, []int{3, 4, 9, 10, 11}},
		{`hasStartAndDueDates()`, []int{1}}, {`NOT hasStartAndDueDates()`, []int{2, 3, 4, 9, 10, 11}},
		{`isOverdue()`, []int{9}}, {`NOT isOverdue()`, []int{1, 2, 10, 11}},
		{`dueDate < today() AND stateGroup IN openStates()`, []int{9}},
		{`isOverdue() OR isEpic() OR isIntake() OR hasNoLabel() OR hasStartAndDueDates()`, []int{1, 2, 3, 4, 9, 10, 11}},
		{`cycle IN activeCycle() AND stateGroup IN closedStates()`, []int{2}},
	} {
		t.Run(tc.query, func(t *testing.T) {
			assertPQLSequences(t, ctx, repo, base, tc.query, tc.want)
		})
	}
	t.Run("multiple relationships do not inflate count or pagination", func(t *testing.T) {
		q := base
		q.Params = map[string]string{"pql": "cycle IS NOT NULL", "fields": "sequence_id", "order_by": "sequence_id", "per_page": "1"}
		q.Present = map[string]bool{"per_page": true}
		first, readErr := repo.Execute(ctx, q, nil, "")
		require.NoError(t, readErr)
		page := first.Body.(map[string]any)
		require.Equal(t, 5, page["total_results"])
		require.Equal(t, []any{map[string]any{"sequence_id": json.Number("1")}}, page["results"])
		q.Params["cursor"], q.Present["cursor"] = page["next_cursor"].(string), true
		next, readErr := repo.Execute(ctx, q, nil, "")
		require.NoError(t, readErr)
		require.Equal(t, 5, next.Body.(map[string]any)["total_results"])
		require.Equal(t, []any{map[string]any{"sequence_id": json.Number("2")}}, next.Body.(map[string]any)["results"])
	})
	t.Run("cycle endpoints use requester calendar dates", func(t *testing.T) {
		q := base
		q.Timezone = "UTC"
		assertPQLSequences(t, ctx, repo, q, `cycle IN activeCycle()`, []int{1, 2, 9})
		assertPQLSequences(t, ctx, repo, q, `cycle IN completedCycles()`, []int{})
	})
	t.Run("foreign workspace and missing scalar references remain unknown", func(t *testing.T) {
		_, updateErr := db.Exec(ctx, `UPDATE states SET workspace_id='00000000-0000-4000-8000-000000000099' WHERE id='00000000-0000-4000-8000-000000000055';
UPDATE issues SET state_id='ffffffff-ffff-4fff-8fff-ffffffffffff' WHERE sequence_id=2;
UPDATE issues SET state_id=NULL WHERE sequence_id=9;`)
		require.NoError(t, updateErr)
		assertPQLSequences(t, ctx, repo, base, `state IS NULL`, []int{1, 2, 3, 4, 9})
		assertPQLSequences(t, ctx, repo, base, `stateGroup IS NOT NULL`, []int{10, 11})
		assertPQLSequences(t, ctx, repo, base, `NOT stateGroup = "started"`, []int{10, 11})
	})
	t.Run("intake presence hides deleted foreign and missing targets", func(t *testing.T) {
		_, updateErr := db.Exec(ctx, `UPDATE intakes SET project_id='00000000-0000-4000-8000-000000000099';
INSERT INTO intakes(id,workspace_id,project_id,deleted_at) VALUES
 ('40000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002',now()),
 ('40000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000099','00000000-0000-4000-8000-000000000002',NULL),
 ('40000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002',NULL);
INSERT INTO intake_issues(issue_id,intake_id,workspace_id,project_id)
SELECT id,('40000000-0000-4000-8000-'||lpad((CASE sequence_id WHEN 3 THEN 1 WHEN 4 THEN 2 WHEN 9 THEN 3 ELSE 4 END)::text,12,'0'))::uuid,
 CASE WHEN sequence_id=9 THEN '00000000-0000-4000-8000-000000000099'::uuid ELSE workspace_id END,project_id
FROM issues WHERE sequence_id IN(3,4,9,10);`)
		require.NoError(t, updateErr)
		assertPQLSequences(t, ctx, repo, base, `isIntake()`, []int{})
		assertPQLSequences(t, ctx, repo, base, `NOT isIntake()`, []int{1, 2, 3, 4, 9, 10, 11})
	})
}

func assertPQLSequences(t *testing.T, ctx context.Context, repo *Repository, base Request, query string, want []int) {
	t.Helper()
	q := base
	q.Params = map[string]string{"pql": query, "fields": "sequence_id", "order_by": "sequence_id", "per_page": "100"}
	q.Present = map[string]bool{"per_page": true}
	result, err := repo.Execute(ctx, q, nil, "")
	require.NoError(t, err)
	page := result.Body.(map[string]any)
	require.Equal(t, len(want), page["total_results"])
	rows := page["results"].([]any)
	got := make([]int, len(rows))
	for i, row := range rows {
		got[i], err = strconv.Atoi(string(row.(map[string]any)["sequence_id"].(json.Number)))
		require.NoError(t, err)
	}
	require.Equal(t, want, got)
}
