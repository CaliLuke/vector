//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"
	"uuid"

	cycles "github.com/CaliLuke/go-plane/internal/cyclereads"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const cycleProject = "30000000-0000-4000-8000-000000000001"
const cycleID = "50000000-0000-4000-8000-000000000001"
const archivedCycleID = "50000000-0000-4000-8000-000000000005"
const cycleReadSchema = `
ALTER TABLE cycles ADD COLUMN id uuid PRIMARY KEY,ADD COLUMN workspace_id uuid,ADD COLUMN name text,ADD COLUMN description text DEFAULT '',
 ADD COLUMN owned_by_id uuid DEFAULT '10000000-0000-4000-8000-000000000001',ADD COLUMN start_date timestamptz,ADD COLUMN end_date timestamptz,
 ADD COLUMN archived_at timestamptz,ADD COLUMN created_at timestamptz DEFAULT '2024-01-01T00:00:00Z',ADD COLUMN updated_at timestamptz DEFAULT '2024-01-01T00:00:00Z',
 ADD COLUMN created_by_id uuid,ADD COLUMN updated_by_id uuid,ADD COLUMN view_props jsonb DEFAULT '{}',ADD COLUMN progress_snapshot jsonb DEFAULT '{}',
 ADD COLUMN logo_props jsonb DEFAULT '{}',ADD COLUMN sort_order double precision DEFAULT 100,ADD COLUMN external_source text,ADD COLUMN external_id text,
 ADD COLUMN timezone text DEFAULT 'UTC',ADD COLUMN version int DEFAULT 1;
INSERT INTO cycles(id,project_id,workspace_id,name,start_date,end_date,archived_at) VALUES
 ('50000000-0000-4000-8000-000000000001','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','A current',now()-interval '1 day',now()+interval '1 day',NULL),
 ('50000000-0000-4000-8000-000000000002','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','B upcoming',now()+interval '2 days',now()+interval '3 days',NULL),
 ('50000000-0000-4000-8000-000000000003','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','C completed',now()-interval '3 days',now()-interval '2 days',NULL),
 ('50000000-0000-4000-8000-000000000004','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','D draft',NULL,NULL,NULL),
 ('50000000-0000-4000-8000-000000000005','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','E archived',NULL,NULL,now()),
 ('50000000-0000-4000-8000-000000000006','30000000-0000-4000-8000-000000000003','20000000-0000-4000-8000-000000000002','Foreign secret',NULL,NULL,NULL);
CREATE TABLE cycle_issues(id uuid,cycle_id uuid,issue_id uuid,project_id uuid,workspace_id uuid,deleted_at timestamptz);
CREATE TABLE issues(id uuid,project_id uuid,workspace_id uuid,state_id uuid,estimate_point_id uuid,archived_at timestamptz,is_draft boolean,deleted_at timestamptz);
CREATE TABLE states(id uuid,project_id uuid,workspace_id uuid,"group" text);
CREATE TABLE estimate_points(id uuid,project_id uuid,workspace_id uuid,key int);
INSERT INTO states VALUES('60000000-0000-4000-8000-000000000001','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','completed');
INSERT INTO estimate_points VALUES('70000000-0000-4000-8000-000000000001','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001',3);
INSERT INTO issues VALUES('80000000-0000-4000-8000-000000000001','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','60000000-0000-4000-8000-000000000001','70000000-0000-4000-8000-000000000001',NULL,false,NULL);
INSERT INTO cycle_issues SELECT gen_random_uuid(),id,'80000000-0000-4000-8000-000000000001',project_id,workspace_id,NULL FROM cycles WHERE id IN('50000000-0000-4000-8000-000000000001','50000000-0000-4000-8000-000000000005');
`

func addCycleFixture(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	addMCPProjectFixture(t, pool)
	_, err := pool.Exec(t.Context(), cycleReadSchema)
	require.NoError(t, err)
}
func cyclePageWire(t *testing.T, result rpcResult) map[string]any {
	t.Helper()
	require.False(t, result.IsError, result.Content)
	var page map[string]any
	require.NoError(t, json.Unmarshal(result.StructuredContent, &page))
	return page
}

func TestCycleReadsMatchPinnedContractsAcrossCredentialsAndProtocols(t *testing.T) {
	f, pool, auth, _ := oauthFixture(t)
	addCycleFixture(t, pool)
	pat, oauth := f.handler(t, nil), oauthHandler(t, f, auth.Authenticate)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, mode := range []string{"PAT", "OAuth"} {
			t.Run(version+"/"+mode, func(t *testing.T) {
				call := func(input string) rpcResult {
					if mode == "OAuth" {
						return oauthToolCall(t, oauth, version, "oauth-user", "workspace-one", "cycle", input, 200)
					}
					return rpcVersion(t, pat, "test-one", "tools/call", cycleCall(input), version)
				}
				page := cyclePageWire(t, call(`{"action":"list","project_id":"`+cycleProject+`","order_by":"name"}`))
				require.Equal(t, float64(4), page["total_results"])
				require.Equal(t, "1000:1:0", page["next_cursor"])
				row := page["results"].([]any)[0].(map[string]any)
				require.Equal(t, "A current", row["name"])
				require.Equal(t, cycleProject, row["project"])
				require.Contains(t, row, "external_source")
				require.Nil(t, row["external_source"])
				require.NotContains(t, row, "total_issues")
				require.ElementsMatch(t, strings.Fields("id created_at updated_at deleted_at name description start_date end_date view_props sort_order external_source external_id progress_snapshot archived_at logo_props timezone version created_by updated_by project workspace owned_by"), slices.Collect(maps.Keys(row)))
				zero := cyclePageWire(t, call(`{"action":"list","project_id":"`+cycleProject+`","per_page":0}`))
				require.Equal(t, "1000:1:0", zero["next_cursor"])
				next := cyclePageWire(t, call(`{"action":"list","project_id":"`+cycleProject+`","per_page":1,"cursor":"1:1:0","order_by":"name"}`))
				require.Equal(t, "B upcoming", next["results"].([]any)[0].(map[string]any)["name"])
				require.Equal(t, true, next["prev_page_results"])
				for _, tc := range []struct {
					status string
					count  float64
				}{{"current", 1}, {"upcoming", 1}, {"completed", 1}, {"draft", 1}, {"incomplete", 3}} {
					filtered := cyclePageWire(t, call(`{"action":"list","project_id":"`+cycleProject+`","status":"`+tc.status+`"}`))
					require.Equal(t, tc.count, filtered["total_results"], tc.status)
				}
				detail := cyclePageWire(t, call(`{"action":"retrieve","project_id":"`+cycleProject+`","cycle_id":"`+cycleID+`"}`))
				require.Equal(t, float64(1), detail["total_issues"])
				require.Equal(t, float64(1), detail["completed_issues"])
				require.Contains(t, detail, "total_estimates")
				require.Nil(t, detail["total_estimates"])
				require.ElementsMatch(t, strings.Fields("id total_issues cancelled_issues completed_issues started_issues unstarted_issues backlog_issues total_estimates completed_estimates started_estimates created_at updated_at deleted_at name description start_date end_date view_props sort_order external_source external_id progress_snapshot archived_at logo_props timezone version created_by updated_by project workspace owned_by"), slices.Collect(maps.Keys(detail)))
				archived := cyclePageWire(t, call(`{"action":"list","project_id":"`+cycleProject+`","archived":true,"status":"intentionally ignored"}`))
				archivedRow := archived["results"].([]any)[0].(map[string]any)
				require.Equal(t, archivedCycleID, archivedRow["id"])
				require.Equal(t, float64(3), archivedRow["total_estimates"])
				require.Equal(t, float64(3), archivedRow["completed_estimates"])
				require.Nil(t, archivedRow["started_estimates"])
				require.ElementsMatch(t, slices.Collect(maps.Keys(detail)), slices.Collect(maps.Keys(archivedRow)))
				for _, id := range []string{archivedCycleID, "50000000-0000-4000-8000-000000000006", "50000000-0000-4000-8000-000000000099"} {
					denied := call(`{"action":"retrieve","project_id":"` + cycleProject + `","cycle_id":"` + id + `"}`)
					require.True(t, denied.IsError)
					require.Contains(t, denied.Content[0].Text, "[not_found]")
				}
				invalid := call(`{"action":"list","project_id":"` + cycleProject + `","status":"invalid"}`)
				require.True(t, invalid.IsError)
				require.Contains(t, invalid.Content[0].Text, "[invalid_params]")
			})
		}
	}
}

type changingCycleReader struct {
	*cycles.Repository
	change func()
}

func (r changingCycleReader) ReadAPIV1(ctx context.Context, user string, request cycles.APIV1Request) (any, error) {
	if r.change != nil {
		r.change()
	}
	return r.Repository.ReadAPIV1(ctx, user, request)
}

func TestCycleDataRechecksScopeAfterActionAdmission(t *testing.T) {
	for _, mutation := range []string{
		`UPDATE users SET is_active=false`,
		`UPDATE workspaces SET deleted_at=now()`,
		`UPDATE workspace_members SET is_active=false`,
		`UPDATE workspace_members SET deleted_at=now()`,
		`UPDATE workspace_members SET role=0`,
		`UPDATE project_members SET is_active=false`,
		`UPDATE project_members SET deleted_at=now()`,
		`UPDATE project_members SET workspace_id='20000000-0000-4000-8000-000000000002'`,
		`UPDATE projects SET deleted_at=now()`,
		`UPDATE workspaces SET slug='renamed' WHERE slug='workspace-one'; UPDATE workspaces SET slug='workspace-one' WHERE slug='workspace-two'`,
	} {
		t.Run(mutation, func(t *testing.T) {
			f, pool, auth, _ := oauthFixture(t)
			addCycleFixture(t, pool)
			f.service = databaseService(t, pool, func() { _, err := pool.Exec(t.Context(), mutation); require.NoError(t, err) })
			h := oauthHandler(t, f, auth.Authenticate)
			result := oauthToolCall(t, h, "2026-07-28", "oauth-user", "workspace-one", "cycle", `{"action":"list","project_id":"`+cycleProject+`"}`, 200)
			require.True(t, result.IsError, result.Content)
			require.Contains(t, result.Content[0].Text, "[forbidden]")
			require.Empty(t, result.StructuredContent)
		})
	}
}

func TestCycleLegacyStatusAliasRetainsFullListShape(t *testing.T) {
	f, pool := databaseFixture(t)
	_ = f
	addCycleFixture(t, pool)
	repo, err := cycles.NewRepository(pool)
	require.NoError(t, err)
	request := cycles.APIV1Request{Operation: "list", Params: map[string]string{"project_id": cycleProject, "slug": "workspace-one", "cycle_view": "current", "status": "draft"}, Present: map[string]bool{"status": true}}
	value, err := repo.ReadAPIV1(t.Context(), "10000000-0000-4000-8000-000000000001", request)
	require.NoError(t, err)
	require.Equal(t, 1, value.(map[string]any)["total_count"])
	request.Params["status"] = "current"
	value, err = repo.ReadAPIV1(t.Context(), "10000000-0000-4000-8000-000000000001", request)
	require.NoError(t, err)
	require.Len(t, value.([]any), 1)
	workspace := uuid.MustParse("20000000-0000-4000-8000-000000000001")
	request.ExpectedWorkspaceID = &workspace
	_, err = repo.ReadAPIV1(t.Context(), "10000000-0000-4000-8000-000000000001", request)
	require.ErrorIs(t, err, cycles.ErrForbidden)
}

func TestCycleWorkspaceSelectionAndEmptyPageRemainDistinctFromDenial(t *testing.T) {
	f, pool, auth, _ := oauthFixture(t)
	addCycleFixture(t, pool)
	h := oauthHandler(t, f, auth.Authenticate)
	missing := oauthToolCall(t, h, "2026-07-28", "oauth-user", "", "cycle", `{"action":"list","project_id":"`+cycleProject+`"}`, 200)
	require.True(t, missing.IsError)
	require.Contains(t, missing.Content[0].Text, "[invalid_params]")
	selected := oauthToolCall(t, h, "2026-07-28", "oauth-user", "", "cycle", `{"action":"list","project_id":"`+cycleProject+`","workspace_slug":"workspace-one"}`, 200)
	require.False(t, selected.IsError, selected.Content)
	foreign := oauthToolCall(t, h, "2026-07-28", "oauth-user", "workspace-one", "cycle", `{"action":"list","project_id":"30000000-0000-4000-8000-000000000003"}`, 200)
	require.True(t, foreign.IsError)
	require.Contains(t, foreign.Content[0].Text, "[forbidden]")
	_, err := pool.Exec(t.Context(), `DELETE FROM cycles`)
	require.NoError(t, err)
	for _, archived := range []string{"false", "true"} {
		result := oauthToolCall(t, h, "2026-07-28", "oauth-user", "workspace-one", "cycle", `{"action":"list","project_id":"`+cycleProject+`","archived":`+archived+`}`, 200)
		page := cyclePageWire(t, result)
		require.Equal(t, float64(0), page["total_results"])
		require.Equal(t, []any{}, page["results"])
	}
}

func TestCycleMetricsExcludeForeignRelationships(t *testing.T) {
	f, pool := databaseFixture(t)
	addCycleFixture(t, pool)
	_, err := pool.Exec(t.Context(), `
 INSERT INTO issues(id,project_id,workspace_id,state_id,estimate_point_id,is_draft) VALUES
 ('80000000-0000-4000-8000-000000000002','30000000-0000-4000-8000-000000000003','20000000-0000-4000-8000-000000000002','60000000-0000-4000-8000-000000000001','70000000-0000-4000-8000-000000000001',false);
 INSERT INTO cycle_issues SELECT gen_random_uuid(),'50000000-0000-4000-8000-000000000005',id,'30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001',NULL FROM issues WHERE id='80000000-0000-4000-8000-000000000002';
 INSERT INTO cycle_issues VALUES(gen_random_uuid(),'50000000-0000-4000-8000-000000000005','80000000-0000-4000-8000-000000000001','30000000-0000-4000-8000-000000000003','20000000-0000-4000-8000-000000000002',NULL);
 INSERT INTO project_members VALUES('30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000001',15,true,NULL,now(),NULL);
 `)
	require.NoError(t, err)
	result := rpc(t, f.handler(t, nil), "test-one", "tools/call", cycleCall(`{"action":"list","project_id":"`+cycleProject+`","archived":true}`))
	page := cyclePageWire(t, result)
	row := page["results"].([]any)[0].(map[string]any)
	require.Equal(t, float64(1), row["total_issues"])
	require.Equal(t, float64(3), row["total_estimates"])
}

func TestEveryCycleProjectionRechecksPrimaryMembership(t *testing.T) {
	_, pool := databaseFixture(t)
	addCycleFixture(t, pool)
	tx, err := pool.Begin(t.Context())
	require.NoError(t, err)
	defer func() { require.NoError(t, tx.Rollback(context.Background())) }()
	repo, err := cycles.NewRepository(tx)
	require.NoError(t, err)
	workspace := uuid.MustParse("20000000-0000-4000-8000-000000000001")
	for _, op := range []string{"lite", "archived", "detail"} {
		request := cycles.APIV1Request{Operation: op, ExpectedWorkspaceID: &workspace, Params: map[string]string{"project_id": cycleProject, "slug": "workspace-one", "pk": cycleID}}
		_, err = repo.ReadAPIV1(t.Context(), "10000000-0000-4000-8000-000000000001", request)
		require.NoError(t, err)
		_, err = tx.Exec(t.Context(), `UPDATE project_members SET is_active=false`)
		require.NoError(t, err)
		_, err = repo.ReadAPIV1(t.Context(), "10000000-0000-4000-8000-000000000001", request)
		require.ErrorIs(t, err, cycles.ErrForbidden)
		_, err = tx.Exec(t.Context(), `UPDATE project_members SET is_active=true`)
		require.NoError(t, err)
	}
}

func TestCycleTimestampsFollowUserTimezoneWithoutChangingSnapshots(t *testing.T) {
	f, pool, auth, _ := oauthFixture(t)
	addCycleFixture(t, pool)
	_, err := pool.Exec(t.Context(), `UPDATE users SET user_timezone='Asia/Kolkata'; UPDATE cycles SET progress_snapshot='{"created_at":"2024-01-01T00:00:00Z"}'`)
	require.NoError(t, err)
	pat, oauth := f.handler(t, nil), oauthHandler(t, f, auth.Authenticate)
	input := `{"action":"retrieve","project_id":"` + cycleProject + `","cycle_id":"` + cycleID + `"}`
	for _, result := range []rpcResult{rpc(t, pat, "test-one", "tools/call", cycleCall(input)), oauthToolCall(t, oauth, "2026-07-28", "oauth-user", "workspace-one", "cycle", input, 200)} {
		record := cyclePageWire(t, result)
		require.Equal(t, "2024-01-01T05:30:00+05:30", record["created_at"])
		require.Equal(t, map[string]any{"created_at": "2024-01-01T00:00:00Z"}, record["progress_snapshot"])
	}
}
