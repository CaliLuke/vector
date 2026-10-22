package useraccounts

import (
	"context"
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type rowFunc func(...any) error

func (function rowFunc) Scan(dest ...any) error { return function(dest...) }

type databaseStub struct {
	query string
	args  []any
	row   pgx.Row
	tag   pgconn.CommandTag
	err   error
}

func (stub *databaseStub) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	stub.query = query
	stub.args = args
	return stub.row
}

func (stub *databaseStub) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	stub.query = query
	stub.args = args
	return stub.tag, stub.err
}

func TestRepositoryListPreservesDjangoShapeOrderingAndOwnership(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	body := `[{"id":"00000000-0000-4000-8000-000000000021"}]`
	stub := &databaseStub{row: rowFunc(func(dest ...any) error {
		*dest[0].(*[]byte) = []byte(body)
		return nil
	})}
	repository, err := NewRepository(stub)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repository.List(context.Background(), userID)
	if err != nil || string(got) != body {
		t.Fatalf("List() = %s, %v", got, err)
	}
	for _, fragment := range []string{"WHERE user_id = $1", "ORDER BY created_at DESC", "'access_token'", "'refresh_token'", "'metadata'"} {
		if !strings.Contains(stub.query, fragment) {
			t.Fatalf("list query omits %q:\n%s", fragment, stub.query)
		}
	}
	if len(stub.args) != 1 || stub.args[0] != userID {
		t.Fatalf("list args = %#v", stub.args)
	}
}

func TestRepositoryGetAndDeleteAreOwnerScoped(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	accountID := uuid.MustParse("00000000-0000-4000-8000-000000000021")
	stub := &databaseStub{row: rowFunc(func(dest ...any) error {
		*dest[0].(*[]byte) = []byte(`{"id":"00000000-0000-4000-8000-000000000021"}`)
		return nil
	})}
	repository, _ := NewRepository(stub)
	if _, err := repository.Get(context.Background(), userID, accountID); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stub.query, "WHERE id = $1 AND user_id = $2") || len(stub.args) != 2 || stub.args[0] != accountID || stub.args[1] != userID {
		t.Fatalf("get query/args = %s %#v", stub.query, stub.args)
	}

	stub.tag = pgconn.NewCommandTag("DELETE 1")
	if err := repository.Delete(context.Background(), userID, accountID); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stub.query, "DELETE FROM accounts WHERE id = $1 AND user_id = $2") || stub.args[0] != accountID || stub.args[1] != userID {
		t.Fatalf("delete query/args = %s %#v", stub.query, stub.args)
	}
}

func TestRepositoryMapsMissingRows(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	accountID := uuid.MustParse("00000000-0000-4000-8000-000000000021")
	stub := &databaseStub{row: rowFunc(func(...any) error { return pgx.ErrNoRows })}
	repository, _ := NewRepository(stub)
	if _, err := repository.Get(context.Background(), userID, accountID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v", err)
	}
	stub.tag = pgconn.NewCommandTag("DELETE 0")
	if err := repository.Delete(context.Background(), userID, accountID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) error = nil")
	}
}
