package analyticviews

import (
	"context"
	"errors"
	"strings"
	"testing"

	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type analyticRow func(...any) error

func (row analyticRow) Scan(destinations ...any) error { return row(destinations...) }

type analyticDB struct {
	query string
	args  []any
	row   pgx.Row
	tag   pgconn.CommandTag
	err   error
}

func (database *analyticDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	database.query, database.args = query, args
	return database.row
}
func (database *analyticDB) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	database.query, database.args = query, args
	return database.tag, database.err
}

func analyticJSONRow(destinations ...any) error {
	*destinations[0].(*[]byte) = []byte(`{"id":"00000000-0000-4000-8000-000000000001"}`)
	return nil
}

func TestRepositoryScopesReadsAndWritesToActiveAdminMembers(t *testing.T) {
	database := &analyticDB{row: analyticRow(analyticJSONRow)}
	repository, err := NewRepository(database)
	if err != nil {
		t.Fatal(err)
	}
	user := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	id := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	if _, err = repository.Get(context.Background(), user, "demo", id); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"wm.role IN (15,20)", "wm.is_active=true", "a.deleted_at IS NULL", "w.slug=$2"} {
		if !strings.Contains(database.query, fragment) {
			t.Errorf("get query does not contain %q", fragment)
		}
	}
	query, queryDict := []byte(`{"state__in":["id"]}`), []byte(`{"state":["id"]}`)
	if _, err = repository.Create(context.Background(), user, "demo", "Name", "Description", query, queryDict); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(database.query, "$3::jsonb,$4::jsonb,$5") || len(database.args) != 6 {
		t.Fatalf("create query=%s args=%#v", database.query, database.args)
	}
	if _, err = repository.Update(context.Background(), user, "demo", id, nil, nil, []byte(`{}`), true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(database.query, "updated_by_id=$3") || !strings.Contains(database.query, "query=CASE WHEN $6") {
		t.Fatalf("update query=%s", database.query)
	}

	database.tag = pgconn.NewCommandTag("UPDATE 1")
	if err = repository.Delete(context.Background(), user, "demo", id); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(database.query, "deleted_at=NOW()") || !strings.Contains(database.query, "updated_by_id=$3") {
		t.Fatalf("delete query=%s", database.query)
	}
}

func TestRepositoryMapsMissingRowsAndValidatesDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("expected nil database error")
	}
	database := &analyticDB{row: analyticRow(func(...any) error { return pgx.ErrNoRows })}
	repository, _ := NewRepository(database)
	if _, err := repository.Get(context.Background(), uuid.Nil(), "demo", uuid.Nil()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get error=%v", err)
	}
	database.tag = pgconn.NewCommandTag("UPDATE 0")
	if err := repository.Delete(context.Background(), uuid.Nil(), "demo", uuid.Nil()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete error=%v", err)
	}
}
