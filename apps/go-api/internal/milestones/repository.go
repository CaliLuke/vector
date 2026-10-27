// Package milestones owns project milestone operations shared by HTTP and MCP.
package milestones

import (
	"context"
	"errors"
	"time"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden    = errors.New("active project membership with an allowed role required")
	ErrNotFound     = errors.New("milestone resource not found")
	ErrInvalidInput = errors.New("invalid milestone input")
)

// Scope retains the immutable authenticated workspace, including for HTTP.
// Resolving a slug alone must never replace this ID after request admission.
type Scope struct {
	ActorID, WorkspaceID, ProjectID uuid.UUID
	WorkspaceSlug, Timezone         string
}

type Repository struct{ database mutationreceipts.Database }

func NewRepository(database mutationreceipts.Database) (*Repository, error) {
	if database == nil {
		return nil, errors.New("milestone primary database is required")
	}
	return &Repository{database: database}, nil
}

func (scope Scope) valid() bool {
	return scope.ActorID != uuid.Nil && scope.WorkspaceID != uuid.Nil && scope.ProjectID != uuid.Nil && scope.WorkspaceSlug != ""
}

func readSnapshot[T any](ctx context.Context, r *Repository, scope Scope, read func(pgx.Tx) (T, error)) (T, error) {
	var zero T
	if !scope.valid() {
		return zero, ErrInvalidInput
	}
	return readTransaction(ctx, r, func(tx pgx.Tx) (T, error) {
		if err := authorizeRead(ctx, tx, scope); err != nil {
			return zero, err
		}
		return read(tx)
	})
}

func readTransaction[T any](ctx context.Context, r *Repository, read func(pgx.Tx) (T, error)) (T, error) {
	var zero T
	tx, err := r.database.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return zero, err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	result, err := read(tx)
	if err != nil {
		return zero, err
	}
	if err = tx.Commit(ctx); err != nil {
		return zero, err
	}
	return result, nil
}

func authorizeRead(ctx context.Context, tx pgx.Tx, scope Scope) error {
	var admitted bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE id=$2::uuid AND slug=$4) AND `+
		issuequery.MilestoneReadScopeSQL("$1::uuid", "$2::uuid", "$3::uuid"), scope.ProjectID, scope.WorkspaceID, scope.ActorID, scope.WorkspaceSlug).Scan(&admitted)
	if err != nil {
		return err
	}
	if !admitted {
		return ErrForbidden
	}
	return nil
}
