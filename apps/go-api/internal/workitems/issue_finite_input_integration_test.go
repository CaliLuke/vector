//go:build integration

package workitems

import (
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/stretchr/testify/require"
)

func TestIssueNonfiniteOrderLeavesRowsSequencesAndEffectsUnchanged(t *testing.T) {
	ctx, db := postgresfixture.OpenSQL(t, "")
	require.NoError(t, bootstrap.MigrateSchema(ctx, db.Config().ConnString()))
	_, err := db.Exec(ctx, testsupport.MilestoneWriteSeed)
	require.NoError(t, err)
	publisher := &versionTaskRecorder{}
	repo, err := NewRepository(db, publisher, "http://localhost:8000")
	require.NoError(t, err)
	request := Request{ActorID: testsupport.LabelActor, Slug: "milestones", ProjectID: testsupport.LabelProject, Timezone: "UTC", API: true,
		Method: "POST", Operation: "post_api_v1_workspaces_by_slug_projects_by_project_id_issues"}
	_, err = repo.Execute(ctx, request, strings.NewReader(`{"name":"Existing","sort_order":"-0.25"}`), "application/json")
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(ctx, `SELECT id::text FROM issues`).Scan(&request.IssueID))
	var order float64
	require.NoError(t, db.QueryRow(ctx, `SELECT sort_order FROM issues`).Scan(&order))
	require.Equal(t, -0.25, order)
	snapshot := func() string {
		t.Helper()
		var rows string
		require.NoError(t, db.QueryRow(ctx, `SELECT jsonb_build_object(
   'issues',(SELECT jsonb_agg(to_jsonb(i) ORDER BY id) FROM issues i),
   'sequences',(SELECT jsonb_agg(to_jsonb(s) ORDER BY id) FROM issue_sequences s),
   'effects',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM work_item_effects e),
   'revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY issue_id) FROM issue_edit_revisions r))::text`).Scan(&rows))
		return rows
	}
	before := snapshot()
	for _, route := range []struct {
		name, method string
		api          bool
	}{
		{"API create", "POST", true}, {"API patch", "PATCH", true}, {"app create", "POST", false}, {"app patch", "PATCH", false}, {"app put", "PUT", false},
	} {
		for _, value := range []string{"NaN", "Infinity", "-Infinity", "1e999"} {
			t.Run(route.name+"/"+value, func(t *testing.T) {
				q := request
				q.Method, q.API = route.method, route.api
				_, writeErr := repo.Execute(ctx, q, strings.NewReader(`{"name":"Rejected","sort_order":"`+value+`"}`), "application/json")
				var failure *djangoapi.HTTPError
				require.ErrorAs(t, writeErr, &failure)
				require.Equal(t, 400, failure.Status)
				require.Equal(t, []string{"A valid number is required."}, failure.Body["sort_order"])
				require.Equal(t, before, snapshot())
				require.Empty(t, publisher.names)
			})
		}
	}
}
