//go:build integration

package assets

import (
	"context"
	"testing"
	"time"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestDraftAssetWritersWaitForConsumptionAndRejectConsumedDraft(t *testing.T) {
	for _, action := range []string{"associate", "upload", "legacy upload", "confirm"} {
		t.Run(action, func(t *testing.T) {
			f := newDraftAssetFixture(t)
			id := f.asset("DRAFT_ISSUE_DESCRIPTION", f.actor)
			if action == "confirm" {
				if err := f.repository.BulkAssociateDraftAssets(f.ctx, f.workspace, f.actor, f.draft, []uuid.UUID{id}); err != nil {
					t.Fatal(err)
				}
			}
			worker, err := pgx.ConnectConfig(f.ctx, f.db.Config())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { testsupport.CloseContext(t, worker, f.ctx) }()
			repo, err := NewPostgresRepository(worker)
			if err != nil {
				t.Fatal(err)
			}
			tx, err := f.db.Begin(f.ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { testsupport.Rollback(t, tx, f.ctx) }()
			if _, err = tx.Exec(f.ctx, `SELECT 1 FROM draft_issues WHERE id=$1 FOR UPDATE`, f.draft); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(f.ctx, 10*time.Second)
			defer cancel()
			finished := make(chan error, 1)
			go func() {
				switch action {
				case "associate":
					finished <- repo.BulkAssociateDraftAssets(ctx, f.workspace, f.actor, f.draft, []uuid.UUID{id})
				case "upload":
					finished <- repo.CreateWorkspaceEntityUpload(ctx, uuid.New(), f.actor, f.workspace, uuid.Nil, "image.png", []byte(`{}`), 10, "DRAFT_ISSUE_DESCRIPTION", f.draft.String(), false)
				case "legacy upload":
					_, createErr := repo.CreateLegacyFileAsset(ctx, uuid.New(), f.actor, map[string]any{"workspace": f.workspace, "draft_issue": f.draft, "is_uploaded": true}, "image.png")
					finished <- createErr
				case "confirm":
					finished <- repo.MutateDraftAsset(ctx, f.workspace, f.actor, id, "confirm", nil, true)
				}
			}()
			waitForDraftAssetLock(t, ctx, tx, worker.PgConn().PID())
			if _, err = tx.Exec(ctx, `UPDATE draft_issues SET deleted_at=now() WHERE id=$1`, f.draft); err != nil {
				t.Fatal(err)
			}
			if err = tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			assertScopeNotFound(t, <-finished, "writer after draft consumption")
			var count int
			if err := f.db.QueryRow(ctx, `SELECT count(*) FROM file_assets`).Scan(&count); err != nil || count != 1 {
				t.Fatalf("late writer created an asset: count=%d err=%v", count, err)
			}
		})
	}
}

func waitForDraftAssetLock(t *testing.T, ctx context.Context, tx pgx.Tx, pid uint32) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var waiting bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE pid=$1 AND wait_event_type='Lock')`, pid).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatal("asset writer never waited for draft consumption lock")
		}
	}
}
