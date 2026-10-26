//go:build integration

package workitemtypes_test

import (
	"context"
	"testing"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/jackc/pgx/v5"
)

func hierarchyConnection(t *testing.T, f projectTypeFixture) *pgx.Conn {
	t.Helper()
	connection, err := pgx.ConnectConfig(f.ctx, f.db.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(f.ctx)) })
	return connection
}
func TestHierarchyQueuedReviewRechecksAdministrator(t *testing.T) {
	for _, commit := range []bool{false, true} {
		t.Run(map[bool]string{false: "preview", true: "commit"}[commit], func(t *testing.T) {
			f := newProjectTypeFixture(t)
			input := hierarchyInput(t, f, 1)
			if commit {
				review, err := f.repo.PreviewHierarchy(f.ctx, f.owner, "types", input)
				if err != nil {
					t.Fatal(err)
				}
				input.ReviewID = review.ID
			}
			connection := hierarchyConnection(t, f)
			repository := types.NewRepository(connection)
			lock := hierarchyTransaction(t, f)
			done := make(chan error, 1)
			go func() {
				var err error
				if commit {
					_, err = repository.CommitHierarchy(f.ctx, f.owner, "types", input)
				} else {
					_, err = repository.PreviewHierarchy(f.ctx, f.owner, "types", input)
				}
				done <- err
			}()
			waitForSchemaWriter(t, f, lock, int(connection.PgConn().PID()), done)
			if _, err := lock.Exec(f.ctx, `UPDATE workspace_members SET role=15 WHERE workspace_id=$1 AND member_id=$2`, f.workspace, f.owner); err != nil {
				t.Fatal(err)
			}
			if err := lock.Commit(f.ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				requireTypeStatus(t, err, 403)
			case <-f.ctx.Done():
				t.Fatal("queued hierarchy write did not finish")
			}
			var enabled bool
			if err := f.db.QueryRow(f.ctx, `SELECT hierarchy_enabled FROM work_item_type_settings WHERE workspace_id=$1`, f.workspace).Scan(&enabled); err != nil || enabled {
				t.Fatalf("demoted admin activated hierarchy: %t %v", enabled, err)
			}
		})
	}
}
func TestHierarchyOppositeEdgesSerializeAndRejectCycle(t *testing.T) {
	f := newProjectTypeFixture(t)
	first := addHierarchyItem(t, f, f.task, "")
	second := addHierarchyItem(t, f, f.task, "")
	connection := hierarchyConnection(t, f)
	lock := hierarchyTransaction(t, f)
	if _, err := lock.Exec(f.ctx, `UPDATE issues SET parent_id=$2 WHERE id=$1`, first, second); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- func() error {
			tx, err := connection.Begin(f.ctx)
			if err != nil {
				return err
			}
			defer func() { _ = tx.Rollback(context.WithoutCancel(f.ctx)) }()
			if _, err = tx.Exec(f.ctx, `SELECT work_item_schema_lock($1::uuid)`, f.workspace); err != nil {
				return err
			}
			if _, err = tx.Exec(f.ctx, `UPDATE issues SET parent_id=$2 WHERE id=$1`, second, first); err != nil {
				return err
			}
			return tx.Commit(f.ctx)
		}()
	}()
	waitForSchemaWriter(t, f, lock, int(connection.PgConn().PID()), done)
	if err := lock.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		requireHierarchyConstraint(t, err)
	case <-f.ctx.Done():
		t.Fatal("opposite-edge writer did not finish")
	}
	var linked int
	if err := f.db.QueryRow(f.ctx, `SELECT count(*) FROM issues WHERE id=ANY($1::uuid[]) AND parent_id IS NOT NULL`, []string{first, second}).Scan(&linked); err != nil || linked != 1 {
		t.Fatalf("cycle or partial rollback: %d %v", linked, err)
	}
}
func TestHierarchyLongUnconfiguredChainTerminatesAndRejectsCycle(t *testing.T) {
	f := newProjectTypeFixture(t)
	first := addHierarchyItem(t, f, f.task, "")
	last := first
	for range 128 {
		last = addHierarchyItem(t, f, f.task, last)
	}
	_, err := f.db.Exec(f.ctx, `UPDATE issues SET parent_id=$2 WHERE id=$1`, first, last)
	requireHierarchyConstraint(t, err)
	if _, err = f.db.Exec(f.ctx, `SELECT validate_workspace_work_item_hierarchy($1::uuid)`, f.workspace); err != nil {
		t.Fatalf("valid long chain rejected: %v", err)
	}
}
