package usernotificationpreferences

import (
	"context"
	"encoding/json"
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
	queries []string
	args    [][]any
	rows    []pgx.Row
	tag     pgconn.CommandTag
	err     error
}

func (stub *databaseStub) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	stub.queries = append(stub.queries, query)
	stub.args = append(stub.args, args)
	if len(stub.rows) == 0 {
		return rowFunc(func(...any) error { return errors.New("unexpected query") })
	}
	row := stub.rows[0]
	stub.rows = stub.rows[1:]
	return row
}

func (stub *databaseStub) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	stub.queries = append(stub.queries, query)
	stub.args = append(stub.args, args)
	return stub.tag, stub.err
}

func activeRow(id uuid.UUID, count int64) pgx.Row {
	return rowFunc(func(dest ...any) error {
		*dest[0].(*string) = id.String()
		*dest[1].(*int64) = count
		return nil
	})
}

func jsonRow(body string) pgx.Row {
	return rowFunc(func(dest ...any) error {
		*dest[0].(*[]byte) = []byte(body)
		return nil
	})
}

func boolRow(value bool) pgx.Row {
	return rowFunc(func(dest ...any) error {
		*dest[0].(*bool) = value
		return nil
	})
}

func TestRepositoryGetPreservesSerializerShapeAndCardinality(t *testing.T) {
	preferenceID := uuid.MustParse("00000000-0000-4000-8000-000000000011")
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	body := `{"id":"00000000-0000-4000-8000-000000000011","user":"00000000-0000-4000-8000-000000000010"}`
	stub := &databaseStub{rows: []pgx.Row{activeRow(preferenceID, 1), jsonRow(body)}}
	repository, err := NewRepository(stub)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repository.Get(context.Background(), userID)
	if err != nil || !json.Valid(got) || string(got) != body {
		t.Fatalf("Get() = %s, %v", got, err)
	}
	joined := strings.Join(stub.queries, "\n")
	for _, fragment := range []string{"deleted_at IS NULL", "'created_by', p.created_by_id", "'user', p.user_id"} {
		if !strings.Contains(joined, fragment) {
			t.Fatalf("queries omit %q:\n%s", fragment, joined)
		}
	}

	duplicate, _ := NewRepository(&databaseStub{rows: []pgx.Row{activeRow(preferenceID, 2)}})
	if _, err := duplicate.Get(context.Background(), userID); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate Get() error = %v", err)
	}
}

func TestRepositoryPatchAlwaysUpdatesAuditFieldsAndUsesStableID(t *testing.T) {
	preferenceID := uuid.MustParse("00000000-0000-4000-8000-000000000011")
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	stub := &databaseStub{rows: []pgx.Row{activeRow(preferenceID, 1)}, tag: pgconn.NewCommandTag("UPDATE 1")}
	repository, _ := NewRepository(stub)
	got, err := repository.Patch(context.Background(), userID, map[string]any{})
	if err != nil || got != preferenceID {
		t.Fatalf("Patch() = %s, %v", got, err)
	}
	query := stub.queries[len(stub.queries)-1]
	if !strings.Contains(query, "updated_by = $2") || !strings.Contains(query, "updated_at = NOW()") || !strings.Contains(query, "WHERE id = $1") {
		t.Fatalf("empty PATCH query does not preserve Django save behavior: %s", query)
	}
	if stub.args[len(stub.args)-1][0] != preferenceID || stub.args[len(stub.args)-1][1] != userID {
		t.Fatalf("PATCH args = %#v", stub.args[len(stub.args)-1])
	}
}

func TestRepositoryPatchValidatesRelationsBeforeUpdate(t *testing.T) {
	preferenceID := uuid.MustParse("00000000-0000-4000-8000-000000000011")
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	workspaceID := uuid.MustParse("00000000-0000-4000-8000-000000000020")
	stub := &databaseStub{rows: []pgx.Row{activeRow(preferenceID, 1), boolRow(false)}}
	repository, _ := NewRepository(stub)
	_, err := repository.Patch(context.Background(), userID, map[string]any{"workspace": workspaceID.String()})
	var fieldError *FieldError
	if !errors.As(err, &fieldError) || fieldError.Field != "workspace" || !strings.Contains(fieldError.Message, "object does not exist") {
		t.Fatalf("Patch() error = %#v", err)
	}
	if len(stub.queries) != 2 || !strings.Contains(stub.queries[1], "FROM workspaces") {
		t.Fatalf("relation queries = %#v", stub.queries)
	}
}

func TestRepositoryGetByIDCanReturnJustSoftDeletedUpdate(t *testing.T) {
	preferenceID := uuid.MustParse("00000000-0000-4000-8000-000000000011")
	stub := &databaseStub{rows: []pgx.Row{jsonRow(`{"deleted_at":"2026-08-26T21:00:00Z"}`)}}
	repository, _ := NewRepository(stub)
	if _, err := repository.GetByID(context.Background(), preferenceID); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stub.queries[0], "deleted_at IS NULL") {
		t.Fatalf("GetByID incorrectly hides the row just updated: %s", stub.queries[0])
	}
}

func TestRepositoryRejectsNilDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) error = nil")
	}
}
