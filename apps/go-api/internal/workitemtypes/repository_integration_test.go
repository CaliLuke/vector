//go:build integration

package workitemtypes_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/jackc/pgx/v5"
)

type typeFixture struct {
	ctx                                         context.Context
	db                                          *pgx.Conn
	owner, member, outsider, workspace, project string
	exec                                        func(string, ...any)
	addProject                                  func(string)
}

func newTypeFixture(t *testing.T) typeFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	database, err := typeSchema.CloneDatabase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if closeErr := database.Close(cleanup); closeErr != nil {
			t.Errorf("close test database: %v", closeErr)
		}
	})
	db, err := pgx.Connect(ctx, database.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close(context.WithoutCancel(ctx)) })
	owner, member, outsider, workspace, project := uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, failure := db.Exec(ctx, sql, args...); failure != nil {
			t.Fatal(failure)
		}
	}
	for _, user := range []string{owner, member, outsider} {
		exec(`INSERT INTO users(id,username,email,password,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,token,user_timezone,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,display_name,is_email_valid,is_password_reset_required) VALUES($1::uuid,$1::text,$1::text||'@example.test','!','','','',now(),now(),now(),'','',false,false,false,true,false,true,false,'','UTC','','','','',false,'Types',true,false)`, user)
	}
	exec(`INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Types',$2,'types','UTC','',now(),now())`, workspace, owner)
	for _, user := range []string{owner, member} {
		role := 15
		if user == owner {
			role = 20
		}
		exec(`INSERT INTO workspace_members(id,workspace_id,member_id,role,created_at,updated_at,view_props,default_props,issue_props,is_active,getting_started_checklist,tips,explored_features) VALUES(gen_random_uuid(),$1,$2,$3,now(),now(),'{}','{}','{}',true,'{}','{}','{}')`, workspace, user, role)
	}
	addProject := func(id string) {
		t.Helper()
		exec(`INSERT INTO projects(id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES($1::uuid,$1::text,'',0,$2,left($1::text,12),false,false,false,true,false,false,false,false,0,0,'{}','UTC',now(),now())`, id, workspace)
	}
	addProject(project)
	return typeFixture{ctx, db, owner, member, outsider, workspace, project, exec, addProject}
}
func TestWorkspaceTypeLifecycle(t *testing.T) {
	f := newTypeFixture(t)
	ctx, db, owner, member, outsider, workspace, project := f.ctx, f.db, f.owner, f.member, f.outsider, f.workspace, f.project
	exec, addProject := f.exec, f.addProject
	repo := types.NewRepository(db)
	var err error
	checkError := func(err error, status int) {
		t.Helper()
		var failure *types.Error
		if !errors.As(err, &failure) || failure.Status != status {
			t.Fatalf("want status %d, got %v", status, err)
		}
	}
	t.Run("access isolation", func(t *testing.T) {
		_, err = repo.Catalog(ctx, outsider, "types")
		checkError(err, 403)
		_, err = repo.Mutate(ctx, member, "types", "enable", "", types.Input{})
		checkError(err, 403)
	})
	catalog, err := repo.Mutate(ctx, owner, "types", "enable", "", types.Input{})
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.Enabled || len(catalog.Types) != 1 || !catalog.Types[0].Default || !catalog.Types[0].Active || catalog.ProjectCount != 1 {
		t.Fatalf("unexpected enabled catalog: %+v", catalog)
	}
	task := catalog.Types[0]
	input := types.Input{Revision: catalog.Revision, Name: "Bug", Icon: "bug", Color: "#123456"}
	catalog, err = repo.Mutate(ctx, owner, "types", "create", "", input)
	if err != nil {
		t.Fatal(err)
	}
	bug := catalog.Types[1]
	if bug.Active || bug.Mandatory || bug.Level != 0 {
		t.Fatalf("new types must start inactive at level zero: %+v", bug)
	}
	t.Run("stale writes and duplicate names do not mutate", func(t *testing.T) {
		_, err = repo.Mutate(ctx, owner, "types", "create", "", input)
		checkError(err, 409)
		input.Revision = catalog.Revision
		input.Name = "  bUg  "
		_, err = repo.Mutate(ctx, owner, "types", "create", "", input)
		checkError(err, 409)
		after, readErr := repo.Catalog(ctx, owner, "types")
		if readErr != nil || after.Revision != catalog.Revision || len(after.Types) != 2 {
			t.Fatalf("failed writes changed catalog: %+v %v", after, readErr)
		}
	})
	t.Run("default protection", func(t *testing.T) {
		_, err = repo.Mutate(ctx, owner, "types", "delete", task.ID, types.Input{Revision: catalog.Revision})
		checkError(err, 422)
		_, err = repo.Mutate(ctx, owner, "types", "update", task.ID, types.Input{Revision: catalog.Revision, Name: task.Name, Icon: task.Icon, Color: task.Color})
		checkError(err, 422)
	})
	input = types.Input{Revision: catalog.Revision, Name: "Defect", Description: "Something broken", Icon: "Database", Color: "#02b55c", Active: true, Mandatory: true}
	catalog, err = repo.Mutate(ctx, owner, "types", "update", bug.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("catalog icon and color survive reload", func(t *testing.T) {
		reloaded, readErr := repo.Catalog(ctx, owner, "types")
		if readErr != nil {
			t.Fatal(readErr)
		}
		if reloaded.Types[1].Icon != "Database" || reloaded.Types[1].Color != "#02b55c" {
			t.Fatalf("appearance was not persisted: %+v", reloaded.Types[1])
		}
	})
	t.Run("mandatory rollout covers existing and future projects", func(t *testing.T) {
		addProject(uuid.New().String())
		var count int
		err = db.QueryRow(ctx, `SELECT count(*) FROM project_issue_types WHERE issue_type_id=$1 AND is_active AND deleted_at IS NULL`, bug.ID).Scan(&count)
		if err != nil || count != 2 {
			t.Fatalf("mandatory rollout: %d %v", count, err)
		}
		input.Revision = catalog.Revision
		input.Active = false
		_, err = repo.Mutate(ctx, owner, "types", "update", bug.ID, input)
		checkError(err, 422)
		input.Active = true
		input.Mandatory = false
		catalog, err = repo.Mutate(ctx, owner, "types", "update", bug.ID, input)
		if err != nil {
			t.Fatal(err)
		}
		err = db.QueryRow(ctx, `SELECT count(*) FROM project_issue_types WHERE issue_type_id=$1 AND is_active AND deleted_at IS NULL`, bug.ID).Scan(&count)
		if err != nil || count != 2 {
			t.Fatalf("switching to import removed assignments: %d %v", count, err)
		}
	})
	t.Run("used drafts block deletion and inactive types block new assignments", func(t *testing.T) {
		draft := uuid.New().String()
		exec(`INSERT INTO draft_issues(id,workspace_id,project_id,type_id,name,description_json,description_html,priority,sort_order,created_at,updated_at) VALUES($1,$2,$3,$4,'Draft','{}','','none',0,now(),now())`, draft, workspace, project, bug.ID)
		_, err = repo.Mutate(ctx, owner, "types", "delete", bug.ID, types.Input{Revision: catalog.Revision})
		checkError(err, 409)
		input.Revision = catalog.Revision
		input.Active = false
		catalog, err = repo.Mutate(ctx, owner, "types", "update", bug.ID, input)
		if err != nil {
			t.Fatal(err)
		}
		// Existing drafts remain editable after deactivation.
		exec(`UPDATE draft_issues SET name='Existing draft' WHERE id=$1`, draft)
		_, err = db.Exec(ctx, `INSERT INTO draft_issues(id,workspace_id,project_id,type_id,description_json,description_html,priority,sort_order,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,'{}','','none',0,now(),now())`, workspace, project, bug.ID)
		if err == nil {
			t.Fatal("inactive type accepted a new draft")
		}
		exec(`DELETE FROM draft_issues WHERE id=$1`, draft)
	})
	t.Run("unused deletion persists and stays tenant scoped", func(t *testing.T) {
		_, err = repo.Mutate(ctx, owner, "types", "delete", uuid.New().String(), types.Input{Revision: catalog.Revision})
		checkError(err, 404)
		catalog, err = repo.Mutate(ctx, owner, "types", "delete", bug.ID, types.Input{Revision: catalog.Revision})
		if err != nil {
			t.Fatal(err)
		}
		after, readErr := repo.Catalog(ctx, owner, "types")
		if readErr != nil || len(after.Types) != 1 || after.Types[0].ID != task.ID {
			t.Fatalf("deletion not durable: %+v %v", after, readErr)
		}
	})
}
