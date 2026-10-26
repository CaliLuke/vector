//go:build integration

package projectlabelreads

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/projectconfiguration"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	parentActor           = "00000000-0000-4000-8000-000000000001"
	parentWorkspace       = "00000000-0000-4000-8000-000000000002"
	parentProject         = "00000000-0000-4000-8000-000000000003"
	otherWorkspace        = "00000000-0000-4000-8000-000000000004"
	otherProject          = "00000000-0000-4000-8000-000000000005"
	labelA                = "00000000-0000-4000-8000-000000000011"
	labelB                = "00000000-0000-4000-8000-000000000012"
	labelC                = "00000000-0000-4000-8000-000000000013"
	foreignLabel          = "00000000-0000-4000-8000-000000000014"
	foreignWorkspaceChild = "00000000-0000-4000-8000-000000000015"
	deletedAncestor       = "00000000-0000-4000-8000-000000000016"
)

func parentIntegrityFixture(t *testing.T) *pgx.Conn {
	t.Helper()
	_, connection := postgresfixture.OpenSQL(t, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, archived_at timestamptz);
CREATE TABLE project_members (project_id uuid NOT NULL, member_id uuid NOT NULL, is_active boolean NOT NULL DEFAULT true);
CREATE TABLE labels (id uuid PRIMARY KEY, name text NOT NULL, color text NOT NULL, parent_id uuid,
 project_id uuid, workspace_id uuid NOT NULL, sort_order double precision NOT NULL,
 description text NOT NULL DEFAULT '', external_source text, external_id text, created_at timestamptz, updated_at timestamptz,
 created_by_id uuid, updated_by_id uuid, deleted_at timestamptz);
CREATE TABLE work_item_effects (id uuid PRIMARY KEY, kind text NOT NULL, payload jsonb NOT NULL);
INSERT INTO workspaces VALUES
 ('00000000-0000-4000-8000-000000000002', 'demo'),
 ('00000000-0000-4000-8000-000000000004', 'other');
INSERT INTO projects(id,workspace_id) VALUES
 ('00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000002'),
 ('00000000-0000-4000-8000-000000000005', '00000000-0000-4000-8000-000000000004');
INSERT INTO project_members(project_id,member_id,is_active) VALUES
 ('00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000001', true),
 ('00000000-0000-4000-8000-000000000005', '00000000-0000-4000-8000-000000000001', true);
INSERT INTO labels (id,name,color,parent_id,project_id,workspace_id,sort_order,created_at) VALUES
 ('00000000-0000-4000-8000-000000000011','A','#111111',NULL,'00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002',10,now()),
 ('00000000-0000-4000-8000-000000000012','B','#222222',NULL,'00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002',20,now()),
 ('00000000-0000-4000-8000-000000000013','C','#333333','00000000-0000-4000-8000-000000000012','00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002',30,now()),
 ('00000000-0000-4000-8000-000000000014','Foreign','#444444',NULL,'00000000-0000-4000-8000-000000000005','00000000-0000-4000-8000-000000000004',40,now()),
 ('00000000-0000-4000-8000-000000000015','Foreign workspace child','#555555',NULL,'00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000004',50,now()),
 ('00000000-0000-4000-8000-000000000016','Deleted ancestor','#666666',NULL,'00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002',60,now());`)
	return connection
}

func TestWriteLabelRejectsInvalidParentRelationships(t *testing.T) {
	for _, test := range []struct {
		name, labelID, parentID, projectID, slug string
		setup                                    func(*testing.T, *pgx.Conn)
	}{
		{"cross-project", labelA, foreignLabel, parentProject, "demo", nil},
		{"self", labelA, labelA, parentProject, "demo", nil},
		{"descendant", labelB, labelC, parentProject, "demo", nil},
		{"preexisting-cycle", labelC, labelA, parentProject, "demo", func(t *testing.T, db *pgx.Conn) {
			if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=CASE id WHEN $1 THEN $2 WHEN $2 THEN $1 ELSE parent_id END WHERE id IN ($1,$2)`, labelA, labelB); err != nil {
				t.Fatal(err)
			}
		}},
		{"deleted-ancestor", labelA, labelB, parentProject, "demo", func(t *testing.T, db *pgx.Conn) {
			if _, err := db.Exec(context.Background(), `UPDATE labels SET deleted_at=now() WHERE id=$1`, deletedAncestor); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=$2 WHERE id=$1`, labelB, deletedAncestor); err != nil {
				t.Fatal(err)
			}
		}},
		{"foreign-ancestor", labelA, labelB, parentProject, "demo", func(t *testing.T, db *pgx.Conn) {
			if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=NULL WHERE id=$1`, foreignLabel); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=$2 WHERE id=$1`, labelB, foreignLabel); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := parentIntegrityFixture(t)
			if test.setup != nil {
				test.setup(t, db)
			}
			if test.name == "preexisting-cycle" {
				if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=CASE id WHEN $1 THEN $2 WHEN $2 THEN $1 ELSE parent_id END WHERE id IN ($1,$2)`, labelA, labelB); err != nil {
					t.Fatal(err)
				}
			}
			var originalParent string
			if err := db.QueryRow(context.Background(), `SELECT COALESCE(parent_id::text,'') FROM labels WHERE id=$1`, test.labelID).Scan(&originalParent); err != nil {
				t.Fatal(err)
			}
			repository, err := NewWriteRepository(db)
			if err != nil {
				t.Fatal(err)
			}
			_, err = repository.Write(context.Background(), parentActor, test.slug, test.projectID, test.labelID,
				http.MethodPatch, strings.NewReader(fmt.Sprintf(`{"parent":%q}`, test.parentID)), "application/json")
			if err == nil {
				t.Fatalf("accepted invalid parent %s for label %s", test.parentID, test.labelID)
			}
			var parent string
			if scanErr := db.QueryRow(context.Background(), `SELECT COALESCE(parent_id::text,'') FROM labels WHERE id=$1`, test.labelID).Scan(&parent); scanErr != nil {
				t.Fatal(scanErr)
			}
			if parent != originalParent {
				t.Fatalf("invalid parent changed existing parent: got %s, want %s", parent, originalParent)
			}
		})
	}
}

func TestConcurrentReciprocalLabelParentMovesCannotCreateCycle(t *testing.T) {
	db := parentIntegrityFixture(t)
	secondDB, err := pgx.Connect(context.Background(), db.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := secondDB.Close(context.WithoutCancel(context.Background())); closeErr != nil {
			t.Errorf("close second postgres connection: %v", closeErr)
		}
	})
	firstRepository, err := NewWriteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	secondRepository, err := NewWriteRepository(secondDB)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	ready := make(chan struct{}, 2)
	results := make(chan error, 2)
	write := func(repository *WriteRepository, childID, parentID string) {
		ready <- struct{}{}
		<-start
		_, writeErr := repository.Write(context.Background(), parentActor, "demo", parentProject, childID,
			http.MethodPatch, strings.NewReader(fmt.Sprintf(`{"parent":%q}`, parentID)), "application/json")
		results <- writeErr
	}
	go write(firstRepository, labelA, labelB)
	go write(secondRepository, labelB, labelA)
	<-ready
	<-ready
	close(start)
	firstErr, secondErr := <-results, <-results
	successes := 0
	for _, resultErr := range []error{firstErr, secondErr} {
		if resultErr == nil {
			successes++
			continue
		}
		var httpError *HTTPError
		if !errors.As(resultErr, &httpError) || httpError.Status != 400 {
			t.Fatalf("reciprocal parent move error = %v, want HTTP 400", resultErr)
		}
	}
	if successes != 1 {
		t.Fatalf("successful reciprocal moves = %d, want exactly one (errors: %v, %v)", successes, firstErr, secondErr)
	}
	var cycle bool
	if err = db.QueryRow(context.Background(), `WITH RECURSIVE ancestry(id,parent_id,path,cycle) AS (
 SELECT id,parent_id,ARRAY[id],false FROM labels WHERE id=$1
 UNION ALL
 SELECT parent.id,parent.parent_id,ancestry.path || parent.id,parent.id=ANY(ancestry.path)
 FROM ancestry JOIN labels parent ON parent.id=ancestry.parent_id WHERE NOT ancestry.cycle
) SELECT COALESCE(bool_or(cycle),false) FROM ancestry`, labelA).Scan(&cycle); err != nil {
		t.Fatal(err)
	}
	if cycle {
		t.Fatal("reciprocal parent moves persisted a cycle")
	}
}

func TestRenameReloadsParentAfterWaitingForProjectMutationLock(t *testing.T) {
	db := parentIntegrityFixture(t)
	lockDB, err := pgx.Connect(context.Background(), db.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := lockDB.Close(context.WithoutCancel(context.Background())); closeErr != nil {
			t.Errorf("close lock connection: %v", closeErr)
		}
	})
	monitorDB, err := pgx.Connect(context.Background(), db.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := monitorDB.Close(context.WithoutCancel(context.Background())); closeErr != nil {
			t.Errorf("close monitor connection: %v", closeErr)
		}
	})
	if _, err = db.Exec(context.Background(), `UPDATE labels SET parent_id=$2 WHERE id=$1`, labelA, labelB); err != nil {
		t.Fatal(err)
	}
	lock, err := lockDB.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.Rollback(context.WithoutCancel(context.Background())) }()
	var lockedProject string
	if err = lock.QueryRow(context.Background(), `SELECT id::text FROM projects WHERE id=$1 FOR UPDATE`, parentProject).Scan(&lockedProject); err != nil {
		t.Fatal(err)
	}
	repository, err := NewWriteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, writeErr := repository.Write(context.Background(), parentActor, "demo", parentProject, labelA,
			http.MethodPatch, strings.NewReader(`{"name":"Renamed"}`), "application/json")
		result <- writeErr
	}()
	deadline := time.NewTimer(5 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		var waiting bool
		if err = monitorDB.QueryRow(context.Background(), `SELECT EXISTS(
 SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND pid<>pg_backend_pid()
 AND wait_event_type='Lock' AND query LIKE '%FOR UPDATE OF project%')`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		select {
		case writeErr := <-result:
			t.Fatalf("rename completed before the held project lock was released: %v", writeErr)
		case <-deadline.C:
			t.Fatal("rename never waited on the project row lock")
		case <-ticker.C:
		}
	}
	if _, err = lock.Exec(context.Background(), `UPDATE labels SET parent_id=NULL WHERE id=$1`, labelA); err != nil {
		t.Fatal(err)
	}
	if _, err = lock.Exec(context.Background(), `UPDATE labels SET parent_id=$2 WHERE id=$1`, labelB, labelA); err != nil {
		t.Fatal(err)
	}
	if err = lock.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err = <-result; err != nil {
		t.Fatalf("rename failed after lock release: %v", err)
	}
	var aParent, bParent string
	if err = db.QueryRow(context.Background(), `SELECT COALESCE(parent_id::text,'') FROM labels WHERE id=$1`, labelA).Scan(&aParent); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(context.Background(), `SELECT COALESCE(parent_id::text,'') FROM labels WHERE id=$1`, labelB).Scan(&bParent); err != nil {
		t.Fatal(err)
	}
	if aParent != "" || bParent != labelA {
		t.Fatalf("rename restored stale parent: A.parent=%q B.parent=%q", aParent, bParent)
	}
}

func TestDeleteLabelSoftDeletesDescendantsAndQueuesEachCleanup(t *testing.T) {
	db := parentIntegrityFixture(t)
	if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=$2 WHERE id=$1`, labelB, labelA); err != nil {
		t.Fatal(err)
	}
	repository, err := NewWriteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repository.Write(context.Background(), parentActor, "demo", parentProject, labelA, http.MethodDelete, strings.NewReader(`not parsed`), ""); err != nil {
		t.Fatal(err)
	}
	var deleted, present int
	if err = db.QueryRow(context.Background(), `SELECT count(*) FILTER (WHERE deleted_at IS NOT NULL),
 count(*) FILTER (WHERE id IN ($1,$2,$3)) FROM labels WHERE id IN ($1,$2,$3)`, labelA, labelB, labelC).Scan(&deleted, &present); err != nil {
		t.Fatal(err)
	}
	if deleted != 3 || present != 3 {
		t.Fatalf("deleted descendants=%d, present rows=%d; want all three soft-deleted", deleted, present)
	}
	var outOfScopeActive int
	if err = db.QueryRow(context.Background(), `SELECT count(*) FROM labels WHERE id IN ($1,$2) AND deleted_at IS NULL`, foreignLabel, foreignWorkspaceChild).Scan(&outOfScopeActive); err != nil {
		t.Fatal(err)
	}
	if outOfScopeActive != 2 {
		t.Fatalf("out-of-scope descendants still active=%d, want both foreign rows untouched", outOfScopeActive)
	}
	var effects int
	if err = db.QueryRow(context.Background(), `SELECT count(*) FROM work_item_effects WHERE kind='soft_delete'`).Scan(&effects); err != nil {
		t.Fatal(err)
	}
	if effects != 3 {
		t.Fatalf("soft-delete effects=%d, want one durable effect per deleted label", effects)
	}
}

func TestDeleteLabelRejectsForeignScopeChildBeforeQueuingCleanup(t *testing.T) {
	for _, test := range []struct {
		name    string
		deleted bool
	}{{"live foreign child", false}, {"soft-deleted foreign child", true}} {
		t.Run(test.name, func(t *testing.T) {
			db := parentIntegrityFixture(t)
			if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=$2,deleted_at=CASE WHEN $3 THEN now() ELSE deleted_at END WHERE id=$1`, foreignLabel, labelA, test.deleted); err != nil {
				t.Fatal(err)
			}
			repository, err := NewWriteRepository(db)
			if err != nil {
				t.Fatal(err)
			}
			_, err = repository.Write(context.Background(), parentActor, "demo", parentProject, labelA, http.MethodDelete, nil, "")
			assertWriteHTTPError(t, err, 409, map[string]any{"detail": "The label hierarchy contains an out-of-scope child."})
			var deleted, effects int
			if err = db.QueryRow(context.Background(), `SELECT count(*) FILTER (WHERE deleted_at IS NOT NULL) FROM labels WHERE id=$1`, labelA).Scan(&deleted); err != nil {
				t.Fatal(err)
			}
			if err = db.QueryRow(context.Background(), `SELECT count(*) FROM work_item_effects`).Scan(&effects); err != nil {
				t.Fatal(err)
			}
			if deleted != 0 || effects != 0 {
				t.Fatalf("rejected foreign-child delete changed root=%d effects=%d", deleted, effects)
			}
		})
	}
}

func TestDeleteLabelCascadeReachesLiveDescendantsBehindDeletedAncestor(t *testing.T) {
	db := parentIntegrityFixture(t)
	if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=$2 WHERE id=$1`, labelB, labelA); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(context.Background(), `UPDATE labels SET deleted_at=now() WHERE id=$1`, labelB); err != nil {
		t.Fatal(err)
	}
	repository, err := NewWriteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repository.Write(context.Background(), parentActor, "demo", parentProject, labelA, http.MethodDelete, nil, ""); err != nil {
		t.Fatal(err)
	}
	var rootDeleted, childDeleted bool
	if err = db.QueryRow(context.Background(), `SELECT deleted_at IS NOT NULL FROM labels WHERE id=$1`, labelA).Scan(&rootDeleted); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(context.Background(), `SELECT deleted_at IS NOT NULL FROM labels WHERE id=$1`, labelC).Scan(&childDeleted); err != nil {
		t.Fatal(err)
	}
	var effects int
	if err = db.QueryRow(context.Background(), `SELECT count(*) FROM work_item_effects`).Scan(&effects); err != nil {
		t.Fatal(err)
	}
	if !rootDeleted || !childDeleted || effects != 2 {
		t.Fatalf("root deleted=%v live grandchild deleted=%v effects=%d, want both deleted and one effect per changed label", rootDeleted, childDeleted, effects)
	}
}

func TestWriteLabelRejectsParentFromAnotherWorkspace(t *testing.T) {
	db := parentIntegrityFixture(t)
	_, err := db.Exec(context.Background(), `UPDATE labels SET project_id=$2,workspace_id=$3 WHERE id=$1`, foreignLabel, parentProject, otherWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := NewWriteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.Write(context.Background(), parentActor, "demo", parentProject, labelA,
		http.MethodPatch, strings.NewReader(fmt.Sprintf(`{"parent":%q}`, foreignLabel)), "application/json")
	if err == nil {
		t.Fatal("accepted parent whose workspace differs from the project")
	}
}

func TestWriteLabelKeepsValidSameProjectParent(t *testing.T) {
	db := parentIntegrityFixture(t)
	repository, err := NewWriteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.Write(context.Background(), parentActor, "demo", parentProject, labelA,
		http.MethodPatch, strings.NewReader(fmt.Sprintf(`{"parent":%q}`, labelB)), "application/json")
	if err != nil {
		t.Fatalf("valid same-project parent rejected: %v", err)
	}
	var parent string
	if err = db.QueryRow(context.Background(), `SELECT parent_id::text FROM labels WHERE id=$1`, labelA).Scan(&parent); err != nil {
		t.Fatal(err)
	}
	if parent != labelB {
		t.Fatalf("parent = %s, want %s", parent, labelB)
	}
}

func TestDeleteLabelCascadeRollsBackIfAnyCleanupCannotBeSaved(t *testing.T) {
	db := parentIntegrityFixture(t)
	if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=$2 WHERE id=$1`, labelB, labelA); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(context.Background(), `CREATE FUNCTION reject_label_cleanup() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected label cleanup failure'; END $$;
CREATE TRIGGER reject_label_cleanup BEFORE INSERT ON work_item_effects FOR EACH STATEMENT EXECUTE FUNCTION reject_label_cleanup()`); err != nil {
		t.Fatal(err)
	}
	repository, err := NewWriteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.Write(context.Background(), parentActor, "demo", parentProject, labelA, http.MethodDelete, strings.NewReader(`{"broken":`), "application/json")
	var postgres *pgconn.PgError
	if !errors.As(err, &postgres) || postgres.Message != "injected label cleanup failure" {
		t.Fatalf("delete error = %v, want injected cleanup failure", err)
	}
	var deleted, effects int
	if err = db.QueryRow(context.Background(), `SELECT count(*) FILTER (WHERE deleted_at IS NOT NULL) FROM labels WHERE id IN ($1,$2,$3)`, labelA, labelB, labelC).Scan(&deleted); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(context.Background(), `SELECT count(*) FROM work_item_effects`).Scan(&effects); err != nil {
		t.Fatal(err)
	}
	if deleted != 0 || effects != 0 {
		t.Fatalf("failed cascade committed %d deletions and %d effects", deleted, effects)
	}
}

func TestRepeatedLabelCascadeDeleteDoesNotQueueDuplicateEffects(t *testing.T) {
	db := parentIntegrityFixture(t)
	if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=$2 WHERE id=$1`, labelB, labelA); err != nil {
		t.Fatal(err)
	}
	repository, err := NewWriteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repository.Write(context.Background(), parentActor, "demo", parentProject, labelA, http.MethodDelete, nil, ""); err != nil {
		t.Fatal(err)
	}
	_, err = repository.Write(context.Background(), parentActor, "demo", parentProject, labelA, http.MethodDelete, nil, "")
	assertWriteHTTPError(t, err, 404, map[string]any{"detail": "No Label matches the given query."})
	var effects int
	if err = db.QueryRow(context.Background(), `SELECT count(*) FROM work_item_effects`).Scan(&effects); err != nil {
		t.Fatal(err)
	}
	if effects != 3 {
		t.Fatalf("repeated delete queued %d effects, want exactly one per deleted label", effects)
	}
}

func TestConcurrentLabelCascadeDeletesQueueEffectsOnce(t *testing.T) {
	db := parentIntegrityFixture(t)
	if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=$2 WHERE id=$1`, labelB, labelA); err != nil {
		t.Fatal(err)
	}
	secondDB, err := pgx.Connect(context.Background(), db.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := secondDB.Close(context.WithoutCancel(context.Background())); closeErr != nil {
			t.Errorf("close second postgres connection: %v", closeErr)
		}
	})
	firstRepository, err := NewWriteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	secondRepository, err := NewWriteRepository(secondDB)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	ready := make(chan struct{}, 2)
	results := make(chan error, 2)
	remove := func(repository *WriteRepository) {
		ready <- struct{}{}
		<-start
		_, deleteErr := repository.Write(context.Background(), parentActor, "demo", parentProject, labelA, http.MethodDelete, nil, "")
		results <- deleteErr
	}
	go remove(firstRepository)
	go remove(secondRepository)
	<-ready
	<-ready
	close(start)
	firstErr, secondErr := <-results, <-results
	successes, missing := 0, 0
	for _, resultErr := range []error{firstErr, secondErr} {
		if resultErr == nil {
			successes++
			continue
		}
		var httpError *HTTPError
		if !errors.As(resultErr, &httpError) || httpError.Status != 404 {
			t.Fatalf("concurrent delete error = %v, want HTTP 404 for the already-deleted tree", resultErr)
		}
		missing++
	}
	if successes != 1 || missing != 1 {
		t.Fatalf("concurrent deletes: success=%d missing=%d", successes, missing)
	}
	var effects int
	if err = db.QueryRow(context.Background(), `SELECT count(*) FROM work_item_effects`).Scan(&effects); err != nil {
		t.Fatal(err)
	}
	if effects != 3 {
		t.Fatalf("concurrent delete queued %d effects, want exactly three", effects)
	}
}

func TestConcurrentAppAndAPIParentMovesShareProjectSerialization(t *testing.T) {
	db := parentIntegrityFixture(t)
	secondDB, err := pgx.Connect(context.Background(), db.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := secondDB.Close(context.WithoutCancel(context.Background())); closeErr != nil {
			t.Errorf("close API connection: %v", closeErr)
		}
	})
	appRepository, err := NewWriteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	apiRepository, err := projectconfiguration.NewRepository(secondDB)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	ready := make(chan struct{}, 2)
	results := make(chan error, 2)
	go func() {
		ready <- struct{}{}
		<-start
		_, writeErr := appRepository.Write(context.Background(), parentActor, "demo", parentProject, labelA,
			http.MethodPatch, strings.NewReader(fmt.Sprintf(`{"parent":%q}`, labelB)), "application/json")
		results <- writeErr
	}()
	go func() {
		ready <- struct{}{}
		<-start
		_, writeErr := apiRepository.WriteAPILabel(context.Background(), parentActor, "demo", parentProject, labelB,
			http.MethodPatch, strings.NewReader(fmt.Sprintf(`{"parent":%q}`, labelA)), "application/json")
		results <- writeErr
	}()
	<-ready
	<-ready
	close(start)
	firstErr, secondErr := <-results, <-results
	successes := 0
	for _, resultErr := range []error{firstErr, secondErr} {
		if resultErr == nil {
			successes++
			continue
		}
		var appError *HTTPError
		var apiError *projectconfiguration.HTTPError
		if (!errors.As(resultErr, &appError) || appError.Status != 400) &&
			(!errors.As(resultErr, &apiError) || apiError.Status != 400) {
			t.Fatalf("mixed app/API reciprocal move error = %v, want HTTP 400", resultErr)
		}
	}
	if successes != 1 {
		t.Fatalf("successful mixed app/API moves = %d, want exactly one (errors: %v, %v)", successes, firstErr, secondErr)
	}
	var cycle bool
	if err = db.QueryRow(context.Background(), `WITH RECURSIVE ancestry(id,parent_id,path,cycle) AS (
 SELECT id,parent_id,ARRAY[id],false FROM labels WHERE id=$1
 UNION ALL
 SELECT parent.id,parent.parent_id,ancestry.path || parent.id,parent.id=ANY(ancestry.path)
 FROM ancestry JOIN labels parent ON parent.id=ancestry.parent_id WHERE NOT ancestry.cycle
) SELECT COALESCE(bool_or(cycle),false) FROM ancestry`, labelA).Scan(&cycle); err != nil {
		t.Fatal(err)
	}
	if cycle {
		t.Fatal("mixed app/API parent moves persisted a cycle")
	}
}

func TestClearingParentCanRepairBrokenAncestry(t *testing.T) {
	db := parentIntegrityFixture(t)
	if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=$2 WHERE id=$1`, labelB, foreignLabel); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=$2 WHERE id=$1`, labelA, labelB); err != nil {
		t.Fatal(err)
	}
	repository, err := NewWriteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	label, err := repository.Write(context.Background(), parentActor, "demo", parentProject, labelA,
		http.MethodPatch, strings.NewReader(`{"parent":null}`), "application/json")
	if err != nil {
		t.Fatalf("clearing parent should repair invalid ancestry: %v", err)
	}
	if label.Parent != nil {
		t.Fatalf("parent = %v, want nil", label.Parent)
	}
}
