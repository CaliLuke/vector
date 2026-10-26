//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"testing"
	"uuid"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func TestQueuedSchemaWritesRecheckWorkspaceAdministrator(t *testing.T) {
	for _, operation := range []string{"type", "property", "property preview", "property commit"} {
		t.Run(operation, func(t *testing.T) {
			f := newProjectTypeFixture(t)
			def := property.Definition{ID: uuid.New().String(), Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}}
			attachValueProperties(t, f, def)
			before, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
			if err != nil {
				t.Fatal(err)
			}
			command := types.PropertyInput{Action: "delete", ID: def.ID, Revision: before.Revision}
			if operation == "property commit" {
				review, reviewErr := f.repo.PreviewPropertyRemoval(f.ctx, f.owner, "types", command)
				if reviewErr != nil {
					t.Fatal(reviewErr)
				}
				command.ReviewID = review.ID
			}
			var reviewsBefore int
			if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM work_item_schema_reviews WHERE workspace_id=$1`, f.workspace).Scan(&reviewsBefore); err != nil {
				t.Fatal(err)
			}
			connection := hierarchyConnection(t, f)
			repository := types.NewRepository(connection)
			lock := hierarchyTransaction(t, f)
			done := make(chan error, 1)
			go func() {
				var writeErr error
				switch operation {
				case "type":
					_, writeErr = repository.Mutate(f.ctx, f.owner, "types", "delete", f.bug, types.Input{Revision: before.Revision})
				case "property preview":
					_, writeErr = repository.PreviewPropertyRemoval(f.ctx, f.owner, "types", command)
				default:
					_, writeErr = repository.MutateProperty(f.ctx, f.owner, "types", command)
				}
				done <- writeErr
			}()
			waitForSchemaWriter(t, f, lock, int(connection.PgConn().PID()), done)
			if _, err = lock.Exec(f.ctx, `UPDATE workspace_members SET role=15 WHERE workspace_id=$1 AND member_id=$2`, f.workspace, f.owner); err != nil {
				t.Fatal(err)
			}
			if err = lock.Commit(f.ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case err = <-done:
				requireTypeStatus(t, err, 403)
			case <-f.ctx.Done():
				t.Fatal("queued write did not finish")
			}
			after, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
			if err != nil {
				t.Fatal(err)
			}
			before.CanManage = false
			beforeJSON, _ := json.Marshal(before)
			afterJSON, _ := json.Marshal(after)
			if string(beforeJSON) != string(afterJSON) {
				t.Fatal("demoted administrator changed configuration")
			}
			var reviewsAfter, committed int
			if err = f.db.QueryRow(f.ctx, `SELECT count(*),count(committed_at) FROM work_item_schema_reviews WHERE workspace_id=$1`, f.workspace).Scan(&reviewsAfter, &committed); err != nil {
				t.Fatal(err)
			}
			if reviewsAfter != reviewsBefore || committed != 0 {
				t.Fatal("demoted administrator created or committed a review")
			}
		})
	}
}

func TestQueuedSchemaWriteRejectsWorkspaceSlugReassignment(t *testing.T) {
	f := newProjectTypeFixture(t)
	before := f.catalog(t)
	connection := hierarchyConnection(t, f)
	repository := types.NewRepository(connection)
	lock := hierarchyTransaction(t, f)
	done := make(chan error, 1)
	go func() {
		_, err := repository.Mutate(f.ctx, f.owner, "types", "delete", f.bug, types.Input{Revision: before.Revision})
		done <- err
	}()
	waitForSchemaWriter(t, f, lock, int(connection.PgConn().PID()), done)
	if _, err := lock.Exec(f.ctx, `UPDATE workspaces SET slug='renamed' WHERE id=$1`, f.workspace); err != nil {
		t.Fatal(err)
	}
	replacement := uuid.New().String()
	if _, err := lock.Exec(f.ctx, `INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Replacement',$2,'types','UTC','',now(),now())`, replacement, f.owner); err != nil {
		t.Fatal(err)
	}
	if _, err := lock.Exec(f.ctx, `INSERT INTO workspace_members(id,workspace_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,issue_props,getting_started_checklist,tips,explored_features) VALUES(gen_random_uuid(),$1,$2,20,true,now(),now(),'{}','{}','{}','{}','{}','{}')`, replacement, f.owner); err != nil {
		t.Fatal(err)
	}
	if err := lock.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		requireTypeStatus(t, err, 409)
	case <-f.ctx.Done():
		t.Fatal("queued write did not finish")
	}
	after, err := f.repo.ProjectCatalog(f.ctx, f.owner, "renamed", f.project)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Types) != len(before.Types) || after.Revision != before.Revision {
		t.Fatal("queued write changed original workspace after slug reassignment")
	}
}
