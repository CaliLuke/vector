package workitemhistory

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
)

func TestParseCursorMatchesGlobalPaginatorInputs(t *testing.T) {
	tests := []struct {
		name  string
		raw   *string
		limit int
		page  int
	}{
		{name: "default", limit: 1000, page: 0},
		{name: "explicit", raw: stringPointer("25:2:1"), limit: 25, page: 2},
		{name: "clamps limit", raw: stringPointer("1200:1:0"), limit: 1000, page: 1},
		{name: "python decimal digits", raw: stringPointer("٢٥:١:0"), limit: 25, page: 1},
		{name: "negative page uses first page", raw: stringPointer("10:-1:0"), limit: 10, page: -1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseCursor(test.raw)
			if err != nil {
				t.Fatal(err)
			}
			if got.limit != test.limit || got.page != test.page {
				t.Fatalf("parseCursor() = {%d,%d}, want {%d,%d}", got.limit, got.page, test.limit, test.page)
			}
		})
	}
}

func TestParseCursorRejectsInvalidPagination(t *testing.T) {
	for _, raw := range []string{"", "1:2", "1:2:3:4", "x:0:0", "1.0:0:0", "0:0:0", "-1:0:0", "1:999999999999999999999999999:0"} {
		t.Run(raw, func(t *testing.T) {
			_, err := parseCursor(&raw)
			var httpError *djangoapi.HTTPError
			if !errors.As(err, &httpError) || httpError.Status != 400 || httpError.Body["detail"] != "Invalid cursor parameter." {
				t.Fatalf("parseCursor(%q) error = %v, want 400 Invalid cursor parameter", raw, err)
			}
		})
	}
}

func TestCursorOffsetRejectsNonPositiveLimitAndOverflow(t *testing.T) {
	for _, page := range []cursorPage{{limit: 0, page: 0}, {limit: -1, page: 0}, {limit: 2, page: math.MaxInt}} {
		_, err := page.offset()
		var httpError *djangoapi.HTTPError
		if !errors.As(err, &httpError) || httpError.Status != 400 {
			t.Fatalf("offset(%+v) error = %v, want HTTP 400", page, err)
		}
	}
}

func TestTransformVersionTimestampsUsesUserTimezone(t *testing.T) {
	input := []byte(`{"created_at":"2026-10-21T12:00:00.123456+00:00","last_saved_at":"2026-10-21T12:00:00+00:00"}`)
	result, err := transformedJSON(input, nil, "test", "America/Los_Angeles", false)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatal(err)
	}
	if got["created_at"] != "2026-10-21T05:00:00.123456-07:00" {
		t.Fatalf("created_at = %q", got["created_at"])
	}
	if got["last_saved_at"] != "2026-10-21T12:00:00Z" {
		t.Fatalf("last_saved_at = %q", got["last_saved_at"])
	}
}

func TestVersionDetailConvertsLastSavedAt(t *testing.T) {
	input := []byte(`{"last_saved_at":"2026-10-21T12:00:00+00:00"}`)
	result, err := transformedJSON(input, nil, "test", "America/Los_Angeles", true)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatal(err)
	}
	if got["last_saved_at"] != "2026-10-21T05:00:00-07:00" {
		t.Fatalf("last_saved_at = %q", got["last_saved_at"])
	}
}

func TestVersionListSQLUsesSourcePaginatorKeys(t *testing.T) {
	for name, query := range map[string]string{
		"issue":       issueVersionListSQL,
		"description": descriptionVersionListSQL,
	} {
		for _, forbidden := range []string{"total_count", "grouped_by", "sub_grouped_by", "'count'"} {
			if strings.Contains(query, forbidden) {
				t.Errorf("%s version list SQL includes non-source paginator key %q", name, forbidden)
			}
		}
		if strings.Contains(query, "jsonb),\n)") {
			t.Errorf("%s version list SQL has a trailing JSON argument comma", name)
		}
	}
}

func TestMixedHistoryOmitsIntakeSourceProjection(t *testing.T) {
	if strings.Contains(historyMixedSQL, "FROM intake_issues") {
		t.Fatal("mixed history query includes intake source data that Django only prefetches for issue-property history")
	}
	if !strings.Contains(historyMixedSQL, "'source_data',NULL") {
		t.Fatal("mixed history query must preserve the serializer's null source_data field")
	}
	if !strings.Contains(historyActivitiesSQL, "FROM intake_issues") {
		t.Fatal("issue-property history query must retain the prefetched intake source data")
	}
}

func stringPointer(value string) *string { return &value }
