//go:build integration

package milestones_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestMilestoneReadsShareAuthorizationCountAndRowSnapshot(t *testing.T) {
	f := newReadFixture(t)
	// A fixture view pauses the first data query after scope admission. The
	// concurrent writer bypasses the view; production code has no test hooks.
	_, err := f.db.Exec(f.ctx, `ALTER TABLE milestones RENAME TO stored_milestones;
 CREATE FUNCTION milestone_read_gate() RETURNS boolean LANGUAGE plpgsql VOLATILE AS $$
 BEGIN PERFORM pg_advisory_xact_lock(38118); RETURN true; END; $$;
 CREATE VIEW milestones AS SELECT * FROM stored_milestones WHERE milestone_read_gate();
 SELECT pg_advisory_lock(38118);`)
	require.NoError(t, err)
	defer func() { _, _ = f.db.Exec(context.WithoutCancel(f.ctx), `SELECT pg_advisory_unlock(38118)`) }()
	reader, err := pgx.ConnectConfig(f.ctx, f.db.Config().Copy())
	require.NoError(t, err)
	repo, err := milestones.NewRepository(reader)
	require.NoError(t, err)
	readCtx, cancel := context.WithTimeout(f.ctx, 10*time.Second)
	defer cancel()
	type outcome struct {
		page milestones.Page[milestones.Record]
		err  error
	}
	finished := make(chan outcome, 1)
	go func() {
		page, failure := repo.List(readCtx, f.scope, djangoapi.Query{})
		closeErr := reader.Close(context.WithoutCancel(readCtx))
		finished <- outcome{page, errors.Join(failure, closeErr)}
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		failure := f.db.QueryRow(f.ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE locktype='advisory' AND objid=38118 AND NOT granted)`).Scan(&waiting)
		return failure == nil && waiting
	}, 3*time.Second, 10*time.Millisecond, "milestone count must reach the database barrier")
	_, err = f.db.Exec(f.ctx, `UPDATE stored_milestones SET deleted_at=now() WHERE id=$1`, firstMilestone)
	require.NoError(t, err)
	_, err = f.db.Exec(f.ctx, `UPDATE project_members SET is_active=false; SELECT pg_advisory_unlock(38118)`)
	require.NoError(t, err)
	result := <-finished
	require.NoError(t, result.err)
	require.Equal(t, 3, result.page.Total)
	require.Len(t, result.page.Results, 3)
	require.Equal(t, firstMilestone, result.page.Results[0].ID)
	_, err = f.repo.List(f.ctx, f.scope, djangoapi.Query{})
	require.ErrorIs(t, err, milestones.ErrForbidden)
	_, err = f.db.Exec(f.ctx, `UPDATE project_members SET is_active=true`)
	require.NoError(t, err)
	page, err := f.repo.List(f.ctx, f.scope, djangoapi.Query{})
	require.NoError(t, err)
	require.Equal(t, 2, page.Total)
	require.Len(t, page.Results, 2)
	require.Equal(t, laterMilestone, page.Results[0].ID)
}
