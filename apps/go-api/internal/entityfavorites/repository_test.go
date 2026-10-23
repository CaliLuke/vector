package entityfavorites

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

func (f rowFunc) Scan(dest ...any) error { return f(dest...) }

type testDB struct {
	queries []string
	args    [][]any
	rows    []pgx.Row
	tag     pgconn.CommandTag
	err     error
}

func (d *testDB) QueryRow(_ context.Context, q string, a ...any) pgx.Row {
	d.queries = append(d.queries, q)
	d.args = append(d.args, a)
	if len(d.rows) == 0 {
		return rowFunc(func(...any) error { return errors.New("unexpected query") })
	}
	r := d.rows[0]
	d.rows = d.rows[1:]
	return r
}
func (d *testDB) Exec(_ context.Context, q string, a ...any) (pgconn.CommandTag, error) {
	d.queries = append(d.queries, q)
	d.args = append(d.args, a)
	return d.tag, d.err
}
func jsonRow(s string) pgx.Row {
	return rowFunc(func(d ...any) error { *d[0].(*[]byte) = []byte(s); return nil })
}

var testUser = uuid.MustParse("00000000-0000-4000-8000-000000000001")
var testProject = uuid.MustParse("00000000-0000-4000-8000-000000000002")
var testEntity = uuid.MustParse("00000000-0000-4000-8000-000000000003")

func TestRepositoryRequiresDB(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestEntityFavoriteListsSerializerJSONAndScopesProject(t *testing.T) {
	db := &testDB{rows: []pgx.Row{jsonRow(`[]`), jsonRow(`[{"id":"favorite"}]`)}}
	r, _ := NewRepository(db)
	if body, err := r.ListProjects(context.Background(), testUser, "demo"); err != nil || string(body) != `[]` {
		t.Fatalf("project list=%s %v", body, err)
	}
	if body, err := r.ListCycles(context.Background(), testUser, "demo", testProject); err != nil || string(body) != `[{"id":"favorite"}]` {
		t.Fatalf("cycle list=%s %v", body, err)
	}
	joined := strings.Join(db.queries, "\n")
	for _, want := range []string{"json_build_object", "entity_data", "entity_type=$3", "f.project_id=$4", "deleted_at IS NULL"} {
		if !strings.Contains(joined, want) {
			t.Errorf("query missing %q", want)
		}
	}
}

func TestEntityFavoriteWritesReturn204SemanticsAndMissing(t *testing.T) {
	db := &testDB{tag: pgconn.NewCommandTag("DELETE 1")}
	r, _ := NewRepository(db)
	if err := r.CreateCycle(context.Background(), testUser, "demo", testProject, testEntity); err != nil {
		t.Fatal(err)
	}
	if err := r.DeleteCycle(context.Background(), testUser, "demo", testProject, testEntity); err != nil {
		t.Fatal(err)
	}
	db.tag = pgconn.NewCommandTag("DELETE 0")
	if err := r.DeletePage(context.Background(), testUser, "demo", testProject, testEntity); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing delete=%v", err)
	}
	if len(db.args[0]) != 6 || !strings.Contains(db.queries[0], "MAX(previous.sequence)") || !strings.Contains(db.queries[0], "created_at,updated_at") {
		t.Fatalf("create query=%s args=%#v", db.queries[0], db.args[0])
	}
}

func TestEntityFavoriteDuplicate(t *testing.T) {
	db := &testDB{err: &pgconn.PgError{Code: "23505"}}
	r, _ := NewRepository(db)
	if err := r.CreateView(context.Background(), testUser, "demo", testProject, testEntity); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate=%v", err)
	}
}
