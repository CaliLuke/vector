package workspacefavorites

import (
	"context"
	"errors"
	"strings"
	"testing"

	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type favoriteRow func(...any) error

func (r favoriteRow) Scan(dest ...any) error { return r(dest...) }

type favoriteDB struct {
	queries []string
	args    [][]any
	rows    []pgx.Row
	tag     pgconn.CommandTag
	err     error
}

func (d *favoriteDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	d.queries = append(d.queries, query)
	d.args = append(d.args, args)
	if len(d.rows) == 0 {
		return favoriteRow(func(...any) error { return errors.New("unexpected query") })
	}
	row := d.rows[0]
	d.rows = d.rows[1:]
	return row
}

func (d *favoriteDB) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	d.queries = append(d.queries, query)
	d.args = append(d.args, args)
	return d.tag, d.err
}

func (d *favoriteDB) Begin(context.Context) (pgx.Tx, error) {
	return favoriteTx{db: d}, nil
}

type favoriteTx struct {
	pgx.Tx
	db *favoriteDB
}

func (tx favoriteTx) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return tx.db.QueryRow(ctx, query, args...)
}

func (tx favoriteTx) Commit(context.Context) error   { return tx.db.err }
func (tx favoriteTx) Rollback(context.Context) error { return nil }

func favoriteJSONRow(value string) pgx.Row {
	return favoriteRow(func(dest ...any) error {
		*dest[0].(*[]byte) = []byte(value)
		return nil
	})
}

func favoriteUUIDRow(value uuid.UUID) pgx.Row {
	return favoriteRow(func(dest ...any) error {
		*dest[0].(*uuid.UUID) = value
		return nil
	})
}

func favoriteNoRows() pgx.Row {
	return favoriteRow(func(...any) error { return pgx.ErrNoRows })
}

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) returned nil error")
	}
}

func TestWorkspaceFavoritesRepositoryScopesAndResponse(t *testing.T) {
	db := &favoriteDB{rows: []pgx.Row{
		favoriteRow(func(dest ...any) error { *dest[0].(*bool) = true; return nil }),
		favoriteJSONRow(`[]`),
		favoriteJSONRow(`[]`),
		favoriteJSONRow(`[]`),
		favoriteJSONRow(`{"id":"favorite"}`),
	}, tag: pgconn.NewCommandTag("DELETE 1")}
	repository, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	user := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	parent := uuid.MustParse("00000000-0000-4000-8000-000000000011")
	id := uuid.MustParse("00000000-0000-4000-8000-000000000012")

	if ok, err := repository.Authorized(context.Background(), user, "demo"); err != nil || !ok {
		t.Fatalf("Authorized() = %v, %v", ok, err)
	}
	if _, err = repository.List(context.Background(), user, "demo", false); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.List(context.Background(), user, "demo", true); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.Group(context.Background(), user, "demo", parent); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.Get(context.Background(), user, "demo", id); err != nil {
		t.Fatal(err)
	}
	if err = repository.Delete(context.Background(), user, "demo", id); err != nil {
		t.Fatal(err)
	}

	joined := strings.Join(db.queries, "\n")
	for _, want := range []string{
		"role IN (15,20)",
		"$3 OR f.parent_id IS NULL",
		"project_members",
		"scoped_member.is_active=true",
		"scoped_parent.entity_type='folder'",
		"entity_data",
		"deleted_at IS NULL",
		"DELETE FROM user_favorites",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("query missing %q in %s", want, joined)
		}
	}
	if len(db.args) < 3 || len(db.args[1]) != 3 || db.args[1][2] != false || db.args[2][2] != true {
		t.Fatalf("List calls did not preserve root-only and all modes: %#v", db.args)
	}
}

func TestWorkspaceFavoritesCreateAllocatesSequenceAndReturnsDuplicate(t *testing.T) {
	db := &favoriteDB{rows: []pgx.Row{favoriteJSONRow(`{"id":"favorite"}`)}}
	repository, _ := NewRepository(db)
	user := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	entity := uuid.MustParse("00000000-0000-4000-8000-000000000013")
	sequence := 42.0
	input := CreateInput{EntityType: "project", EntityIdentifier: &entity, IsFolder: false, Sequence: &sequence}
	body, err := repository.Create(context.Background(), user, "demo", input)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"id":"favorite"}` {
		t.Fatalf("body = %s", body)
	}
	if len(db.args) != 1 || len(db.args[0]) != 10 {
		t.Fatalf("Create args = %#v", db.args)
	}
	query := db.queries[0]
	for _, want := range []string{
		"WITH candidate",
		"existing AS",
		"NOT EXISTS (SELECT 1 FROM existing)",
		"MAX(previous.sequence)",
		"+10000",
		"COALESCE($7,65535)",
		"entity_identifier",
		"created_at,updated_at",
		"project_members",
		"parent.entity_type='folder'",
	} {
		if !strings.Contains(query, want) {
			t.Errorf("Create query missing %q", want)
		}
	}
}

func TestWorkspaceFavoritesUpdatePresenceAndNotFound(t *testing.T) {
	name := "Renamed"
	db := &favoriteDB{rows: []pgx.Row{favoriteUUIDRow(uuid.MustParse("00000000-0000-4000-8000-000000000013")), favoriteJSONRow(`{"id":"favorite"}`), favoriteNoRows()}}
	repository, _ := NewRepository(db)
	user := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	id := uuid.MustParse("00000000-0000-4000-8000-000000000012")
	body, err := repository.Update(context.Background(), user, "demo", id, UpdateInput{NamePresent: true, Name: &name, ParentPresent: true})
	if err != nil || string(body) != `{"id":"favorite"}` {
		t.Fatalf("Update() = %s, %v", body, err)
	}
	if _, err = repository.Update(context.Background(), user, "demo", id, UpdateInput{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update missing = %v, want ErrNotFound", err)
	}
	query := db.queries[1]
	if !strings.Contains(db.queries[0], "FOR UPDATE OF wm") {
		t.Fatalf("parent update did not serialize by workspace membership: %s", db.queries[0])
	}
	for _, want := range []string{"CASE WHEN $8 THEN $9", "parent_id=CASE WHEN $14 THEN $15", "updated_by_id=$1"} {
		if !strings.Contains(query, want) {
			t.Errorf("Update query missing %q", want)
		}
	}
}

func TestWorkspaceFavoritesDeleteNotFound(t *testing.T) {
	db := &favoriteDB{tag: pgconn.NewCommandTag("DELETE 0")}
	repository, _ := NewRepository(db)
	user := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	id := uuid.MustParse("00000000-0000-4000-8000-000000000012")
	if err := repository.Delete(context.Background(), user, "demo", id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete() = %v, want ErrNotFound", err)
	}
}
