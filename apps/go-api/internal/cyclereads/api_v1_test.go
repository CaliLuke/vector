package cyclereads

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestAPIV1ProjectionPrecedesExpansionAndKeepsMetadataOpaque(t *testing.T) {
	repository, _ := NewRepository(&databaseStub{})
	metadata := map[string]any{"created_at": "2011-06-17T13:14:15Z"}
	row := map[string]any{"id": "cycle-id", "name": "Sprint", "view_props": metadata, "project_id": "project-id"}
	result, err := repository.apiSerialize(context.Background(), row, "cycle", "id,view_props", "id,project", "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, map[string]any{"id": nil, "view_props": metadata}) {
		t.Fatalf("projection/expansion = %#v", result)
	}
}

func TestAPIV1RelatedOrderPreservesBaseCountAndWindowContinuation(t *testing.T) {
	db := &databaseStub{rows: []pgx.Row{jsonRow(`{"total_count":1,"has_next":true,"results":[{"id":"a","name":"one"},{"id":"a","name":"one"}]}`)}}
	repository, _ := NewRepository(db)
	body, err := repository.ReadAPIV1(context.Background(), "user", APIV1Request{Operation: "issues", Params: map[string]string{"per_page": "2", "order_by": "assignees__first_name"}, Present: map[string]bool{"per_page": true}})
	if err != nil {
		t.Fatal(err)
	}
	result := body.(map[string]any)
	if result["total_count"] != 1 || result["count"] != 2 || result["next_page_results"] != true {
		t.Fatalf("pagination = %#v", result)
	}
}

func TestAPIV1CurrentCyclesIgnoreMalformedPagination(t *testing.T) {
	db := &databaseStub{rows: []pgx.Row{boolRow(true), jsonRow(`{"total_count":1,"has_next":false,"results":[{"id":"a","name":"one","total_issues":0}]}`)}}
	repository, _ := NewRepository(db)
	body, err := repository.ReadAPIV1(context.Background(), "user", APIV1Request{Operation: "list", Params: map[string]string{"cycle_view": "current", "per_page": "bad", "cursor": "bad"}, Present: map[string]bool{"per_page": true, "cursor": true}})
	if err != nil {
		t.Fatal(err)
	}
	rows, ok := body.([]any)
	if !ok || len(rows) != 1 || rows[0].(map[string]any)["total_issues"] != json.Number("0") {
		t.Fatalf("current cycles = %#v", body)
	}
}

func TestAPIV1NullEstimateExpansionUsesSerializerInitialFields(t *testing.T) {
	repository, _ := NewRepository(&databaseStub{})
	row := map[string]any{"id": "issue", "estimate_point_id": nil}
	got, err := repository.apiSerialize(context.Background(), row, "issue", "id,estimate_point", "estimate_point", "")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"id": "issue", "estimate_point": map[string]any{"deleted_at": nil, "key": nil, "description": "", "value": "", "created_by": nil, "updated_by": nil}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("null estimate expansion = %#v, want %#v", got, want)
	}
}
