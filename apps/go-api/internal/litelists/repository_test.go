package litelists

import (
	"context"
	"errors"
	"strings"
	"testing"

	"uuid"

	"github.com/jackc/pgx/v5"
)

type rowFunc func(...any) error

func (f rowFunc) Scan(dest ...any) error { return f(dest...) }

type dbStub struct {
	query string
	args  []any
	row   pgx.Row
}

func (d *dbStub) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	d.query, d.args = q, args
	return d.row
}

func pageRow(body string, count int) pgx.Row {
	return rowFunc(func(dest ...any) error { *dest[0].(*[]byte) = []byte(body); *dest[1].(*int) = count; return nil })
}

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) returned nil error")
	}
}

func TestListProjectsScopesVisibilityAndReturnsCount(t *testing.T) {
	db := &dbStub{row: pageRow(`[{"id":"x"}]`, 2)}
	r, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	p, err := r.ListProjects(context.Background(), uuid.Nil(), "demo", ListOptions{Limit: 1})
	if err != nil || p.Count != 2 || string(p.Results) != `[{"id":"x"}]` {
		t.Fatalf("page=%+v err=%v", p, err)
	}
	for _, want := range []string{"project_members", "network=2", "deleted_at IS NULL", "LIMIT 1", "p.archived_at IS NULL"} {
		if !strings.Contains(db.query, want) {
			t.Errorf("query missing %q", want)
		}
	}
}

func TestListMembersUsesSerializerFieldsAndPropagatesDatabaseErrors(t *testing.T) {
	db := &dbStub{row: rowFunc(func(...any) error { return errors.New("boom") })}
	r, _ := NewRepository(db)
	if _, err := r.ListWorkspaceMembers(context.Background(), "demo", ListOptions{}); err == nil {
		t.Fatal("expected database error")
	}
	if !strings.Contains(db.query, "display_name") || !strings.Contains(db.query, "is_bot") || !strings.Contains(db.query, "workspace_members") {
		t.Fatalf("member serializer/scope missing: %s", db.query)
	}
}

func TestListCyclesAndModulesExcludeArchived(t *testing.T) {
	db := &dbStub{row: pageRow(`[]`, 0)}
	r, _ := NewRepository(db)
	if _, err := r.ListCycles(context.Background(), "demo", uuid.Nil(), ListOptions{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(db.query, "c.archived_at IS NULL") {
		t.Fatal("cycles did not exclude archived rows")
	}
	if _, err := r.ListModules(context.Background(), "demo", uuid.Nil(), ListOptions{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(db.query, "m.archived_at IS NULL") || !strings.Contains(db.query, "module_members") {
		t.Fatal("modules query missing archive/member scope")
	}
}
