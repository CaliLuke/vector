package stickies

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	initialSortOrder = 65535
	sortOrderStep    = 10000
)

var ErrStickyCreateFailed = errors.New("Sticky create failed")

// StickyID is a Sticky identifier. It is distinct from workspace and user identifiers.
type StickyID uuid.UUID

// CreatedSticky contains the fields produced by the M1 sort-order allocation proof.
type CreatedSticky struct {
	ID        StickyID
	SortOrder float64
}

type transactionStarter interface {
	Begin(context.Context) (pgx.Tx, error)
}

// SortRepository owns the transaction that locks a workspace and creates a Sticky.
// M1.17 extends this narrow repository with the complete Sticky field contract.
type SortRepository struct {
	database transactionStarter
}

// NewSortRepository creates the repository for the M1 sort-order proof.
func NewSortRepository(database transactionStarter) (*SortRepository, error) {
	if database == nil {
		return nil, fmt.Errorf("Sticky sort database is required")
	}
	return &SortRepository{database: database}, nil
}

// CreateWithAllocatedSort locks the owning workspace before it reads and writes
// the next sort order. The active-membership check uses the same transaction.
func (repository *SortRepository) CreateWithAllocatedSort(
	ctx context.Context,
	stickyID StickyID,
	scope Scope,
) (CreatedSticky, error) {
	if uuid.UUID(stickyID) == uuid.Nil() || !validCreateScope(scope) {
		return CreatedSticky{}, ErrInvalidScope
	}

	transaction, err := repository.database.Begin(ctx)
	if err != nil {
		return CreatedSticky{}, fmt.Errorf("%w: begin transaction: %w", ErrStickyCreateFailed, err)
	}
	defer func() {
		_ = transaction.Rollback(context.WithoutCancel(ctx))
	}()

	var lockedWorkspace uuid.UUID
	err = transaction.QueryRow(ctx, `
SELECT id
FROM workspaces
WHERE id = $1
  AND deleted_at IS NULL
FOR UPDATE
`, uuid.UUID(scope.WorkspaceID)).Scan(&lockedWorkspace)
	if errors.Is(err, pgx.ErrNoRows) {
		return CreatedSticky{}, ErrWorkspaceAccessDenied
	}
	if err != nil {
		return CreatedSticky{}, fmt.Errorf("%w: lock workspace: %w", ErrStickyCreateFailed, err)
	}

	var active bool
	err = transaction.QueryRow(
		ctx,
		activeMembershipQuery,
		uuid.UUID(scope.WorkspaceID),
		uuid.UUID(scope.UserID),
	).Scan(&active)
	if err != nil {
		return CreatedSticky{}, fmt.Errorf("%w: read workspace membership: %w", ErrStickyCreateFailed, err)
	}
	if !active {
		return CreatedSticky{}, ErrWorkspaceAccessDenied
	}

	var largest pgtype.Float8
	err = transaction.QueryRow(ctx, `
SELECT MAX(sort_order)
FROM stickies
WHERE workspace_id = $1
  AND deleted_at IS NULL
`, uuid.UUID(scope.WorkspaceID)).Scan(&largest)
	if err != nil {
		return CreatedSticky{}, fmt.Errorf("%w: read largest sort order: %w", ErrStickyCreateFailed, err)
	}
	sortOrder := nextSortOrder(largest)

	_, err = transaction.Exec(ctx, `
INSERT INTO stickies (
    created_at,
    updated_at,
    deleted_at,
    id,
    name,
    description,
    description_html,
    description_stripped,
    description_binary,
    logo_props,
    color,
    background_color,
    created_by_id,
    owner_id,
    updated_by_id,
    workspace_id,
    sort_order
)
VALUES (
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    NULL,
    $1,
    NULL,
    '{}'::jsonb,
    '<p></p>',
    '',
    NULL,
    '{}'::jsonb,
    NULL,
    NULL,
    $2,
    $2,
    NULL,
    $3,
    $4
)
`, uuid.UUID(stickyID), uuid.UUID(scope.UserID), uuid.UUID(scope.WorkspaceID), sortOrder)
	if err != nil {
		return CreatedSticky{}, fmt.Errorf("%w: insert Sticky: %w", ErrStickyCreateFailed, err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return CreatedSticky{}, fmt.Errorf("%w: commit transaction: %w", ErrStickyCreateFailed, err)
	}
	return CreatedSticky{ID: stickyID, SortOrder: sortOrder}, nil
}

func validCreateScope(scope Scope) bool {
	return scope.Operation == OperationCreate &&
		uuid.UUID(scope.WorkspaceID) != uuid.Nil() &&
		uuid.UUID(scope.UserID) != uuid.Nil() &&
		scope.Ownership == AssignOwner &&
		scope.AssignCreator &&
		!scope.RequireCreator
}

func nextSortOrder(largest pgtype.Float8) float64 {
	if !largest.Valid {
		return initialSortOrder
	}
	return largest.Float64 + sortOrderStep
}
