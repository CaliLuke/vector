//go:build integration

package standardviews

import (
	"context"
	"errors"
	"testing"

	"uuid"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestDeleteViewCleansScopedReferences(t *testing.T) {
	ctx := context.Background()
	connection := deleteViewDatabase(t)
	user := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	project := uuid.MustParse("00000000-0000-4000-8000-000000000020")
	for _, scope := range []string{"workspace", "project"} {
		t.Run(scope, func(t *testing.T) {
			view := uuid.MustParse("00000000-0000-4000-8000-000000000030")
			var projectID *uuid.UUID
			if scope == "project" {
				view = uuid.MustParse("00000000-0000-4000-8000-000000000031")
				projectID = &project
			}
			for _, scenario := range []string{"success", "denied"} {
				t.Run(scenario, func(t *testing.T) {
					tx, err := connection.Begin(ctx)
					if err != nil {
						t.Fatal(err)
					}
					defer func() { _ = tx.Rollback(ctx) }()
					// Both users' favorites are removed, while unrelated tenant, kind and
					// project references to the same identifier must remain intact.
					if _, err = tx.Exec(ctx, `INSERT INTO user_favorites(user_id,workspace_id,project_id,entity_identifier,entity_type)
 SELECT u.id,v.workspace_id,v.project_id,v.id,'view' FROM issue_views v
 CROSS JOIN (VALUES ($2::uuid),('00000000-0000-4000-8000-000000000002'::uuid)) u(id) WHERE v.id=$1;
 `, pgx.QueryExecModeSimpleProtocol, view, user); err != nil {
						t.Fatal(err)
					}
					if _, err = tx.Exec(ctx, `INSERT INTO user_favorites(workspace_id,project_id,entity_identifier,entity_type)
 SELECT '00000000-0000-4000-8000-000000000011',project_id,id,'view' FROM issue_views WHERE id=$1
 UNION ALL SELECT workspace_id,project_id,id,'issue' FROM issue_views WHERE id=$1
 UNION ALL SELECT workspace_id,'00000000-0000-4000-8000-000000000099',id,'view' FROM issue_views WHERE id=$1;
 INSERT INTO user_recent_visits SELECT workspace_id,project_id,id,'view' FROM issue_views WHERE id=$1;
 INSERT INTO user_recent_visits SELECT '00000000-0000-4000-8000-000000000011',project_id,id,'view' FROM issue_views WHERE id=$1;
 INSERT INTO user_recent_visits SELECT workspace_id,project_id,id,'issue' FROM issue_views WHERE id=$1;`, pgx.QueryExecModeSimpleProtocol, view); err != nil {
						t.Fatal(err)
					}
					repository, err := NewRepository(tx)
					if err != nil {
						t.Fatal(err)
					}
					actor := user
					if scenario == "denied" {
						actor = uuid.MustParse("00000000-0000-4000-8000-000000000002")
					}
					if projectID == nil {
						err = repository.DeleteWorkspace(ctx, actor, "alpha", view)
					} else {
						err = repository.DeleteProject(ctx, actor, "alpha", *projectID, view)
					}
					if scenario == "success" && err != nil {
						t.Fatal(err)
					}
					if scenario == "denied" && !errors.Is(err, ErrForbidden) {
						t.Fatalf("delete error = %v, want denied", err)
					}
					var deleted bool
					var favorites, visits int
					if err := tx.QueryRow(ctx, `SELECT deleted_at IS NOT NULL FROM issue_views WHERE id=$1`, view).Scan(&deleted); err != nil {
						t.Fatal(err)
					}
					if err := tx.QueryRow(ctx, `SELECT count(*) FROM user_favorites WHERE deleted_at IS NULL`).Scan(&favorites); err != nil {
						t.Fatal(err)
					}
					if err := tx.QueryRow(ctx, `SELECT count(*) FROM user_recent_visits`).Scan(&visits); err != nil {
						t.Fatal(err)
					}
					wantFavorites, wantVisits := 5, 3
					if scenario == "success" {
						wantFavorites = 3
						if scope == "project" {
							wantVisits = 2
						}
					}
					if deleted != (scenario == "success") || favorites != wantFavorites || visits != wantVisits {
						t.Fatalf("deleted=%v active favorites=%d visits=%d; want %v/%d/%d", deleted, favorites, visits, scenario == "success", wantFavorites, wantVisits)
					}
				})
			}
		})
	}
}

func deleteViewDatabase(t *testing.T) *pgx.Conn {
	t.Helper()
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, seedMutationValidation)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pair.Close() })
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close(ctx) })
	if _, err := connection.Exec(ctx, `CREATE TABLE user_recent_visits (workspace_id uuid, project_id uuid, entity_identifier uuid, entity_name text)`); err != nil {
		t.Fatal(err)
	}

	return connection
}

func TestDeleteViewCleanupFailureDoesNotPersistPartialRemoval(t *testing.T) {
	ctx := context.Background()
	connection := deleteViewDatabase(t)
	if _, err := connection.Exec(ctx, `INSERT INTO user_favorites(workspace_id,project_id,entity_identifier,entity_type)
 SELECT workspace_id,project_id,id,'view' FROM issue_views;
 INSERT INTO user_recent_visits SELECT workspace_id,project_id,id,'view' FROM issue_views;
 CREATE FUNCTION reject_view_cleanup() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'cleanup rejected'; END $$;
 CREATE TRIGGER reject_view_cleanup BEFORE UPDATE ON user_favorites FOR EACH ROW EXECUTE FUNCTION reject_view_cleanup()`); err != nil {
		t.Fatal(err)
	}
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	user := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	project := uuid.MustParse("00000000-0000-4000-8000-000000000020")
	// Use autocommit, as production does: a caller-owned transaction/rollback
	// would mask partial writes from an implementation using separate statements.
	if err := repository.DeleteWorkspace(ctx, user, "alpha", uuid.MustParse("00000000-0000-4000-8000-000000000030")); err == nil {
		t.Fatal("workspace cleanup failure ignored")
	}
	if err := repository.DeleteProject(ctx, user, "alpha", project, uuid.MustParse("00000000-0000-4000-8000-000000000031")); err == nil {
		t.Fatal("project cleanup failure ignored")
	}
	var deleted, favorites, visits int
	if err := connection.QueryRow(ctx, `SELECT (SELECT count(*) FROM issue_views WHERE deleted_at IS NOT NULL),
 (SELECT count(*) FROM user_favorites WHERE deleted_at IS NULL),(SELECT count(*) FROM user_recent_visits)`).Scan(&deleted, &favorites, &visits); err != nil {
		t.Fatal(err)
	}
	if deleted != 0 || favorites != 2 || visits != 2 {
		t.Fatalf("partial deletion: removed=%d favorites=%d visits=%d", deleted, favorites, visits)
	}
}
