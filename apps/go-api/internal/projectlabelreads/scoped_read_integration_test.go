//go:build integration

package projectlabelreads

import (
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func scopedLabelFixture(t *testing.T) (*Repository, *pgx.Conn, APIV1Request) {
	t.Helper()
	db := parentIntegrityFixture(t)
	_, err := db.Exec(t.Context(), `
 ALTER TABLE workspaces ADD COLUMN deleted_at timestamptz;
 ALTER TABLE projects ADD COLUMN deleted_at timestamptz;
 ALTER TABLE project_members ADD COLUMN deleted_at timestamptz,ADD COLUMN workspace_id uuid;
 UPDATE project_members pm SET workspace_id=p.workspace_id FROM projects p WHERE p.id=pm.project_id;
 CREATE TABLE users(id uuid PRIMARY KEY,is_active boolean);
 INSERT INTO users VALUES('00000000-0000-4000-8000-000000000001',true);
 CREATE TABLE workspace_members(workspace_id uuid,member_id uuid,role int,is_active boolean,deleted_at timestamptz);
 INSERT INTO workspace_members VALUES('00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001',15,true,NULL);
 `)
	require.NoError(t, err)
	repo, err := NewRepository(db)
	require.NoError(t, err)
	workspace := uuid.MustParse(parentWorkspace)
	return repo, db, APIV1Request{Operation: "list", ExpectedWorkspaceID: &workspace, Params: map[string]string{"project_id": parentProject, "slug": "demo"}}
}

func TestScopedLabelReadsDistinguishEmptyFromRevokedAccess(t *testing.T) {
	repo, db, request := scopedLabelFixture(t)
	page, err := repo.ReadAPIV1(t.Context(), parentActor, request)
	require.NoError(t, err)
	require.Len(t, page.(map[string]any)["results"], 4)
	_, err = db.Exec(t.Context(), `UPDATE labels SET deleted_at=now() WHERE workspace_id=$1`, parentWorkspace)
	require.NoError(t, err)
	page, err = repo.ReadAPIV1(t.Context(), parentActor, request)
	require.NoError(t, err)
	require.Empty(t, page.(map[string]any)["results"])
	_, err = db.Exec(t.Context(), `UPDATE workspace_members SET is_active=false`)
	require.NoError(t, err)
	_, err = repo.ReadAPIV1(t.Context(), parentActor, request)
	require.ErrorIs(t, err, ErrForbidden)
	request.Operation = "detail"
	request.Params["pk"] = labelA
	_, err = repo.ReadAPIV1(t.Context(), parentActor, request)
	require.ErrorIs(t, err, ErrForbidden)
	request.Operation = "unsupported"
	_, err = repo.ReadAPIV1(t.Context(), parentActor, request)
	require.ErrorIs(t, err, ErrForbidden)
}

func TestScopedLabelParentMustBelongToSelectedProjectAndWorkspace(t *testing.T) {
	repo, db, request := scopedLabelFixture(t)
	request.Operation = "detail"
	request.Params["pk"] = labelC
	_, err := db.Exec(t.Context(), `UPDATE labels SET deleted_at=now() WHERE id=$1`, labelB)
	require.NoError(t, err)
	detail, err := repo.ReadAPIV1(t.Context(), parentActor, request)
	require.NoError(t, err)
	require.Equal(t, labelB, detail.(map[string]any)["parent"])
	_, err = db.Exec(t.Context(), `UPDATE labels SET parent_id=$1 WHERE id=$2`, foreignLabel, labelC)
	require.NoError(t, err)
	detail, err = repo.ReadAPIV1(t.Context(), parentActor, request)
	require.NoError(t, err)
	require.Nil(t, detail.(map[string]any)["parent"])
	// Reassigning a slug cannot retarget an already admitted workspace UUID.
	_, err = db.Exec(t.Context(), `UPDATE workspaces SET slug='renamed' WHERE id=$1;`, parentWorkspace)
	require.NoError(t, err)
	_, err = db.Exec(t.Context(), `UPDATE workspaces SET slug='demo' WHERE id=$1`, otherWorkspace)
	require.NoError(t, err)
	_, err = repo.ReadAPIV1(t.Context(), parentActor, request)
	require.ErrorIs(t, err, ErrForbidden)
}
