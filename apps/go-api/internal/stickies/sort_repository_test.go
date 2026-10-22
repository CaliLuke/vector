package stickies

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type transactionStarterFunc func(context.Context) (pgx.Tx, error)

func (function transactionStarterFunc) Begin(ctx context.Context) (pgx.Tx, error) {
	return function(ctx)
}

func TestNextSortOrderMatchesDjangoSequence(t *testing.T) {
	tests := []struct {
		name    string
		largest pgtype.Float8
		want    float64
	}{
		{name: "first Sticky", want: 65535},
		{name: "next Sticky", largest: pgtype.Float8{Float64: 65535, Valid: true}, want: 75535},
		{name: "existing custom order", largest: pgtype.Float8{Float64: 20000, Valid: true}, want: 30000},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := nextSortOrder(test.largest); got != test.want {
				t.Fatalf("nextSortOrder() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSortRepositoryRejectsInvalidScopeBeforeTransaction(t *testing.T) {
	calls := 0
	repository, err := NewSortRepository(transactionStarterFunc(func(context.Context) (pgx.Tx, error) {
		calls++
		return nil, errors.New("unexpected transaction")
	}))
	if err != nil {
		t.Fatalf("NewSortRepository() error = %v", err)
	}

	validSticky := StickyID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290401"))
	validWorkspace := WorkspaceID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290402"))
	validUser := UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290403"))
	validScope := Scope{
		Operation:     OperationCreate,
		WorkspaceID:   validWorkspace,
		UserID:        validUser,
		Ownership:     AssignOwner,
		AssignCreator: true,
	}
	tests := []struct {
		sticky StickyID
		scope  Scope
	}{
		{scope: validScope},
		{sticky: validSticky},
		{sticky: validSticky, scope: Scope{Operation: OperationList, WorkspaceID: validWorkspace, UserID: validUser, Ownership: RequireOwner}},
		{sticky: validSticky, scope: Scope{Operation: OperationCreate, WorkspaceID: validWorkspace, UserID: validUser, Ownership: AssignOwner}},
	}
	for _, test := range tests {
		_, createErr := repository.CreateWithAllocatedSort(
			context.Background(),
			test.sticky,
			test.scope,
		)
		if !errors.Is(createErr, ErrInvalidScope) {
			t.Fatalf("CreateWithAllocatedSort() error = %v, want invalid scope", createErr)
		}
	}
	if calls != 0 {
		t.Fatalf("Begin() calls = %d, want 0", calls)
	}
}

func TestNewSortRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewSortRepository(nil); err == nil {
		t.Fatal("NewSortRepository() error = nil, want missing database error")
	}
}
