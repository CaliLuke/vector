//go:build integration

package workitems

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/stretchr/testify/require"
)

const pqlReadSeed = `
ALTER TABLE issue_subscribers ADD COLUMN project_id uuid, ADD COLUMN workspace_id uuid;
CREATE TABLE issue_types(id uuid PRIMARY KEY,is_epic boolean);
UPDATE issues SET name='Literal 50%_\\ title',priority='high',target_date='2026-10-27',start_date='2026-10-20' WHERE sequence_id=1;
UPDATE issues SET name='',priority='low',target_date=NULL,start_date=NULL WHERE sequence_id=2;
INSERT INTO issues(id,workspace_id,project_id,name,sequence_id,priority,created_by_id,archived_at,is_draft,deleted_at)
SELECT ('90000000-0000-4000-8000-'||lpad(n::text,12,'0'))::uuid,
 '00000000-0000-4000-8000-000000000001',
 CASE WHEN n=5 THEN '00000000-0000-4000-8000-000000000099' ELSE '00000000-0000-4000-8000-000000000002' END::uuid,
 'Ordinary', n, CASE WHEN n=3 THEN 'urgent' WHEN n=4 THEN NULL ELSE 'high' END,
 CASE WHEN n=4 THEN NULL ELSE '00000000-0000-4000-8000-000000000003'::uuid END,
 CASE WHEN n=6 THEN now() END,n=7,CASE WHEN n=8 THEN now() END
FROM generate_series(3,8) n;
UPDATE issues SET created_at='2026-10-27 00:00:00Z'::timestamptz+sequence_id*interval '1 hour',updated_at='2026-10-27 12:00:00Z';
UPDATE issues SET external_id='external-one',external_source='source-one' WHERE sequence_id=1;
`

func TestAPIPQLFiltersBeforePaginationWithinExistingScope(t *testing.T) {
	ctx, db := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed+pqlReadSeed)
	repo, err := NewRepository(db, modelPublisher{}, "https://example.test")
	require.NoError(t, err)
	repo.now = func() time.Time { return time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC) }
	base := Request{API: true, Method: "GET", Operation: "get_api_v1_workspaces_by_slug_projects_by_project_id_issues", ActorID: testsupport.LabelActor, Slug: "demo", ProjectID: testsupport.LabelProject, Timezone: "UTC"}
	require.NoError(t, repo.AuthorizeInitial(ctx, base))
	for _, tc := range []struct {
		query string
		want  []int
	}{
		{`priority = "high" OR priority = "urgent" AND title IS EMPTY`, []int{1}},
		{`(priority = "high" OR priority = "urgent") AND title IS NOT EMPTY`, []int{1, 3}},
		{`NOT priority = "high"`, []int{2, 3}},
		{`priority IS NULL`, []int{4}},
		{`priority IS NOT NULL`, []int{1, 2, 3}},
		{`title ~ "50%_\\"`, []int{1}},
		{`title ~ "%"`, []int{1}},
		{`title IS EMPTY`, []int{2}},
		{`id ~ "CUR-"`, []int{1, 2, 3, 4}},
		{`id ~ "UR"`, []int{}},
		{`id = "CUR-3"`, []int{3}},
		{`createdBy = currentUser()`, []int{1, 2, 3}},
		{`createdBy IS NULL`, []int{4}},
		{`project = "00000000-0000-4000-8000-000000000099"`, []int{}},
		{`project != "00000000-0000-4000-8000-000000000099"`, []int{1, 2, 3, 4}},
		{`priority IN ("urgent","high")`, []int{1, 3}},
		{`priority NOT IN ("urgent","high")`, []int{2}},
		{`isDraft() OR isArchived()`, []int{}},
		{`isDraft = false AND isArchived = false`, []int{1, 2, 3, 4}},
		{`dueDate BETWEEN (daysAgo(7), today())`, []int{1}},
		{`dueDate BETWEEN "2026-10-27" AND "2026-10-27"`, []int{1}},
		{`startDate <= "2026-10-20"`, []int{1}},
		{`startDate > "2026-10-20"`, []int{}},
		{`createdAt >= "2026-10-27T02:00:00Z" AND createdAt < "2026-10-27T04:00:00Z"`, []int{2, 3}},
		{`updatedAt = now()`, []int{1, 2, 3, 4}},
		{`title ~ "' OR TRUE --"`, []int{}},
		{"", []int{1, 2, 3, 4}}, {" \t\n", []int{1, 2, 3, 4}},
	} {
		t.Run(tc.query, func(t *testing.T) {
			q := base
			q.Params = map[string]string{"pql": tc.query, "per_page": "1", "order_by": "sequence_id", "fields": "sequence_id"}
			q.Present = map[string]bool{"per_page": true}
			result, readErr := repo.Execute(ctx, q, nil, "")
			require.NoError(t, readErr)
			page := result.Body.(map[string]any)
			require.Equal(t, len(tc.want), page["total_results"])
			rows := page["results"].([]any)
			require.Len(t, rows, min(1, len(tc.want)))
			if len(tc.want) == 0 {
				return
			}
			require.Equal(t, json.Number(strconv.Itoa(tc.want[0])), rows[0].(map[string]any)["sequence_id"])
			if len(tc.want) == 1 {
				return
			}
			q.Params["cursor"] = page["next_cursor"].(string)
			q.Present["cursor"] = true
			next, nextErr := repo.Execute(ctx, q, nil, "")
			require.NoError(t, nextErr)
			nextPage := next.Body.(map[string]any)
			require.Equal(t, len(tc.want), nextPage["total_results"])
			require.Equal(t, json.Number(strconv.Itoa(tc.want[1])), nextPage["results"].([]any)[0].(map[string]any)["sequence_id"])
		})
	}
	t.Run("archived candidates remain separately scoped", func(t *testing.T) {
		q := base
		q.Operation = "get_api_v1_workspaces_by_slug_projects_by_project_id_archived_issues"
		q.Params = map[string]string{"pql": "isArchived() AND priority = \"high\"", "fields": "sequence_id"}
		result, readErr := repo.Execute(ctx, q, nil, "")
		require.NoError(t, readErr)
		require.Equal(t, 1, result.Body.(map[string]any)["total_results"])
		require.Equal(t, json.Number("6"), result.Body.(map[string]any)["results"].([]any)[0].(map[string]any)["sequence_id"])
	})
	t.Run("external lookup without a PQL predicate retains object response", func(t *testing.T) {
		q := base
		q.Params = map[string]string{"external_id": "external-one", "external_source": "source-one", "fields": "sequence_id", "pql": " "}
		result, readErr := repo.Execute(ctx, q, nil, "")
		require.NoError(t, readErr)
		require.Equal(t, map[string]any{"sequence_id": json.Number("1")}, result.Body)
	})
	t.Run("invalid and conflicting filters return safe 400 errors", func(t *testing.T) {
		for _, params := range []map[string]string{
			{"pql": `title = "private" OR unknown = "private"`},
			{"pql": `priority = "high"`, "external_id": "private"},
			{"pql": `priority = "high"`, "external_source": "private"},
			{"filters": "private"},
			{"pql": strings.Repeat(" ", 16385), "external_id": "private", "external_source": "private"},
		} {
			q := base
			q.Params = params
			_, readErr := repo.Execute(ctx, q, nil, "")
			var invalid *djangoapi.HTTPError
			require.ErrorAs(t, readErr, &invalid)
			require.Equal(t, 400, invalid.Status)
			raw, marshalErr := json.Marshal(invalid.Body)
			require.NoError(t, marshalErr)
			require.NotContains(t, string(raw), "private")
		}
	})
}
