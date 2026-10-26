//go:build integration

package workitems

import (
	"context"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Exercise the writer's public review/commit boundary against the migrated schema.
func TestReviewedSaveRejectsExpiredReceiptAndCommitsRepairedDraftOnce(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, bootstrap.MigrateSchema)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pair.Close() }()
	db, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close(ctx) }()
	actor, workspace, project, draft := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, failure := db.Exec(ctx, query, args...); failure != nil {
			t.Fatal(failure)
		}
	}
	exec(`INSERT INTO users(id,username,email,password,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,token,user_timezone,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,display_name,is_email_valid,is_password_reset_required) VALUES($1::uuid,$1::text,'editor@example.test','!','','','',now(),now(),now(),'','',false,false,false,true,false,true,false,'','UTC','','','','',false,'Editor',true,false)`, actor)
	exec(`INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Review',$2,'review','UTC','',now(),now())`, workspace, actor)
	exec(`INSERT INTO workspace_members(id,workspace_id,member_id,role,created_at,updated_at,view_props,default_props,issue_props,is_active,getting_started_checklist,tips,explored_features) VALUES(gen_random_uuid(),$1,$2,20,now(),now(),'{}','{}','{}',true,'{}','{}','{}')`, workspace, actor)
	exec(`INSERT INTO projects(id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES($1,'Review','',0,$2,'REV',true,true,false,true,false,false,false,false,0,0,'{}','UTC',now(),now())`, project, workspace)
	exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,20,true,now(),now(),'{}','{}','{}',0)`, workspace, project, actor)
	catalog, err := workitemtypes.NewRepository(db).Mutate(ctx, actor, "review", "enable", "", workitemtypes.Input{})
	if err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO draft_issues(id,workspace_id,project_id,type_id,name,description_json,description_html,priority,sort_order,created_at,updated_at,created_by_id) VALUES($1,$2,$3,$4,'Original','{}','<p>Body</p>','none',0,now(),now(),$5)`, draft, workspace, project, catalog.Types[0].ID, actor)
	publisher := &versionTaskRecorder{}
	repo, err := NewRepository(db, publisher, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	q := Request{ActorID: actor, Slug: "review", Timezone: "UTC"}
	input := ItemEditInput{Mode: "draft", ItemID: draft, ProjectID: project, Revision: catalog.Revision, Changes: map[string]any{"name": "Edited draft"}}
	source, err := repo.ReadItemEditSource(ctx, q, "draft", draft, project)
	if err != nil {
		t.Fatal(err)
	}
	input.ExpectedSourceVersion = source.Body.(ItemEditSource).SourceVersion
	review, err := repo.PreviewItemChange(ctx, q, input)
	if err != nil {
		t.Fatal(err)
	}
	input.ReviewID = review.Body.(ItemEditReview).ID
	exec(`UPDATE work_item_schema_reviews SET created_at=now()-interval '31 minutes' WHERE id=$1`, input.ReviewID)
	_, err = repo.CommitItemChange(ctx, q, input)
	var rejected *djangoapi.HTTPError
	if !errors.As(err, &rejected) || rejected.Status != 409 || rejected.Body["review_again"] != true {
		t.Fatalf("expired uncommitted review did not allow repair: %v", err)
	}
	var title string
	if err = db.QueryRow(ctx, `SELECT name FROM draft_issues WHERE id=$1`, draft).Scan(&title); err != nil || title != "Original" {
		t.Fatalf("rejection changed draft: %s %v", title, err)
	}
	input.Mode = "publish"
	review, err = repo.PreviewItemChange(ctx, q, input)
	if err != nil {
		t.Fatal(err)
	}
	input.ReviewID = review.Body.(ItemEditReview).ID
	saved, err := repo.CommitItemChange(ctx, q, input)
	if err != nil {
		t.Fatal(err)
	}
	body := saved.Body.(map[string]any)
	if body["name"] != "Edited draft" || body["project_id"] != project || body["type_id"] != catalog.Types[0].ID {
		t.Fatalf("lost canonical saved fields: %+v", body)
	}
	// An acknowledged commit remains idempotent even after the receipt expires.
	exec(`UPDATE work_item_schema_reviews SET created_at=now()-interval '31 minutes' WHERE id=$1`, input.ReviewID)
	replay, err := repo.CommitItemChange(ctx, q, input)
	if err != nil || replay.Body.(map[string]any)["id"] != body["id"] {
		t.Fatalf("lost committed receipt: %+v %v", replay, err)
	}
	var count int
	if err = db.QueryRow(ctx, `SELECT count(*) FROM issues`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("publication count=%d err=%v", count, err)
	}
	if len(publisher.names) != 0 {
		t.Fatal("admitted effects before outbox dispatch")
	}
}
