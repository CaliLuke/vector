package standardviews

import (
	"context"
	"errors"
	"testing"

	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type serviceDB struct{ rows []pgx.Row }

func (d *serviceDB) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	if len(d.rows) == 0 {
		return serviceRow(func(...any) error { return errors.New("unexpected query") })
	}
	r := d.rows[0]
	d.rows = d.rows[1:]
	return r
}
func (d *serviceDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("DELETE 1"), nil
}

type serviceRow func(...any) error

func (r serviceRow) Scan(dest ...any) error { return r(dest...) }

func serviceBool(value bool) pgx.Row {
	return serviceRow(func(dest ...any) error { *dest[0].(*bool) = value; return nil })
}

func TestServiceRequiresActiveScopeBeforeListing(t *testing.T) {
	db := &serviceDB{rows: []pgx.Row{serviceBool(false)}}
	r, _ := NewRepository(db)
	s, _ := NewService(r)
	user := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	if _, err := s.ListWorkspace(context.Background(), user, "demo"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("ListWorkspace() = %v, want ErrForbidden", err)
	}
	if len(db.rows) != 0 {
		t.Fatal("repository list ran after authorization failure")
	}
}

func TestServiceValidatesCreateInput(t *testing.T) {
	db := &serviceDB{rows: []pgx.Row{serviceBool(true)}}
	r, _ := NewRepository(db)
	s, _ := NewService(r)
	user := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	if _, err := s.CreateWorkspace(context.Background(), user, "demo", ViewInput{}); err == nil {
		t.Fatal("CreateWorkspace() accepted an empty input")
	}
	if len(db.rows) != 0 {
		t.Fatal("create validation should run after scope and before persistence")
	}
}
