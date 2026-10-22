// Package stickies defines the authorization scope for the future Sticky repository.
package stickies

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
)

// Operation identifies one frontend Sticky repository operation.
type Operation string

const (
	OperationList     Operation = "list"
	OperationCreate   Operation = "create"
	OperationRetrieve Operation = "retrieve"
	OperationUpdate   Operation = "update"
	OperationSort     Operation = "sort"
	OperationDelete   Operation = "delete"
)

// OwnershipRule identifies the owner predicate that a repository query must apply.
type OwnershipRule uint8

const (
	// AssignOwner assigns the authenticated user as the owner of a new Sticky.
	AssignOwner OwnershipRule = iota + 1
	// RequireOwner restricts an existing Sticky to its authenticated owner.
	RequireOwner
)

var (
	// ErrWorkspaceAccessDenied does not reveal whether a membership or workspace exists.
	ErrWorkspaceAccessDenied = errors.New("workspace access denied")
	// ErrInvalidScope identifies missing identifiers or an unknown operation.
	ErrInvalidScope = errors.New("invalid Sticky scope")
)

var operationPolicies = map[Operation]struct {
	ownership      OwnershipRule
	assignCreator  bool
	requireCreator bool
}{
	OperationList:     {ownership: RequireOwner},
	OperationCreate:   {ownership: AssignOwner, assignCreator: true},
	OperationRetrieve: {ownership: RequireOwner},
	OperationUpdate:   {ownership: RequireOwner, requireCreator: true},
	OperationSort:     {ownership: RequireOwner, requireCreator: true},
	OperationDelete:   {ownership: RequireOwner, requireCreator: true},
}

// WorkspaceID is a workspace identifier. It is distinct from other domain identifiers.
type WorkspaceID uuid.UUID

// UserID is a user identifier. It is distinct from other domain identifiers.
type UserID uuid.UUID

// Scope contains all mandatory predicates for one future Sticky repository operation.
type Scope struct {
	Operation      Operation
	WorkspaceID    WorkspaceID
	UserID         UserID
	Ownership      OwnershipRule
	AssignCreator  bool
	RequireCreator bool
}

// MembershipReader reads the active-membership predicate without changing database state.
type MembershipReader interface {
	HasActiveMembership(context.Context, WorkspaceID, UserID) (bool, error)
}

// Authorizer produces the complete repository scope for a frontend Sticky operation.
type Authorizer struct {
	memberships MembershipReader
}

// NewAuthorizer creates a Sticky scope authorizer.
func NewAuthorizer(memberships MembershipReader) (*Authorizer, error) {
	if memberships == nil {
		return nil, fmt.Errorf("Sticky membership reader is required")
	}
	return &Authorizer{memberships: memberships}, nil
}

// ScopeFor requires active membership before it returns tenant and ownership predicates.
func (authorizer *Authorizer) ScopeFor(
	ctx context.Context,
	operation Operation,
	workspaceID WorkspaceID,
	userID UserID,
) (Scope, error) {
	policy, ok := operationPolicies[operation]
	if !ok || uuid.UUID(workspaceID) == uuid.Nil() || uuid.UUID(userID) == uuid.Nil() {
		return Scope{}, ErrInvalidScope
	}
	active, err := authorizer.memberships.HasActiveMembership(ctx, workspaceID, userID)
	if err != nil {
		return Scope{}, fmt.Errorf("read Sticky workspace membership: %w", err)
	}
	if !active {
		return Scope{}, ErrWorkspaceAccessDenied
	}
	return Scope{
		Operation:      operation,
		WorkspaceID:    workspaceID,
		UserID:         userID,
		Ownership:      policy.ownership,
		AssignCreator:  policy.assignCreator,
		RequireCreator: policy.requireCreator,
	}, nil
}

const activeMembershipQuery = `
SELECT EXISTS (
    SELECT 1
    FROM workspace_members
    WHERE workspace_id = $1
      AND member_id = $2
      AND is_active IS TRUE
      AND deleted_at IS NULL
)
`

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// PostgresMembershipReader reads Plane's workspace_members table.
type PostgresMembershipReader struct {
	database queryRower
}

// NewPostgresMembershipReader creates a read-only membership reader.
func NewPostgresMembershipReader(database queryRower) (*PostgresMembershipReader, error) {
	if database == nil {
		return nil, fmt.Errorf("Sticky membership database is required")
	}
	return &PostgresMembershipReader{database: database}, nil
}

// HasActiveMembership applies the active, non-deleted workspace-membership predicate.
func (reader *PostgresMembershipReader) HasActiveMembership(
	ctx context.Context,
	workspaceID WorkspaceID,
	userID UserID,
) (bool, error) {
	var active bool
	err := reader.database.QueryRow(
		ctx,
		activeMembershipQuery,
		uuid.UUID(workspaceID),
		uuid.UUID(userID),
	).Scan(&active)
	if err != nil {
		return false, fmt.Errorf("query active workspace membership: %w", err)
	}
	return active, nil
}
