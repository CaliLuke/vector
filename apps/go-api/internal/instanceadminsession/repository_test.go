package instanceadminsession

import (
	"context"
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
)

type rowStub struct {
	value any
	err   error
}

func (r rowStub) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	switch d := dest[0].(type) {
	case *bool:
		*d = r.value.(bool)
	case *[]byte:
		*d = []byte(r.value.(string))
	}
	return nil
}

type databaseStub struct {
	query string
	args  []any
	row   rowStub
}

func (d *databaseStub) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	d.query = q
	d.args = args
	return d.row
}

func TestRepositoryQueries(t *testing.T) {
	d := &databaseStub{row: rowStub{value: true}}
	r, err := NewRepository(d)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	ok, err := r.IsAdmin(context.Background(), id)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if !strings.Contains(d.query, "admins.role >= 15") ||
		!strings.Contains(d.query, "admins.instance_id") ||
		!strings.Contains(d.query, "ORDER BY created_at DESC") ||
		d.args[0] != id {
		t.Fatalf("query=%s", d.query)
	}
	d.row = rowStub{value: true}
	ok, err = r.HasAdminMembership(context.Background(), id)
	if err != nil || !ok || strings.Contains(d.query, "role >= 15") || !strings.Contains(d.query, "deleted_at IS NULL") {
		t.Fatalf("membership ok=%v err=%v query=%s", ok, err, d.query)
	}
	d.row = rowStub{value: `{"id":"00000000-0000-0000-0000-000000000001"}`}
	data, err := r.User(context.Background(), id)
	if err != nil || string(data) != string(`{"id":"00000000-0000-0000-0000-000000000001"}`) {
		t.Fatalf("data=%s err=%v", data, err)
	}
	if !strings.Contains(d.query, "/api/assets/v2/static/") || !strings.Contains(d.query, "avatar_asset_id") {
		t.Fatalf("avatar query=%s", d.query)
	}
}
func TestRepositoryError(t *testing.T) {
	r, _ := NewRepository(&databaseStub{row: rowStub{err: errors.New("boom")}})
	if _, err := r.IsAdmin(context.Background(), uuid.Nil()); err == nil {
		t.Fatal("expected error")
	}
}
