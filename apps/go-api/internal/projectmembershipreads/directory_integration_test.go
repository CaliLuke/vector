//go:build integration

package projectmembershipreads

import (
	"context"
	"encoding/json"
	"testing"
	"uuid"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const directoryActor = "10000000-0000-4000-8000-000000000001"
const directoryGuest = "10000000-0000-4000-8000-000000000003"

func directoryFixture(t *testing.T) (*DirectoryReader, *pgx.Conn) {
	t.Helper()
	pair, err := postgresfixture.StartUncloned(t.Context(), func(ctx context.Context, url string) error {
		db, err := pgx.Connect(ctx, url)
		if err != nil {
			return err
		}
		defer func() { testsupport.CloseContext(t, db, context.WithoutCancel(ctx)) }()
		_, err = db.Exec(ctx, directorySchema)
		return err
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pair.Close()) })
	db, err := pgx.Connect(t.Context(), pair.SeedURL())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close(context.Background())) })
	reader, err := NewDirectoryReader(db)
	require.NoError(t, err)
	return reader, db
}

func directoryRows(t *testing.T, page WorkspaceMembersPage) []map[string]any {
	t.Helper()
	var rows []map[string]any
	require.NoError(t, json.Unmarshal(page.Results, &rows))
	return rows
}

func TestWorkspaceMemberDirectoryFiltersBeforeCountsAndPaging(t *testing.T) {
	reader, _ := directoryFixture(t)
	active, inactive := true, false
	for _, tc := range []struct {
		name  string
		query WorkspaceMembersQuery
		names []string
		total int
	}{
		{"default includes inactive", WorkspaceMembersQuery{Limit: 10}, []string{"Alice", "Bob", "Carol", "Robot", "Unknown"}, 5},
		{"case insensitive substring", WorkspaceMembersQuery{Limit: 10, FirstName: "LI"}, []string{"Alice"}, 1},
		{"combined text filters", WorkspaceMembersQuery{Limit: 10, LastName: "SMITH", Email: "example", DisplayName: "b"}, []string{"Bob"}, 1},
		{"AND excludes nonmatch", WorkspaceMembersQuery{Limit: 10, FirstName: "ali", LastName: "Jones"}, []string{}, 0},
		{"percent literal", WorkspaceMembersQuery{Limit: 10, DisplayName: "%"}, []string{"Robot"}, 1},
		{"underscore literal", WorkspaceMembersQuery{Limit: 10, DisplayName: "_"}, []string{"Unknown"}, 1},
		{"owner mapping", WorkspaceMembersQuery{Limit: 10, RoleSlug: "owner"}, []string{"Alice"}, 1},
		{"admin mapping", WorkspaceMembersQuery{Limit: 10, RoleSlug: "admin"}, []string{"Bob"}, 1},
		{"guest mapping", WorkspaceMembersQuery{Limit: 10, RoleSlug: "guest"}, []string{"Carol"}, 1},
		{"slug exact", WorkspaceMembersQuery{Limit: 10, RoleSlug: "Admin"}, []string{}, 0},
		{"active true", WorkspaceMembersQuery{Limit: 10, IsActive: &active}, []string{"Alice", "Bob", "Carol", "Unknown"}, 4},
		{"active false", WorkspaceMembersQuery{Limit: 10, IsActive: &inactive}, []string{"Robot"}, 1},
		{"bot true", WorkspaceMembersQuery{Limit: 10, IsBot: &active}, []string{"Robot"}, 1},
		{"bot false", WorkspaceMembersQuery{Limit: 10, IsBot: &inactive}, []string{"Alice", "Bob", "Carol", "Unknown"}, 4},
		{"multi field ordering", WorkspaceMembersQuery{Limit: 10, OrderBy: "last_name,-first_name"}, []string{"Robot", "Carol", "Unknown", "Bob", "Alice"}, 5},
		{"paging counts match", WorkspaceMembersQuery{Limit: 1, Page: 1, IsActive: &active}, []string{"Bob", "Carol"}, 4},
		{"filtered empty page", WorkspaceMembersQuery{Limit: 2, Page: 4, IsActive: &inactive}, []string{}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page, err := reader.ListWorkspaceMembers(t.Context(), uuid.MustParse(directoryActor), "current", tc.query)
			require.NoError(t, err)
			assert.Equal(t, tc.total, page.Total)
			rows := directoryRows(t, page)
			names := make([]string, 0, len(rows))
			for _, row := range rows {
				names = append(names, row["first_name"].(string))
			}
			assert.Equal(t, tc.names, names)
		})
	}
	page, err := reader.ListWorkspaceMembers(t.Context(), uuid.MustParse(directoryActor), "current", WorkspaceMembersQuery{Limit: 10})
	require.NoError(t, err)
	rows := directoryRows(t, page)
	assert.Equal(t, "owner", rows[0]["role_slug"])
	assert.Equal(t, "/api/assets/v2/static/40000000-0000-4000-8000-000000000001/", rows[1]["avatar_url"])
	assert.Equal(t, float64(99), rows[4]["role"])
	assert.Nil(t, rows[4]["role_slug"])
	assert.NotContains(t, string(page.Results), "Foreign")
	assert.NotContains(t, string(page.Results), "Deleted")
}

func TestMemberDirectoryRoleCatalogueUsesTenantStableIdentity(t *testing.T) {
	reader, db := directoryFixture(t)
	actor := uuid.MustParse(directoryActor)
	roles, total, err := reader.ListRoles(t.Context(), actor, "current", "", 20, 0)
	require.NoError(t, err)
	require.Len(t, roles, 7)
	assert.Equal(t, 7, total)
	seen := make(map[string]bool)
	for _, role := range roles {
		assert.False(t, seen[role.ID])
		seen[role.ID] = true
		id, parseErr := uuid.Parse(role.ID)
		require.NoError(t, parseErr)
		got, readErr := reader.RetrieveRole(t.Context(), actor, "current", id)
		require.NoError(t, readErr)
		assert.Equal(t, role, got)
		_, readErr = reader.RetrieveRole(t.Context(), actor, "other", id)
		assert.ErrorIs(t, readErr, ErrNotFound)
		assert.NotEqual(t, "commenter", role.Slug)
	}
	project, total, err := reader.ListRoles(t.Context(), actor, "current", "project", 2, 0)
	require.NoError(t, err)
	require.Len(t, project, 2)
	assert.Equal(t, 3, total)
	assert.Equal(t, "admin", project[0].Slug)
	assert.Equal(t, "contributor", project[1].Slug)
	last, total, err := reader.ListRoles(t.Context(), actor, "current", "project", 2, 1)
	require.NoError(t, err)
	require.Len(t, last, 1)
	assert.Equal(t, "guest", last[0].Slug)
	assert.Equal(t, 3, total)
	empty, total, err := reader.ListRoles(t.Context(), actor, "current", "workspace", 1, 9)
	require.NoError(t, err)
	assert.Empty(t, empty)
	assert.Equal(t, 4, total)
	guest, _, err := reader.ListRoles(t.Context(), uuid.MustParse(directoryGuest), "current", "", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, roles, guest)
	_, err = db.Exec(t.Context(), `UPDATE workspaces SET slug='renamed' WHERE slug='current'`)
	require.NoError(t, err)
	renamed, _, err := reader.ListRoles(t.Context(), actor, "renamed", "", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, roles, renamed)
}

func TestMemberDirectoryRequiresCurrentAccountAndWorkspaceAccess(t *testing.T) {
	_, db := directoryFixture(t)
	for _, tc := range []struct{ name, mutation string }{
		{"revoked", `DELETE FROM workspace_members WHERE member_id='` + directoryActor + `'`},
		{"inactive membership", `UPDATE workspace_members SET is_active=false WHERE member_id='` + directoryActor + `'`},
		{"deleted membership", `UPDATE workspace_members SET deleted_at=now() WHERE member_id='` + directoryActor + `'`},
		{"inactive actor", `UPDATE users SET is_active=false WHERE id='` + directoryActor + `'`},
		{"deleted workspace", `UPDATE workspaces SET deleted_at=now() WHERE slug='current'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := db.Begin(t.Context())
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, tx.Rollback(context.Background())) })
			reader, err := NewDirectoryReader(tx)
			require.NoError(t, err)
			actor := uuid.MustParse(directoryActor)
			_, _, err = reader.ListRoles(t.Context(), actor, "current", "", 20, 0)
			require.NoError(t, err)
			_, err = tx.Exec(t.Context(), tc.mutation)
			require.NoError(t, err)
			_, _, err = reader.ListRoles(t.Context(), actor, "current", "", 20, 0)
			assert.ErrorIs(t, err, ErrForbidden)
			_, err = reader.ListWorkspaceMembers(t.Context(), actor, "current", WorkspaceMembersQuery{Limit: 10})
			assert.ErrorIs(t, err, ErrForbidden)
		})
	}
	reader, err := NewDirectoryReader(db)
	require.NoError(t, err)
	for _, tc := range []struct {
		actor uuid.UUID
		slug  string
	}{
		{uuid.MustParse(directoryActor), "missing"}, {uuid.MustParse(directoryGuest), "other"}, {uuid.UUID{}, "current"}, {uuid.MustParse(directoryActor), ""},
	} {
		_, _, err = reader.ListRoles(t.Context(), tc.actor, tc.slug, "", 20, 0)
		assert.ErrorIs(t, err, ErrForbidden)
		_, err = reader.ListWorkspaceMembers(t.Context(), tc.actor, tc.slug, WorkspaceMembersQuery{Limit: 10})
		assert.ErrorIs(t, err, ErrForbidden)
	}
	_, err = reader.ListWorkspaceMembers(t.Context(), uuid.MustParse(directoryGuest), "current", WorkspaceMembersQuery{Limit: 10})
	assert.ErrorIs(t, err, ErrForbidden, "guests can read definitions but not the directory")
}

const directorySchema = `
CREATE TABLE users(id uuid PRIMARY KEY,is_active boolean NOT NULL,first_name text,last_name text,email text,avatar text,avatar_asset_id uuid,display_name text,is_bot boolean NOT NULL);
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text NOT NULL,owner_id uuid NOT NULL,deleted_at timestamptz);
CREATE TABLE workspace_members(id uuid PRIMARY KEY,workspace_id uuid NOT NULL,member_id uuid NOT NULL,role integer NOT NULL,is_active boolean NOT NULL,deleted_at timestamptz,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now());
INSERT INTO users VALUES
 ('10000000-0000-4000-8000-000000000001',true,'Alice','Smith','alice@example.test','',NULL,'alice',false),
 ('10000000-0000-4000-8000-000000000002',true,'Bob','Smith','bob@example.test','old.png','40000000-0000-4000-8000-000000000001','bob',false),
 ('10000000-0000-4000-8000-000000000003',true,'Carol','Jones','carol@example.test','',NULL,'carol',false),
 ('10000000-0000-4000-8000-000000000004',true,'Robot','Bot','robot@example.test','',NULL,'100%bot',true),
 ('10000000-0000-4000-8000-000000000005',true,'Unknown','Other','unknown@example.test','',NULL,'unknown_user',false),
 ('10000000-0000-4000-8000-000000000006',true,'Deleted','User','deleted@example.test','',NULL,'deleted',false),
 ('10000000-0000-4000-8000-000000000007',true,'Foreign','User','foreign@example.test','',NULL,'foreign',false);
INSERT INTO workspaces VALUES
 ('20000000-0000-4000-8000-000000000001','current','10000000-0000-4000-8000-000000000001',NULL),
 ('20000000-0000-4000-8000-000000000002','other','10000000-0000-4000-8000-000000000001',NULL);
INSERT INTO workspace_members(id,workspace_id,member_id,role,is_active,deleted_at) VALUES
 ('30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000001',20,true,NULL),
 ('30000000-0000-4000-8000-000000000002','20000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000002',20,true,NULL),
 ('30000000-0000-4000-8000-000000000003','20000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000003',5,true,NULL),
 ('30000000-0000-4000-8000-000000000004','20000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000004',15,false,NULL),
 ('30000000-0000-4000-8000-000000000005','20000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000005',99,true,NULL),
 ('30000000-0000-4000-8000-000000000006','20000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000006',15,true,now()),
 ('30000000-0000-4000-8000-000000000007','20000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000007',15,true,NULL),
 ('30000000-0000-4000-8000-000000000008','20000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000001',20,true,NULL);
`
