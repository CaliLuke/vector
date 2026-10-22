package userapitokens

import (
	"context"
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type tokenRow func(...any) error

func (r tokenRow) Scan(dest ...any) error { return r(dest...) }

type tokenDB struct {
	queries []string
	args    [][]any
	rows    []pgx.Row
	tag     pgconn.CommandTag
	err     error
}

func (d *tokenDB) QueryRow(_ context.Context, q string, a ...any) pgx.Row {
	d.queries = append(d.queries, q)
	d.args = append(d.args, a)
	if len(d.rows) == 0 {
		return tokenRow(func(...any) error { return errors.New("unexpected query") })
	}
	r := d.rows[0]
	d.rows = d.rows[1:]
	return r
}
func (d *tokenDB) Exec(_ context.Context, q string, a ...any) (pgconn.CommandTag, error) {
	d.queries = append(d.queries, q)
	d.args = append(d.args, a)
	return d.tag, d.err
}
func tokenJSONRow(body string) pgx.Row {
	return tokenRow(func(dest ...any) error { *dest[0].(*[]byte) = []byte(body); return nil })
}
func TestRepositoryFiltersSoftDeletedAndRedactsReadToken(t *testing.T) {
	db := &tokenDB{rows: []pgx.Row{tokenJSONRow("[]"), tokenJSONRow(`{"id":"x"}`)}}
	r, _ := NewRepository(db)
	u := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	if _, e := r.List(context.Background(), u); e != nil {
		t.Fatal(e)
	}
	if _, e := r.Get(context.Background(), u, u); e != nil {
		t.Fatal(e)
	}
	joined := strings.Join(db.queries, "\n")
	if !strings.Contains(joined, "deleted_at IS NULL") || strings.Contains(joined, "'token', token") {
		t.Fatalf("queries=%s", joined)
	}
}
func TestRepositoryCreatePatchAndSoftDeleteAudit(t *testing.T) {
	u := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	id := uuid.MustParse("00000000-0000-4000-8000-000000000011")
	db := &tokenDB{rows: []pgx.Row{tokenJSONRow(`{"id":"x"}`), tokenJSONRow(`{"id":"x"}`)}, tag: pgconn.NewCommandTag("UPDATE 1")}
	r, _ := NewRepository(db)
	expiry := "2026-09-01T00:00:00Z"
	if _, e := r.Create(context.Background(), u, "label", "desc", "plane_api_x", &expiry); e != nil {
		t.Fatal(e)
	}
	label := "new"
	if _, e := r.Patch(context.Background(), u, id, &label, nil); e != nil {
		t.Fatal(e)
	}
	if e := r.Delete(context.Background(), u, id); e != nil {
		t.Fatal(e)
	}
	joined := strings.Join(db.queries, "\n")
	for _, fragment := range []string{"created_by_id", "expired_at", "updated_by_id=$2", "deleted_at=NOW()", "UPDATE api_tokens"} {
		if !strings.Contains(joined, fragment) {
			t.Fatalf("missing %q in %s", fragment, joined)
		}
	}
}
func TestRepositoryMissingDelete(t *testing.T) {
	db := &tokenDB{tag: pgconn.NewCommandTag("UPDATE 0")}
	r, _ := NewRepository(db)
	if e := r.Delete(context.Background(), uuid.Nil(), uuid.Nil()); !errors.Is(e, ErrNotFound) {
		t.Fatalf("error=%v", e)
	}
}
