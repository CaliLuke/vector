//go:build integration

package workitemtypes_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitemsearch"
)

func TestRichTextSearchBackfillBatchesRollbackAndRetry(t *testing.T) {
	f, _, field := newHistoryRecorderFixture(t)
	for range 101 {
		id := addHierarchyItem(t, f, f.task, "")
		f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($2::text,'<p>Batch value</p>') WHERE id=$1`, id, field)
	}
	var malformed string
	if err := f.db.QueryRow(f.ctx, `SELECT id::text FROM issues WHERE workspace_id=$1 ORDER BY id DESC LIMIT 1`, f.workspace).Scan(&malformed); err != nil {
		t.Fatal(err)
	}
	// Choose the last item so a complete first batch was written before failure.
	// A malformed legacy value must abort the workspace, not leave a partial index
	// or a completion marker that prevents a repaired retry.
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($2::text,42) WHERE id=$1`, malformed, field)
	if err := workitemsearch.Backfill(f.ctx, f.db); err == nil {
		t.Fatal("malformed rich text was accepted")
	}
	assertSearchBackfillCounts(t, f, 0, 0)
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($2::text,'<p>Repaired value</p>') WHERE id=$1`, malformed, field)
	// Concurrent bootstrap attempts serialize on the workspace lock and recheck
	// the journal; both succeed without duplicate projections or duplicate markers.
	done := make(chan error, 2)
	for range 2 {
		connection, err := pgx.ConnectConfig(f.ctx, f.db.Config().Copy())
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			defer func() { testsupport.CloseContext(t, connection, context.Background()) }()
			done <- workitemsearch.Backfill(f.ctx, connection)
		}()
	}
	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	assertSearchBackfillCounts(t, f, 102, 1)
	expectRichSearch(t, f, "Repaired value", malformed, field)
}

func TestRichTextSearchBackfillCancellationReleasesLockForRetry(t *testing.T) {
	f, state, field := newHistoryRecorderFixture(t)
	tx, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err = tx.Exec(f.ctx, `SELECT work_item_schema_lock($1::uuid)`, f.workspace); err != nil {
		t.Fatal(err)
	}
	connection, err := pgx.ConnectConfig(f.ctx, f.db.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.CloseContext(t, connection, context.Background()) }()
	ctx, cancel := context.WithTimeout(f.ctx, 100*time.Millisecond)
	defer cancel()
	if err = workitemsearch.Backfill(ctx, connection); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline: %v", err)
	}
	assertSearchBackfillCounts(t, f, 0, 0)
	if err = tx.Rollback(f.ctx); err != nil {
		t.Fatal(err)
	}
	if err = workitemsearch.Backfill(f.ctx, f.db); err != nil {
		t.Fatal(err)
	}
	assertSearchBackfillCounts(t, f, 1, 1)
	expectRichSearch(t, f, "Before", state.ItemID, field)
}
func assertSearchBackfillCounts(t *testing.T, f projectTypeFixture, rows, marks int) {
	t.Helper()
	var gotRows, gotMarks int
	if err := f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM work_item_property_search WHERE workspace_id=$1),(SELECT count(*) FROM work_item_search_backfills WHERE workspace_id=$1)`, f.workspace).Scan(&gotRows, &gotMarks); err != nil {
		t.Fatal(err)
	}
	if gotRows != rows || gotMarks != marks {
		t.Fatalf("projection rows=%d markers=%d; want %d,%d", gotRows, gotMarks, rows, marks)
	}
}
