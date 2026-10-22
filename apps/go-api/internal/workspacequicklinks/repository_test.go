package workspacequicklinks

import (
	"context"
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type quickRow func(...any) error

func (r quickRow) Scan(dest ...any) error { return r(dest...) }

type quickDB struct {
	queries []string
	rows    []pgx.Row
	tag     pgconn.CommandTag
}

func (d *quickDB) QueryRow(_ context.Context, q string, _ ...any) pgx.Row {
	d.queries = append(d.queries, q)
	if len(d.rows) == 0 {
		return quickRow(func(...any) error { return errors.New("unexpected query") })
	}
	r := d.rows[0]
	d.rows = d.rows[1:]
	return r
}
func (d *quickDB) Exec(_ context.Context, q string, _ ...any) (pgconn.CommandTag, error) {
	d.queries = append(d.queries, q)
	return d.tag, nil
}
func quickJSONRow(v string) pgx.Row {
	return quickRow(func(dest ...any) error { *dest[0].(*[]byte) = []byte(v); return nil })
}
func TestWorkspaceQuickLinksRepositoryScopeDefaultsAndDelete(t *testing.T) {
	db := &quickDB{rows: []pgx.Row{quickRow(func(dest ...any) error { *dest[0].(*bool) = true; return nil }), quickJSONRow(`[]`), quickJSONRow(`{"id":"x"}`), quickJSONRow(`{"id":"x"}`)}, tag: pgconn.NewCommandTag("UPDATE 1")}
	r, e := NewRepository(db)
	if e != nil {
		t.Fatal(e)
	}
	u := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	id := uuid.MustParse("00000000-0000-4000-8000-000000000011")
	if ok, e := r.Authorized(context.Background(), u, "demo"); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if _, e = r.List(context.Background(), u, "demo"); e != nil {
		t.Fatal(e)
	}
	if _, e = r.Update(context.Background(), u, "demo", id, true, nil, nil, false, nil); e != nil {
		t.Fatal(e)
	}
	if e = r.Delete(context.Background(), u, "demo", id); e != nil {
		t.Fatal(e)
	}
	joined := strings.Join(db.queries, "\n")
	for _, want := range []string{"owner_id", "workspace_id", "deleted_at IS NULL", "updated_by_id", "metadata"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %s", want, joined)
		}
	}
}
