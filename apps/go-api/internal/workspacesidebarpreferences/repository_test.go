package workspacesidebarpreferences

import (
	"context"
	"errors"
	"strings"
	"testing"

	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type sidebarRow func(...any) error

func (r sidebarRow) Scan(dest ...any) error { return r(dest...) }

type sidebarDB struct {
	queries []string
	args    [][]any
	row     pgx.Row
}

func (d *sidebarDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	d.queries = append(d.queries, query)
	d.args = append(d.args, args)
	if d.row == nil {
		return sidebarRow(func(...any) error { return errors.New("row") })
	}
	return d.row
}

func (d *sidebarDB) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	d.queries = append(d.queries, query)
	d.args = append(d.args, args)
	return pgconn.CommandTag{}, nil
}

func TestListLazilyCreatesDjangoKeysAndDefaults(t *testing.T) {
	db := &sidebarDB{row: sidebarRow(func(dest ...any) error {
		*dest[0].(*[]byte) = []byte(`{}`)
		return nil
	})}
	repository, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.List(context.Background(), uuid.Nil(), "demo"); err != nil {
		t.Fatal(err)
	}
	query := db.queries[0]
	for _, fragment := range []string{"'views'", "'active_cycles'", "'your_work'", "'stickies'", "row_number() OVER (ORDER BY m.idx)", "65535", "10000", "'drafts','your_work','stickies'", "json_object_agg"} {
		if !strings.Contains(query, fragment) {
			t.Errorf("list query missing %q", fragment)
		}
	}
	if !strings.Contains(query, "created_at,updated_at") || !strings.Contains(query, "NOW(),NOW()") {
		t.Error("list insert must populate BaseModel timestamps")
	}
	if strings.Contains(query, "created_by") {
		t.Error("bulk-created Django preferences do not set created_by")
	}
}

func TestAuthorizedRequiresActiveAdminMemberOrGuest(t *testing.T) {
	db := &sidebarDB{row: sidebarRow(func(dest ...any) error {
		*dest[0].(*bool) = true
		return nil
	})}
	repository, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := repository.Authorized(context.Background(), uuid.Nil(), "demo"); err != nil || !ok {
		t.Fatalf("Authorized() = %v, %v", ok, err)
	}
	for _, fragment := range []string{"wm.role IN (5,15,20)", "wm.is_active=true", "wm.deleted_at IS NULL", "w.deleted_at IS NULL"} {
		if !strings.Contains(db.queries[0], fragment) {
			t.Errorf("authorization query missing %q", fragment)
		}
	}
}

func TestPatchPreservesOmittedFieldsAndIgnoresUnknownKeys(t *testing.T) {
	db := &sidebarDB{row: sidebarRow(func(dest ...any) error {
		*dest[0].(*[]byte) = []byte(`{"message":"Successfully updated"}`)
		return nil
	})}
	repository, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	pinned := true
	if _, err := repository.Patch(context.Background(), uuid.Nil(), "demo", []Patch{{Key: "drafts", IsPinned: &pinned}, {Key: "unknown"}}); err != nil {
		t.Fatal(err)
	}

	if _, err := repository.Patch(context.Background(), uuid.Nil(), "demo", []Patch{{IsPinned: &pinned}}); err != nil {
		t.Fatal(err)
	}
	got, ok := db.args[1][2].([]byte)
	if !ok || !strings.Contains(string(got), `"key":""`) {
		t.Fatalf("missing key should decode as an empty key for SQL filtering, got %s", got)
	}
	query := db.queries[0]
	for _, fragment := range []string{"jsonb_array_elements", "item ? 'is_pinned'", "item ? 'sort_order'", "updates.key=p.key", "p.user_id=$1", "p.deleted_at IS NULL"} {
		if !strings.Contains(query, fragment) {
			t.Errorf("patch query missing %q", fragment)
		}
	}
}

func TestNewRepositoryRejectsNilDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) returned nil error")
	}
}
