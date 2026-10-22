package stickies

import (
	"context"
	"errors"
	"testing"
	"uuid"
)

var allOperations = []Operation{
	OperationList,
	OperationCreate,
	OperationRetrieve,
	OperationUpdate,
	OperationSort,
	OperationDelete,
}

type membershipReaderFunc func(context.Context, WorkspaceID, UserID) (bool, error)

func (function membershipReaderFunc) HasActiveMembership(
	ctx context.Context,
	workspaceID WorkspaceID,
	userID UserID,
) (bool, error) {
	return function(ctx, workspaceID, userID)
}

func TestScopeForRequiresActiveMembershipForEveryOperation(t *testing.T) {
	workspaceID := WorkspaceID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290311"))
	userID := UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290312"))
	calls := 0
	authorizer, err := NewAuthorizer(membershipReaderFunc(func(
		_ context.Context,
		gotWorkspaceID WorkspaceID,
		gotUserID UserID,
	) (bool, error) {
		calls++
		if gotWorkspaceID != workspaceID || gotUserID != userID {
			t.Fatalf("membership identifiers = %s, %s", uuid.UUID(gotWorkspaceID), uuid.UUID(gotUserID))
		}
		return true, nil
	}))
	if err != nil {
		t.Fatalf("NewAuthorizer() error = %v", err)
	}

	for _, operation := range allOperations {
		t.Run(string(operation), func(t *testing.T) {
			scope, scopeErr := authorizer.ScopeFor(context.Background(), operation, workspaceID, userID)
			if scopeErr != nil {
				t.Fatalf("ScopeFor() error = %v", scopeErr)
			}
			if scope.Operation != operation || scope.WorkspaceID != workspaceID || scope.UserID != userID {
				t.Fatalf("ScopeFor() scope = %+v", scope)
			}
		})
	}
	if calls != len(allOperations) {
		t.Fatalf("active membership calls = %d, want %d", calls, len(allOperations))
	}
}

func TestScopeForReturnsOperationOwnershipRules(t *testing.T) {
	workspaceID := WorkspaceID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290321"))
	userID := UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290322"))
	authorizer, err := NewAuthorizer(membershipReaderFunc(func(context.Context, WorkspaceID, UserID) (bool, error) {
		return true, nil
	}))
	if err != nil {
		t.Fatalf("NewAuthorizer() error = %v", err)
	}

	tests := map[Operation]struct {
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
	for operation, want := range tests {
		scope, scopeErr := authorizer.ScopeFor(context.Background(), operation, workspaceID, userID)
		if scopeErr != nil {
			t.Fatalf("ScopeFor(%s) error = %v", operation, scopeErr)
		}
		if scope.Ownership != want.ownership ||
			scope.AssignCreator != want.assignCreator ||
			scope.RequireCreator != want.requireCreator {
			t.Fatalf(
				"ScopeFor(%s) ownership = %d, assign creator = %t, require creator = %t",
				operation,
				scope.Ownership,
				scope.AssignCreator,
				scope.RequireCreator,
			)
		}
	}
}

func TestScopeForHidesMissingInactiveAndDeletedMemberships(t *testing.T) {
	authorizer, err := NewAuthorizer(membershipReaderFunc(func(context.Context, WorkspaceID, UserID) (bool, error) {
		return false, nil
	}))
	if err != nil {
		t.Fatalf("NewAuthorizer() error = %v", err)
	}
	workspaceID := WorkspaceID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290331"))
	userID := UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290332"))

	for _, operation := range allOperations {
		_, scopeErr := authorizer.ScopeFor(context.Background(), operation, workspaceID, userID)
		if !errors.Is(scopeErr, ErrWorkspaceAccessDenied) {
			t.Fatalf("ScopeFor(%s) error = %v, want access denied", operation, scopeErr)
		}
	}
}

func TestScopeForRejectsUnknownOperationAndMissingIdentifiers(t *testing.T) {
	calls := 0
	authorizer, err := NewAuthorizer(membershipReaderFunc(func(context.Context, WorkspaceID, UserID) (bool, error) {
		calls++
		return true, nil
	}))
	if err != nil {
		t.Fatalf("NewAuthorizer() error = %v", err)
	}
	workspaceID := WorkspaceID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290341"))
	userID := UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290342"))
	tests := []struct {
		operation   Operation
		workspaceID WorkspaceID
		userID      UserID
	}{
		{operation: "replace", workspaceID: workspaceID, userID: userID},
		{operation: OperationList, userID: userID},
		{operation: OperationList, workspaceID: workspaceID},
	}
	for _, test := range tests {
		if _, scopeErr := authorizer.ScopeFor(
			context.Background(),
			test.operation,
			test.workspaceID,
			test.userID,
		); !errors.Is(scopeErr, ErrInvalidScope) {
			t.Fatalf("ScopeFor(%s) error = %v, want invalid scope", test.operation, scopeErr)
		}
	}
	if calls != 0 {
		t.Fatalf("membership calls = %d, want 0", calls)
	}
}

func TestScopeForPreservesMembershipReadFailure(t *testing.T) {
	databaseError := errors.New("database unavailable")
	authorizer, err := NewAuthorizer(membershipReaderFunc(func(context.Context, WorkspaceID, UserID) (bool, error) {
		return false, databaseError
	}))
	if err != nil {
		t.Fatalf("NewAuthorizer() error = %v", err)
	}
	workspaceID := WorkspaceID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290351"))
	userID := UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290352"))

	_, scopeErr := authorizer.ScopeFor(context.Background(), OperationList, workspaceID, userID)
	if !errors.Is(scopeErr, databaseError) {
		t.Fatalf("ScopeFor() error = %v, want database failure", scopeErr)
	}
}

func TestConstructorsRequireDependencies(t *testing.T) {
	if _, err := NewAuthorizer(nil); err == nil {
		t.Fatal("NewAuthorizer() error = nil, want missing reader error")
	}
	if _, err := NewPostgresMembershipReader(nil); err == nil {
		t.Fatal("NewPostgresMembershipReader() error = nil, want missing database error")
	}
}
