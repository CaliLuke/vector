package workspacethemes

import (
	"context"
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type themeRow struct {
	value any
	err   error
}

func (r themeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	switch d := dest[0].(type) {
	case *[]byte:
		*d = []byte(r.value.(string))
	case *bool:
		*d = r.value.(bool)
	}
	return nil
}

type themeDB struct {
	query   string
	args    []any
	row     pgx.Row
	tag     pgconn.CommandTag
	execErr error
}

func (d *themeDB) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	d.query = q
	d.args = args
	return d.row
}
func (d *themeDB) Exec(_ context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	d.query = q
	d.args = args
	return d.tag, d.execErr
}

func TestRepositorySQLScopingAndJSON(t *testing.T) {
	d := &themeDB{row: themeRow{value: `[{"id":"x"}]`}}
	r, _ := NewRepository(d)
	id := uuid.New()
	user := uuid.New()
	if _, err := r.List(context.Background(), user, "acme"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"json_agg(theme ORDER BY created_at DESC)", "t.deleted_at IS NULL", "wm.member_id=$2", "w.slug=$1"} {
		if !strings.Contains(d.query, want) {
			t.Errorf("list query missing %q", want)
		}
	}
	d.row = themeRow{value: `{"id":"x"}`}
	if _, err := r.Update(context.Background(), user, "acme", id, nil, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Split(d.query, "FROM workspaces")[0], "updated_by_id=$3") {
		t.Error("update must record the authenticated audit user")
	}
	if !strings.Contains(d.query, "COALESCE($4,name)") || !strings.Contains(d.query, "t.workspace_id=w.id") {
		t.Error("update lost partial/workspace scoping")
	}
}
func TestRepositoryDeleteRowsAffected(t *testing.T) {
	d := &themeDB{tag: pgconn.NewCommandTag("UPDATE 0")}
	r, _ := NewRepository(d)
	err := r.Delete(context.Background(), uuid.New(), "acme", uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(d.query, "t.deleted_at IS NULL") || !strings.Contains(d.query, "wm.member_id=$1") {
		t.Fatal("delete query missing soft-delete/member scope")
	}
}
