//go:build integration

package projectstatereads

import (
	"context"
	"encoding/json"
	"testing"
	"time"
	"uuid"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

const (
	boundActor     = "00000000-0000-4000-8000-000000000001"
	boundWorkspace = "00000000-0000-4000-8000-000000000002"
	boundProject   = "00000000-0000-4000-8000-000000000003"
	boundState     = "00000000-0000-4000-8000-000000000004"
	boundStateNext = "00000000-0000-4000-8000-000000000005"
	boundForeign   = "00000000-0000-4000-8000-000000000006"
)

func boundReadRequest(project, state string) APIV1Request {
	workspace := uuid.MustParse(boundWorkspace)
	operation := "list"
	if state != "" {
		operation = "detail"
	}
	return APIV1Request{ExpectedWorkspaceID: &workspace, Operation: operation,
		Params: map[string]string{"slug": "states", "project_id": project, "state_id": state}, Present: map[string]bool{}}
}

func boundStateDatabase(t *testing.T) *pgx.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, url string) error {
		db, err := pgx.Connect(ctx, url)
		if err != nil {
			return err
		}
		defer func() { testsupport.CloseContext(t, db, context.WithoutCancel(ctx)) }()
		_, err = db.Exec(ctx, `
CREATE TABLE users(id uuid PRIMARY KEY,is_active boolean NOT NULL DEFAULT true);
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text,deleted_at timestamptz);
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,deleted_at timestamptz,archived_at timestamptz);
CREATE TABLE workspace_members(workspace_id uuid,member_id uuid,role integer DEFAULT 15,is_active boolean DEFAULT true,deleted_at timestamptz);
CREATE TABLE project_members(project_id uuid,workspace_id uuid,member_id uuid,role integer DEFAULT 15,is_active boolean DEFAULT true,deleted_at timestamptz);
CREATE TABLE states(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,name text DEFAULT 'Todo',color text DEFAULT '#abc',
 description text DEFAULT '',slug text DEFAULT 'todo',sequence double precision DEFAULT 0,"group" text DEFAULT 'unstarted',
 is_triage boolean DEFAULT false,"default" boolean DEFAULT false,external_source text,external_id text,
 created_at timestamptz DEFAULT '2026-01-01Z',updated_at timestamptz DEFAULT '2026-01-01Z',deleted_at timestamptz,created_by_id uuid,updated_by_id uuid);
INSERT INTO users(id) VALUES('00000000-0000-4000-8000-000000000001');
INSERT INTO workspaces(id,slug) VALUES('00000000-0000-4000-8000-000000000002','states'),('00000000-0000-4000-8000-000000000006','foreign');
INSERT INTO projects(id,workspace_id) VALUES('00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002');
INSERT INTO workspace_members(workspace_id,member_id) VALUES('00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001');
INSERT INTO project_members(project_id,workspace_id,member_id) VALUES('00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001');
INSERT INTO states(id,project_id,workspace_id) VALUES
 ('00000000-0000-4000-8000-000000000005','00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002'),
 ('00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002');
`)
		return err
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pair.Close()) })
	db, err := pgx.Connect(ctx, pair.SeedURL())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close(context.WithoutCancel(ctx))) })
	return db
}

func TestBoundStateReadsPreserveProjectionAndDeterministicPages(t *testing.T) {
	db := boundStateDatabase(t)
	repository, repositoryErr := NewRepository(db)
	require.NoError(t, repositoryErr)
	for _, project := range []string{boundProject, ""} {
		t.Run("project="+project, func(t *testing.T) {
			request := boundReadRequest(project, "")
			request.Params["per_page"], request.Present["per_page"] = "1", true
			value, err := repository.ReadAPIV1(t.Context(), boundActor, request)
			require.NoError(t, err)
			page := value.(map[string]any)
			require.Equal(t, 2, page["total_count"])
			require.Equal(t, true, page["next_page_results"])
			rows := page["results"].([]any)
			require.Len(t, rows, 1)
			record := rows[0].(map[string]any)
			require.Equal(t, boundState, record["id"])
			require.Equal(t, json.Number("0.0"), record["sequence"])
			require.Equal(t, false, record["default"])
			require.Len(t, record, 18)
			detail, err := repository.ReadAPIV1(t.Context(), boundActor, boundReadRequest(project, boundState))
			require.NoError(t, err)
			require.Equal(t, record, detail)
			legacy := boundReadRequest(boundProject, boundState)
			legacy.ExpectedWorkspaceID = nil
			native, err := repository.ReadAPIV1(t.Context(), boundActor, legacy)
			require.NoError(t, err)
			require.Equal(t, native, detail)
			request.Params["cursor"], request.Present["cursor"] = page["next_cursor"].(string), true
			value, err = repository.ReadAPIV1(t.Context(), boundActor, request)
			require.NoError(t, err)
			page = value.(map[string]any)
			require.Equal(t, false, page["next_page_results"])
			require.Equal(t, boundStateNext, page["results"].([]any)[0].(map[string]any)["id"])
		})
	}
}

func TestBoundStateReadsApplyCurrentScopeBeforeCounts(t *testing.T) {
	db := boundStateDatabase(t)
	for _, test := range []struct {
		name, change                   string
		workspaceDenied, projectDenied bool
	}{
		{"inactive actor", `UPDATE users SET is_active=false`, true, true},
		{"deleted workspace", `UPDATE workspaces SET deleted_at=now()`, true, true},
		{"revoked workspace membership", `UPDATE workspace_members SET is_active=false`, true, true},
		{"deleted workspace membership", `UPDATE workspace_members SET deleted_at=now()`, true, true},
		{"unknown workspace role", `UPDATE workspace_members SET role=99`, true, true},
		{"deleted project", `UPDATE projects SET deleted_at=now()`, false, true},
		{"archived project", `UPDATE projects SET archived_at=now()`, false, true},
		{"revoked project membership", `UPDATE project_members SET is_active=false`, false, true},
		{"deleted project membership", `UPDATE project_members SET deleted_at=now()`, false, true},
		{"unknown project role", `UPDATE project_members SET role=99`, false, true},
		{"foreign membership owner", `UPDATE project_members SET workspace_id='00000000-0000-4000-8000-000000000006'`, false, true},
		{"foreign project owner", `UPDATE projects SET workspace_id='00000000-0000-4000-8000-000000000006'`, false, true},
		{"foreign state owner", `UPDATE states SET workspace_id='00000000-0000-4000-8000-000000000006'`, false, false},
		{"deleted state", `UPDATE states SET deleted_at=now()`, false, false},
		{"triage group", `UPDATE states SET "group"='triage'`, false, false},
		{"triage flag", `UPDATE states SET is_triage=true`, false, false},
		{"slug reassignment", `UPDATE workspaces SET slug=CASE WHEN slug='states' THEN 'retired' ELSE 'states' END`, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			tx, err := db.Begin(t.Context())
			require.NoError(t, err)
			defer func() { _ = tx.Rollback(context.WithoutCancel(t.Context())) }()
			_, err = tx.Exec(t.Context(), test.change)
			require.NoError(t, err)
			repository, err := NewRepository(tx)
			require.NoError(t, err)
			assertBoundStateDeniedOrEmpty(t, repository, boundProject, test.projectDenied)
			assertBoundStateDeniedOrEmpty(t, repository, "", test.workspaceDenied)
		})
	}
}

func TestBoundWorkspaceStatesFilterHiddenProjectsBeforePaging(t *testing.T) {
	db := boundStateDatabase(t)
	_, err := db.Exec(t.Context(), `
INSERT INTO projects(id,workspace_id) VALUES($1,$2);
`, boundForeign, boundWorkspace)
	require.NoError(t, err)
	hiddenState := uuid.New()
	_, err = db.Exec(t.Context(), `INSERT INTO states(id,project_id,workspace_id,sequence) VALUES($1,$2,$3,-100)`, hiddenState, boundForeign, boundWorkspace)
	require.NoError(t, err)
	repository, repositoryErr := NewRepository(db)
	require.NoError(t, repositoryErr)
	request := boundReadRequest("", "")
	request.Params["per_page"], request.Present["per_page"] = "1", true
	value, err := repository.ReadAPIV1(t.Context(), boundActor, request)
	require.NoError(t, err)
	page := value.(map[string]any)
	require.Equal(t, 2, page["total_count"])
	require.Equal(t, boundState, page["results"].([]any)[0].(map[string]any)["id"])
	_, err = repository.ReadAPIV1(t.Context(), boundActor, boundReadRequest("", hiddenState.String()))
	require.ErrorIs(t, err, ErrNotFound)
}

func TestBoundStateReadsAllowKnownRolesWithoutGuestFeatureFlags(t *testing.T) {
	db := boundStateDatabase(t)
	repository, repositoryErr := NewRepository(db)
	require.NoError(t, repositoryErr)
	for _, role := range []int{5, 15, 20} {
		_, writeErr := db.Exec(t.Context(), `UPDATE workspace_members SET role=$1`, role)
		require.NoError(t, writeErr)
		_, writeErr = db.Exec(t.Context(), `UPDATE project_members SET role=$1`, role)
		require.NoError(t, writeErr)
		value, readErr := repository.ReadAPIV1(t.Context(), boundActor, boundReadRequest(boundProject, boundState))
		require.NoError(t, readErr)
		require.Equal(t, boundState, value.(map[string]any)["id"])
	}
	_, deleteErr := db.Exec(t.Context(), `DELETE FROM project_members`)
	require.NoError(t, deleteErr)
	assertBoundStateDeniedOrEmpty(t, repository, "", false)
	assertBoundStateDeniedOrEmpty(t, repository, boundProject, true)
}

func assertBoundStateDeniedOrEmpty(t *testing.T, repository *Repository, project string, denied bool) {
	t.Helper()
	value, err := repository.ReadAPIV1(t.Context(), boundActor, boundReadRequest(project, ""))
	if denied {
		require.ErrorIs(t, err, ErrForbidden)
	} else {
		require.NoError(t, err)
		page := value.(map[string]any)
		require.Equal(t, 0, page["total_count"])
		require.Empty(t, page["results"])
	}
	_, err = repository.ReadAPIV1(t.Context(), boundActor, boundReadRequest(project, boundState))
	if denied {
		require.ErrorIs(t, err, ErrForbidden)
	} else {
		require.ErrorIs(t, err, ErrNotFound)
	}
}
