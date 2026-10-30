package workspaceprojectreads

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type projectCleanupDatabase struct {
	*projectLifecycleDBFake
	tx pgx.Tx
}

func (db *projectCleanupDatabase) Begin(context.Context) (pgx.Tx, error) { return db.tx, nil }

type projectCleanupTransaction struct {
	pgx.Tx
	cancel            context.CancelFunc
	failure           error
	cleanup           context.Context
	canceled, bounded bool
}

func (tx *projectCleanupTransaction) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	tx.cancel()
	return pgconn.CommandTag{}, tx.failure
}
func (tx *projectCleanupTransaction) Rollback(ctx context.Context) error {
	tx.cleanup = ctx
	tx.canceled = ctx.Err() != nil
	deadline, ok := ctx.Deadline()
	remaining := time.Until(deadline)
	tx.bounded = ok && remaining > 0 && remaining <= 5*time.Second
	return errors.New("rollback failed")
}
func TestProjectCreationCleansUpAfterCanceledWriteFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	failure := errors.New("insert failed")
	tx := &projectCleanupTransaction{cancel: cancel, failure: failure}
	db := &projectCleanupDatabase{
		projectLifecycleDBFake: &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{projectLifecycleWorkspaceRow(), projectLifecycleBoolRow(false)}}}, tx: tx,
	}
	effects := &projectLifecycleEffectsFake{}
	repo, err := NewLifecycleRepository(db, effects, "", validProjectCover)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.CreateProject(ctx, "00000000-0000-4000-8000-000000000001", "space", djangoinput.Input{Fields: map[string]any{"name": "Created", "identifier": "NEW"}}, false)
	if !errors.Is(err, failure) {
		t.Fatalf("lost insert failure: %v", err)
	}
	if tx.cleanup == nil || tx.canceled || !tx.bounded {
		t.Fatalf("cleanup called=%t canceled=%t bounded=%t", tx.cleanup != nil, tx.canceled, tx.bounded)
	}
	if !errors.Is(tx.cleanup.Err(), context.Canceled) {
		t.Fatalf("cleanup context was not released: %v", tx.cleanup.Err())
	}
	if len(effects.tasks) != 0 {
		t.Fatalf("published %d tasks after insert failure", len(effects.tasks))
	}
}
