package workspaceslugcheck

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

type rowStub struct {
	exists bool
	err    error
}

func (row rowStub) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	*dest[0].(*bool) = row.exists
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

func TestRepositoryExists(t *testing.T) {
	database := &databaseStub{row: rowStub{exists: true}}
	repository, err := NewRepository(database)
	if err != nil {
		t.Fatal(err)
	}
	exists, err := repository.Exists(context.Background(), "CaseSensitive")
	if err != nil || !exists {
		t.Fatalf("exists = %v, error = %v", exists, err)
	}
	if database.args[0] != "CaseSensitive" {
		t.Fatalf("slug argument = %#v", database.args[0])
	}
	if !strings.Contains(database.query, "slug = $1") || !strings.Contains(database.query, "deleted_at IS NULL") {
		t.Fatalf("query does not preserve Django default-manager predicates: %s", database.query)
	}
}

func TestRepositoryExistsError(t *testing.T) {
	repository, err := NewRepository(&databaseStub{row: rowStub{err: errors.New("boom")}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Exists(context.Background(), "slug"); err == nil {
		t.Fatal("Exists() returned nil error")
	}
}

func TestRepositorySoftDeletedWorkspaceIsNotVisible(t *testing.T) {
	repository, err := NewRepository(&databaseStub{row: rowStub{exists: false}})
	if err != nil {
		t.Fatal(err)
	}
	exists, err := repository.Exists(context.Background(), "deleted-slug")
	if err != nil || exists {
		t.Fatalf("exists = %v, error = %v, want false for soft-deleted workspace", exists, err)
	}
}
