//go:build integration

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/bootstrap"
	store "github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	api "github.com/CaliLuke/go-plane/milestones/gen/milestones"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type milestoneHTTPFixture struct {
	ctx         context.Context
	pool        *pgxpool.Pool
	repo        *store.Repository
	keys        *apikey.Validator
	handler     http.Handler
	base, token string
}
type milestoneHTTPThrottle struct{ denied bool }

func (t milestoneHTTPThrottle) Allow(context.Context, string) (apikey.Rate, error) {
	return apikey.Rate{Allowed: !t.denied, Remaining: "99", Reset: 60}, nil
}

func newMilestoneHTTPFixture(t *testing.T) milestoneHTTPFixture {
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
	token := uuid.NewString()
	_, err = pool.Exec(ctx, `INSERT INTO api_tokens(id,created_at,updated_at,token,label,user_type,user_id,description,is_active,is_service,allowed_rate_limit)
 VALUES(gen_random_uuid(),now(),now(),$1,'HTTP fixture',0,$2,'',true,false,'100/minute')`, token, testsupport.LabelActor)
	require.NoError(t, err)
	repo, err := store.NewRepository(pool)
	require.NoError(t, err)
	keys, err := apikey.NewWithReaderWriter(pool, pool)
	require.NoError(t, err)
	service, err := systemapi.NewMilestones(repo, keys, milestoneHTTPThrottle{}, "https://vector.example.test")
	require.NoError(t, err)
	handler := milestoneTestHTTP(service)
	return milestoneHTTPFixture{ctx, pool, repo, keys, handler, "/api/v1/workspaces/milestones/projects/" + testsupport.LabelProject + "/milestones/", token}
}
func milestoneTestHTTP(service api.Service) http.Handler {
	return extensionHTTPHandler(pendingReplacementEndpoints{milestones: api.NewEndpoints(service), activityReadsBodyLimit: 1 << 20})
}

func (f milestoneHTTPFixture) call(t *testing.T, method, path, body string, status int) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, f.base+path, strings.NewReader(body)).WithContext(f.ctx)
	request.Header.Set("X-Api-Key", f.token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()
	f.handler.ServeHTTP(response, request)
	require.Equal(t, status, response.Code, response.Body.String())
	return response
}
func milestoneHTTPObject(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var value map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &value))
	return value
}
func (f milestoneHTTPFixture) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	_, err := f.pool.Exec(f.ctx, query, args...)
	require.NoError(t, err)
}
func (f milestoneHTTPFixture) item(t *testing.T) string {
	t.Helper()
	id := uuid.NewString()
	f.exec(t, `INSERT INTO issues(id,created_at,updated_at,workspace_id,project_id,created_by_id,name,description_json,description_html,priority,sequence_id,sort_order,is_draft)
 VALUES($1,now(),now(),$2,$3,$4,'Work','{}','','none',1,0,false)`, id, testsupport.LabelWorkspace, testsupport.LabelProject, testsupport.LabelActor)
	return id
}

func TestMilestoneHTTPProductionLifecycleAndReceiptShapes(t *testing.T) {
	f := newMilestoneHTTPFixture(t)
	created := milestoneHTTPObject(t, f.call(t, "POST", "", `{"title":"  Release  ","target_date":"2026-12-31","external_source":"source"}`, 201))
	require.Len(t, created, 7)
	require.Equal(t, "Release", created["title"])
	require.Nil(t, created["external_id"])
	id := created["id"].(string)
	detail := milestoneHTTPObject(t, f.call(t, "GET", id+"/?include_revision=true", "", 200))
	require.Len(t, detail, 8)
	require.Contains(t, detail["revision"], "milestone:"+id+":")
	page := milestoneHTTPObject(t, f.call(t, "GET", "?per_page=1", "", 200))
	require.Len(t, page, 12)
	require.Len(t, page["results"], 1)
	require.Equal(t, float64(1), page["total_count"])
	item := f.item(t)
	association := `{"issues":["` + item + `"]}`
	require.Empty(t, f.call(t, "POST", id+"/work-items/", association, 204).Body.String())
	links := milestoneHTTPObject(t, f.call(t, "GET", id+"/work-items/", "", 200))
	link := links["results"].([]any)[0].(map[string]any)
	require.Len(t, link, 3)
	require.Equal(t, item, link["issue"])
	require.Equal(t, id, link["milestone"])
	receiptID := uuid.NewString()
	remove := `{"issues":["` + item + `"],"request_id":"` + receiptID + `"}`
	first := milestoneHTTPObject(t, f.call(t, "DELETE", id+"/work-items/", remove, 200))
	require.Len(t, first, 6)
	require.Nil(t, first["result"])
	require.Equal(t, false, first["replayed"])
	replay := milestoneHTTPObject(t, f.call(t, "DELETE", id+"/work-items/", remove, 200))
	require.Equal(t, true, replay["replayed"])
	require.Equal(t, first["revision"], replay["revision"])
	require.Empty(t, f.call(t, "POST", id+"/work-items/", association, 204).Body.String())
	require.Empty(t, f.call(t, "DELETE", id+"/work-items/", association, 204).Body.String())
	patched := milestoneHTTPObject(t, f.call(t, "PATCH", id+"/", `{"title":"Release two","target_date":null}`, 200))
	require.Len(t, patched, 7)
	require.Nil(t, patched["target_date"])
	require.Equal(t, "source", patched["external_source"])
	f.call(t, "DELETE", id+"/?expected_revision="+url.QueryEscape(detail["revision"].(string)), "", 409)
	current := milestoneHTTPObject(t, f.call(t, "GET", id+"/?include_revision=true", "", 200))
	deletePath := id + "/?request_id=" + uuid.NewString() + "&expected_revision=" + url.QueryEscape(current["revision"].(string))
	removed := milestoneHTTPObject(t, f.call(t, "DELETE", deletePath, "", 200))
	require.Nil(t, removed["result"])
	replay = milestoneHTTPObject(t, f.call(t, "DELETE", deletePath, "", 200))
	require.Equal(t, true, replay["replayed"])
	f.call(t, "GET", id+"/", "", 404)
	second := milestoneHTTPObject(t, f.call(t, "POST", "", `{"title":"Second"}`, 201))
	require.Empty(t, f.call(t, "DELETE", second["id"].(string)+"/", "", 204).Body.String())
	empty := milestoneHTTPObject(t, f.call(t, "GET", "", "", 200))
	require.Empty(t, empty["results"])
	require.Equal(t, float64(0), empty["total_count"])
	var itemExists bool
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT EXISTS(SELECT 1 FROM issues WHERE id=$1 AND deleted_at IS NULL)`, item).Scan(&itemExists))
	require.True(t, itemExists)
}

func TestMilestoneHTTPRejectsUnknownFieldsAndPreservesNativePaginationValidation(t *testing.T) {
	f := newMilestoneHTTPFixture(t)
	for _, body := range []string{`{}`, `null`, `{"title":null}`, `{"title":" "}`, `{"title":"Release","module":"foreign"}`, `{"title":"Release","target_date":"not-a-date"}`, `{"title":"Release","expected_revision":"stale"}`} {
		f.call(t, "POST", "", body, 400)
	}
	for _, query := range []string{"per_page=", "per_page=0", "per_page=-1", "per_page=1001", "per_page=garbage", "per_page=1&per_page=bad", "cursor=", "cursor=1:-1:0", "cursor=1:999999999999999999999999:0"} {
		f.call(t, "GET", "?"+query, "", 400)
	}
	created := milestoneHTTPObject(t, f.call(t, "POST", "", `{"title":"Release"}`, 201))
	id := created["id"].(string)
	f.call(t, "PATCH", id+"/", `{"unknown":1}`, 400)
	f.call(t, "POST", id+"/work-items/", `{"issues":[],"unknown":1}`, 400)
	f.call(t, "DELETE", id+"/work-items/", `{"issues":[null]}`, 400)
	f.call(t, "POST", id+"/work-items/", `{"issues":[]}`, 400)
	f.call(t, "PUT", id+"/", `{}`, 405)
	f.call(t, "PATCH", uuid.Nil.String()+"/", `{"title":"Invalid identifier"}`, 400)
	f.call(t, "DELETE", uuid.Nil.String()+"/", "", 400)
	f.call(t, "GET", uuid.Nil.String()+"/", "", 400)
	f.call(t, "GET", uuid.Nil.String()+"/work-items/", "", 400)
	f.call(t, "POST", id+"/work-items/", `{"issues":["`+uuid.Nil.String()+`"]}`, 400)
	var records, effects int
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM milestones),(SELECT count(*) FROM work_item_effects)`).Scan(&records, &effects))
	require.Equal(t, 1, records)
	require.Equal(t, 1, effects)
}

func TestMilestoneHTTPPermissionsAndReplaysUseCurrentScope(t *testing.T) {
	f := newMilestoneHTTPFixture(t)
	requestID := uuid.NewString()
	body := `{"title":"Receipt","request_id":"` + requestID + `"}`
	first := milestoneHTTPObject(t, f.call(t, "POST", "", body, 201))
	require.Len(t, first, 6)
	require.Equal(t, false, first["replayed"])
	replay := milestoneHTTPObject(t, f.call(t, "POST", "", body, 201))
	require.Equal(t, true, replay["replayed"])
	id := first["milestone_id"].(string)
	f.exec(t, `UPDATE project_members SET role=5`)
	f.call(t, "GET", "", "", 403)
	f.call(t, "GET", id+"/", "", 403)
	f.call(t, "POST", "", body, 403)
	f.call(t, "PATCH", id+"/", `{"title":"Denied"}`, 403)
	f.call(t, "DELETE", id+"/", "", 403)
	association := `{"issues":["` + uuid.NewString() + `"]}`
	f.call(t, "POST", id+"/work-items/", association, 403)
	f.call(t, "DELETE", id+"/work-items/", association, 403)
	f.exec(t, `UPDATE projects SET guest_view_all_features=true`)
	f.call(t, "GET", id+"/", "", 200)
	f.call(t, "POST", "", body, 403)
	f.exec(t, `UPDATE project_members SET role=15,is_active=false`)
	f.call(t, "GET", id+"/", "", 403)
	f.exec(t, `UPDATE project_members SET is_active=true; UPDATE projects SET archived_at=now()`)
	f.call(t, "GET", id+"/", "", 403)
	f.exec(t, `UPDATE projects SET archived_at=NULL; UPDATE api_tokens SET is_active=false`)
	f.call(t, "POST", "", body, 403)
	f.token = ""
	f.call(t, "GET", "", "", 401)
}
