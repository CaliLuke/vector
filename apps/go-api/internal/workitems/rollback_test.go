package workitems

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// These setup rows let each operation reach its transaction. Existing PostgreSQL
// suites own query/authorization coverage; this probe injects cleanup failures.
type itemCleanupDatabase struct {
	pgx.Tx
	cancel   context.CancelFunc
	failure  error
	rollback func(context.Context) error
	begins   int
}

func (db *itemCleanupDatabase) Begin(context.Context) (pgx.Tx, error) {
	db.begins++
	db.cancel()
	return db, nil
}
func (db *itemCleanupDatabase) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, db.failure
}
func (db *itemCleanupDatabase) Commit(context.Context) error       { return db.failure }
func (db *itemCleanupDatabase) Rollback(ctx context.Context) error { return db.rollback(ctx) }
func (*itemCleanupDatabase) QueryRow(context.Context, string, ...any) pgx.Row {
	return itemCleanupRow{}
}
func (*itemCleanupDatabase) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return &itemCleanupRows{}, nil
}

type itemCleanupRow struct{}

func (itemCleanupRow) Scan(destinations ...any) error {
	if len(destinations) != 1 {
		return fmt.Errorf("unexpected setup scan arity %d", len(destinations))
	}
	switch value := destinations[0].(type) {
	case *[]byte:
		*value = []byte(`{"workspace_id":"00000000-0000-4000-8000-000000000001"}`)
	case **string:
		*value = nil
	case *bool:
		*value = false
	default:
		return fmt.Errorf("unexpected setup scan type %T", value)
	}
	return nil
}

type itemCleanupRows struct{ pgx.Rows }

func (*itemCleanupRows) Next() bool { return false }
func (*itemCleanupRows) Err() error { return nil }
func (*itemCleanupRows) Close()     {}

func TestItemTransactionsPreserveRollbackFailureAfterCancellation(t *testing.T) {
	q := Request{Method: "POST", Slug: "demo", ProjectID: "00000000-0000-4000-8000-000000000002", IssueID: "00000000-0000-4000-8000-000000000003", ActorID: "00000000-0000-4000-8000-000000000004"}
	for _, test := range []struct {
		name string
		run  func(context.Context, *Repository) error
	}{
		{"create item", func(ctx context.Context, r *Repository) error {
			_, err := r.saveIssueRow(ctx, q.ActorID, defaultIssue(q.ProjectID), nil)
			return err
		}},
		{"collaboration", func(ctx context.Context, r *Repository) error {
			_, err := r.saveCollaboration(ctx, q, "issue_links", collaborationDefault("issue_links", q, "00000000-0000-4000-8000-000000000001"), nil)
			return err
		}},
		{"planning", func(ctx context.Context, r *Repository) error {
			_, err := r.insertPlanning(ctx, "module", q, "workspace", []string{q.IssueID}, "module", q.ActorID, false)
			return err
		}},
		{"date batch", func(ctx context.Context, r *Repository) error {
			_, err := r.issueDates(ctx, q, strings.NewReader(`{}`), "application/json")
			return err
		}},
		{"module batch", func(ctx context.Context, r *Repository) error {
			_, err := r.issueModules(ctx, q, djangoinput.Input{Fields: map[string]any{}})
			return err
		}},
		{"relations", func(ctx context.Context, r *Repository) error {
			_, err := r.relations(ctx, q, strings.NewReader(`{"relation_type":"relates_to","issues":[]}`), "application/json")
			return err
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			primary, secondary := errors.New("operation failed"), errors.New("rollback failed")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var cleanup context.Context
			var canceled, bounded bool
			calls := 0
			db := &itemCleanupDatabase{cancel: cancel, failure: primary, rollback: func(ctx context.Context) error {
				calls++
				cleanup = ctx
				canceled = ctx.Err() != nil
				deadline, ok := ctx.Deadline()
				remaining := time.Until(deadline)
				bounded = ok && remaining > 0 && remaining <= 5*time.Second
				return secondary
			}}
			err := test.run(ctx, &Repository{db: db, now: time.Now})
			if db.begins != 1 || calls != 1 {
				t.Fatalf("begins=%d rollbacks=%d error=%v", db.begins, calls, err)
			}
			if !errors.Is(err, primary) || !errors.Is(err, secondary) {
				t.Fatalf("operation or rollback failure lost: %v", err)
			}
			if canceled || !bounded {
				t.Fatalf("cleanup canceled=%t bounded=%t", canceled, bounded)
			}
			if !errors.Is(cleanup.Err(), context.Canceled) {
				t.Fatalf("cleanup context not released: %v", cleanup.Err())
			}
		})
	}
}
