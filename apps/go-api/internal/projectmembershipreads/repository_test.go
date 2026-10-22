package projectmembershipreads

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"uuid"
)

type rowFunc func(...any) error

func (row rowFunc) Scan(destinations ...any) error { return row(destinations...) }

type databaseStub struct {
	query string
	args  []any
	row   pgx.Row
}

func (database *databaseStub) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	database.query, database.args = query, args
	return database.row
}

func (database *databaseStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) returned nil error")
	}
}

func TestAuthorizeWorkspaceUsesOnlyActiveUndeletedMembership(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*bool) = true
		return nil
	})}
	repository, _ := NewRepository(database)
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	if err := repository.AuthorizeWorkspace(context.Background(), userID, "demo"); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"SELECT EXISTS", "workspace.slug = $2", "workspace_member.member_id = $1",
		"workspace_member.is_active = true", "workspace_member.deleted_at IS NULL", "workspace.deleted_at IS NULL",
	} {
		if !strings.Contains(database.query, fragment) {
			t.Errorf("workspace authorization query missing %q", fragment)
		}
	}
	if len(database.args) != 2 || database.args[0] != userID || database.args[1] != "demo" {
		t.Fatalf("unexpected args: %#v", database.args)
	}
}

func TestAuthorizeWorkspaceMapsMissingMembershipToForbidden(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*bool) = false
		return nil
	})}
	repository, _ := NewRepository(database)
	if err := repository.AuthorizeWorkspace(context.Background(), uuid.Nil(), "demo"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("err=%v", err)
	}
}

func TestMembershipScopesTheActiveMemberAndBuildsNestedSerializerShape(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(`{"id":"membership"}`)
		return nil
	})}
	repository, _ := NewRepository(database)
	projectID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000002")

	data, err := repository.Membership(context.Background(), userID, "demo", projectID)
	if err != nil || string(data) != `{"id":"membership"}` {
		t.Fatalf("data=%s err=%v", data, err)
	}
	for _, fragment := range []string{
		"pm.project_id = $1", "workspace.slug = $2", "pm.member_id = $3",
		"pm.is_active = true", "pm.deleted_at IS NULL", "avatar_url", "cover_image_url", "logo_url",
	} {
		if !strings.Contains(database.query, fragment) {
			t.Errorf("membership query missing %q", fragment)
		}
	}
	if len(database.args) != 3 || database.args[0] != projectID || database.args[1] != "demo" || database.args[2] != userID {
		t.Fatalf("unexpected args: %#v", database.args)
	}
}

func TestMembershipMapsMissingRows(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(...any) error { return pgx.ErrNoRows })}
	repository, _ := NewRepository(database)
	_, err := repository.Membership(context.Background(), uuid.Nil(), "demo", uuid.Nil())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestRolesRequiresAnActiveWorkspaceMemberAndReturnsAJSONMap(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*bool) = true
		*destinations[1].(*[]byte) = []byte(`{"project":15}`)
		return nil
	})}
	repository, _ := NewRepository(database)
	data, err := repository.Roles(context.Background(), uuid.Nil(), "demo")
	if err != nil || string(data) != `{"project":15}` {
		t.Fatalf("data=%s err=%v", data, err)
	}
	for _, fragment := range []string{"workspace_members", "workspace.slug = $2", "project_members", "json_object_agg", "is_active = true", "deleted_at IS NULL"} {
		if !strings.Contains(database.query, fragment) {
			t.Errorf("roles query missing %q", fragment)
		}
	}
}

func TestRolesMapsInactiveWorkspaceMembershipToForbidden(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*bool) = false
		*destinations[1].(*[]byte) = []byte(`{}`)
		return nil
	})}
	repository, _ := NewRepository(database)
	_, err := repository.Roles(context.Background(), uuid.Nil(), "demo")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err=%v", err)
	}
}

func TestWorkspaceMembersScopesAuthorizationSearchAndSerializerRole(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(`[]`)
		return nil
	})}
	repository, _ := NewRepository(database)
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	searchTerms := []string{"alex", "smith"}
	data, err := repository.WorkspaceMembers(context.Background(), userID, "demo", searchTerms)
	if err != nil || string(data) != `[]` {
		t.Fatalf("data=%s err=%v", data, err)
	}
	for _, fragment := range []string{
		"requester.role > 5", "wm.role IN (5, 15, 20)", "set_config('TimeZone'", "workspace_members",
		"unnest($3::text[])", "UPPER(member.display_name::text) LIKE UPPER", "ESCAPE '\\'", "ORDER BY wm.created_at DESC",
		"LEFT JOIN file_assets member_avatar_asset", "member_avatar_asset.entity_type IN",
		"LEFT JOIN workspaces member_avatar_asset_workspace", "member_avatar_asset.entity_type = 'ISSUE_ATTACHMENT'",
		"'ISSUE_DESCRIPTION', 'COMMENT_DESCRIPTION', 'PAGE_DESCRIPTION', 'DRAFT_ISSUE_DESCRIPTION'",
	} {
		if !strings.Contains(database.query, fragment) {
			t.Errorf("workspace-members query missing %q", fragment)
		}
	}
	if len(database.args) != 3 || database.args[0] != userID || database.args[1] != "demo" || !strings.EqualFold(strings.Join(database.args[2].([]string), ","), "alex,smith") {
		t.Fatalf("args=%#v", database.args)
	}
}

func TestMemberDirectoryRequesterRolesMatchDjangoPermissions(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(`{}`)
		return nil
	})}
	repository, _ := NewRepository(database)
	for name, call := range map[string]func() error{
		"workspace members": func() error {
			_, err := repository.WorkspaceMembers(context.Background(), uuid.Nil(), "demo", nil)
			return err
		},
		"workspace member": func() error {
			_, err := repository.WorkspaceMember(context.Background(), uuid.Nil(), "demo", uuid.Nil(), nil)
			return err
		},
		"project members": func() error {
			_, err := repository.ProjectMembers(context.Background(), uuid.Nil(), "demo", uuid.Nil())
			return err
		},
		"project member": func() error {
			_, err := repository.ProjectMember(context.Background(), uuid.Nil(), "demo", uuid.Nil(), uuid.Nil())
			return err
		},
		"preference": func() error {
			_, err := repository.Preference(context.Background(), uuid.Nil(), "demo", uuid.Nil(), uuid.Nil())
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(database.query, "role IN (5, 15, 20)") {
				t.Fatalf("requester query widens Django roles: %s", database.query)
			}
		})
	}
}

func TestWorkspaceProjectMembersAllowsEveryActiveWorkspaceRole(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(`{}`)
		return nil
	})}
	repository, _ := NewRepository(database)
	if _, err := repository.WorkspaceProjectMembers(context.Background(), uuid.Nil(), "demo"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(database.query, "wm.role IN (5, 15, 20)") {
		t.Fatalf("workspace aggregate rejects active custom roles: %s", database.query)
	}
	for _, fragment := range []string{"wm.is_active = true", "wm.deleted_at IS NULL"} {
		if !strings.Contains(database.query, fragment) {
			t.Fatalf("workspace aggregate query missing %q: %s", fragment, database.query)
		}
	}
}

func TestTimestampDirectoryReadsActivateTheRequesterTimezone(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(`{}`)
		return nil
	})}
	repository, _ := NewRepository(database)
	calls := []func() error{
		func() error {
			_, err := repository.WorkspaceMembers(context.Background(), uuid.Nil(), "demo", nil)
			return err
		},
		func() error {
			_, err := repository.WorkspaceMember(context.Background(), uuid.Nil(), "demo", uuid.Nil(), nil)
			return err
		},
		func() error {
			_, err := repository.WorkspaceProjectMembers(context.Background(), uuid.Nil(), "demo")
			return err
		},
		func() error {
			_, err := repository.ProjectMembers(context.Background(), uuid.Nil(), "demo", uuid.Nil())
			return err
		},
		func() error {
			_, err := repository.ProjectMember(context.Background(), uuid.Nil(), "demo", uuid.Nil(), uuid.Nil())
			return err
		},
	}
	for _, call := range calls {
		if err := call(); err != nil {
			t.Fatal(err)
		}
		for _, fragment := range []string{
			"set_config('TimeZone'", "requester_user.user_timezone", "requester.user_timezone IS NOT NULL",
			"to_char(", "'US'", "'TZH:TZM'", "THEN 'Z'",
		} {
			if !strings.Contains(database.query, fragment) {
				t.Fatalf("timestamp query missing %q: %s", fragment, database.query)
			}
		}
	}
}

func TestProjectDirectoryRequesterPreservesWorkspaceAdminFallback(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(`{}`)
		return nil
	})}
	repository, _ := NewRepository(database)
	calls := []func() error{
		func() error {
			_, err := repository.ProjectMembers(context.Background(), uuid.Nil(), "demo", uuid.Nil())
			return err
		},
		func() error {
			_, err := repository.ProjectMember(context.Background(), uuid.Nil(), "demo", uuid.Nil(), uuid.Nil())
			return err
		},
		func() error {
			_, err := repository.Preference(context.Background(), uuid.Nil(), "demo", uuid.Nil(), uuid.Nil())
			return err
		},
	}
	for _, call := range calls {
		if err := call(); err != nil {
			t.Fatal(err)
		}
		for _, fragment := range []string{
			"OR EXISTS", "requester_workspace_member.role = 20", "requester_workspace_member.is_active = true",
		} {
			if !strings.Contains(database.query, fragment) {
				t.Fatalf("project permission query omits workspace-admin fallback %q: %s", fragment, database.query)
			}
		}
	}
}

func TestProjectMemberScopesRequesterAndMapsMissingTarget(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(`null`)
		return nil
	})}
	repository, _ := NewRepository(database)
	_, err := repository.ProjectMember(context.Background(), uuid.Nil(), "demo", uuid.Nil(), uuid.Nil())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
	for _, fragment := range []string{
		"requester.role > 5", "pm.project_id = $3", "pm.id = $4", "member.is_bot = false",
		"LEFT JOIN file_assets member_avatar_asset", "LEFT JOIN file_assets project_cover_asset",
		"LEFT JOIN file_assets workspace_logo_asset", "member_avatar_asset.entity_type IN",
		"project_cover_asset.entity_type IN", "workspace_logo_asset.entity_type IN",
		"LEFT JOIN workspaces member_avatar_asset_workspace", "LEFT JOIN workspaces project_cover_asset_workspace",
		"LEFT JOIN workspaces workspace_logo_asset_workspace", "project_cover_asset.entity_type = 'ISSUE_ATTACHMENT'",
		"workspace_logo_asset.entity_type = 'ISSUE_ATTACHMENT'", "DRAFT_ISSUE_DESCRIPTION",
	} {
		if !strings.Contains(database.query, fragment) {
			t.Errorf("project-member query missing %q", fragment)
		}
	}
}

func TestWorkspaceMemberAppliesSearchTermsToTheTarget(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(`null`)
		return nil
	})}
	repository, _ := NewRepository(database)
	searchTerms := []string{"alex", "smith"}
	_, err := repository.WorkspaceMember(context.Background(), uuid.Nil(), "demo", uuid.Nil(), searchTerms)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(database.query, "unnest($4::text[])") {
		t.Fatalf("workspace-member query omits search terms: %s", database.query)
	}
	if len(database.args) != 4 || strings.Join(database.args[3].([]string), ",") != "alex,smith" {
		t.Fatalf("args=%#v", database.args)
	}
}

func TestWorkspaceMemberSearchTreatsLikeMetacharactersLiterally(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(`null`)
		return nil
	})}
	repository, _ := NewRepository(database)
	_, err := repository.WorkspaceMember(
		context.Background(), uuid.Nil(), "demo", uuid.Nil(), []string{`100\%`, `under\_score`, `back\\slash`},
	)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
	if strings.Count(database.query, "ESCAPE '\\'") != 2 {
		t.Fatalf("workspace-member query does not escape both case-insensitive fields: %s", database.query)
	}
}

func TestWorkspaceMemberSearchUsesDjangoUnicodeCaseFolding(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(`null`)
		return nil
	})}
	repository, _ := NewRepository(database)
	_, err := repository.WorkspaceMember(
		context.Background(), uuid.Nil(), "demo", uuid.Nil(), []string{"ı"},
	)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
	for _, field := range []string{"display_name", "first_name"} {
		fragment := "UPPER(member." + field + "::text) LIKE UPPER('%' || search_term || '%') ESCAPE '\\'"
		if !strings.Contains(database.query, fragment) {
			t.Fatalf("workspace-member query does not mirror Django icontains for %s: %s", field, database.query)
		}
	}
	if got := database.args[3].([]string); len(got) != 1 || got[0] != "ı" {
		t.Fatalf("unicode search args=%#v", got)
	}
}

func TestMemberDirectoryMapsMissingRequesterToForbidden(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(...any) error { return pgx.ErrNoRows })}
	repository, _ := NewRepository(database)
	_, err := repository.WorkspaceProjectMembers(context.Background(), uuid.Nil(), "demo")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err=%v", err)
	}
}
