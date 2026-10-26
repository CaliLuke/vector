//go:build integration

package projectlabelreads

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const deleteActor = "00000000-0000-4000-8000-000000000003"

func labelDeletionFixture(t *testing.T) *pgx.Conn {
	t.Helper()
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		db, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close(context.WithoutCancel(ctx)) }()
		_, err = db.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid NOT NULL);
CREATE TABLE project_members (project_id uuid NOT NULL, member_id uuid NOT NULL);
CREATE TABLE labels (id uuid PRIMARY KEY, name text NOT NULL, color text NOT NULL, parent_id uuid,
 project_id uuid, workspace_id uuid NOT NULL, sort_order double precision NOT NULL,
 deleted_at timestamptz, updated_at timestamptz, updated_by_id uuid);
CREATE TABLE work_item_effects (id uuid PRIMARY KEY, kind text NOT NULL, payload jsonb NOT NULL);
INSERT INTO workspaces VALUES ('00000000-0000-4000-8000-000000000001', 'demo');
INSERT INTO projects VALUES ('11111111-1111-4111-8111-111111111111', '00000000-0000-4000-8000-000000000001');
INSERT INTO project_members VALUES ('11111111-1111-4111-8111-111111111111', '00000000-0000-4000-8000-000000000003');
INSERT INTO labels(id,name,color,project_id,workspace_id,sort_order) VALUES
 ('22222222-2222-4222-8222-222222222222', 'Parent', '#ff0000', '11111111-1111-4111-8111-111111111111', '00000000-0000-4000-8000-000000000001', 10);
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pair.Close() })
	db, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close(context.WithoutCancel(ctx)) })
	return db
}

func TestDeleteLabelCommitsCleanupEffectWithDeletion(t *testing.T) {
	db := labelDeletionFixture(t)
	repository, err := NewWriteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.Write(context.Background(), deleteActor, "demo", writeProject, writeID, http.MethodDelete, strings.NewReader(`{"ignored":`), "text/plain")
	if err != nil {
		t.Fatalf("delete must not depend on cleanup admission: %v", err)
	}
	var deleted bool
	var actor string
	if err = db.QueryRow(context.Background(), `SELECT deleted_at IS NOT NULL,updated_by_id::text FROM labels WHERE id=$1`, writeID).Scan(&deleted, &actor); err != nil {
		t.Fatal(err)
	}
	if !deleted || actor != deleteActor {
		t.Fatalf("deletion = %v, actor = %q", deleted, actor)
	}
	var raw []byte
	var effects int
	if err = db.QueryRow(context.Background(), `SELECT count(*) FROM work_item_effects`).Scan(&effects); err != nil {
		t.Fatal(err)
	}
	if effects != 1 {
		t.Fatalf("cleanup effects = %d, want exactly one", effects)
	}
	if err = db.QueryRow(context.Background(), `SELECT payload FROM work_item_effects WHERE kind='soft_delete'`).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var payload map[string]string
	if err = json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(payload, map[string]string{"app": "db", "model": "label", "id": writeID}) {
		t.Fatalf("cleanup payload = %s", raw)
	}
}

func TestDeleteLabelRollsBackWhenCleanupEffectCannotBeSaved(t *testing.T) {
	db := labelDeletionFixture(t)
	_, err := db.Exec(context.Background(), `CREATE FUNCTION reject_cleanup() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected effect storage failure'; END $$;
CREATE TRIGGER reject_cleanup BEFORE INSERT ON work_item_effects FOR EACH STATEMENT EXECUTE FUNCTION reject_cleanup()`)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := NewWriteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.Write(context.Background(), deleteActor, "demo", writeProject, writeID, http.MethodDelete, nil, "")
	var storageError *pgconn.PgError
	if !errors.As(err, &storageError) || storageError.Message != "injected effect storage failure" {
		t.Fatalf("expected injected effect storage failure, got %v", err)
	}
	var pristine bool
	if err = db.QueryRow(context.Background(), `SELECT deleted_at IS NULL AND updated_at IS NULL AND updated_by_id IS NULL FROM labels WHERE id=$1`, writeID).Scan(&pristine); err != nil {
		t.Fatal(err)
	}
	if !pristine {
		t.Fatal("failed cleanup persistence committed the label deletion")
	}
	var effects int
	if err = db.QueryRow(context.Background(), `SELECT count(*) FROM work_item_effects`).Scan(&effects); err != nil {
		t.Fatal(err)
	}
	if effects != 0 {
		t.Fatalf("failed delete left %d cleanup effects", effects)
	}
}

func TestDeleteLabelRejectsMismatchedScopeWithoutSavingEffect(t *testing.T) {
	db := labelDeletionFixture(t)
	repository, err := NewWriteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, user, workspace, project string }{
		{"workspace", deleteActor, "another-workspace", writeProject},
		{"project", deleteActor, "demo", "00000000-0000-4000-8000-000000000099"},
		{"actor", "00000000-0000-4000-8000-000000000099", "demo", writeProject},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, deleteErr := repository.Write(context.Background(), test.user, test.workspace, test.project, writeID, http.MethodDelete, nil, "")
			assertWriteHTTPError(t, deleteErr, 404, map[string]any{"detail": "No Label matches the given query."})
		})
	}
	var pristine bool
	var effects int
	if err = db.QueryRow(context.Background(), `SELECT deleted_at IS NULL AND updated_at IS NULL AND updated_by_id IS NULL FROM labels WHERE id=$1`, writeID).Scan(&pristine); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(context.Background(), `SELECT count(*) FROM work_item_effects`).Scan(&effects); err != nil {
		t.Fatal(err)
	}
	if !pristine || effects != 0 {
		t.Fatalf("mismatched scope changed label: pristine=%v, effects=%d", pristine, effects)
	}
}

func TestRepeatedLabelDeletionDoesNotQueueAnotherEffect(t *testing.T) {
	db := labelDeletionFixture(t)
	repository, err := NewWriteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repository.Write(context.Background(), deleteActor, "demo", writeProject, writeID, http.MethodDelete, nil, ""); err != nil {
		t.Fatal(err)
	}
	_, err = repository.Write(context.Background(), deleteActor, "demo", writeProject, writeID, http.MethodDelete, nil, "")
	assertWriteHTTPError(t, err, 404, map[string]any{"detail": "No Label matches the given query."})
	var effects int
	if err = db.QueryRow(context.Background(), `SELECT count(*) FROM work_item_effects`).Scan(&effects); err != nil {
		t.Fatal(err)
	}
	if effects != 1 {
		t.Fatalf("repeated deletion queued %d effects", effects)
	}
}
