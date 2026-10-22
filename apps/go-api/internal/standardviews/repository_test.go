package standardviews

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"uuid"
)

type viewRow func(...any) error

func (r viewRow) Scan(dest ...any) error { return r(dest...) }

type viewDB struct {
	queries []string
	rows    []pgx.Row
	tag     pgconn.CommandTag
}

func (d *viewDB) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	d.queries = append(d.queries, query)
	if len(d.rows) == 0 {
		return viewRow(func(...any) error { return errors.New("unexpected query") })
	}
	r := d.rows[0]
	d.rows = d.rows[1:]
	return r
}
func (d *viewDB) Exec(_ context.Context, query string, _ ...any) (pgconn.CommandTag, error) {
	d.queries = append(d.queries, query)
	return d.tag, nil
}

func jsonViewRow(value string) pgx.Row {
	return viewRow(func(dest ...any) error {
		*dest[0].(*[]byte) = []byte(value)
		return nil
	})
}

func boolRow(value bool) pgx.Row {
	return viewRow(func(dest ...any) error {
		*dest[0].(*bool) = value
		return nil
	})
}

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) returned nil error")
	}
}

func TestProjectQueriesPreserveTenantMembershipVisibilityAndFavoriteRules(t *testing.T) {
	db := &viewDB{rows: []pgx.Row{boolRow(true), jsonViewRow(`[]`), jsonViewRow(`{"id":"view"}`)}}
	r, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	user := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	project := uuid.MustParse("00000000-0000-4000-8000-000000000011")
	view := uuid.MustParse("00000000-0000-4000-8000-000000000012")
	if ok, err := r.AuthorizedProject(context.Background(), user, "demo", project); err != nil || !ok {
		t.Fatalf("authorization = %v, %v", ok, err)
	}
	if _, err := r.ListProject(context.Background(), user, "demo", project); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetProject(context.Background(), user, "demo", project, view); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(db.queries, "\n")
	for _, fragment := range []string{"project_members", "pm.is_active=true", "p.archived_at IS NULL", "v.access=1", "guest_view_all_features", "entity_type='view'"} {
		if !strings.Contains(joined, fragment) {
			t.Errorf("query missing %q", fragment)
		}
	}
}

func TestDeleteProjectSoftDeletesAndCleansSideEffects(t *testing.T) {
	project := uuid.MustParse("00000000-0000-4000-8000-000000000011")
	db := &viewDB{rows: []pgx.Row{viewRow(func(dest ...any) error { *dest[0].(**uuid.UUID) = &project; return nil })}, tag: pgconn.NewCommandTag("DELETE 1")}
	r, _ := NewRepository(db)
	user := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	id := uuid.MustParse("00000000-0000-4000-8000-000000000012")
	if err := r.DeleteProject(context.Background(), user, "demo", project, id); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(db.queries, "\n")
	for _, fragment := range []string{"deleted_at=NOW()", "wm.role=20", "UPDATE user_favorites SET deleted_at=NOW()", "DELETE FROM user_recent_visits"} {
		if !strings.Contains(joined, fragment) {
			t.Errorf("delete query missing %q", fragment)
		}
	}
}
