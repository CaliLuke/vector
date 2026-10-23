package workspacememberme

import (
	"context"
	"errors"
	"strings"
	"testing"

	"uuid"

	"github.com/jackc/pgx/v5"
)

type memberMeRow func(...any) error

func (row memberMeRow) Scan(destinations ...any) error { return row(destinations...) }

type memberMeDB struct {
	query string
	args  []any
	row   pgx.Row
}

func (database *memberMeDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	database.query, database.args = query, args
	return database.row
}

func TestGetBuildsActiveMemberShapeAndDraftCount(t *testing.T) {
	database := &memberMeDB{row: memberMeRow(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(`{"member":"00000000-0000-4000-8000-000000000001"}`)
		return nil
	})}
	repository, err := NewRepository(database)
	if err != nil {
		t.Fatal(err)
	}
	user := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	result, err := repository.Get(context.Background(), user, "demo")
	if err != nil || string(result) == "" {
		t.Fatalf("result=%s err=%v", result, err)
	}
	for _, fragment := range []string{"'company_role', CASE WHEN wm.id IS NULL THEN '' ELSE wm.company_role END", "wm.is_active = true", "wm.deleted_at IS NULL", "draft_issues", "'created_at'", "'updated_at'"} {
		if !strings.Contains(database.query, fragment) {
			t.Errorf("query does not contain %q", fragment)
		}
	}
	if strings.Contains(database.query, "COALESCE(wm.company_role") {
		t.Error("query must preserve a null company_role for active memberships")
	}
	if len(database.args) != 2 || database.args[0] != user || database.args[1] != "demo" {
		t.Fatalf("args = %#v", database.args)
	}
}

func TestGetReturnsDjangoNullInstanceShapeWhenWorkspaceIsMissing(t *testing.T) {
	database := &memberMeDB{row: memberMeRow(func(...any) error { return pgx.ErrNoRows })}
	repository, _ := NewRepository(database)
	result, err := repository.Get(context.Background(), uuid.Nil(), "missing")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{`"company_role":""`, `"is_active":false`, `"role":null`, `"workspace":null`} {
		if !strings.Contains(string(result), fragment) {
			t.Errorf("result %s does not contain %s", result, fragment)
		}
	}
}

func TestRepositoryValidationAndDatabaseError(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("expected nil database error")
	}
	want := errors.New("database")
	database := &memberMeDB{row: memberMeRow(func(...any) error { return want })}
	repository, _ := NewRepository(database)
	if _, err := repository.Get(context.Background(), uuid.Nil(), "demo"); !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
}
