package assets

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
)

func TestLegacyWritableFieldsMatchesDRFInputCoercions(t *testing.T) {
	workspace := uuid.New()
	fields, err := legacyWritableFields(map[string]string{
		"unknown":     "ignored",
		"is_archived": "yes",
		"is_deleted":  "off",
		"workspace":   workspace.String(),
		"size":        "0.0",
		"deleted_at":  "2026-10-19",
	})
	if err != nil {
		t.Fatal(err)
	}
	if fields["is_archived"] != true || fields["is_deleted"] != false {
		t.Fatalf("boolean coercions = %#v", fields)
	}
	if fields["workspace"] != workspace {
		t.Fatalf("workspace = %#v, want %s", fields["workspace"], workspace)
	}
	if fields["size"] != float64(0) {
		t.Fatalf("size = %#v", fields["size"])
	}
	if parsed, ok := fields["deleted_at"].(time.Time); !ok || parsed.Format(time.RFC3339) != "2026-10-19T00:00:00Z" {
		t.Fatalf("deleted_at = %#v", fields["deleted_at"])
	}
	if _, ok := fields["unknown"]; ok {
		t.Fatalf("unknown field was retained: %#v", fields)
	}
}

func TestLegacyWritableFieldsAggregatesDeclaredFieldErrors(t *testing.T) {
	_, err := legacyWritableFields(map[string]string{
		"is_archived": "sometimes",
		"is_deleted":  "perhaps",
		"workspace":   "not-a-uuid",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	var httpErr *djangoapi.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error type = %T", err)
	}
	body := httpErr.Body
	for _, name := range []string{"is_archived", "is_deleted", "workspace"} {
		if _, ok := body[name]; !ok {
			t.Errorf("missing %q error in %#v", name, body)
		}
	}
}

func TestLegacyWritableFieldsUsesDRFEmptyAndNonfiniteRules(t *testing.T) {
	fields, err := legacyWritableFields(map[string]string{"is_archived": "", "size": "nan"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["is_archived"]; ok {
		t.Fatalf("empty HTML boolean should be omitted: %#v", fields)
	}
	if number, ok := fields["size"].(float64); !ok || !math.IsNaN(number) {
		t.Fatalf("size = %#v, want NaN", fields["size"])
	}
	for _, value := range []string{"   ", "null", "not-a-uuid"} {
		if _, err := legacyWritableFields(map[string]string{"workspace": value}); err == nil {
			t.Errorf("workspace %q unexpectedly accepted", value)
		}
	}
	if _, err := legacyWritableFields(map[string]string{"deleted_at": "null"}); err == nil {
		t.Fatal("literal null datetime unexpectedly accepted")
	}
}

func TestTaskOriginAndActivityJSONUseConfiguredPythonRepresentation(t *testing.T) {
	service := &Service{appBaseURL: "http://localhost:8000/"}
	if got := service.taskOrigin(WithRequestOrigin(context.Background(), "http", "go-container:8080")); got != "http://localhost:8000" {
		t.Fatalf("task origin = %q", got)
	}
	encoded, err := json.Marshal(pythonJSONNumbers(map[string]any{"size": float64(0), "fraction": float64(37.5)}))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(encoded), `{"fraction":37.5,"size":0.0}`; got != want {
		t.Fatalf("python activity JSON = %s, want %s", got, want)
	}
}
