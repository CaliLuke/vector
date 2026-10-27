//go:build integration

package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rosterActor = "00000000-0000-4000-8000-000000000001"
const rosterProject = "00000000-0000-4000-8000-000000000101"

func projectRosterDatabase(t *testing.T) *pgx.Conn {
	t.Helper()
	pair, err := postgresfixture.StartUncloned(t.Context(), func(ctx context.Context, url string) error {
		conn, err := pgx.Connect(ctx, url)
		if err != nil {
			return err
		}
		defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()
		_, err = conn.Exec(ctx, projectRosterSchema)
		return err
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pair.Close()) })
	conn, err := pgx.Connect(t.Context(), pair.SeedURL())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close(context.Background())) })
	return conn
}

func primaryRosterRepository(t *testing.T, db database) *LifecycleRepository {
	t.Helper()
	replica := &databaseStub{row: rowFunc(func(...any) error { return errors.New("project roster must not use a replica") })}
	repository, err := NewLifecycleRepositoryWithReaderWriter(replica, db, &lifecycleTaskSpy{}, "http://localhost:8000")
	require.NoError(t, err)
	return repository
}

func TestProjectMemberApplicationReadsOnlyTheAuthorizedProjectOnPrimary(t *testing.T) {
	db := projectRosterDatabase(t)
	repository := primaryRosterRepository(t, db)
	reader, err := NewMemberReader(repository)
	require.NoError(t, err)
	body, err := reader.ListProjectMembers(t.Context(), uuid.MustParse(rosterActor), "current", uuid.MustParse(rosterProject))
	require.NoError(t, err)
	var members []map[string]any
	require.NoError(t, json.Unmarshal(body, &members))
	require.Len(t, members, 3)
	assert.Equal(t, "Actor", members[0]["display_name"])
	assert.Equal(t, "Member", members[1]["display_name"])
	assert.Equal(t, "Inactive", members[2]["display_name"])
	assert.Equal(t, "member@example.test", members[1]["email"])
	assert.Contains(t, members[1], "avatar_url")
	assert.NotContains(t, string(body), "Foreign")
	assert.NotContains(t, string(body), "Deleted")
	assert.NotContains(t, members[0], "role", "retain the authoritative HTTP UserLite response")

	_, err = reader.ListProjectMembers(t.Context(), uuid.MustParse(rosterActor), "other", uuid.MustParse(rosterProject))
	require.Error(t, err)
	_, err = reader.ListProjectMembers(t.Context(), uuid.MustParse("00000000-0000-4000-8000-000000000003"), "current", uuid.MustParse(rosterProject))
	require.Error(t, err, "a membership carrying another workspace ID must not grant access")
}

func TestProjectRosterRechecksAccessAfterInitialAuthorization(t *testing.T) {
	db := projectRosterDatabase(t)
	for _, tc := range []struct {
		name, mutation string
		status         int
	}{
		{"membership deactivated", `UPDATE project_members SET is_active=false WHERE member_id='` + rosterActor + `'`, 403},
		{"membership deleted", `UPDATE project_members SET deleted_at=now() WHERE member_id='` + rosterActor + `'`, 403},
		{"membership moved", `UPDATE project_members SET workspace_id='00000000-0000-4000-8000-000000000020' WHERE member_id='` + rosterActor + `'`, 403},
		{"actor deactivated", `UPDATE users SET is_active=false WHERE id='` + rosterActor + `'`, 403},
		{"project deleted", `UPDATE projects SET deleted_at=now() WHERE id='` + rosterProject + `'`, 403},
		{"workspace deleted", `UPDATE workspaces SET deleted_at=now() WHERE slug='current'`, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := db.Begin(t.Context())
			require.NoError(t, err)
			defer func() { require.NoError(t, tx.Rollback(context.Background())) }()
			repository := primaryRosterRepository(t, tx)
			command := LifecycleCommand{Operation: "v1-list-project-members", Method: "GET", UserID: rosterActor, Slug: "current", ProjectID: rosterProject}
			require.NoError(t, repository.AuthorizeLifecycle(t.Context(), command))
			_, err = tx.Exec(t.Context(), tc.mutation)
			require.NoError(t, err)
			result, err := repository.ExecuteLifecycle(t.Context(), command)
			var denied *djangoapi.HTTPError
			require.ErrorAs(t, err, &denied)
			assert.Equal(t, tc.status, denied.Status)
			assert.Nil(t, result)
		})
	}
}

const projectRosterSchema = `
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text,deleted_at timestamptz);
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,deleted_at timestamptz);
CREATE TABLE project_members(project_id uuid,workspace_id uuid,member_id uuid,role integer,is_active boolean,deleted_at timestamptz);
CREATE TABLE users(id uuid PRIMARY KEY,is_active boolean,created_at timestamptz,first_name text,last_name text,email text,avatar text,display_name text,avatar_asset_id uuid);
CREATE TABLE file_assets(id uuid PRIMARY KEY,entity_type text,workspace_id uuid,project_id uuid,issue_id uuid);
INSERT INTO workspaces VALUES
 ('00000000-0000-4000-8000-000000000010','current',NULL),
 ('00000000-0000-4000-8000-000000000020','other',NULL);
INSERT INTO projects VALUES
 ('00000000-0000-4000-8000-000000000101','00000000-0000-4000-8000-000000000010',NULL),
 ('00000000-0000-4000-8000-000000000201','00000000-0000-4000-8000-000000000020',NULL);
INSERT INTO users VALUES
 ('00000000-0000-4000-8000-000000000001',true,'2026-01-05','Actor','','actor@example.test','','Actor',NULL),
 ('00000000-0000-4000-8000-000000000002',true,'2026-01-04','Member','','member@example.test','','Member',NULL),
 ('00000000-0000-4000-8000-000000000003',true,'2026-01-03','Foreign','','foreign@example.test','','Foreign',NULL),
 ('00000000-0000-4000-8000-000000000004',false,'2026-01-02','Inactive','','inactive@example.test','','Inactive',NULL),
 ('00000000-0000-4000-8000-000000000005',true,'2026-01-01','Deleted','','deleted@example.test','','Deleted',NULL);
INSERT INTO project_members VALUES
 ('00000000-0000-4000-8000-000000000101','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000001',15,true,NULL),
 ('00000000-0000-4000-8000-000000000101','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000002',5,true,NULL),
 ('00000000-0000-4000-8000-000000000201','00000000-0000-4000-8000-000000000020','00000000-0000-4000-8000-000000000003',15,true,NULL),
 ('00000000-0000-4000-8000-000000000101','00000000-0000-4000-8000-000000000020','00000000-0000-4000-8000-000000000003',15,true,NULL),
 ('00000000-0000-4000-8000-000000000101','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000004',5,false,NULL),
 ('00000000-0000-4000-8000-000000000101','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000005',5,true,now());
`
