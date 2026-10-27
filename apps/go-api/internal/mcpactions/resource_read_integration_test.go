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

	"github.com/CaliLuke/go-plane/internal/mcpactions"
	modules "github.com/CaliLuke/go-plane/internal/modulereads"
	labels "github.com/CaliLuke/go-plane/internal/projectlabelreads"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const moduleID = "50000000-0000-4000-8000-000000000101"
const archivedModuleID = "50000000-0000-4000-8000-000000000105"
const labelID = "50000000-0000-4000-8000-000000000201"
const resourceReadSchema = `
ALTER TABLE modules ADD COLUMN id uuid PRIMARY KEY,ADD COLUMN workspace_id uuid,ADD COLUMN name text,ADD COLUMN description text DEFAULT '',
 ADD COLUMN start_date date,ADD COLUMN target_date date,ADD COLUMN archived_at timestamptz,
 ADD COLUMN created_at timestamptz DEFAULT '2024-01-01T00:00:00Z',ADD COLUMN updated_at timestamptz DEFAULT '2024-01-01T00:00:00Z',
 ADD COLUMN created_by_id uuid,ADD COLUMN updated_by_id uuid,ADD COLUMN description_text text,ADD COLUMN description_html text,
 ADD COLUMN status text DEFAULT 'planned',ADD COLUMN lead_id uuid,ADD COLUMN view_props jsonb DEFAULT '{}',
 ADD COLUMN logo_props jsonb DEFAULT '{}',ADD COLUMN sort_order double precision DEFAULT 100,ADD COLUMN external_source text,ADD COLUMN external_id text;
INSERT INTO modules(id,project_id,workspace_id,name,created_at,archived_at) VALUES
 ('50000000-0000-4000-8000-000000000101','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','A module','2024-01-02',NULL),
 ('50000000-0000-4000-8000-000000000102','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','B module','2024-01-01',NULL),
 ('50000000-0000-4000-8000-000000000105','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','Archived module','2024-01-01',now()),
 ('50000000-0000-4000-8000-000000000106','30000000-0000-4000-8000-000000000003','20000000-0000-4000-8000-000000000002','Foreign secret','2024-01-01',NULL);
CREATE TABLE module_members(module_id uuid,member_id uuid,project_id uuid,workspace_id uuid,deleted_at timestamptz);
INSERT INTO module_members VALUES('50000000-0000-4000-8000-000000000101','10000000-0000-4000-8000-000000000001','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001',NULL);
CREATE TABLE module_issues(id uuid,module_id uuid,issue_id uuid,project_id uuid,workspace_id uuid,deleted_at timestamptz);
INSERT INTO module_issues SELECT gen_random_uuid(),id,'80000000-0000-4000-8000-000000000001',project_id,workspace_id,NULL FROM modules WHERE id IN('50000000-0000-4000-8000-000000000101','50000000-0000-4000-8000-000000000105');
CREATE TABLE labels(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,name text,description text,color text,sort_order double precision DEFAULT 100,
 parent_id uuid,external_source text,external_id text,created_by_id uuid,updated_by_id uuid,deleted_at timestamptz,
 created_at timestamptz,updated_at timestamptz DEFAULT '2024-01-01T00:00:00Z');
INSERT INTO labels(id,project_id,workspace_id,name,color,created_at) VALUES
 ('50000000-0000-4000-8000-000000000201','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','A label','#123456','2024-01-02'),
 ('50000000-0000-4000-8000-000000000202','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','B label','#654321','2024-01-01'),
 ('50000000-0000-4000-8000-000000000206','30000000-0000-4000-8000-000000000003','20000000-0000-4000-8000-000000000002','Foreign secret','#000000','2024-01-01');
`

func addResourceReadFixture(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	addCycleFixture(t, pool)
	_, err := pool.Exec(t.Context(), resourceReadSchema)
	require.NoError(t, err)
}

func TestModuleAndLabelReadsMatchPinnedContracts(t *testing.T) {
	f, pool, auth, _ := oauthFixture(t)
	addResourceReadFixture(t, pool)
	pat, oauth := f.handler(t, nil), oauthHandler(t, f, auth.Authenticate)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, mode := range []string{"PAT", "OAuth"} {
			t.Run(version+"/"+mode, func(t *testing.T) {
				call := func(tool, input string) rpcResult {
					if mode == "OAuth" {
						return oauthToolCall(t, oauth, version, "oauth-user", "workspace-one", tool, input, 200)
					}
					return rpcVersion(t, pat, "test-one", "tools/call", resourceCall(tool, input), version)
				}
				for _, tool := range []string{"module", "label"} {
					t.Run(tool, func(t *testing.T) {
						page := cyclePageWire(t, call(tool, `{"action":"list","project_id":"`+cycleProject+`","per_page":1,"cursor":""}`))
						require.Equal(t, float64(2), page["total_count"])
						require.Equal(t, "1:1:0", page["next_cursor"])
						require.Equal(t, true, page["next_page_results"])
						row := page["results"].([]any)[0].(map[string]any)
						require.Equal(t, "A "+tool, row["name"])
						require.Nil(t, row["external_source"])
						require.Equal(t, "2024-01-02T00:00:00Z", row["created_at"])
						require.NotContains(t, string(mustJSON(t, page)), "Foreign secret")
						next := cyclePageWire(t, call(tool, `{"action":"list","project_id":"`+cycleProject+`","per_page":1,"cursor":"1:1:0"}`))
						require.Equal(t, "B "+tool, next["results"].([]any)[0].(map[string]any)["name"])
						require.Equal(t, true, next["prev_page_results"])
						zero := cyclePageWire(t, call(tool, `{"action":"list","project_id":"`+cycleProject+`","per_page":0}`))
						require.Equal(t, "1000:1:0", zero["next_cursor"])
					})
				}
				modulePage := cyclePageWire(t, call("module", `{"action":"list","project_id":"`+cycleProject+`","order_by":"name"}`))
				require.Len(t, modulePage, 12)
				module := modulePage["results"].([]any)[0].(map[string]any)
				require.ElementsMatch(t, strings.Fields("id created_at updated_at deleted_at name description description_text description_html start_date target_date status view_props sort_order external_source external_id archived_at logo_props created_by updated_by project workspace lead members"), slices.Collect(maps.Keys(module)))
				require.Equal(t, []any{"10000000-0000-4000-8000-000000000001"}, module["members"])
				detail := cyclePageWire(t, call("module", `{"action":"retrieve","project_id":"`+cycleProject+`","module_id":"`+moduleID+`"}`))
				require.Len(t, detail, 29)
				require.Equal(t, float64(1), detail["total_issues"])
				require.Equal(t, float64(1), detail["completed_issues"])
				require.Equal(t, module["members"], detail["members"])
				archived := cyclePageWire(t, call("module", `{"action":"list","project_id":"`+cycleProject+`","archived":true}`))
				require.Equal(t, float64(1), archived["total_count"])
				require.Equal(t, archivedModuleID, archived["results"].([]any)[0].(map[string]any)["id"])
				labelPage := cyclePageWire(t, call("label", `{"action":"list","project_id":"`+cycleProject+`"}`))
				require.ElementsMatch(t, strings.Fields("results total_count count next_cursor prev_cursor next_page_results prev_page_results"), slices.Collect(maps.Keys(labelPage)))
				label := cyclePageWire(t, call("label", `{"action":"retrieve","project_id":"`+cycleProject+`","label_id":"`+labelID+`"}`))
				require.ElementsMatch(t, strings.Fields("id created_at updated_at deleted_at name description color sort_order external_source external_id created_by updated_by workspace project parent"), slices.Collect(maps.Keys(label)))
				require.Equal(t, labelPage["results"].([]any)[0], label)
			})
		}
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	require.NoError(t, err)
	return body
}

type changingModuleReader struct {
	*modules.Repository
	change func()
}

func (r changingModuleReader) ReadAPIV1(ctx context.Context, user string, request modules.APIV1Request) (any, error) {
	r.change()
	return r.Repository.ReadAPIV1(ctx, user, request)
}

type changingLabelReader struct {
	*labels.Repository
	change func()
}

func (r changingLabelReader) ReadAPIV1(ctx context.Context, user string, request labels.APIV1Request) (any, error) {
	r.change()
	return r.Repository.ReadAPIV1(ctx, user, request)
}

func TestModuleAndLabelScopeIsCheckedInDataSnapshot(t *testing.T) {
	for _, tool := range []string{"module", "label"} {
		for _, mutation := range []string{
			`UPDATE users SET is_active=false`, `UPDATE workspaces SET deleted_at=now()`,
			`UPDATE workspace_members SET is_active=false`, `UPDATE workspace_members SET deleted_at=now()`, `UPDATE workspace_members SET role=0`,
			`UPDATE project_members SET is_active=false`, `UPDATE project_members SET deleted_at=now()`,
			`UPDATE project_members SET workspace_id='20000000-0000-4000-8000-000000000002'`, `UPDATE projects SET deleted_at=now()`,
			`UPDATE workspaces SET slug='renamed' WHERE slug='workspace-one'; UPDATE workspaces SET slug='workspace-one' WHERE slug='workspace-two'`,
		} {
			t.Run(tool+"/"+mutation, func(t *testing.T) {
				f, pool, auth, _ := oauthFixture(t)
				addResourceReadFixture(t, pool)
				moduleReads, err := modules.NewRepository(pool)
				require.NoError(t, err)
				labelReads, err := labels.NewRepository(pool)
				require.NoError(t, err)
				change := func() { _, changeErr := pool.Exec(t.Context(), mutation); require.NoError(t, changeErr) }
				// The wrapper changes primary data after action admission and immediately before the resource query.
				apps := databaseApplications(t, pool)
				apps.Modules, apps.Labels = changingModuleReader{moduleReads, change}, changingLabelReader{labelReads, change}
				f.service, err = mcpactions.New(apps)
				require.NoError(t, err)
				result := oauthToolCall(t, oauthHandler(t, f, auth.Authenticate), "2026-07-28", "oauth-user", "workspace-one", tool, `{"action":"list","project_id":"`+cycleProject+`"}`, 200)
				require.True(t, result.IsError, result.Content)
				require.Contains(t, result.Content[0].Text, "[forbidden]")
				require.Empty(t, result.StructuredContent)
			})
		}
	}
}

func TestModuleAndLabelArchiveMissingAndEmptyPolicies(t *testing.T) {
	f, pool, auth, _ := oauthFixture(t)
	addResourceReadFixture(t, pool)
	h := oauthHandler(t, f, auth.Authenticate)
	call := func(tool, input string) rpcResult {
		return oauthToolCall(t, h, "2026-07-28", "oauth-user", "workspace-one", tool, input, 200)
	}
	for _, tc := range []struct{ tool, id string }{{"module", archivedModuleID}, {"module", "50000000-0000-4000-8000-000000000106"}, {"label", "50000000-0000-4000-8000-000000000206"}, {"label", "50000000-0000-4000-8000-000000000299"}} {
		result := call(tc.tool, `{"action":"retrieve","project_id":"`+cycleProject+`","`+tc.tool+`_id":"`+tc.id+`"}`)
		require.True(t, result.IsError)
		require.Contains(t, result.Content[0].Text, "[not_found]")
	}
	_, err := pool.Exec(t.Context(), `UPDATE projects SET archived_at=now() WHERE id='30000000-0000-4000-8000-000000000001'`)
	require.NoError(t, err)
	require.Equal(t, float64(2), cyclePageWire(t, call("module", `{"action":"list","project_id":"`+cycleProject+`"}`))["total_count"])
	require.Empty(t, cyclePageWire(t, call("label", `{"action":"list","project_id":"`+cycleProject+`"}`))["results"])
	_, err = pool.Exec(t.Context(), `UPDATE modules SET deleted_at=now(); UPDATE projects SET archived_at=NULL; UPDATE labels SET deleted_at=now()`)
	require.NoError(t, err)
	for _, tool := range []string{"module", "label"} {
		require.Empty(t, cyclePageWire(t, call(tool, `{"action":"list","project_id":"`+cycleProject+`"}`))["results"])
		missing := oauthToolCall(t, h, "2026-07-28", "oauth-user", "", tool, `{"action":"list","project_id":"`+cycleProject+`"}`, 200)
		require.True(t, missing.IsError)
		require.Contains(t, missing.Content[0].Text, "[invalid_params]")
		explicit := oauthToolCall(t, h, "2026-07-28", "oauth-user", "", tool, `{"action":"list","project_id":"`+cycleProject+`","workspace_slug":"workspace-one"}`, 200)
		require.False(t, explicit.IsError, explicit.Content)
	}
	workspace := uuid.MustParse("20000000-0000-4000-8000-000000000001")
	moduleRepo, err := modules.NewRepository(pool)
	require.NoError(t, err)
	_, err = moduleRepo.ReadAPIV1(t.Context(), "10000000-0000-4000-8000-000000000001", modules.APIV1Request{Operation: "issues", ExpectedWorkspaceID: &workspace})
	require.ErrorIs(t, err, modules.ErrForbidden)
	labelRepo, err := labels.NewRepository(pool)
	require.NoError(t, err)
	_, err = labelRepo.ReadAPIV1(t.Context(), "10000000-0000-4000-8000-000000000001", labels.APIV1Request{Operation: "other", ExpectedWorkspaceID: &workspace})
	require.ErrorIs(t, err, labels.ErrForbidden)
}

func TestModuleRelationshipsCannotCrossTenantBoundaries(t *testing.T) {
	f, pool := databaseFixture(t)
	addResourceReadFixture(t, pool)
	_, err := pool.Exec(t.Context(), `
 INSERT INTO module_members VALUES
 ('50000000-0000-4000-8000-000000000101','10000000-0000-4000-8000-000000000002','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000002',NULL),
 ('50000000-0000-4000-8000-000000000101','10000000-0000-4000-8000-000000000002','30000000-0000-4000-8000-000000000003','20000000-0000-4000-8000-000000000001',NULL),
 ('50000000-0000-4000-8000-000000000101','10000000-0000-4000-8000-000000000002','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001',now());
 INSERT INTO states VALUES('60000000-0000-4000-8000-000000000002','30000000-0000-4000-8000-000000000003','20000000-0000-4000-8000-000000000002','started');
 INSERT INTO issues(id,project_id,workspace_id,state_id,is_draft) VALUES
 ('80000000-0000-4000-8000-000000000002','30000000-0000-4000-8000-000000000003','20000000-0000-4000-8000-000000000002','60000000-0000-4000-8000-000000000001',false),
 ('80000000-0000-4000-8000-000000000003','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','60000000-0000-4000-8000-000000000002',false);
 INSERT INTO module_issues SELECT gen_random_uuid(),'50000000-0000-4000-8000-000000000101',id,'30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001',NULL FROM issues WHERE id IN('80000000-0000-4000-8000-000000000002','80000000-0000-4000-8000-000000000003');
 INSERT INTO module_issues VALUES(gen_random_uuid(),'50000000-0000-4000-8000-000000000101','80000000-0000-4000-8000-000000000001','30000000-0000-4000-8000-000000000003','20000000-0000-4000-8000-000000000002',NULL);
 UPDATE modules SET workspace_id='20000000-0000-4000-8000-000000000001' WHERE id='50000000-0000-4000-8000-000000000106';
 `)
	require.NoError(t, err)
	h := f.handler(t, nil)
	detail := cyclePageWire(t, rpc(t, h, "test-one", "tools/call", resourceCall("module", `{"action":"retrieve","project_id":"`+cycleProject+`","module_id":"`+moduleID+`"}`)))
	require.Equal(t, []any{"10000000-0000-4000-8000-000000000001"}, detail["members"])
	require.Equal(t, float64(2), detail["total_issues"])
	require.Equal(t, float64(1), detail["completed_issues"])
	require.Equal(t, float64(0), detail["started_issues"])
	page := cyclePageWire(t, rpc(t, h, "test-one", "tools/call", resourceCall("module", `{"action":"list","project_id":"`+cycleProject+`"}`)))
	require.Equal(t, float64(2), page["total_count"])
	require.NotContains(t, string(mustJSON(t, page)), "Foreign secret")
}

func TestLabelParentsRetainOwnedDeletedRecordsButHideForeignRecords(t *testing.T) {
	f, pool := databaseFixture(t)
	addResourceReadFixture(t, pool)
	_, err := pool.Exec(t.Context(), `UPDATE labels SET parent_id='50000000-0000-4000-8000-000000000202' WHERE id='50000000-0000-4000-8000-000000000201'; UPDATE labels SET deleted_at=now() WHERE id='50000000-0000-4000-8000-000000000202'`)
	require.NoError(t, err)
	h := f.handler(t, nil)
	call := func() map[string]any {
		return cyclePageWire(t, rpc(t, h, "test-one", "tools/call", resourceCall("label", `{"action":"retrieve","project_id":"`+cycleProject+`","label_id":"`+labelID+`"}`)))
	}
	require.Equal(t, "50000000-0000-4000-8000-000000000202", call()["parent"])
	_, err = pool.Exec(t.Context(), `UPDATE labels SET parent_id='50000000-0000-4000-8000-000000000206' WHERE id='50000000-0000-4000-8000-000000000201'`)
	require.NoError(t, err)
	require.Nil(t, call()["parent"])
	_, err = pool.Exec(t.Context(), `UPDATE labels SET workspace_id='20000000-0000-4000-8000-000000000001' WHERE id='50000000-0000-4000-8000-000000000206'`)
	require.NoError(t, err)
	require.Nil(t, call()["parent"])
	page := cyclePageWire(t, rpc(t, h, "test-one", "tools/call", resourceCall("label", `{"action":"list","project_id":"`+cycleProject+`"}`)))
	require.Equal(t, float64(1), page["total_count"])
}

func TestEveryModuleAndLabelProjectionRequiresPrimaryMembership(t *testing.T) {
	_, pool := databaseFixture(t)
	addResourceReadFixture(t, pool)
	tx, err := pool.Begin(t.Context())
	require.NoError(t, err)
	defer func() { require.NoError(t, tx.Rollback(context.Background())) }()
	moduleRepo, err := modules.NewRepository(tx)
	require.NoError(t, err)
	labelRepo, err := labels.NewRepository(tx)
	require.NoError(t, err)
	workspace := uuid.MustParse("20000000-0000-4000-8000-000000000001")
	for _, tc := range []struct{ tool, operation, id string }{{"module", "lite", moduleID}, {"module", "detail", moduleID}, {"module", "archived", archivedModuleID}, {"label", "list", labelID}, {"label", "detail", labelID}} {
		t.Run(tc.tool+"/"+tc.operation, func(t *testing.T) {
			request := modules.APIV1Request{Operation: tc.operation, ExpectedWorkspaceID: &workspace, Params: map[string]string{"project_id": cycleProject, "slug": "workspace-one", "pk": tc.id}}
			read := moduleRepo.ReadAPIV1
			denied := modules.ErrForbidden
			if tc.tool == "label" {
				read = labelRepo.ReadAPIV1
				denied = labels.ErrForbidden
			}
			_, err = read(t.Context(), "10000000-0000-4000-8000-000000000001", request)
			require.NoError(t, err)
			_, err = tx.Exec(t.Context(), `UPDATE project_members SET is_active=false`)
			require.NoError(t, err)
			_, err = read(t.Context(), "10000000-0000-4000-8000-000000000001", request)
			require.ErrorIs(t, err, denied)
			_, err = tx.Exec(t.Context(), `UPDATE project_members SET is_active=true`)
			require.NoError(t, err)
		})
	}
}
