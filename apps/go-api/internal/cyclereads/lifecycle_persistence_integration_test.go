//go:build integration

package cyclereads

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

const lifecycleWorkspace = "10000000-0000-4000-8000-000000000001"
const lifecycleProject = "10000000-0000-4000-8000-000000000002"
const lifecycleActor = "10000000-0000-4000-8000-000000000003"

const cycleLifecycleSchema = `
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text);
CREATE TABLE users(id uuid PRIMARY KEY,user_timezone text);
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,timezone text,cycle_view bool,deleted_at timestamptz);
CREATE TABLE cycles(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,name text,description text,
 created_at timestamptz,updated_at timestamptz,deleted_at timestamptz,archived_at timestamptz,
 start_date timestamptz,end_date timestamptz,owned_by_id uuid,created_by_id uuid,updated_by_id uuid,
 view_props jsonb,progress_snapshot jsonb,logo_props jsonb,sort_order float8,external_source text,external_id text,timezone text,version int);
CREATE TABLE user_favorites(entity_type text,entity_identifier uuid,project_id uuid,workspace_id uuid,deleted_at timestamptz);
INSERT INTO workspaces VALUES('10000000-0000-4000-8000-000000000001','lifecycle');
INSERT INTO users VALUES('10000000-0000-4000-8000-000000000003','UTC');
INSERT INTO projects VALUES('10000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000001','UTC',true,NULL);
`

type cycleCreationEffects struct {
	lifecycleEffectsStub
	task string
	data map[string]any
}

func (effects *cycleCreationEffects) Task(_ context.Context, task string, _ []any, data map[string]any) error {
	effects.task, effects.data = task, data
	return nil
}

func cycleLifecycleFixture(t *testing.T) (context.Context, *pgx.Conn, *LifecycleRepository, *cycleCreationEffects) {
	t.Helper()
	ctx, connection := postgresfixture.OpenSQL(t, cycleLifecycleSchema)
	effects := &cycleCreationEffects{}
	repository, err := NewLifecycleRepository(connection, effects, "")
	if err != nil {
		t.Fatal(err)
	}
	return ctx, connection, repository, effects
}

func TestCycleAPICreatePersistsDefaultsAndAdmitsOwnedWebhook(t *testing.T) {
	ctx, connection, repository, effects := cycleLifecycleFixture(t)
	ctx = WithLifecycleRequestOrigin(ctx, "https", "example.test")
	start, end := "2001-01-01", "2001-01-02"
	requested := map[string]any{"name": "Release"}
	result, err := repository.Create(ctx, lifecycleActor, "lifecycle", lifecycleProject,
		CycleCreate{Name: "Release", StartDate: &start, EndDate: &end}, true, requested)
	if err != nil {
		t.Fatal(err)
	}
	if result["status"] != "COMPLETED" || result["owned_by"] != lifecycleActor || result["is_favorite"] != false {
		t.Fatalf("created cycle projection = %#v", result)
	}
	id, ok := result["id"].(string)
	if !ok || id == "" {
		t.Fatalf("created cycle ID = %#v", result["id"])
	}
	var name, owner, zone string
	var sortOrder float64
	var version int
	if err = connection.QueryRow(ctx, `SELECT name,owned_by_id::text,timezone,sort_order,version FROM cycles WHERE id=$1 AND project_id=$2 AND workspace_id=$3`, id, lifecycleProject, lifecycleWorkspace).Scan(&name, &owner, &zone, &sortOrder, &version); err != nil {
		t.Fatal(err)
	}
	if name != "Release" || owner != lifecycleActor || zone != "UTC" || sortOrder != 65535 || version != 1 {
		t.Fatalf("persisted cycle = %s %s %s %g %d", name, owner, zone, sortOrder, version)
	}
	if effects.task != "plane.bgtasks.webhook_task.model_activity" || effects.data["model_id"] != id || effects.data["actor_id"] != lifecycleActor || effects.data["origin"] != "https://example.test" {
		t.Fatalf("webhook ownership = %s %#v", effects.task, effects.data)
	}
}

func TestCycleAPICreateRejectsDisabledProjectWithoutWriting(t *testing.T) {
	ctx, connection, repository, effects := cycleLifecycleFixture(t)
	if _, err := connection.Exec(ctx, `UPDATE projects SET cycle_view=false`); err != nil {
		t.Fatal(err)
	}
	_, err := repository.Create(ctx, lifecycleActor, "lifecycle", lifecycleProject, CycleCreate{Name: "Disabled"}, true, nil)
	var invalid *djangoapi.HTTPError
	if !errors.As(err, &invalid) || invalid.Status != 400 {
		t.Fatalf("disabled project error = %v", err)
	}
	var count int
	if err = connection.QueryRow(ctx, `SELECT count(*) FROM cycles`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 || effects.task != "" {
		t.Fatalf("disabled create wrote %d cycles or webhook %q", count, effects.task)
	}
}

func TestCycleArchiveAndRestorePreserveFavoriteDeletion(t *testing.T) {
	ctx, connection, repository, _ := cycleLifecycleFixture(t)
	const cycle = "10000000-0000-4000-8000-000000000004"
	if _, err := connection.Exec(ctx, `INSERT INTO cycles(id,workspace_id,project_id,name,end_date) VALUES($1,$2,$3,'Completed','2001-01-02')`, cycle, lifecycleWorkspace, lifecycleProject); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, `INSERT INTO user_favorites VALUES('cycle',$1,$3,$2,NULL)`, cycle, lifecycleWorkspace, lifecycleProject); err != nil {
		t.Fatal(err)
	}
	archived, err := repository.SetArchived(ctx, "lifecycle", lifecycleProject, cycle, lifecycleActor, true)
	if err != nil || archived.IsZero() {
		t.Fatalf("archive = %v %v", archived, err)
	}
	var archivedAt, favoriteDeleted *time.Time
	if err = connection.QueryRow(ctx, `SELECT c.archived_at,f.deleted_at FROM cycles c JOIN user_favorites f ON f.entity_identifier=c.id WHERE c.id=$1`, cycle).Scan(&archivedAt, &favoriteDeleted); err != nil {
		t.Fatal(err)
	}
	if archivedAt == nil || favoriteDeleted == nil {
		t.Fatal("archive did not remove the favorite")
	}
	restored, err := repository.SetArchived(ctx, "lifecycle", lifecycleProject, cycle, lifecycleActor, false)
	if err != nil || !restored.IsZero() {
		t.Fatalf("restore = %v %v", restored, err)
	}
	if err = connection.QueryRow(ctx, `SELECT c.archived_at,f.deleted_at FROM cycles c JOIN user_favorites f ON f.entity_identifier=c.id WHERE c.id=$1`, cycle).Scan(&archivedAt, &favoriteDeleted); err != nil {
		t.Fatal(err)
	}
	if archivedAt != nil || favoriteDeleted == nil {
		t.Fatal("restore changed favorite deletion or left the cycle archived")
	}
}
