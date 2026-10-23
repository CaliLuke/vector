package projectlabelreads

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestAPIV1DetailIgnoresProjectionAndPagination(t *testing.T) {
	db := &databaseStub{rows: []pgx.Row{jsonRow(`{"total_count":1,"has_next":false,"results":[{"id":"label","name":"Name","parent_id":"parent","expanded_parent":{"id":"parent","project_id":"project"}}]}`)}}
	r, _ := NewRepository(db)
	got, err := r.ReadAPIV1(context.Background(), "u", APIV1Request{Operation: "detail", Params: map[string]string{"fields": "unknown", "expand": "parent", "cursor": "bad", "per_page": "many"}})
	want := map[string]any{"id": "label", "name": "Name", "parent": "parent"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%#v err=%v", got, err)
	}
}
func TestAPIV1ParentLabelExpansionPreservesSourceShape(t *testing.T) {
	db := &databaseStub{rows: []pgx.Row{jsonRow(`{"total_count":2,"has_next":false,"results":[{"id":"child","name":"Child","parent_id":"parent","expanded_parent":{"id":"parent","project_id":"project"}},{"id":"parent","name":"Parent","parent_id":null,"expanded_parent":{}}]}`)}}
	r, _ := NewRepository(db)
	got, err := r.ReadAPIV1(context.Background(), "u", APIV1Request{Params: map[string]string{"fields": "id,parent", "expand": "parent"}})
	if err != nil {
		t.Fatal(err)
	}
	rows := got.(map[string]any)["results"].([]any)
	want := []any{map[string]any{"id": "child", "parent": map[string]any{"id": "parent", "project_id": "project"}}, map[string]any{"id": "parent", "parent": map[string]any{}}}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows=%#v", rows)
	}
}
func TestAPIV1MissingDetail(t *testing.T) {
	db := &databaseStub{rows: []pgx.Row{jsonRow(`{"total_count":0,"has_next":false,"results":[]}`)}}
	r, _ := NewRepository(db)
	_, err := r.ReadAPIV1(context.Background(), "u", APIV1Request{Operation: "detail", Params: map[string]string{}})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestAPIV1ListRetainsScalarLabelDescription(t *testing.T) {
	db := &databaseStub{rows: []pgx.Row{jsonRow(`{"total_count":1,"has_next":false,"results":[{"id":"label","name":"Name","description":"Label description","created_by_id":null,"updated_by_id":null,"workspace_id":"workspace","project_id":"project","parent_id":null}]}`)}}
	r, _ := NewRepository(db)
	got, err := r.ReadAPIV1(context.Background(), "u", APIV1Request{Params: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	row := got.(map[string]any)["results"].([]any)[0].(map[string]any)
	if row["description"] != "Label description" {
		t.Fatalf("description=%#v, want scalar label text", row["description"])
	}
}
