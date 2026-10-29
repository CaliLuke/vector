package workitems

import (
	"context"
	"io"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
)

// draftLifecycle owns standalone draft transactions. Reviewed changes instead
// bind a draftOperation to their existing savepoint and shared review state.
type draftLifecycle struct {
	db     Database
	now    func() time.Time
	origin string
}

func (lifecycle draftLifecycle) execute(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	tx, err := lifecycle.db.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	operation := newDraftOperation(tx, lifecycle.now, lifecycle.origin, nil)
	result, err := operation.writeDraft(ctx, q, body, contentType)
	if err != nil {
		return Result{}, err
	}
	return result, tx.Commit(ctx)
}

// These capabilities share item validation and model persistence without giving
// the draft lifecycle access to query discovery, list state or other workflows.
type draftItemValidation interface {
	validateIssue(context.Context, djangoinput.Input, issueValidation) (issueChanges, error)
	validateCustomValues(context.Context, string, map[string]any, map[string]any, map[string]any, bool, bool, bool) ([]string, error)
}

type draftItemCreation interface {
	saveIssue(context.Context, string, map[string]any, map[string]any) (map[string]any, error)
	replaceIssueRelations(context.Context, map[string]any, issueChanges, bool) error
	subscribePropertyMembers(context.Context, string, map[string]any, []string) error
}

type draftOperation struct {
	db          Database
	now         func() time.Time
	origin      string
	review      *itemEditState
	validation  draftItemValidation
	publication draftItemCreation
}

func newDraftOperation(db Database, now func() time.Time, origin string, review *itemEditState) draftOperation {
	// Bind only the dependencies used by the shared item model. Do not copy a
	// repository: query state and unrelated mutation flags must not enter drafts.
	model := &Repository{db: db, now: now, origin: origin, itemEdit: review, transactionalEffects: true}
	return draftOperation{db: db, now: now, origin: origin, review: review, validation: model, publication: model}
}

func (operation *draftOperation) activity(ctx context.Context, q Request, kind string, requested, previous any, defaults bool) error {
	return enqueueEffect(ctx, operation.db, "activity", issueActivityData(q, kind, requested, previous, defaults, operation.now(), operation.origin))
}
