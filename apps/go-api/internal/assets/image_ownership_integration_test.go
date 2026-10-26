//go:build integration

package assets

import (
	"context"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type imageOwnershipCase struct {
	kind, table, column string
	userAsset           bool
}

var imageOwnershipCases = []imageOwnershipCase{
	{"USER_AVATAR", "users", "avatar_asset_id", true},
	{"USER_COVER", "users", "cover_image_asset_id", true},
	{"WORKSPACE_LOGO", "workspaces", "logo_asset_id", false},
	{"PROJECT_COVER", "projects", "cover_image_asset_id", false},
}

type imageOwnershipFixture struct {
	connection *pgx.Conn
	repository *PostgresRepository
}

func newImageOwnershipFixture(t *testing.T) imageOwnershipFixture {
	t.Helper()
	ctx := t.Context()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer func() { _ = connection.Close(context.WithoutCancel(ctx)) }()
		_, err = connection.Exec(ctx, `
CREATE TABLE users (id uuid PRIMARY KEY, avatar_asset_id uuid, cover_image_asset_id uuid, avatar text, cover_image text, updated_at timestamptz);
CREATE TABLE workspaces (id uuid PRIMARY KEY, logo_asset_id uuid, logo text, deleted_at timestamptz, updated_at timestamptz, updated_by_id uuid);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid, guest_view_all_features boolean DEFAULT false, cover_image_asset_id uuid, cover_image text, deleted_at timestamptz, updated_at timestamptz, updated_by_id uuid);
CREATE TABLE issues(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,created_by_id uuid,deleted_at timestamptz,is_draft boolean DEFAULT false);
CREATE TABLE issue_comments(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,issue_id uuid,created_by_id uuid,deleted_at timestamptz);
CREATE TABLE pages(id uuid PRIMARY KEY,workspace_id uuid,owned_by_id uuid,access integer,deleted_at timestamptz);
CREATE TABLE project_pages(page_id uuid,project_id uuid,deleted_at timestamptz);
CREATE TABLE workspace_members(workspace_id uuid,member_id uuid,role integer,is_active boolean,deleted_at timestamptz);
CREATE TABLE project_members(workspace_id uuid,project_id uuid,member_id uuid,role integer,is_active boolean,deleted_at timestamptz);
CREATE TABLE file_assets (id uuid PRIMARY KEY, user_id uuid, workspace_id uuid, project_id uuid, draft_issue_id uuid, issue_id uuid, comment_id uuid, page_id uuid, entity_type text,
  is_uploaded boolean DEFAULT false, is_deleted boolean DEFAULT false, deleted_at timestamptz, updated_at timestamptz, attributes jsonb DEFAULT '{}');
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close(context.Background()) })
	repository, err := NewPostgresRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	return imageOwnershipFixture{connection, repository}
}

type imageOwner struct {
	user, workspace, project, previous, next uuid.UUID
}

func (fixture imageOwnershipFixture) seed(t *testing.T, scenario imageOwnershipCase) imageOwner {
	t.Helper()
	owner := imageOwner{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	for _, row := range []struct {
		table string
		id    uuid.UUID
	}{{"users", owner.user}, {"workspaces", owner.workspace}, {"projects", owner.project}} {
		if _, err := fixture.connection.Exec(t.Context(), `INSERT INTO `+row.table+` (id) VALUES ($1)`, row.id); err != nil {
			t.Fatal(err)
		}
	}
	_, err := fixture.connection.Exec(t.Context(), `
INSERT INTO file_assets (id,user_id,workspace_id,project_id,entity_type,is_uploaded) VALUES
($1,$3,$4,$5,$6,true),($2,$3,$4,$5,$6,false)
`, owner.previous, owner.next, owner.user, owner.workspace, owner.project, scenario.kind)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.connection.Exec(t.Context(), `UPDATE `+scenario.table+` SET `+scenario.column+`=$1 WHERE id=$2`, owner.previous, map[string]uuid.UUID{"users": owner.user, "workspaces": owner.workspace, "projects": owner.project}[scenario.table]); err != nil {
		t.Fatal(err)
	}
	return owner
}

func (fixture imageOwnershipFixture) confirm(t *testing.T, scenario imageOwnershipCase, owner imageOwner) {
	t.Helper()
	var err error
	if scenario.userAsset {
		err = fixture.repository.UpdateUserAsset(t.Context(), owner.user, owner.next, []byte(`{"name":"image.png"}`), true, false)
	} else {
		_, err = fixture.repository.UpdateWorkspaceAsset(t.Context(), owner.user, owner.workspace, owner.next, []byte(`{"name":"image.png"}`), false)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func (fixture imageOwnershipFixture) attached(t *testing.T, scenario imageOwnershipCase, owner imageOwner) uuid.UUID {
	t.Helper()
	id := owner.user
	switch scenario.table {
	case "workspaces":
		id = owner.workspace
	case "projects":
		id = owner.project
	}
	var attached uuid.UUID
	if err := fixture.connection.QueryRow(t.Context(), `SELECT `+scenario.column+` FROM `+scenario.table+` WHERE id=$1`, id).Scan(&attached); err != nil {
		t.Fatal(err)
	}
	return attached
}

func TestImageFinalizationRetryKeepsCurrentAssetAvailable(t *testing.T) {
	fixture := newImageOwnershipFixture(t)
	for _, scenario := range imageOwnershipCases {
		t.Run(scenario.kind, func(t *testing.T) {
			owner := fixture.seed(t, scenario)
			fixture.confirm(t, scenario, owner)
			fixture.confirm(t, scenario, owner)
			if got := fixture.attached(t, scenario, owner); got != owner.next {
				t.Fatalf("attached asset = %s, want %s", got, owner.next)
			}
			assertAttachmentState(t, fixture.connection, owner.next.String(), true, false)
			assertAttachmentState(t, fixture.connection, owner.previous.String(), true, true)
		})
	}
}

func TestDeletingUnusedImageDoesNotClearCurrentAsset(t *testing.T) {
	fixture := newImageOwnershipFixture(t)
	for _, scenario := range imageOwnershipCases {
		t.Run(scenario.kind, func(t *testing.T) {
			owner := fixture.seed(t, scenario)
			fixture.remove(t, scenario, owner)
			if got := fixture.attached(t, scenario, owner); got != owner.previous {
				t.Fatalf("attached asset = %s, want %s", got, owner.previous)
			}
			assertAttachmentState(t, fixture.connection, owner.next.String(), false, true)
			assertAttachmentState(t, fixture.connection, owner.previous.String(), true, false)
		})
	}
}

func TestImageMutationsRejectAnotherOwner(t *testing.T) {
	fixture := newImageOwnershipFixture(t)
	for _, scenario := range imageOwnershipCases {
		t.Run(scenario.kind, func(t *testing.T) {
			owner := fixture.seed(t, scenario)
			foreign := uuid.New()
			var confirmErr, deleteErr error
			if scenario.userAsset {
				confirmErr = fixture.repository.UpdateUserAsset(t.Context(), foreign, owner.next, []byte(`{}`), true, false)
				deleteErr = fixture.repository.DeleteUserAsset(t.Context(), foreign, owner.next)
			} else {
				_, confirmErr = fixture.repository.UpdateWorkspaceAsset(t.Context(), owner.user, foreign, owner.next, []byte(`{}`), false)
				deleteErr = fixture.repository.DeleteWorkspaceAsset(t.Context(), owner.user, foreign, owner.next)
			}
			assertScopeNotFound(t, confirmErr, "confirm another owner's image")
			assertScopeNotFound(t, deleteErr, "delete another owner's image")
			if got := fixture.attached(t, scenario, owner); got != owner.previous {
				t.Fatalf("attached asset = %s, want %s", got, owner.previous)
			}
			assertAttachmentState(t, fixture.connection, owner.previous.String(), true, false)
			assertAttachmentState(t, fixture.connection, owner.next.String(), false, false)
		})
	}
}

func (fixture imageOwnershipFixture) remove(t *testing.T, scenario imageOwnershipCase, owner imageOwner) {
	t.Helper()
	var err error
	if scenario.userAsset {
		err = fixture.repository.DeleteUserAsset(t.Context(), owner.user, owner.next)
	} else {
		err = fixture.repository.DeleteWorkspaceAsset(t.Context(), owner.user, owner.workspace, owner.next)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeletingCurrentImageClearsItsOwner(t *testing.T) {
	fixture := newImageOwnershipFixture(t)
	for _, scenario := range imageOwnershipCases {
		t.Run(scenario.kind, func(t *testing.T) {
			owner := fixture.seed(t, scenario)
			fixture.confirm(t, scenario, owner)
			fixture.remove(t, scenario, owner)
			if got := fixture.attached(t, scenario, owner); got != uuid.Nil {
				t.Fatalf("attached asset = %s, want none", got)
			}
			assertAttachmentState(t, fixture.connection, owner.next.String(), true, true)
		})
	}
}
