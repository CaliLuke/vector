package modulereads

import (
	"context"
	"github.com/jackc/pgx/v5"
	"reflect"
	"testing"
)

func TestAPIV1MembersSurviveProjectionAndKeepDuplicates(t *testing.T) {
	db := &moduleDatabaseStub{rows: []pgx.Row{moduleJSONRow(`{"total_count":1,"has_next":false,"results":[{"id":"module","name":"M","members":["u","u","inactive"]}]}`)}}
	r, _ := NewRepository(db)
	got, err := r.ReadAPIV1(context.Background(), "user", APIV1Request{Operation: "detail", Params: map[string]string{"fields": "unknown", "per_page": "bad", "cursor": "bad"}})
	want := map[string]any{"members": []any{"u", "u", "inactive"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%#v err=%v want=%#v", got, err, want)
	}
}
func TestAPIV1IssueRelatedOrderKeepsBaseCountAndContinuation(t *testing.T) {
	db := &moduleDatabaseStub{rows: []pgx.Row{moduleJSONRow(`{"total_count":1,"has_next":true,"results":[{"id":"a","name":"one"},{"id":"a","name":"one"}]}`)}}
	r, _ := NewRepository(db)
	got, err := r.ReadAPIV1(context.Background(), "user", APIV1Request{Operation: "issues", Params: map[string]string{"per_page": "2", "order_by": "labels__name"}, Present: map[string]bool{"per_page": true}})
	if err != nil {
		t.Fatal(err)
	}
	body := got.(map[string]any)
	if body["total_count"] != 1 || body["count"] != 2 || body["next_page_results"] != true {
		t.Fatalf("body=%#v", body)
	}
}
