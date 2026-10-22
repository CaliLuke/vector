package workspacemetadata

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"uuid"
)

type metadataRow func(...any) error

func (r metadataRow) Scan(dest ...any) error { return r(dest...) }

type metadataDB struct {
	queries []string
	rows    []pgx.Row
}

func (d *metadataDB) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	d.queries = append(d.queries, query)
	if len(d.rows) == 0 {
		return metadataRow(func(...any) error { return errors.New("unexpected query") })
	}
	row := d.rows[0]
	d.rows = d.rows[1:]
	return row
}

func metadataJSON(value string) pgx.Row {
	return metadataRow(func(dest ...any) error { *dest[0].(*[]byte) = []byte(value); return nil })
}

func metadataBool(value bool) pgx.Row {
	return metadataRow(func(dest ...any) error { *dest[0].(*bool) = value; return nil })
}

func TestRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("expected nil database error")
	}
}

func TestRepositoryListsWorkspaceLabelsWithProjectVisibilityAndDjangoOrdering(t *testing.T) {
	db := &metadataDB{rows: []pgx.Row{metadataBool(true), metadataJSON(`[{"name":"Newer"}]`)}}
	r, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	user := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	if ok, err := r.Authorized(context.Background(), user, "workspace-metadata"); err != nil || !ok {
		t.Fatalf("Authorized() = %v, %v", ok, err)
	}
	body, err := r.ListLabels(context.Background(), user, "workspace-metadata")
	if err != nil || string(body) != `[{"name":"Newer"}]` {
		t.Fatalf("ListLabels() = %s, %v", body, err)
	}
	query := strings.Join(db.queries, "\n")
	for _, fragment := range []string{"project_members", "pm.is_active=true", "p.archived_at IS NULL", "l.deleted_at IS NULL", "ORDER BY created_at DESC"} {
		if !strings.Contains(query, fragment) {
			t.Errorf("labels query missing %q", fragment)
		}
	}
}

func TestRepositoryListsWorkspaceStatesWithVisibilityTriageAndGroupOrder(t *testing.T) {
	db := &metadataDB{rows: []pgx.Row{metadataJSON(`[{"name":"Backlog One","order":0.5}]`)}}
	r, _ := NewRepository(db)
	body, err := r.ListStates(context.Background(), uuid.Nil(), "workspace-metadata")
	if err != nil || string(body) != `[{"name":"Backlog One","order":0.5}]` {
		t.Fatalf("ListStates() = %s, %v", body, err)
	}
	for _, fragment := range []string{"s.is_triage=false", "s.group <> 'triage'", "row_number() OVER (PARTITION BY s.group", "count(*) OVER (PARTITION BY s.group)", "ORDER BY sequence,id", "project_members"} {
		if !strings.Contains(db.queries[0], fragment) {
			t.Errorf("states query missing %q", fragment)
		}
	}
}
