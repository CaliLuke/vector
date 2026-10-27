//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/mcpactions"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type milestoneFixture struct {
	fixture
	pool   *pgxpool.Pool
	oauth  *mcpauth.OAuth
	issuer *liveIssuer
}

func newMilestoneFixture(t *testing.T) milestoneFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	pair, err := postgresfixture.StartUncloned(ctx, bootstrap.MigrateSchema)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pair.Close()) })
	pool, err := pgxpool.New(ctx, pair.SeedURL())
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	_, err = pool.Exec(ctx, testsupport.MilestoneWriteSeed)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE workspaces SET slug='workspace-one';
 INSERT INTO api_tokens(id,created_at,updated_at,token,label,user_type,user_id,description,is_active,is_service,allowed_rate_limit)
 VALUES(gen_random_uuid(),now(),now(),'test-one','MCP test',0,'00000000-0000-4000-8000-000000000003','',true,false,'100/minute');`)
	require.NoError(t, err)
	apps := databaseApplications(t, pool)
	service, err := mcpactions.New(apps, mcpactions.WithOrigin("https://vector.example"))
	require.NoError(t, err)
	keys, err := apikey.NewWithReaderWriter(pool, pool)
	require.NoError(t, err)
	members, err := mcpauth.NewWorkspaceReader(pool)
	require.NoError(t, err)
	pat, err := mcpauth.NewPersonalTokens(keys, members, throttle{})
	require.NoError(t, err)
	oauth, issuer := oauthForPool(t, pool, testsupport.LabelActor)
	issuer.scope = "vector:read vector:write"
	return milestoneFixture{fixture: fixture{service: service, auth: pat}, pool: pool, oauth: oauth, issuer: issuer}
}
func (f milestoneFixture) call(t *testing.T, mode, version string) func(map[string]any) rpcResult {
	t.Helper()
	pat, oauth := f.handler(t, nil), oauthHandler(t, f.fixture, f.oauth.Authenticate)
	return func(input map[string]any) rpcResult {
		input["project_id"] = testsupport.LabelProject
		body, err := json.Marshal(input)
		require.NoError(t, err)
		if mode == "OAuth" {
			return oauthToolCall(t, oauth, version, "oauth-user", "workspace-one", "milestone", string(body), http.StatusOK)
		}
		return rpcVersion(t, pat, "test-one", "tools/call", milestoneCall(string(body)), version)
	}
}
func milestoneObject(t *testing.T, result rpcResult) map[string]any {
	t.Helper()
	require.False(t, result.IsError, result.Content)
	var body map[string]any
	require.NoError(t, json.Unmarshal(result.StructuredContent, &body))
	require.Len(t, result.Content, 1)
	require.JSONEq(t, string(result.StructuredContent), result.Content[0].Text)
	return body
}
func (f milestoneFixture) item(t *testing.T) string {
	t.Helper()
	id := uuid.NewString()
	_, err := f.pool.Exec(t.Context(), `INSERT INTO issues(id,created_at,updated_at,workspace_id,project_id,created_by_id,name,description_json,description_html,priority,sequence_id,sort_order,is_draft)
 VALUES($1,now(),now(),$2,$3,$4,'Work','{}','','none',1,0,false)`, id, testsupport.LabelWorkspace, testsupport.LabelProject, testsupport.LabelActor)
	require.NoError(t, err)
	return id
}
func milestoneFailure(t *testing.T, result rpcResult, code string) {
	t.Helper()
	require.True(t, result.IsError, result.Content)
	require.Empty(t, result.StructuredContent)
	require.Contains(t, result.Content[0].Text, "["+code+"]")
}

func TestMilestoneMCPShapesAcrossCredentialsAndProtocols(t *testing.T) {
	f := newMilestoneFixture(t)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, mode := range []string{"personal token", "OAuth"} {
			t.Run(version+"/"+mode, func(t *testing.T) {
				call := f.call(t, mode, version)
				record := milestoneObject(t, call(map[string]any{"action": "create", "title": "Delivery", "target_date": "", "external_source": "", "external_id": ""}))
				require.Len(t, record, 7)
				require.Nil(t, record["target_date"])
				require.Nil(t, record["external_source"])
				require.Nil(t, record["external_id"])
				id := record["id"].(string)
				detail := milestoneObject(t, call(map[string]any{"action": "retrieve", "milestone_id": id}))
				require.Equal(t, record, detail)
				page := milestoneObject(t, call(map[string]any{"action": "list", "per_page": 0, "cursor": ""}))
				require.Len(t, page, 7)
				require.Equal(t, float64(1), page["count"])
				require.Equal(t, float64(1), page["total_count"])
				require.Equal(t, []any{record}, page["results"])
				require.Equal(t, "1000:1:0", page["next_cursor"])
				updated := milestoneObject(t, call(map[string]any{"action": "update", "milestone_id": id, "title": "", "target_date": "2028-02-29", "external_source": "  source  "}))
				require.Equal(t, "Delivery", updated["title"])
				require.Equal(t, "2028-02-29", updated["target_date"])
				require.Equal(t, "  source  ", updated["external_source"])
				item, removed := f.item(t), f.item(t)
				result := call(map[string]any{"action": "manage_workitems", "milestone_id": id, "add_ids": item + "," + removed + "," + item, "remove_ids": "[\"" + removed + "\"]"})
				require.False(t, result.IsError, result.Content)
				require.Empty(t, result.Content)
				require.Empty(t, result.StructuredContent)
				links := milestoneObject(t, call(map[string]any{"action": "list_workitems", "milestone_id": id, "per_page": 1}))
				require.Len(t, links, 7)
				require.Len(t, links["results"], 1)
				link := links["results"].([]any)[0].(map[string]any)
				require.Len(t, link, 3)
				require.Equal(t, item, link["issue"])
				require.Equal(t, id, link["milestone"])
				result = call(map[string]any{"action": "delete", "milestone_id": id})
				require.False(t, result.IsError, result.Content)
				require.Empty(t, result.Content)
				require.Empty(t, result.StructuredContent)
				empty := milestoneObject(t, call(map[string]any{"action": "list"}))
				require.Len(t, empty, 7)
				require.Empty(t, empty["results"])
				var alive bool
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT deleted_at IS NULL FROM issues WHERE id=$1`, item).Scan(&alive))
				require.True(t, alive)
			})
		}
	}
}

func TestMilestoneMCPReceiptsAndCASAcrossCredentialsAndProtocols(t *testing.T) {
	f := newMilestoneFixture(t)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, mode := range []string{"personal token", "OAuth"} {
			t.Run(version+"/"+mode, func(t *testing.T) {
				call := f.call(t, mode, version)
				create := map[string]any{"action": "create", "title": uuid.NewString(), "request_id": uuid.NewString()}
				receipt := milestoneObject(t, call(create))
				require.Len(t, receipt, 6)
				require.False(t, receipt["replayed"].(bool))
				require.Len(t, receipt["result"], 7)
				expires, err := time.Parse(time.RFC3339Nano, receipt["replay_expires_at"].(string))
				require.NoError(t, err)
				require.True(t, time.Until(expires) > 23*time.Hour)
				id, revision := receipt["milestone_id"].(string), receipt["revision"].(string)
				create["external_source"] = "" // Exact empty optional strings normalize to omission for replay.
				replay := milestoneObject(t, call(create))
				require.Equal(t, receipt["result"], replay["result"])
				require.True(t, replay["replayed"].(bool))
				create["title"] = "changed"
				milestoneFailure(t, call(create), "request_conflict")
				detail := milestoneObject(t, call(map[string]any{"action": "retrieve", "milestone_id": id, "include_revision": true}))
				require.Len(t, detail, 8)
				require.Equal(t, revision, detail["revision"])
				update := map[string]any{"action": "update", "milestone_id": id, "title": uuid.NewString(), "expected_revision": revision, "request_id": uuid.NewString()}
				changed := milestoneObject(t, call(update))
				require.NotEqual(t, revision, changed["revision"])
				require.True(t, milestoneObject(t, call(update))["replayed"].(bool))
				update["request_id"] = uuid.NewString()
				milestoneFailure(t, call(update), "revision_conflict")
				item := f.item(t)
				manage := map[string]any{"action": "manage_workitems", "milestone_id": id, "add_ids": item, "request_id": uuid.NewString(), "expected_revision": changed["revision"]}
				linked := milestoneObject(t, call(manage))
				require.Nil(t, linked["result"])
				manage["add_ids"] = fmt.Sprintf("[%q,%q]", item, item)
				require.True(t, milestoneObject(t, call(manage))["replayed"].(bool))
				var before int
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects`).Scan(&before))
				deletion := map[string]any{"action": "delete", "milestone_id": id, "request_id": uuid.NewString(), "expected_revision": linked["revision"]}
				deleted := milestoneObject(t, call(deletion))
				require.Nil(t, deleted["result"])
				var after int
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects`).Scan(&after))
				require.Greater(t, after, before)
				require.True(t, milestoneObject(t, call(deletion))["replayed"].(bool))
				var replayed int
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects`).Scan(&replayed))
				require.Equal(t, after, replayed)
				milestoneFailure(t, call(map[string]any{"action": "retrieve", "milestone_id": id}), "not_found")
				// Membership is checked before replay, even when the target is deleted.
				_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET is_active=false`)
				require.NoError(t, err)
				milestoneFailure(t, call(deletion), "forbidden")
				_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET is_active=true`)
				require.NoError(t, err)
			})
		}
	}
}

func TestMilestoneMCPInvalidInputIsAtomic(t *testing.T) {
	f := newMilestoneFixture(t)
	call := f.call(t, "personal token", "2026-07-28")
	record := milestoneObject(t, call(map[string]any{"action": "create", "title": "Atomic"}))
	id := record["id"].(string)
	for _, input := range []map[string]any{
		{"action": "create", "title": "   "}, {"action": "create", "title": "CAS create", "expected_revision": "x"},
		{"action": "update", "milestone_id": id, "title": nil}, {"action": "update", "milestone_id": id, "target_date": "2027-02-29"},
		{"action": "update", "milestone_id": id, "unknown": true}, {"action": "list", "per_page": 1001}, {"action": "list", "cursor": "bad"},
		{"action": "manage_workitems", "milestone_id": id}, {"action": "manage_workitems", "milestone_id": id, "add_ids": "[1]"},
		{"action": "delete", "milestone_id": id, "request_id": uuid.Nil.String()},
	} {
		t.Run(fmt.Sprint(input), func(t *testing.T) { milestoneFailure(t, call(input), "invalid_params") })
	}
	item := f.item(t)
	milestoneFailure(t, call(map[string]any{"action": "manage_workitems", "milestone_id": id, "add_ids": item, "remove_ids": uuid.NewString(), "request_id": uuid.NewString()}), "not_found")
	page := milestoneObject(t, call(map[string]any{"action": "list_workitems", "milestone_id": id}))
	require.Empty(t, page["results"])
	milestoneFailure(t, call(map[string]any{"action": "create", "title": "Atomic"}), "conflict")
}
