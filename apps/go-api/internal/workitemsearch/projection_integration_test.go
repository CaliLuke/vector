//go:build integration

package workitemsearch_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	search "github.com/CaliLuke/go-plane/internal/workitemsearch"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type searchFixture struct {
	ctx                                       context.Context
	db                                        *pgx.Conn
	workspace, project, item, itemType, field string
}

func newSearchFixture(t *testing.T) searchFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	pair, err := postgresfixture.StartUncloned(ctx, bootstrap.MigrateSchema)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.Close(t, pair) })
	db, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.CloseContext(t, db, context.WithoutCancel(ctx)) })
	f := searchFixture{ctx: ctx, db: db, workspace: uuid.NewString(), project: uuid.NewString(), item: uuid.NewString(), itemType: uuid.NewString(), field: uuid.NewString()}
	actor := uuid.NewString()
	searchExec(t, f, `INSERT INTO users(id,username,email,password,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,token,user_timezone,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,display_name,is_email_valid,is_password_reset_required) VALUES($1::uuid,$1::text,$1::text||'@example.test','!','','','',now(),now(),now(),'','',false,false,false,true,false,true,false,'','UTC','','','','',false,'Search',true,false)`, actor)
	searchExec(t, f, `INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Search',$2,'search','UTC','',now(),now())`, f.workspace, actor)
	searchExec(t, f, `INSERT INTO projects(id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES($1,'Search','',0,$2,'SEARCH',false,false,false,true,false,false,false,false,0,0,'{}','UTC',now(),now())`, f.project, f.workspace)
	searchExec(t, f, `INSERT INTO issue_types(id,workspace_id,name,description,logo_props,is_active,is_default,is_epic,level,created_at,updated_at) VALUES($1,$2,'Task','','{}',true,true,false,0,now(),now());`, f.itemType, f.workspace)
	searchExec(t, f, `INSERT INTO work_item_properties(id,workspace_id,definition) VALUES($1,$2,jsonb_build_object('kind','Rich text','title','Notes','configuration',jsonb_build_object('active',true)))`, f.field, f.workspace)
	searchExec(t, f, `INSERT INTO work_item_type_properties(type_id,property_id) VALUES($1,$2)`, f.itemType, f.field)
	searchExec(t, f, `INSERT INTO issues(id,workspace_id,project_id,type_id,name,description_json,description_html,priority,sort_order,sequence_id,is_draft,custom_properties,created_at,updated_at) VALUES($1,$2,$3,$4,'Search item','{}','','none',0,1,false,jsonb_build_object($5::text,'<p>Alpha needle</p>'),now(),now())`, f.item, f.workspace, f.project, f.itemType, f.field)
	searchExec(t, f, `INSERT INTO work_item_type_settings(workspace_id,enabled) VALUES($1,true)`, f.workspace)
	return f
}
func searchExec(t *testing.T, f searchFixture, sql string, args ...any) {
	t.Helper()
	if _, err := f.db.Exec(f.ctx, sql, args...); err != nil {
		t.Fatal(err)
	}
}
func refreshSearchItem(t *testing.T, f searchFixture) {
	t.Helper()
	tx, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.Rollback(t, tx, f.ctx) }()
	if _, err = tx.Exec(f.ctx, `SELECT work_item_schema_lock($1::uuid)`, f.workspace); err != nil {
		t.Fatal(err)
	}
	if err = search.Refresh(f.ctx, tx, f.workspace, f.item); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
}
func searchSnippet(t *testing.T, f searchFixture, term string) map[string]any {
	t.Helper()
	var raw []byte
	err := f.db.QueryRow(f.ctx, `SELECT property_match.snippet FROM issues issue`+search.JoinSQL("issue", "$2")+`WHERE issue.id=$1`, f.item, term).Scan(&raw)
	if err != nil {
		t.Fatal(err)
	}
	var snippet map[string]any
	if len(raw) > 0 {
		if err = json.Unmarshal(raw, &snippet); err != nil {
			t.Fatal(err)
		}
	}
	return snippet
}
func TestCurrentProjectionRejectsStaleValuesAndInactiveContexts(t *testing.T) {
	f := newSearchFixture(t)
	refreshSearchItem(t, f)
	snippet := searchSnippet(t, f, "needle")
	if snippet["property_id"] != f.field || snippet["text"] != "Alpha needle" {
		t.Fatalf("snippet=%v", snippet)
	}
	searchExec(t, f, `UPDATE issues SET custom_properties=jsonb_build_object($2::text,'<p>Replacement needle</p>') WHERE id=$1`, f.item, f.field)
	if got := searchSnippet(t, f, "needle"); got != nil {
		t.Fatalf("stale projection leaked: %v", got)
	}
	refreshSearchItem(t, f)
	if got := searchSnippet(t, f, "Replacement"); got["text"] != "Replacement needle" {
		t.Fatalf("replacement=%v", got)
	}
	searchExec(t, f, `UPDATE work_item_properties SET definition=jsonb_set(definition,'{contexts}',jsonb_build_array(jsonb_build_object('project_ids',jsonb_build_array($2::text),'type_ids','[]'::jsonb,'override_keys',jsonb_build_array('active'),'configuration',jsonb_build_object('active',false)))) WHERE id=$1`, f.field, f.project)
	if got := searchSnippet(t, f, "Replacement"); got != nil {
		t.Fatalf("inactive context leaked: %v", got)
	}
	searchExec(t, f, `UPDATE issues SET custom_properties='{}' WHERE id=$1`, f.item)
	refreshSearchItem(t, f)
	var count int
	if err := f.db.QueryRow(f.ctx, `SELECT count(*) FROM work_item_property_search WHERE issue_id=$1`, f.item).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("clear retained %d index rows", count)
	}
}

func TestBackfillRollsBackCompletedBatchesAndRepairsOnRetry(t *testing.T) {
	f := newSearchFixture(t)
	searchExec(t, f, `INSERT INTO issues(id,workspace_id,project_id,type_id,name,description_json,description_html,priority,sort_order,sequence_id,is_draft,custom_properties,created_at,updated_at) SELECT gen_random_uuid(),$1,$2,$3,'Batch','{}','','none',n,n+1,false,jsonb_build_object($4::text,'<p>Batch needle</p>'),now(),now() FROM generate_series(1,101)n`, f.workspace, f.project, f.itemType, f.field)
	var last string
	if err := f.db.QueryRow(f.ctx, `SELECT id::text FROM issues ORDER BY id DESC LIMIT 1`).Scan(&last); err != nil {
		t.Fatal(err)
	}
	searchExec(t, f, `UPDATE issues SET custom_properties=jsonb_build_object($2::text,42) WHERE id=$1`, last, f.field)
	if err := search.Backfill(f.ctx, f.db); err == nil {
		t.Fatal("malformed saved value accepted")
	}
	assertSearchCounts(t, f, 0, 0)
	searchExec(t, f, `UPDATE issues SET custom_properties=jsonb_build_object($2::text,'<p>Repaired</p>') WHERE id=$1`, last, f.field)
	if err := search.Backfill(f.ctx, f.db); err != nil {
		t.Fatal(err)
	}
	assertSearchCounts(t, f, 102, 1)
	if err := search.Backfill(f.ctx, f.db); err != nil {
		t.Fatal(err)
	}
	assertSearchCounts(t, f, 102, 1)
}
func assertSearchCounts(t *testing.T, f searchFixture, rows, markers int) {
	t.Helper()
	var gotRows, gotMarkers int
	if err := f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM work_item_property_search),(SELECT count(*) FROM work_item_search_backfills)`).Scan(&gotRows, &gotMarkers); err != nil {
		t.Fatal(err)
	}
	if gotRows != rows || gotMarkers != markers {
		t.Fatalf("rows/markers=%d/%d; want %d/%d", gotRows, gotMarkers, rows, markers)
	}
}
