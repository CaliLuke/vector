//go:build integration

package workspaceprojectreads_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	projects "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type archiveEffects struct{}

func (archiveEffects) Task(context.Context, string, []any, map[string]any) error {
	panic("archive must not dispatch tasks")
}
func (archiveEffects) SoftDelete(context.Context, string, string, string) error {
	panic("archive must retain the transactional favorite update")
}

type archiveFixture struct {
	db  *pgxpool.Pool
	app *projects.LifecycleRepository
	cmd projects.ArchiveCommand
}

func newArchiveFixture(t *testing.T) archiveFixture {
	t.Helper()
	receiptSQL, err := os.ReadFile("../bootstrap/schema_migrations/000016_mutation_receipts.up.sql")
	require.NoError(t, err)
	revisionSQL, err := os.ReadFile("../bootstrap/schema_migrations/000017_project_edit_revisions.up.sql")
	require.NoError(t, err)
	ctx, conn := postgres.OpenSQL(t, projectReadSchema+`
ALTER TABLE user_favorites ADD COLUMN workspace_id uuid;
INSERT INTO user_favorites(user_id,project_id,workspace_id) VALUES('10000000-0000-4000-8000-000000000001','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001');
`+string(receiptSQL)+string(revisionSQL))
	pool, err := pgxpool.New(ctx, conn.Config().ConnString())
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	app, err := projects.NewLifecycleRepository(pool, archiveEffects{}, "https://vector.example", func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error) { return false, nil })
	require.NoError(t, err)
	return archiveFixture{pool, app, projects.ArchiveCommand{
		ActorID: uuid.MustParse("10000000-0000-4000-8000-000000000001"), WorkspaceID: uuid.MustParse("20000000-0000-4000-8000-000000000001"),
		ProjectID: uuid.MustParse("30000000-0000-4000-8000-000000000001"), WorkspaceSlug: "space", Archive: true,
	}}
}

func archiveState(t *testing.T, f archiveFixture, archived bool, revision int64, favorites, receipts int) {
	t.Helper()
	var actualArchived bool
	var actualRevision int64
	var actualFavorites, actualReceipts int
	require.NoError(t, f.db.QueryRow(t.Context(), `SELECT archived_at IS NOT NULL,(SELECT revision FROM project_edit_revisions WHERE project_id=projects.id),(SELECT count(*) FROM user_favorites WHERE deleted_at IS NULL),(SELECT count(*) FROM mutation_receipts) FROM projects WHERE id=$1`, f.cmd.ProjectID).Scan(&actualArchived, &actualRevision, &actualFavorites, &actualReceipts))
	require.Equal(t, archived, actualArchived)
	require.Equal(t, revision, actualRevision)
	require.Equal(t, favorites, actualFavorites)
	require.Equal(t, receipts, actualReceipts)
}

func TestArchiveRetainsHTTPPoliciesAndCleansFavoritesAtomically(t *testing.T) {
	for _, tc := range []struct {
		name               string
		app, archive       bool
		workspace, project int
		allowed            bool
	}{
		{"API member archives without project membership", false, true, 15, 0, true},
		{"API member cannot restore without membership", false, false, 15, 0, false},
		{"API member cannot restore with member role", false, false, 15, 15, false},
		{"API project admin restores", false, false, 15, 20, true},
		{"API workspace admin needs project membership to restore", false, false, 20, 0, false},
		{"API workspace admin restores with guest project membership", false, false, 20, 5, true},
		{"API workspace guest cannot archive", false, true, 5, 20, false},
		{"API project admin can restore from workspace guest role", false, false, 5, 20, true},
		{"app member archives", true, true, 15, 15, true},
		{"app member restores", true, false, 15, 15, true},
		{"app project guest cannot archive", true, true, 15, 5, false},
		{"app workspace admin needs project membership", true, true, 20, 0, false},
		{"app workspace admin with project guest membership", true, true, 20, 5, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newArchiveFixture(t)
			_, err := f.db.Exec(t.Context(), `UPDATE workspace_members SET role=$1`, tc.workspace)
			require.NoError(t, err)
			if tc.project != 0 {
				_, err = f.db.Exec(t.Context(), `INSERT INTO project_members(project_id,workspace_id,member_id,role,is_active) VALUES($1,$2,$3,$4,true)`, f.cmd.ProjectID, f.cmd.WorkspaceID, f.cmd.ActorID, tc.project)
				require.NoError(t, err)
			}
			f.cmd.App, f.cmd.Archive = tc.app, tc.archive
			// HTTP does not carry a preselected immutable workspace ID.
			if tc.app {
				f.cmd.WorkspaceID = uuid.Nil
			}
			result, err := f.app.ArchiveProject(t.Context(), f.cmd)
			if !tc.allowed {
				require.ErrorIs(t, err, projects.ErrForbidden)
				archiveState(t, f, false, 1, 1, 0)
				return
			}
			require.NoError(t, err)
			require.False(t, result.At.IsZero())
			require.Equal(t, "project:"+f.cmd.ProjectID.String()+":2", result.Revision)
			require.Nil(t, result.Receipt.ReplayExpiresAt)
			favorites := 1
			if tc.archive {
				favorites = 0
			}
			archiveState(t, f, tc.archive, 2, favorites, 0)
		})
	}
}

func TestArchiveReceiptReplaysBeforeStaleRevisionButAfterCurrentAuthorization(t *testing.T) {
	f := newArchiveFixture(t)
	request := uuid.New()
	revision := "project:" + f.cmd.ProjectID.String() + ":1"
	f.cmd.RequestID, f.cmd.ExpectedRevision = &request, &revision
	first, err := f.app.ArchiveProject(t.Context(), f.cmd)
	require.NoError(t, err)
	replay, err := f.app.ArchiveProject(t.Context(), f.cmd)
	require.NoError(t, err)
	require.True(t, replay.Receipt.Replayed)
	require.Equal(t, first.At, replay.At)
	require.Equal(t, first.Revision, replay.Revision)
	archiveState(t, f, true, 2, 0, 1)
	changed := "project:" + f.cmd.ProjectID.String() + ":2"
	f.cmd.ExpectedRevision = &changed
	_, err = f.app.ArchiveProject(t.Context(), f.cmd)
	require.ErrorIs(t, err, mutationreceipts.ErrConflict)
	_, err = f.db.Exec(t.Context(), `UPDATE workspace_members SET is_active=false`)
	require.NoError(t, err)
	_, err = f.app.ArchiveProject(t.Context(), f.cmd)
	require.ErrorIs(t, err, projects.ErrForbidden)
	archiveState(t, f, true, 2, 0, 1)
}

func TestArchiveFailureRollsBackProjectFavoritesRevisionAndReceipt(t *testing.T) {
	f := newArchiveFixture(t)
	request := uuid.New()
	f.cmd.RequestID = &request
	_, err := f.db.Exec(t.Context(), `CREATE FUNCTION reject_archive_favorite() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'controlled favorite failure'; END $$; CREATE TRIGGER reject_archive_favorite BEFORE UPDATE ON user_favorites FOR EACH ROW EXECUTE FUNCTION reject_archive_favorite()`)
	require.NoError(t, err)
	_, err = f.app.ArchiveProject(t.Context(), f.cmd)
	require.Error(t, err)
	archiveState(t, f, false, 1, 1, 0)
}

func TestArchiveRejectsChangedScopeAndCredentialAfterAdmission(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"account deactivated", `UPDATE users SET is_active=false`},
		{"workspace deleted", `UPDATE workspaces SET deleted_at=now()`},
		{"membership deleted", `UPDATE workspace_members SET deleted_at=now()`},
		{"membership role removed", `UPDATE workspace_members SET role=0`},
		{"slug changed", `UPDATE workspaces SET slug='renamed'`},
		{"credential revoked", ``},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newArchiveFixture(t)
			f.cmd.Revalidate = func(ctx context.Context) error {
				if tc.sql == "" {
					return projects.ErrForbidden
				}
				_, err := f.db.Exec(ctx, tc.sql)
				return err
			}
			result, err := f.app.ArchiveProject(t.Context(), f.cmd)
			require.ErrorIs(t, err, projects.ErrForbidden)
			require.Empty(t, result)
			archiveState(t, f, false, 1, 1, 0)
		})
	}
}

func TestProjectRevisionIncludesOtherWritersAndNewProjects(t *testing.T) {
	f := newArchiveFixture(t)
	_, err := f.db.Exec(t.Context(), `UPDATE projects SET name='HTTP or worker edit'`)
	require.NoError(t, err)
	_, err = f.db.Exec(t.Context(), `INSERT INTO projects(id,workspace_id,name) VALUES('30000000-0000-4000-8000-000000000002',$1,'Created later')`, f.cmd.WorkspaceID)
	require.NoError(t, err)
	var added int64
	require.NoError(t, f.db.QueryRow(t.Context(), `SELECT revision FROM project_edit_revisions WHERE project_id='30000000-0000-4000-8000-000000000002'`).Scan(&added))
	require.Equal(t, int64(1), added)
	stale := "project:" + f.cmd.ProjectID.String() + ":1"
	f.cmd.ExpectedRevision = &stale
	_, err = f.app.ArchiveProject(t.Context(), f.cmd)
	require.ErrorIs(t, err, projects.ErrRevisionConflict)
	archiveState(t, f, false, 2, 1, 0)
}

func TestConcurrentArchiveCompareAndSetCommitsOneMutation(t *testing.T) {
	f := newArchiveFixture(t)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	ready := make(chan struct{}, 2)
	release := make(chan struct{})
	done := make(chan error, 2)
	revision := "project:" + f.cmd.ProjectID.String() + ":1"
	for range 2 {
		go func() {
			cmd := f.cmd
			request := uuid.New()
			cmd.RequestID, cmd.ExpectedRevision = &request, &revision
			cmd.Revalidate = func(ctx context.Context) error {
				ready <- struct{}{}
				select {
				case <-release:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			_, err := f.app.ArchiveProject(ctx, cmd)
			done <- err
		}()
	}
	for range 2 {
		select {
		case <-ready:
		case err := <-done:
			t.Fatalf("write finished before barrier: %v", err)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	close(release)
	first, second := <-done, <-done
	require.True(t, first == nil && errors.Is(second, projects.ErrRevisionConflict) || second == nil && errors.Is(first, projects.ErrRevisionConflict), "outcomes: %v, %v", first, second)
	archiveState(t, f, true, 2, 0, 1)
}

func TestArchiveCASSeesConcurrentOrdinaryWriterAfterLockWait(t *testing.T) {
	f := newArchiveFixture(t)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	tx, err := f.db.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	var writerPID int
	require.NoError(t, tx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&writerPID))
	_, err = tx.Exec(ctx, `UPDATE projects SET name='HTTP update won' WHERE id=$1`, f.cmd.ProjectID)
	require.NoError(t, err)
	revision := "project:" + f.cmd.ProjectID.String() + ":1"
	f.cmd.ExpectedRevision = &revision
	done := make(chan error, 1)
	go func() { _, err := f.app.ArchiveProject(ctx, f.cmd); done <- err }()
	require.Eventually(t, func() bool {
		var waiting bool
		queryErr := f.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid)))`, writerPID).Scan(&waiting)
		return queryErr == nil && waiting
	}, 5*time.Second, 10*time.Millisecond, "archive must wait behind the ordinary writer")
	require.NoError(t, tx.Commit(ctx))
	select {
	case err := <-done:
		require.ErrorIs(t, err, projects.ErrRevisionConflict)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	archiveState(t, f, false, 2, 1, 0)
	var name string
	require.NoError(t, f.db.QueryRow(ctx, `SELECT name FROM projects WHERE id=$1`, f.cmd.ProjectID).Scan(&name))
	require.Equal(t, "HTTP update won", name)
}
