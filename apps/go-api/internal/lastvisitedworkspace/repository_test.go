package lastvisitedworkspace

import (
	"context"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
)

type rowFn func(...any) error

func (r rowFn) Scan(v ...any) error { return r(v...) }

type dbStub struct{ query string }

func (d *dbStub) QueryRow(_ context.Context, q string, _ ...any) pgx.Row {
	d.query = q
	return rowFn(func(v ...any) error {
		*v[0].(*[]byte) = []byte(`{"project_details":[],"workspace_details":{}}`)
		return nil
	})
}
func TestGetUsesProfileAndSerializerKeys(t *testing.T) {
	d := &dbStub{}
	r, _ := NewRepository(d)
	if _, err := r.Get(context.Background(), uuid.Nil()); err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"profiles", "last_workspace_id", "background_color", "logo_asset", "avatar_asset_id", "cover_image_asset_id", "/api/assets/v2/static/", "project_details", "created_by", "updated_by"} {
		if !strings.Contains(d.query, part) {
			t.Errorf("query missing %q", part)
		}
	}
	if strings.Contains(d.query, "u.last_workspace_id") {
		t.Fatal("query references removed users column")
	}
}
func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("expected error")
	}
}
