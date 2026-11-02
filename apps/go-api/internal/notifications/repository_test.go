package notifications

import (
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"testing"
)

func TestNotificationPaginationUsesNativeWindows(t *testing.T) {
	page, err := parsePage(url.Values{"per_page": {"20"}, "cursor": {"10:2:1"}})
	if err != nil || page.limit != 20 || page.number != 2 || page.offset != 40 {
		t.Fatalf("window=%+v error=%v", page, err)
	}
	page, err = parsePage(url.Values{})
	if err != nil || page != nil {
		t.Fatalf("unpaginated window=%+v error=%v", page, err)
	}
	for _, query := range []url.Values{
		{"per_page": {"0"}}, {"per_page": {"1001"}}, {"per_page": {"٢"}},
		{"per_page": {"2"}, "cursor": {"2:-1:0"}}, {"cursor": {"2.0:1:-3"}},
		{"cursor": {"2:100000000000000000000:0"}}, {"cursor": {""}},
	} {
		_, err := parsePage(query)
		var failure *HTTPError
		if !errors.As(err, &failure) || failure.Status != 400 {
			t.Fatalf("query=%v error=%v", query, err)
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
