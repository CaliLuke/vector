package notifications

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPaginationErrorsMatchDjangoFixtures(t *testing.T) {
	for _, name := range []string{"pagination-invalid-limit", "pagination-excessive-limit", "pagination-invalid-cursor", "pagination-zero-limit", "pagination-negative-limit", "pagination-mismatched-backward"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("../parity/testdata/fixtures/notification_reads", name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var fixture struct {
				Request  struct{ Query map[string]string }
				Response struct {
					Status int
					Body   map[string]any
				}
			}
			if unmarshalErr := json.Unmarshal(raw, &fixture); unmarshalErr != nil {
				t.Fatal(unmarshalErr)
			}
			query := url.Values{}
			for key, value := range fixture.Request.Query {
				query.Set(key, value)
			}
			_, err = parsePage(query)
			var response *HTTPError
			if !errors.As(err, &response) {
				t.Fatalf("expected HTTP error, got %v", err)
			}
			if response.Status != fixture.Response.Status || !reflect.DeepEqual(response.Body, fixture.Response.Body) {
				t.Fatalf("got %d %v, want Django %d %v", response.Status, response.Body, fixture.Response.Status, fixture.Response.Body)
			}
		})
	}
}

func TestPaginationPreservesPythonCursorGrammar(t *testing.T) {
	for _, test := range []struct{ name, limit, cursor, offset string }{
		{"forward ignores cursor limit", "2", "9:3:0", "6"},
		{"float equal backward", "2", "2.0:1:-3", "2"},
		{"Unicode digits", "٢", "٢:٣:٠", "6"},
		{"underscores", "1_0", "10:1_0:0", "100"},
		{"arbitrary precision offset", "2", "2:100000000000000000000:0", "200000000000000000000"},
	} {
		t.Run(test.name, func(t *testing.T) {
			page, err := parsePage(url.Values{"per_page": {test.limit}, "cursor": {test.cursor}})
			if err != nil {
				t.Fatal(err)
			}
			if page.offset.String() != test.offset {
				t.Fatalf("offset=%s, want %s", page.offset, test.offset)
			}
		})
	}
}

func TestPaginationNegativeOffsetIsBadPagination(t *testing.T) {
	_, err := parsePage(url.Values{"per_page": {"2"}, "cursor": {"2:-1:1"}})
	var response *HTTPError
	if !errors.As(err, &response) || response.Status != 400 || response.Body["detail"] != "Error in parsing" {
		t.Fatalf("negative offset must preserve Django BadPaginationError response: %#v", err)
	}
}

func TestPaginationRequiresBothNonemptyParameters(t *testing.T) {
	for _, query := range []url.Values{{}, {"per_page": {"bad"}}, {"cursor": {"bad"}}, {"per_page": {"2"}, "cursor": {""}}, {"per_page": {""}, "cursor": {"bad"}}} {
		page, err := parsePage(query)
		if page != nil || err != nil {
			t.Fatalf("query %v activates pagination: %v, %v", query, page, err)
		}
	}
}

func TestNotificationDatesDoNotRewriteUserJSON(t *testing.T) {
	const raw = `{"created_at":"2026-01-02T03:04:05.12Z","updated_at":"2026-01-02T03:04:05Z","read_at":null,"data":{"created_at":"2025-03-04T05:06:07Z","large":9007199254740993},"message":{"nested":{"archived_at":"2025-03-04T05:06:07Z"}}}`
	record, err := decodeRecord([]byte(raw), "Asia/Kolkata")
	if err != nil {
		t.Fatal(err)
	}
	if record["created_at"] != "2026-01-02T08:34:05.120000+05:30" || record["updated_at"] != "2026-01-02T08:34:05+05:30" || record["read_at"] != nil {
		t.Fatalf("model dates changed incorrectly: %v", record)
	}
	if want := map[string]any{"created_at": "2025-03-04T05:06:07Z", "large": json.Number("9007199254740993")}; !reflect.DeepEqual(record["data"], want) {
		t.Fatalf("user data changed: %#v", record["data"])
	}
	if want := map[string]any{"nested": map[string]any{"archived_at": "2025-03-04T05:06:07Z"}}; !reflect.DeepEqual(record["message"], want) {
		t.Fatalf("user message changed: %#v", record["message"])
	}
}
