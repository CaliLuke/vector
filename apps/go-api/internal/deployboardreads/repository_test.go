package deployboardreads

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"uuid"
)

type deployBoardRowFunc func(...any) error

func (row deployBoardRowFunc) Scan(destinations ...any) error { return row(destinations...) }

type deployBoardDatabaseStub struct {
	queries []string
	args    [][]any
	rows    []pgx.Row
}

func (database *deployBoardDatabaseStub) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	database.queries = append(database.queries, query)
	database.args = append(database.args, args)
	if len(database.rows) == 0 {
		return deployBoardRowFunc(func(...any) error { return errors.New("unexpected query") })
	}
	row := database.rows[0]
	database.rows = database.rows[1:]
	return row
}

func deployBoardBoolRow(value bool) pgx.Row {
	return deployBoardRowFunc(func(destinations ...any) error {
		*destinations[0].(*bool) = value
		return nil
	})
}

func deployBoardJSONRow(value string) pgx.Row {
	return deployBoardRowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(value)
		return nil
	})
}

const deployBoardUnsavedDefaults = `{"created_by":null,"deleted_at":null,"entity_identifier":null,"entity_name":"","intake":null,"is_activity_enabled":false,"is_comments_enabled":false,"is_disabled":false,"is_reactions_enabled":false,"is_votes_enabled":false,"updated_by":null,"view_props":null}`

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) returned nil error")
	}
}

func TestAuthorizeProjectRequiresExactActiveTenantMembership(t *testing.T) {
	database := &deployBoardDatabaseStub{rows: []pgx.Row{deployBoardBoolRow(true)}}
	repository, _ := NewRepository(database)
	userID, projectID := uuid.New(), uuid.New()
	if err := repository.AuthorizeProject(context.Background(), userID, "demo", projectID); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"project_member.member_id = $1",
		"workspace.slug = $2",
		"project.id = $3",
		"project.workspace_id = workspace.id",
		"project_member.is_active = true",
		"project_member.deleted_at IS NULL",
		"workspace.deleted_at IS NULL",
		"project.deleted_at IS NULL",
	} {
		if !strings.Contains(database.queries[0], fragment) {
			t.Errorf("authorization query missing %q", fragment)
		}
	}
	if len(database.args[0]) != 3 || database.args[0][0] != userID || database.args[0][1] != "demo" || database.args[0][2] != projectID {
		t.Fatalf("authorization args = %#v", database.args[0])
	}
}

func TestAuthorizeProjectRejectsOtherTenant(t *testing.T) {
	database := &deployBoardDatabaseStub{rows: []pgx.Row{deployBoardBoolRow(false)}}
	repository, _ := NewRepository(database)
	if err := repository.AuthorizeProject(context.Background(), uuid.New(), "other", uuid.New()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want forbidden", err)
	}
}

func TestCollectionReturnsUnsavedDefaultsOrSingleObjectWithoutPagination(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "unsaved defaults", body: deployBoardUnsavedDefaults},
		{name: "single", body: `{"id":"00000000-0000-4000-8000-000000000004"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			database := &deployBoardDatabaseStub{rows: []pgx.Row{deployBoardBoolRow(true), deployBoardJSONRow(test.body)}}
			repository, _ := NewRepository(database)
			body, err := repository.Collection(context.Background(), uuid.New(), "demo", uuid.New())
			if err != nil || string(body) != test.body {
				t.Fatalf("body=%s err=%v", body, err)
			}
			query := database.queries[1]
			for _, fragment := range []string{
				"deploy_board.entity_name = 'project'",
				"deploy_board.entity_identifier = $3",
				"project.id = $3",
				"project.workspace_id = workspace.id",
				"workspace.slug = $2",
				"deploy_board.deleted_at IS NULL",
				"workspace.deleted_at IS NULL",
				"project.deleted_at IS NULL",
				"ORDER BY deploy_board.created_at DESC",
				"LIMIT 1",
				"COALESCE((",
				"'created_by', NULL",
				"'deleted_at', NULL",
				"'entity_identifier', NULL",
				"'entity_name', ''",
				"'intake', NULL",
				"'is_activity_enabled', false",
				"'is_comments_enabled', false",
				"'is_disabled', false",
				"'is_reactions_enabled', false",
				"'is_votes_enabled', false",
				"'updated_by', NULL",
				"'view_props', NULL",
			} {
				if !strings.Contains(query, fragment) {
					t.Errorf("collection query missing %q", fragment)
				}
			}
			if strings.Contains(query, "json_agg") {
				t.Fatalf("collection query paginates or aggregates rows: %s", query)
			}
		})
	}
}

func TestDetailScopesBoardIDToRouteWorkspaceProjectAndDeploymentIdentity(t *testing.T) {
	database := &deployBoardDatabaseStub{rows: []pgx.Row{
		deployBoardBoolRow(true),
		deployBoardRowFunc(func(...any) error { return pgx.ErrNoRows }),
	}}
	repository, _ := NewRepository(database)
	_, err := repository.Detail(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v, want not found", err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"workspace.slug = $2",
		"project.id = $3",
		"project.workspace_id = workspace.id",
		"deploy_board.id = $4",
		"deploy_board.entity_name = 'project'",
		"deploy_board.entity_identifier = $3",
		"deploy_board.deleted_at IS NULL",
		"workspace.deleted_at IS NULL",
		"project.deleted_at IS NULL",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("detail query missing %q", fragment)
		}
	}
}

func TestProjectionPreservesNestedLocalAssetURLs(t *testing.T) {
	for _, query := range []string{deployBoardCollectionSQL, deployBoardDetailSQL} {
		for _, fragment := range []string{
			"'project_details'",
			"'workspace_detail'",
			"project.cover_image_asset_id",
			"workspace.logo_asset_id",
			"'/api/assets/v2/static/'",
			"'ISSUE_ATTACHMENT'",
			"'/attachments/'",
			"'PAGE_DESCRIPTION'",
			"NULLIF(project.cover_image, '')",
			"NULLIF(workspace.logo, '')",
		} {
			if !strings.Contains(query, fragment) {
				t.Errorf("projection query missing %q", fragment)
			}
		}
	}
}
