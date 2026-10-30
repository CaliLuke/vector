package milestones

import (
	"context"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type failedMilestoneSnapshot struct {
	pgx.Tx
	cancel              context.CancelFunc
	readErr, cleanupErr error
}

func (db *failedMilestoneSnapshot) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	db.cancel()
	return db, nil
}
func (db *failedMilestoneSnapshot) QueryRow(context.Context, string, ...any) pgx.Row {
	return failedMilestoneRow{db.readErr}
}
func (db *failedMilestoneSnapshot) Rollback(context.Context) error { return db.cleanupErr }

type failedMilestoneRow struct{ err error }

func (row failedMilestoneRow) Scan(...any) error { return row.err }
func TestMilestoneListRetainsSnapshotAndCleanupErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	db := &failedMilestoneSnapshot{cancel: cancel, readErr: errors.New("authorization read failed"), cleanupErr: errors.New("snapshot rollback failed")}
	repo, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.List(ctx, Scope{ActorID: uuid.New(), WorkspaceID: uuid.New(), ProjectID: uuid.New(), WorkspaceSlug: "space"}, djangoapi.Query{})
	if !errors.Is(err, db.readErr) || !errors.Is(err, db.cleanupErr) {
		t.Fatalf("read or cleanup failure lost: %v", err)
	}
}
