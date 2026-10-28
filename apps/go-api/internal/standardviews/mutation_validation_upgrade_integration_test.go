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

func TestMutationValidationPreservesDjangoPrecedence(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, seedMutationValidation)
	if err != nil {
		t.Fatal(err)
	}
	defer pair.Close()
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(ctx)
	user := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	project := uuid.MustParse("00000000-0000-4000-8000-000000000020")
	workspaceView := uuid.MustParse("00000000-0000-4000-8000-000000000030")
	projectView := uuid.MustParse("00000000-0000-4000-8000-000000000031")
	missingView := uuid.MustParse("00000000-0000-4000-8000-000000000099")
	for _, test := range []struct {
		name    string
		project *uuid.UUID
		view    uuid.UUID
		partial bool
		setup   string
		want    error
	}{
		{name: "workspace creator owner permits field validation", view: workspaceView, partial: true},
		{name: "workspace membership precedes locked error", view: workspaceView, partial: true, setup: `UPDATE workspace_members SET is_active=false; UPDATE issue_views SET is_locked=true`, want: ErrPermission},
		{name: "workspace missing denied by creator decorator", view: missingView, partial: true, want: ErrPermission},
		{name: "workspace noncreator owner denied before locked", view: workspaceView, partial: true, setup: `UPDATE issue_views SET created_by_id='00000000-0000-4000-8000-000000000002',is_locked=true`, want: ErrPermission},
		{name: "workspace creator wrong tenant missing", view: workspaceView, partial: true, setup: `UPDATE issue_views SET workspace_id='00000000-0000-4000-8000-000000000011'`, want: ErrNotFound},
		{name: "workspace locked precedes ownership", view: workspaceView, partial: true, setup: `UPDATE issue_views SET owned_by_id='00000000-0000-4000-8000-000000000002',is_locked=true`, want: ErrLocked},
		{name: "workspace creator nonowner denied", view: workspaceView, partial: true, setup: `UPDATE issue_views SET owned_by_id='00000000-0000-4000-8000-000000000002'`, want: ErrForbidden},
		{name: "workspace patch can target project row", view: projectView, partial: true},
		{name: "project creator owner permits field validation", project: &project, view: projectView, partial: true},
		{name: "project regular member missing denied", project: &project, view: missingView, partial: true, want: ErrPermission},
		{name: "project admin missing reaches lookup", project: &project, view: missingView, partial: true, setup: `UPDATE workspace_members SET role=20`, want: ErrNotFound},
		{name: "project admin bypass still checks ownership", project: &project, view: projectView, partial: true, setup: `UPDATE workspace_members SET role=20; UPDATE issue_views SET created_by_id='00000000-0000-4000-8000-000000000002',owned_by_id='00000000-0000-4000-8000-000000000002'`, want: ErrForbidden},
		{name: "project admin bypass requires project membership", project: &project, view: projectView, partial: true, setup: `UPDATE workspace_members SET role=20; UPDATE project_members SET is_active=false; UPDATE issue_views SET created_by_id='00000000-0000-4000-8000-000000000002',is_locked=true`, want: ErrPermission},
		{name: "project creator wrong project missing", project: &project, view: workspaceView, partial: true, want: ErrNotFound},
		{name: "project locked precedes ownership", project: &project, view: projectView, partial: true, setup: `UPDATE issue_views SET owned_by_id='00000000-0000-4000-8000-000000000002',is_locked=true`, want: ErrLocked},
		{name: "workspace PUT guest public row permits validation", view: workspaceView, setup: `UPDATE workspace_members SET role=5; UPDATE issue_views SET owned_by_id='00000000-0000-4000-8000-000000000002',is_locked=true`},
		{name: "project PUT guest public row permits validation", project: &project, view: projectView, setup: `UPDATE project_members SET role=5; UPDATE issue_views SET owned_by_id='00000000-0000-4000-8000-000000000002',is_locked=true`},
		{name: "workspace PUT private nonowner missing", view: workspaceView, setup: `UPDATE issue_views SET access=0,owned_by_id='00000000-0000-4000-8000-000000000002'`, want: ErrNotFound},
		{name: "project PUT archived project missing", project: &project, view: projectView, setup: `UPDATE projects SET archived_at=NOW()`, want: ErrNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			tx, err := connection.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			if test.setup != "" {
				if _, execErr := tx.Exec(ctx, test.setup); execErr != nil {
					t.Fatal(execErr)
				}
			}
			repository, err := NewRepository(tx)
			if err != nil {
				t.Fatal(err)
			}
			if err := repository.CheckMutation(ctx, user, "alpha", test.project, test.view, test.partial); !errors.Is(err, test.want) {
				t.Fatalf("CheckMutation error = %v, want %v before JSON-field validation", err, test.want)
			}
		})
	}
}

func seedMutationValidation(ctx context.Context, databaseURL string) error {
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer connection.Close(ctx)
	_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text, deleted_at timestamptz);
CREATE TABLE workspace_members (workspace_id uuid, member_id uuid, role integer, is_active boolean, deleted_at timestamptz);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid, archived_at timestamptz, deleted_at timestamptz, guest_view_all_features boolean DEFAULT false);
CREATE TABLE project_members (workspace_id uuid, project_id uuid, member_id uuid, role integer, is_active boolean, deleted_at timestamptz);
CREATE TABLE user_favorites (user_id uuid, entity_type text, entity_identifier uuid, project_id uuid, workspace_id uuid, deleted_at timestamptz);
CREATE TABLE issue_views (id uuid PRIMARY KEY, workspace_id uuid, project_id uuid, created_by_id uuid, owned_by_id uuid, is_locked boolean DEFAULT false, access integer DEFAULT 1, deleted_at timestamptz,
created_at timestamptz, updated_at timestamptz, name text, description text, query jsonb, filters jsonb, display_filters jsonb, display_properties jsonb, rich_filters jsonb, sort_order float8, logo_props jsonb, archived_at timestamptz, updated_by_id uuid);
INSERT INTO workspaces (id,slug) VALUES ('00000000-0000-4000-8000-000000000010','alpha'),('00000000-0000-4000-8000-000000000011','beta');
INSERT INTO workspace_members VALUES ('00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000001',15,true,NULL);
INSERT INTO projects (id,workspace_id) VALUES ('00000000-0000-4000-8000-000000000020','00000000-0000-4000-8000-000000000010');
INSERT INTO project_members VALUES ('00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000020','00000000-0000-4000-8000-000000000001',15,true,NULL);
INSERT INTO issue_views (id,workspace_id,project_id,created_by_id,owned_by_id) VALUES
('00000000-0000-4000-8000-000000000030','00000000-0000-4000-8000-000000000010',NULL,'00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000001'),
('00000000-0000-4000-8000-000000000031','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000020','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000001');`)
	return err
}
