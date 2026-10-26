//go:build integration

package projectconfiguration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

const (
	apiLabelActor            = "00000000-0000-4000-8000-000000000001"
	apiLabelWorkspace        = "00000000-0000-4000-8000-000000000002"
	apiLabelProject          = "00000000-0000-4000-8000-000000000003"
	apiOtherWorkspace        = "00000000-0000-4000-8000-000000000004"
	apiOtherProject          = "00000000-0000-4000-8000-000000000005"
	apiLabelA                = "00000000-0000-4000-8000-000000000011"
	apiLabelB                = "00000000-0000-4000-8000-000000000012"
	apiLabelC                = "00000000-0000-4000-8000-000000000013"
	apiForeignLabel          = "00000000-0000-4000-8000-000000000014"
	apiForeignWorkspaceChild = "00000000-0000-4000-8000-000000000015"
)

func apiLabelFixture(t *testing.T) *pgx.Conn {
	t.Helper()
	_, connection := postgresfixture.OpenSQL(t, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, archived_at timestamptz);
CREATE TABLE project_members (project_id uuid NOT NULL, member_id uuid NOT NULL, is_active boolean NOT NULL);
CREATE TABLE labels (id uuid PRIMARY KEY, name text NOT NULL, color text NOT NULL, parent_id uuid,
 project_id uuid, workspace_id uuid NOT NULL, sort_order double precision NOT NULL,
 description text NOT NULL DEFAULT '', external_source text, external_id text,
 created_at timestamptz, updated_at timestamptz, created_by_id uuid, updated_by_id uuid, deleted_at timestamptz);
CREATE TABLE work_item_effects (id uuid PRIMARY KEY, kind text NOT NULL, payload jsonb NOT NULL);
INSERT INTO workspaces VALUES
 ('00000000-0000-4000-8000-000000000002', 'demo'),
 ('00000000-0000-4000-8000-000000000004', 'other');
INSERT INTO projects VALUES
 ('00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000002', NULL),
 ('00000000-0000-4000-8000-000000000005', '00000000-0000-4000-8000-000000000004', NULL);
INSERT INTO project_members VALUES
 ('00000000-0000-4000-8000-000000000003', '00000000-0000-4000-8000-000000000001', true),
 ('00000000-0000-4000-8000-000000000005', '00000000-0000-4000-8000-000000000001', true);
INSERT INTO labels (id,name,color,parent_id,project_id,workspace_id,sort_order,created_at) VALUES
 ('00000000-0000-4000-8000-000000000011','A','#111111',NULL,'00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002',10,now()),
 ('00000000-0000-4000-8000-000000000012','B','#222222',NULL,'00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002',20,now()),
 ('00000000-0000-4000-8000-000000000013','C','#333333','00000000-0000-4000-8000-000000000012','00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002',30,now()),
 ('00000000-0000-4000-8000-000000000014','Foreign','#444444',NULL,'00000000-0000-4000-8000-000000000005','00000000-0000-4000-8000-000000000004',40,now()),
 ('00000000-0000-4000-8000-000000000015','Foreign workspace child','#555555',NULL,'00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000004',50,now());`)
	return connection
}

func TestWriteAPILabelRejectsInvalidParentRelationships(t *testing.T) {
	for _, test := range []struct {
		name, targetID, parentID string
		setup                    func(*testing.T, *pgx.Conn)
	}{
		{"cross-project", apiLabelA, apiForeignLabel, nil},
		{"self", apiLabelA, apiLabelA, nil},
		{"descendant", apiLabelB, apiLabelC, nil},
		{"preexisting-cycle", apiLabelC, apiLabelA, func(t *testing.T, db *pgx.Conn) {
			if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=CASE id WHEN $1 THEN $2 WHEN $2 THEN $1 ELSE parent_id END WHERE id IN ($1,$2)`, apiLabelA, apiLabelB); err != nil {
				t.Fatal(err)
			}
		}},
		{"cross-workspace", apiLabelA, apiForeignLabel, func(t *testing.T, db *pgx.Conn) {
			if _, err := db.Exec(context.Background(), `UPDATE labels SET project_id=$2,workspace_id=$3 WHERE id=$1`, apiForeignLabel, apiLabelProject, apiOtherWorkspace); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := apiLabelFixture(t)
			if test.setup != nil {
				test.setup(t, db)
			}
			repository, err := NewRepository(db)
			if err != nil {
				t.Fatal(err)
			}
			_, err = repository.WriteAPILabel(context.Background(), apiLabelActor, "demo", apiLabelProject, test.targetID,
				http.MethodPatch, strings.NewReader(fmt.Sprintf(`{"parent":%q}`, test.parentID)), "application/json")
			var httpError *HTTPError
			if !errors.As(err, &httpError) || httpError.Status != 400 {
				t.Fatalf("invalid parent error = %v, want HTTP 400", err)
			}
		})
	}
}

func TestConcurrentReciprocalAPILabelMovesCannotCreateCycle(t *testing.T) {
	db := apiLabelFixture(t)
	secondDB, err := pgx.Connect(context.Background(), db.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := secondDB.Close(context.WithoutCancel(context.Background())); closeErr != nil {
			t.Errorf("close second postgres connection: %v", closeErr)
		}
	})
	firstRepository, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	secondRepository, err := NewRepository(secondDB)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	ready := make(chan struct{}, 2)
	results := make(chan error, 2)
	write := func(repository *Repository, childID, parentID string) {
		ready <- struct{}{}
		<-start
		_, writeErr := repository.WriteAPILabel(context.Background(), apiLabelActor, "demo", apiLabelProject, childID,
			http.MethodPatch, strings.NewReader(fmt.Sprintf(`{"parent":%q}`, parentID)), "application/json")
		results <- writeErr
	}
	go write(firstRepository, apiLabelA, apiLabelB)
	go write(secondRepository, apiLabelB, apiLabelA)
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
			t.Fatalf("reciprocal API parent move error = %v, want HTTP 400", resultErr)
		}
	}
	if successes != 1 {
		t.Fatalf("successful reciprocal API moves = %d, want exactly one (errors: %v, %v)", successes, firstErr, secondErr)
	}
}

func TestDeleteAPILabelSoftDeletesDescendantsAndQueuesEachCleanup(t *testing.T) {
	db := apiLabelFixture(t)
	if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=$2 WHERE id=$1`, apiLabelB, apiLabelA); err != nil {
		t.Fatal(err)
	}
	repository, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.WriteAPILabel(context.Background(), apiLabelActor, "demo", apiLabelProject, apiLabelA, http.MethodDelete, strings.NewReader(`{"broken":`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	var deleted, count int
	if err = db.QueryRow(context.Background(), `SELECT count(*) FILTER (WHERE deleted_at IS NOT NULL),count(*)
 FROM labels WHERE id IN ($1,$2,$3)`, apiLabelA, apiLabelB, apiLabelC).Scan(&deleted, &count); err != nil {
		t.Fatal(err)
	}
	if deleted != 3 || count != 3 {
		t.Fatalf("deleted descendants=%d, present rows=%d; want all three soft-deleted", deleted, count)
	}
	var outOfScopeActive int
	if err = db.QueryRow(context.Background(), `SELECT count(*) FROM labels WHERE id IN ($1,$2) AND deleted_at IS NULL`, apiForeignLabel, apiForeignWorkspaceChild).Scan(&outOfScopeActive); err != nil {
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

func TestDeleteAPILabelRejectsForeignScopeChildBeforeQueuingCleanup(t *testing.T) {
	db := apiLabelFixture(t)
	if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=$2 WHERE id=$1`, apiForeignLabel, apiLabelA); err != nil {
		t.Fatal(err)
	}
	repository, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.WriteAPILabel(context.Background(), apiLabelActor, "demo", apiLabelProject, apiLabelA, http.MethodDelete, nil, "")
	var httpError *HTTPError
	if !errors.As(err, &httpError) || httpError.Status != 409 {
		t.Fatalf("foreign-child API delete error = %v, want HTTP 409", err)
	}
	var deleted, effects int
	if err = db.QueryRow(context.Background(), `SELECT count(*) FILTER (WHERE deleted_at IS NOT NULL) FROM labels WHERE id=$1`, apiLabelA).Scan(&deleted); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(context.Background(), `SELECT count(*) FROM work_item_effects`).Scan(&effects); err != nil {
		t.Fatal(err)
	}
	if deleted != 0 || effects != 0 {
		t.Fatalf("rejected foreign-child API delete changed root=%d effects=%d", deleted, effects)
	}
}

func TestClearingAPIParentCanRepairBrokenAncestry(t *testing.T) {
	db := apiLabelFixture(t)
	if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=$2 WHERE id=$1`, apiLabelB, apiForeignLabel); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(context.Background(), `UPDATE labels SET parent_id=$2 WHERE id=$1`, apiLabelA, apiLabelB); err != nil {
		t.Fatal(err)
	}
	repository, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.WriteAPILabel(context.Background(), apiLabelActor, "demo", apiLabelProject, apiLabelA,
		http.MethodPatch, strings.NewReader(`{"parent":null}`), "application/json")
	if err != nil {
		t.Fatalf("clearing API parent should repair invalid ancestry: %v", err)
	}
	var parent string
	if err = db.QueryRow(context.Background(), `SELECT COALESCE(parent_id::text,'') FROM labels WHERE id=$1`, apiLabelA).Scan(&parent); err != nil {
		t.Fatal(err)
	}
	if parent != "" {
		t.Fatalf("parent = %q, want cleared", parent)
	}
}
