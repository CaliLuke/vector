//go:build integration

package milestones_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

const firstMilestone = "30000000-0000-4000-8000-000000000001"
const laterMilestone = "30000000-0000-4000-8000-000000000002"
const undatedMilestone = "30000000-0000-4000-8000-000000000003"

type readFixture struct {
	ctx   context.Context
	db    *pgx.Conn
	repo  *milestones.Repository
	scope milestones.Scope
}

func newReadFixture(t *testing.T) readFixture {
	t.Helper()
	ctx, db := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed+milestoneReadSeed)
	repo, err := milestones.NewRepository(db)
	require.NoError(t, err)
	return readFixture{ctx, db, repo, milestones.Scope{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), ProjectID: uuid.MustParse(testsupport.LabelProject), WorkspaceSlug: "demo", Timezone: "America/Los_Angeles"}}
}

func TestMilestoneReadRecordsAndPagesMatchNativeContract(t *testing.T) {
	f := newReadFixture(t)
	query := djangoapi.Query{Params: map[string]string{"per_page": "2"}, Present: map[string]bool{"per_page": true}}
	page, err := f.repo.List(f.ctx, f.scope, query)
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.Equal(t, []string{firstMilestone, laterMilestone}, []string{page.Results[0].ID, page.Results[1].ID})
	wire, err := json.Marshal(page.Envelope())
	require.NoError(t, err)
	require.JSONEq(t, `{"count":2,"total_count":3,"total_results":3,"total_pages":2,"next_cursor":"2:1:0","prev_cursor":"2:-1:1","next_page_results":true,"prev_page_results":false,"grouped_by":null,"sub_grouped_by":null,"extra_stats":null,"results":[{"id":"30000000-0000-4000-8000-000000000001","title":"First","target_date":"2026-01-01","external_source":" source ","external_id":"identity","created_at":"2026-01-01T01:04:05.123456-08:00","updated_at":"2026-01-01T01:04:05.123456-08:00"},{"id":"30000000-0000-4000-8000-000000000002","title":"Later","target_date":"2026-03-01","external_source":null,"external_id":null,"created_at":"2026-01-01T01:04:05.123456-08:00","updated_at":"2026-01-01T01:04:05.123456-08:00"}]}`, string(wire))
	query.Params["cursor"], query.Present["cursor"] = page.Envelope()["next_cursor"].(string), true
	page, err = f.repo.List(f.ctx, f.scope, query)
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.Len(t, page.Results, 1)
	require.Equal(t, undatedMilestone, page.Results[0].ID)
	require.Nil(t, page.Results[0].TargetDate)
	query.Params["cursor"] = "2:10:0"
	page, err = f.repo.List(f.ctx, f.scope, query)
	require.NoError(t, err)
	require.Empty(t, page.Results)
	require.Equal(t, 3, page.Total)
	require.NotNil(t, page.Results)
	record, err := f.repo.Retrieve(f.ctx, f.scope, uuid.MustParse(firstMilestone), false)
	require.NoError(t, err)
	require.Nil(t, record.Revision)
	record, err = f.repo.Retrieve(f.ctx, f.scope, uuid.MustParse(firstMilestone), true)
	require.NoError(t, err)
	require.Equal(t, "milestone:"+firstMilestone+":7", *record.Revision)
	page, err = f.repo.List(f.ctx, f.scope, djangoapi.Query{})
	require.NoError(t, err)
	require.Equal(t, 1000, page.Limit)
}

func TestMilestoneAssociationReadsRetainArchivedAndIntakeItems(t *testing.T) {
	f := newReadFixture(t)
	query := djangoapi.Query{Params: map[string]string{"per_page": "1"}, Present: map[string]bool{"per_page": true}}
	page, err := f.repo.ListWorkItems(f.ctx, f.scope, uuid.MustParse(firstMilestone), query)
	require.NoError(t, err)
	require.Equal(t, 2, page.Total)
	wire, err := json.Marshal(page.Results)
	require.NoError(t, err)
	require.JSONEq(t, `[{"id":"40000000-0000-4000-8000-000000000001","issue":"00000000-0000-4000-8000-000000000004","milestone":"30000000-0000-4000-8000-000000000001"}]`, string(wire))
	query.Params["cursor"], query.Present["cursor"] = page.Envelope()["next_cursor"].(string), true
	page, err = f.repo.ListWorkItems(f.ctx, f.scope, uuid.MustParse(firstMilestone), query)
	require.NoError(t, err)
	require.Equal(t, 2, page.Total)
	require.Equal(t, []milestones.Link{{ID: "40000000-0000-4000-8000-000000000002", Issue: testsupport.LabelEmptyIssue, Milestone: firstMilestone}}, page.Results)
	empty, err := f.repo.ListWorkItems(f.ctx, f.scope, uuid.MustParse(laterMilestone), djangoapi.Query{})
	require.NoError(t, err)
	require.Zero(t, empty.Total)
	require.NotNil(t, empty.Results)
	require.Empty(t, empty.Results)
}

func TestMilestoneReadsMaskTargetsAndRecheckScope(t *testing.T) {
	f := newReadFixture(t)
	for _, id := range []string{"30000000-0000-4000-8000-000000000004", "30000000-0000-4000-8000-000000000005", "30000000-0000-4000-8000-000000000006", "30000000-0000-4000-8000-000000000099"} {
		_, err := f.repo.Retrieve(f.ctx, f.scope, uuid.MustParse(id), true)
		require.ErrorIs(t, err, milestones.ErrNotFound)
		_, err = f.repo.ListWorkItems(f.ctx, f.scope, uuid.MustParse(id), djangoapi.Query{})
		require.ErrorIs(t, err, milestones.ErrNotFound)
	}
	wrongWorkspace := f.scope
	wrongWorkspace.WorkspaceID = uuid.MustParse("00000000-0000-4000-8000-000000000099")
	_, err := f.repo.List(f.ctx, wrongWorkspace, djangoapi.Query{})
	require.ErrorIs(t, err, milestones.ErrForbidden)
	_, err = f.db.Exec(f.ctx, `UPDATE workspaces SET slug='moved' WHERE id=$1`, f.scope.WorkspaceID)
	require.NoError(t, err)
	_, err = f.repo.Retrieve(f.ctx, f.scope, uuid.MustParse(firstMilestone), false)
	require.ErrorIs(t, err, milestones.ErrForbidden)
	f.scope.WorkspaceSlug = "moved"
	_, err = f.repo.Retrieve(f.ctx, f.scope, uuid.MustParse(firstMilestone), false)
	require.NoError(t, err)
	_, err = f.db.Exec(f.ctx, `UPDATE project_members SET is_active=false`)
	require.NoError(t, err)
	_, err = f.repo.List(f.ctx, f.scope, djangoapi.Query{})
	require.ErrorIs(t, err, milestones.ErrForbidden)
	_, err = f.repo.Retrieve(f.ctx, f.scope, uuid.MustParse(firstMilestone), false)
	require.ErrorIs(t, err, milestones.ErrForbidden)
	_, err = f.repo.ListWorkItems(f.ctx, f.scope, uuid.MustParse(firstMilestone), djangoapi.Query{})
	require.ErrorIs(t, err, milestones.ErrForbidden)
}

func TestMilestoneReadsRejectInvalidPagesWithoutWeakeningGuestPolicy(t *testing.T) {
	f := newReadFixture(t)
	for _, params := range []map[string]string{{"per_page": ""}, {"per_page": "0"}, {"per_page": "-1"}, {"per_page": "1001"}, {"cursor": ""}, {"cursor": "bad"}, {"cursor": "1:-1:0"}, {"cursor": "1:9223372036854775807:0"}, {"cursor": "1:9999999999999999999999999:0"}} {
		present := map[string]bool{}
		for key := range params {
			present[key] = true
		}
		query := djangoapi.Query{Params: params, Present: present}
		_, err := f.repo.List(f.ctx, f.scope, query)
		require.ErrorIs(t, err, milestones.ErrInvalidInput)
		_, err = f.repo.ListWorkItems(f.ctx, f.scope, uuid.MustParse(firstMilestone), query)
		require.ErrorIs(t, err, milestones.ErrInvalidInput)
	}
	_, err := f.db.Exec(f.ctx, `UPDATE project_members SET role=5`)
	require.NoError(t, err)
	_, err = f.repo.List(f.ctx, f.scope, djangoapi.Query{})
	require.ErrorIs(t, err, milestones.ErrForbidden)
	_, err = f.db.Exec(f.ctx, `UPDATE projects SET guest_view_all_features=true`)
	require.NoError(t, err)
	_, err = f.repo.ListWorkItems(f.ctx, f.scope, uuid.MustParse(firstMilestone), djangoapi.Query{})
	require.NoError(t, err)
	_, err = f.db.Exec(f.ctx, `UPDATE projects SET archived_at=now()`)
	require.NoError(t, err)
	_, err = f.repo.ListWorkItems(f.ctx, f.scope, uuid.MustParse(firstMilestone), djangoapi.Query{})
	require.ErrorIs(t, err, milestones.ErrForbidden)
}
