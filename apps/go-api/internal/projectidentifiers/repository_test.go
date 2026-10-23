package projectidentifiers

import (
	"context"
	"errors"
	"strings"
	"testing"

	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type identifierRow func(...any) error

func (row identifierRow) Scan(dest ...any) error { return row(dest...) }

type identifierDB struct {
	queries []string
	args    [][]any
	rows    []pgx.Row
	execs   int
}

func (db *identifierDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	db.queries = append(db.queries, query)
	db.args = append(db.args, args)
	if len(db.rows) == 0 {
		return identifierRow(func(...any) error { return errors.New("unexpected query") })
	}
	row := db.rows[0]
	db.rows = db.rows[1:]
	return row
}

func (db *identifierDB) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	db.queries = append(db.queries, query)
	db.args = append(db.args, args)
	db.execs++
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func identifierJSON(value string) pgx.Row {
	return identifierRow(func(dest ...any) error {
		*dest[0].(*[]byte) = []byte(value)
		return nil
	})
}

func identifierBool(value bool) pgx.Row {
	return identifierRow(func(dest ...any) error {
		*dest[0].(*bool) = value
		return nil
	})
}

func TestRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("expected nil database error")
	}
}

func TestGetNormalizesNameAndScopesWorkspaceMember(t *testing.T) {
	db := &identifierDB{rows: []pgx.Row{identifierJSON(`{"exists":1,"identifiers":[]}`)}}
	r, _ := NewRepository(db)
	user := uuid.New()
	body, err := r.Get(context.Background(), user, "acme", "  ab  ")
	if err != nil || string(body) != `{"exists":1,"identifiers":[]}` {
		t.Fatalf("Get() = %s, %v", body, err)
	}
	if got := db.args[0][0]; got != "AB" {
		t.Fatalf("normalized name = %#v, want AB", got)
	}
	for _, fragment := range []string{"project_identifiers", "wm.role IN (15,20)", "pi.deleted_at IS NULL", "w.slug=$2"} {
		if !strings.Contains(db.queries[0], fragment) {
			t.Errorf("query missing %q", fragment)
		}
	}
}

func TestNormalizeNameUsesFullUnicodeUppercase(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"  straße  ", "STRASSE"},
		{"ﬃ", "FFI"},
	} {
		got, err := normalizeName(test.input)
		if err != nil || got != test.want {
			t.Errorf("normalizeName(%q) = %q, %v; want %s", test.input, got, err, test.want)
		}
	}
}

func TestNameRequiredBeforeDatabaseAccess(t *testing.T) {
	db := &identifierDB{}
	r, _ := NewRepository(db)
	if _, err := r.Get(context.Background(), uuid.New(), "acme", " \t"); !errors.Is(err, ErrNameRequired) {
		t.Fatalf("Get() error = %v", err)
	}
	if len(db.queries) != 0 {
		t.Fatal("empty name queried the database")
	}
}

func TestDeleteBlocksExistingProjectAndSoftDeletesUnusedName(t *testing.T) {
	user := uuid.New()
	db := &identifierDB{rows: []pgx.Row{identifierBool(true)}}
	r, _ := NewRepository(db)
	if err := r.Delete(context.Background(), user, "acme", " ab "); !errors.Is(err, ErrProjectExists) {
		t.Fatalf("Delete(existing) error = %v", err)
	}
	if db.execs != 0 {
		t.Fatal("existing project should not be deleted")
	}

	db = &identifierDB{rows: []pgx.Row{identifierBool(false)}}
	r, _ = NewRepository(db)
	if err := r.Delete(context.Background(), user, "acme", " ab "); err != nil {
		t.Fatal(err)
	}
	if db.execs != 1 || db.args[1][0] != "AB" {
		t.Fatalf("delete args/execution = %#v, %d", db.args, db.execs)
	}
	for _, fragment := range []string{"SET deleted_at=NOW()", "pi.workspace_id=w.id", "wm.member_id=$3", "pi.deleted_at IS NULL"} {
		if !strings.Contains(db.queries[1], fragment) {
			t.Errorf("delete query missing %q", fragment)
		}
	}
}
