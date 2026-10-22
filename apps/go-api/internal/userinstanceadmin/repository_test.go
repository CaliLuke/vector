package userinstanceadmin

import (
	"context"
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
)

type rowStub struct {
	value bool
	err   error
}

func (row rowStub) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	*dest[0].(*bool) = row.value
	return nil
}

type databaseStub struct {
	query string
	args  []any
	row   rowStub
}

func (database *databaseStub) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	database.query, database.args = query, args
	return database.row
}

func TestRepositoryIsInstanceAdmin(t *testing.T) {
	database := &databaseStub{row: rowStub{value: true}}
	repository, err := NewRepository(database)
	if err != nil {
		t.Fatal(err)
	}
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	isAdmin, err := repository.IsInstanceAdmin(context.Background(), userID)
	if err != nil || !isAdmin {
		t.Fatalf("isAdmin = %v, error = %v", isAdmin, err)
	}
	if database.args[0] != userID || !strings.Contains(database.query, "ORDER BY created_at DESC") || strings.Contains(database.query, "role >=") || strings.Contains(database.query, "is_verified") {
		t.Fatalf("query does not preserve membership contract: %s", database.query)
	}
}

func TestRepositoryIsInstanceAdminError(t *testing.T) {
	repository, err := NewRepository(&databaseStub{row: rowStub{err: errors.New("boom")}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.IsInstanceAdmin(context.Background(), uuid.MustParse("00000000-0000-0000-0000-000000000001")); err == nil {
		t.Fatal("IsInstanceAdmin returned nil error")
	}
}
