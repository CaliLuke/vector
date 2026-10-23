package userproperties

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"uuid"

	"github.com/jackc/pgx/v5"
)

type row func(...any) error

func (r row) Scan(dest ...any) error { return r(dest...) }

type db struct {
	queries []string
	rows    []pgx.Row
}

func (d *db) QueryRow(_ context.Context, q string, _ ...any) pgx.Row {
	d.queries = append(d.queries, q)
	if len(d.rows) == 0 {
		return row(func(...any) error { return errors.New("unexpected query") })
	}
	r := d.rows[0]
	d.rows = d.rows[1:]
	return r
}
func jsonRow(s string) pgx.Row {
	return row(func(dest ...any) error { *dest[0].(*[]byte) = []byte(s); return nil })
}

func TestRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("expected nil database error")
	}
}

func TestWorkspaceGetCreatesDjangoDefaults(t *testing.T) {
	d := &db{rows: []pgx.Row{jsonRow(`{"filters":{},"navigation_project_limit":10}`)}}
	r, _ := NewRepository(d)
	got, err := r.GetWorkspace(context.Background(), uuid.Nil(), "demo")
	if err != nil || string(got) != `{"filters":{},"navigation_project_limit":10}` {
		t.Fatalf("get=%s err=%v", got, err)
	}
	q := d.queries[0]
	for _, fragment := range []string{"workspace_user_properties", "ON CONFLICT", "'ACCORDION'", "'{}'::jsonb", "deleted_at IS NULL"} {
		if !strings.Contains(q, fragment) {
			t.Errorf("query missing %q", fragment)
		}
	}
}

func TestScopedPropertiesPatchMergesOnlyPresentFields(t *testing.T) {
	d := &db{rows: []pgx.Row{jsonRow(`{"filters":{"state":"started"}}`)}}
	r, _ := NewRepository(d)
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	got, err := r.PatchProject(context.Background(), uuid.Nil(), "demo", id, map[string]json.RawMessage{"filters": json.RawMessage(`{"state":"started"}`)})
	if err != nil || len(got) == 0 {
		t.Fatalf("patch=%s err=%v", got, err)
	}
	q := d.queries[0]
	for _, fragment := range []string{"UPDATE project_user_properties", "CASE WHEN $4", "display_filters", "updated_by_id", "p.user_id=$2", "p.deleted_at IS NULL"} {
		if !strings.Contains(q, fragment) {
			t.Errorf("query missing %q", fragment)
		}
	}
}

func TestProjectAuthorizationRequiresActiveProjectMembership(t *testing.T) {
	d := &db{rows: []pgx.Row{row(func(dest ...any) error { *dest[0].(*bool) = true; return nil })}}
	r, _ := NewRepository(d)
	if ok, err := r.AuthorizedProject(context.Background(), uuid.Nil(), "demo", uuid.Nil()); err != nil || !ok {
		t.Fatalf("authorized=%v err=%v", ok, err)
	}
	if !strings.Contains(d.queries[0], "project_members") || !strings.Contains(d.queries[0], "pm.is_active=true") {
		t.Fatalf("project authorization query is not scoped: %s", d.queries[0])
	}
}

func TestWorkspacePatchSupportsNavigationFields(t *testing.T) {
	d := &db{rows: []pgx.Row{jsonRow(`{"navigation_project_limit":25,"navigation_control_preference":"TABBED"}`)}}
	r, _ := NewRepository(d)
	_, err := r.PatchWorkspace(context.Background(), uuid.Nil(), "demo", map[string]json.RawMessage{
		"navigation_project_limit":      json.RawMessage(`25`),
		"navigation_control_preference": json.RawMessage(`"TABBED"`),
	})
	if err != nil {
		t.Fatalf("patch workspace: %v", err)
	}
	for _, fragment := range []string{"navigation_project_limit", "navigation_control_preference", "CASE WHEN $11", "CASE WHEN $13"} {
		if !strings.Contains(d.queries[0], fragment) {
			t.Errorf("query missing %q", fragment)
		}
	}
}

func TestProjectPatchSupportsSortOrderAndPreferences(t *testing.T) {
	d := &db{rows: []pgx.Row{jsonRow(`{"sort_order":7.5,"preferences":{"pages":{"block_display":false}}}`)}}
	r, _ := NewRepository(d)
	_, err := r.PatchProject(context.Background(), uuid.Nil(), "demo", uuid.Nil(), map[string]json.RawMessage{
		"sort_order":  json.RawMessage(`7.5`),
		"preferences": json.RawMessage(`{"pages":{"block_display":false}}`),
	})
	if err != nil {
		t.Fatalf("patch project: %v", err)
	}
	for _, fragment := range []string{"preferences=CASE WHEN $12", "sort_order=CASE WHEN $14"} {
		if !strings.Contains(d.queries[0], fragment) {
			t.Errorf("query missing %q", fragment)
		}
	}
}
