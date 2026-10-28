package workitems

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// ReadItemInTransaction projects one currently visible item inside a caller's
// repeatable-read snapshot. Association catalogues use this same native serializer.
func ReadItemInTransaction(ctx context.Context, tx pgx.Tx, q Request) (Result, error) {
	if tx == nil || q.ExpectedWorkspaceID == nil || !q.API || q.Method != "GET" || q.ProjectID == "" || q.IssueID == "" {
		return Result{}, forbidden()
	}
	reader := &Repository{db: tx}
	if err := reader.authorizeWorkitemRead(ctx, q); err != nil {
		return Result{}, err
	}
	q.Operation = "retrieve"
	return reader.boundWorkitemDetail(ctx, q)
}
