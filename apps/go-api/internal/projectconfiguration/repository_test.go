package projectconfiguration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
)

func TestRequireObjectInputRejectsValidJSONNonObjectsAs400(t *testing.T) {
	for _, input := range []djangoinput.Input{
		{Kind: "list"},
		{Kind: "string"},
		{Kind: "number"},
		{Kind: "boolean"},
		{Null: true, Kind: "NoneType"},
	} {
		err := requireObjectInput(input)
		httpErr, ok := err.(*HTTPError)
		if !ok || httpErr.Status != 400 {
			t.Fatalf("requireObjectInput(%#v) = %#v, want HTTP 400", input, err)
		}
		body, ok := httpErr.Body.(map[string]any)
		if !ok || body["error"] != "Please provide valid detail" {
			t.Fatalf("error body = %#v", httpErr.Body)
		}
	}
}

// Invalid create inputs enter and roll back a transaction without issuing SQL.
type labelValidationDatabase struct{ pgx.Tx }

func (db labelValidationDatabase) Begin(context.Context) (pgx.Tx, error) { return db, nil }
func (labelValidationDatabase) Rollback(context.Context) error           { return nil }

func TestAPILabelCreateRejectsArrayBeforeFieldValidation(t *testing.T) {
	repo := &Repository{db: labelValidationDatabase{}}
	_, err := repo.WriteAPILabel(t.Context(), "user", "workspace", "project", "", "POST", strings.NewReader(`[]`), "application/json")
	httpErr, ok := err.(*HTTPError)
	if !ok || httpErr.Status != 400 {
		t.Fatalf("expected 400, got %v", err)
	}
	body, marshalErr := json.Marshal(httpErr.Body)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if string(body) != `{"non_field_errors":["Invalid data. Expected a dictionary, but got list."]}` {
		t.Fatalf("unexpected error body: %s", body)
	}
}

func TestAPILabelSerializerUsesDRFFieldErrorsAndLengthRules(t *testing.T) {
	repo := &Repository{db: labelValidationDatabase{}}
	_, err := repo.WriteAPILabel(t.Context(), "user", "workspace", "project", "", "POST", strings.NewReader(`{"name":"`+strings.Repeat("x", 256)+`","sort_order":"not-a-number"}`), "application/json")
	httpErr, ok := err.(*HTTPError)
	if !ok || httpErr.Status != 400 {
		t.Fatalf("expected serializer HTTP 400, got %v", err)
	}
	body, ok := httpErr.Body.(map[string][]string)
	if !ok {
		t.Fatalf("serializer error body = %#v, want field-to-message arrays", httpErr.Body)
	}
	if got := body["name"]; len(got) != 1 || got[0] != "Ensure this field has no more than 255 characters." {
		t.Fatalf("name errors = %#v", got)
	}
	if got := body["sort_order"]; len(got) != 1 || got[0] != "A valid number is required." {
		t.Fatalf("sort_order errors = %#v", got)
	}
}

func TestAPILabelSerializerRejectsMalformedParentAsDRFUUIDError(t *testing.T) {
	repo := &Repository{db: labelValidationDatabase{}}
	_, err := repo.WriteAPILabel(t.Context(), "user", "workspace", "project", "", "POST", strings.NewReader(`{"name":"Child","parent":"not-a-uuid"}`), "application/json")
	httpErr, ok := err.(*HTTPError)
	if !ok || httpErr.Status != 400 {
		t.Fatalf("expected serializer HTTP 400, got %v", err)
	}
	body, ok := httpErr.Body.(map[string][]string)
	if !ok || len(body["parent"]) != 1 || body["parent"][0] != "“not-a-uuid” is not a valid UUID." {
		t.Fatalf("parent errors = %#v", httpErr.Body)
	}
}

func TestPatchEstimatePointMalformedStringIDReturnsSourceDetailError(t *testing.T) {
	repo := &Repository{}
	_, err := repo.PatchEstimatePoint(t.Context(), "user", "workspace", "project", "estimate", "not-a-uuid", strings.NewReader(`{}`), "application/json")
	httpErr, ok := err.(*HTTPError)
	if !ok || httpErr.Status != 400 {
		t.Fatalf("expected source-compatible HTTP 400, got %v", err)
	}
	body, marshalErr := json.Marshal(httpErr.Body)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if string(body) != `{"error":"Please provide valid detail"}` {
		t.Fatalf("response body = %s", body)
	}
}

func TestBulkCreateLabelsEmptyReturnsEmptyList(t *testing.T) {
	repo := &Repository{}
	body, err := repo.BulkCreateLabels(t.Context(), "user", "workspace", "project", strings.NewReader(`{"label_data":[]}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"labels":[]}` {
		t.Fatalf("response body = %s", body)
	}
}
