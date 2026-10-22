package djangoapi

import (
	"fmt"
	"math"
	"testing"
)

func TestPageWindowUsesCurrentSizeInBothDirections(t *testing.T) {
	for _, cursor := range []string{"5:1:0", "5:1:1", "1:1:1", "2.5:1:1", "2:1:1"} {
		t.Run(cursor, func(t *testing.T) {
			limit, page, offset, err := PageWindow(Query{
				Params:  map[string]string{"per_page": "2", "cursor": cursor},
				Present: map[string]bool{"per_page": true, "cursor": true},
			})
			if err != nil || limit != 2 || page != 1 || offset != 2 {
				t.Fatalf("window=(%d,%d,%d), error=%v", limit, page, offset, err)
			}
			rows := []any{"a", "b", "c", "d", "e"}
			result := Envelope(rows[offset:offset+limit], len(rows), limit, page)
			if fmt.Sprint(result["results"]) != "[c d]" || result["next_cursor"] != "2:2:0" || result["prev_cursor"] != "2:0:1" || result["next_page_results"] != true || result["prev_page_results"] != true {
				t.Fatalf("page envelope=%#v", result)
			}
		})
	}
}

func TestEnvelopeKeepsPythonCursorPrecision(t *testing.T) {
	result := Envelope([]any{}, 5, 1, math.MaxInt64)
	if result["next_cursor"] != "1:9223372036854775808:0" || result["prev_cursor"] != "1:9223372036854775806:1" || result["next_page_results"] != false {
		t.Fatalf("overflowing cursor %#v", result)
	}
}
