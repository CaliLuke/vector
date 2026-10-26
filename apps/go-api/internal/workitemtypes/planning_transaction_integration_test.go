//go:build integration

package workitemtypes_test

import (
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
)

func planningCycleRequest(t *testing.T, f projectTypeFixture) (workitems.Request, string) {
	t.Helper()
	cycle := uuid.New().String()
	f.exec(`INSERT INTO cycles(id,name,description,owned_by_id,project_id,workspace_id,view_props,sort_order,progress_snapshot,logo_props,timezone,version,created_at,updated_at) VALUES($1,'Cycle','',$2,$3,$4,'{}',0,'{}','{}','UTC',1,now(),now())`, cycle, f.owner, f.project, f.workspace)
	item := addHierarchyItem(t, f, f.task, "")
	return workitems.Request{Operation: "api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_cycle_issues", Method: "POST", ActorID: f.owner, Slug: "types", ProjectID: f.project, Timezone: "UTC", Params: map[string]string{"cycle_id": cycle}}, `{"issues":["` + item + `"]}`
}

func TestPlanningQueuedWriteRechecksMembership(t *testing.T) {
	f := newProjectTypeFixture(t)
	request, body := planningCycleRequest(t, f)
	connection := hierarchyConnection(t, f)
	repo, err := workitems.NewRepository(connection, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	lock := hierarchyTransaction(t, f)
	done := make(chan error, 1)
	go func() {
		_, writeErr := repo.Execute(f.ctx, request, strings.NewReader(body), "application/json")
		done <- writeErr
	}()
	waitForSchemaWriter(t, f, lock, int(connection.PgConn().PID()), done)
	if _, err = lock.Exec(f.ctx, `UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, f.project, f.owner); err != nil {
		t.Fatal(err)
	}
	if err = lock.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-done:
		var denied *djangoapi.HTTPError
		if !errors.As(err, &denied) || denied.Status != 403 {
			t.Fatalf("queued permission revocation ignored: %v", err)
		}
	case <-f.ctx.Done():
		t.Fatal("queued planning write did not finish")
	}
	var count int
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM cycle_issues WHERE cycle_id=$1`, request.Params["cycle_id"]).Scan(&count); err != nil || count != 0 {
		t.Fatalf("revoked writer changed cycle membership: %d %v", count, err)
	}
}

func TestPlanningLateEffectFailureRollsBackMembership(t *testing.T) {
	f := newProjectTypeFixture(t)
	request, body := planningCycleRequest(t, f)
	repo, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	f.exec(`CREATE FUNCTION reject_planning_effect() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'reject planning effect'; END $$; CREATE TRIGGER reject_planning_effect BEFORE INSERT ON work_item_effects FOR EACH ROW EXECUTE FUNCTION reject_planning_effect()`)
	_, err = repo.Execute(f.ctx, request, strings.NewReader(body), "application/json")
	if err == nil {
		t.Fatal("injected effect failure did not reject planning write")
	}
	var count int
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM cycle_issues WHERE cycle_id=$1`, request.Params["cycle_id"]).Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed planning write left membership: %d %v", count, err)
	}
	f.exec(`DROP TRIGGER reject_planning_effect ON work_item_effects`)
	if _, err = repo.Execute(f.ctx, request, strings.NewReader(body), "application/json"); err != nil {
		t.Fatal(err)
	}
	var effects int
	if err = f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM cycle_issues WHERE cycle_id=$1),(SELECT count(*) FROM work_item_effects)`, request.Params["cycle_id"]).Scan(&count, &effects); err != nil || count != 1 || effects != 1 {
		t.Fatalf("successful write missing relation/effect: %d %d %v", count, effects, err)
	}
}
