//go:build integration

package workitemtypes_test

import (
	"context"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/projectconfiguration"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5"
)

func TestEstimateReassignmentWaitsForPropertyPreservation(t *testing.T) {
	f := newProjectTypeFixture(t)
	items, _, _ := seedPublicationDraft(t, f)
	created, err := writeValueItem(f, items, "POST", "", `{"name":"Estimate item"}`)
	if err != nil {
		t.Fatal(err)
	}
	item := resultIssueID(t, created)
	estimate, oldPoint, newPoint := seedEstimatePoints(t, f, item)
	connection, err := pgx.ConnectConfig(f.ctx, f.db.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.CloseContext(t, connection, context.WithoutCancel(f.ctx)) }()
	repo, err := projectconfiguration.NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(f.ctx)) }()
	if _, err = tx.Exec(f.ctx, `SELECT work_item_schema_lock($1::uuid)`, f.workspace); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, failure := repo.DeleteEstimatePoint(f.ctx, f.owner, "types", f.project, estimate, oldPoint, strings.NewReader(`{"new_estimate_id":"`+newPoint+`"}`), "application/json")
		done <- failure
	}()
	waitForSchemaWriter(t, f, tx, int(connection.PgConn().PID()), done)
	if _, err = tx.Exec(f.ctx, `UPDATE issues SET description_html='<p>Preserved field</p>' WHERE id=$1`, item); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-f.ctx.Done():
		t.Fatal(f.ctx.Err())
	}
	var point, description string
	var cleanup int
	if err = f.db.QueryRow(f.ctx, `SELECT estimate_point_id::text,description_html,(SELECT count(*) FROM work_item_effects WHERE kind='soft_delete' AND payload->>'id'=$2) FROM issues WHERE id=$1`, item, oldPoint).Scan(&point, &description, &cleanup); err != nil {
		t.Fatal(err)
	}
	if point != newPoint || description != "<p>Preserved field</p>" || cleanup != 1 {
		t.Fatalf("reassignment lost preservation/effect: %s %q %d", point, description, cleanup)
	}
}

func TestEstimateDeletionRollsBackReassignmentAndEffects(t *testing.T) {
	f := newProjectTypeFixture(t)
	items, _, _ := seedPublicationDraft(t, f)
	created, err := writeValueItem(f, items, "POST", "", `{"name":"Estimate item"}`)
	if err != nil {
		t.Fatal(err)
	}
	item := resultIssueID(t, created)
	estimate, oldPoint, newPoint := seedEstimatePoints(t, f, item)
	repo, err := projectconfiguration.NewRepository(f.db)
	if err != nil {
		t.Fatal(err)
	}
	var before int
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM work_item_effects`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	f.exec(`CREATE FUNCTION reject_estimate_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.deleted_at IS NOT NULL THEN RAISE EXCEPTION 'reject deletion'; END IF; RETURN NEW; END $$; CREATE CONSTRAINT TRIGGER reject_estimate_commit AFTER UPDATE ON estimate_points DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_estimate_commit()`)
	_, err = repo.DeleteEstimatePoint(f.ctx, f.owner, "types", f.project, estimate, oldPoint, strings.NewReader(`{"new_estimate_id":"`+newPoint+`"}`), "application/json")
	if err == nil {
		t.Fatal("forced failure succeeded")
	}
	var point string
	var deleted bool
	var after int
	if err = f.db.QueryRow(f.ctx, `SELECT estimate_point_id::text,(SELECT deleted_at IS NOT NULL FROM estimate_points WHERE id=$2),(SELECT count(*) FROM work_item_effects) FROM issues WHERE id=$1`, item, oldPoint).Scan(&point, &deleted, &after); err != nil {
		t.Fatal(err)
	}
	if point != oldPoint || deleted || after != before {
		t.Fatalf("partial reassignment: %s deleted=%v effects=%d/%d", point, deleted, after, before)
	}
}

func seedEstimatePoints(t *testing.T, f projectTypeFixture, item string) (string, string, string) {
	t.Helper()
	estimate, oldPoint, newPoint := uuid.New().String(), uuid.New().String(), uuid.New().String()
	f.exec(`INSERT INTO estimates(id,name,description,project_id,workspace_id,type,last_used,created_at,updated_at) VALUES($1,'Estimate','',$2,$3,'points',true,now(),now())`, estimate, f.project, f.workspace)
	f.exec(`INSERT INTO estimate_points(id,key,description,value,estimate_id,project_id,workspace_id,created_at,updated_at) VALUES($1,0,'','1',$3,$4,$5,now(),now()),($2,1,'','2',$3,$4,$5,now(),now())`, oldPoint, newPoint, estimate, f.project, f.workspace)
	f.exec(`UPDATE issues SET estimate_point_id=$2 WHERE id=$1`, item, oldPoint)
	return estimate, oldPoint, newPoint
}

func TestEstimateDeletionWithoutReplacementClearsCurrentValueAtCommit(t *testing.T) {
	f := newProjectTypeFixture(t)
	items, _, _ := seedPublicationDraft(t, f)
	created, err := writeValueItem(f, items, "POST", "", `{"name":"Estimate item"}`)
	if err != nil {
		t.Fatal(err)
	}
	item := resultIssueID(t, created)
	estimate, oldPoint, _ := seedEstimatePoints(t, f, item)
	repo, err := projectconfiguration.NewRepository(f.db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.DeleteEstimatePoint(f.ctx, f.owner, "types", f.project, estimate, oldPoint, strings.NewReader(`{}`), "application/json"); err != nil {
		t.Fatal(err)
	}
	var cleared, deleted bool
	if err = f.db.QueryRow(f.ctx, `SELECT estimate_point_id IS NULL,(SELECT deleted_at IS NOT NULL FROM estimate_points WHERE id=$2) FROM issues WHERE id=$1`, item, oldPoint).Scan(&cleared, &deleted); err != nil {
		t.Fatal(err)
	}
	if !cleared || !deleted {
		t.Fatalf("delete left a current reference: cleared=%v deleted=%v", cleared, deleted)
	}
}
