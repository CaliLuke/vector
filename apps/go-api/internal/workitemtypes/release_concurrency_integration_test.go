//go:build integration

package workitemtypes_test

import (
	"testing"
	"uuid"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

func queuedReleaseDelete(t *testing.T, f projectTypeFixture, input types.ReleaseInput, change func(pgx.Tx)) error {
	t.Helper()
	connection := hierarchyConnection(t, f)
	repository := types.NewRepository(connection)
	lock := hierarchyTransaction(t, f)
	done := make(chan error, 1)
	go func() {
		_, err := repository.MutateReleaseCatalog(f.ctx, f.owner, "types", f.project, input)
		done <- err
	}()
	waitForSchemaWriter(t, f, lock, int(connection.PgConn().PID()), done)
	change(lock)
	if err := lock.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		return err
	case <-f.ctx.Done():
		t.Fatal("Queued release mutation did not finish")
		return f.ctx.Err()
	}
}
func TestReleaseDeletionRechecksReferencesAfterWaiting(t *testing.T) {
	f := newProjectTypeFixture(t)
	field := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: field, Title: "Delivery", Kind: property.Release, Configuration: property.Configuration{Active: true}})
	input := createTestRelease(t, f)
	item := addHierarchyItem(t, f, f.task, "")
	input.Action = "delete"
	input.Revision = releaseCatalog(t, f).Revision
	err := queuedReleaseDelete(t, f, input, func(tx pgx.Tx) {
		if _, err := tx.Exec(f.ctx, `UPDATE issues SET custom_properties=jsonb_build_object($2::text,$3::text) WHERE id=$1`, item, field, input.ID); err != nil {
			t.Fatal(err)
		}
	})
	requireTypeStatus(t, err, 422)
	if after := releaseCatalog(t, f); len(after.Releases) != 1 || after.Revision != input.Revision {
		t.Fatalf("referenced release changed: %+v", after)
	}
}
func TestReleaseDeletionRechecksAdministratorAfterWaiting(t *testing.T) {
	f := newProjectTypeFixture(t)
	input := createTestRelease(t, f)
	input.Action = "delete"
	input.Revision = releaseCatalog(t, f).Revision
	err := queuedReleaseDelete(t, f, input, func(tx pgx.Tx) {
		if _, err := tx.Exec(f.ctx, `UPDATE project_members SET role=15 WHERE project_id=$1 AND member_id=$2`, f.project, f.owner); err != nil {
			t.Fatal(err)
		}
	})
	requireTypeStatus(t, err, 403)
	if after := releaseCatalog(t, f); len(after.Releases) != 1 || after.Revision != input.Revision {
		t.Fatalf("demoted admin changed release: %+v", after)
	}
}
func TestReleaseCatalogBeforeTypeActivation(t *testing.T) {
	f := newTypeFixture(t)
	f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,20,true,now(),now(),'{}','{}','{}',0)`, f.workspace, f.project, f.owner)
	repo := types.NewRepository(f.db)
	created, err := repo.MutateReleaseCatalog(f.ctx, f.owner, "types", f.project, types.ReleaseInput{Revision: 0, Action: "create", ID: uuid.New().String(), Values: types.ReleaseValues{Name: "First"}})
	if err != nil || created.Revision != 1 || len(created.Releases) != 1 {
		t.Fatalf("catalog before rollout: %+v %v", created, err)
	}
	catalog, err := repo.Catalog(f.ctx, f.owner, "types")
	if err != nil || catalog.Enabled {
		t.Fatalf("release creation enabled types: %+v %v", catalog, err)
	}
	enabled, err := repo.Mutate(f.ctx, f.owner, "types", "enable", "", types.Input{Revision: catalog.Revision})
	if err != nil || !enabled.Enabled || len(enabled.Types) != 1 {
		t.Fatalf("activation after catalog creation: %+v %v", enabled, err)
	}
}
