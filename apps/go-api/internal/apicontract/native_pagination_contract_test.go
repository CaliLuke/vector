package apicontract

import (
	"errors"
	"testing"
)

func TestNativePaginationRejectsInvalidRequestsAsClientErrors(t *testing.T) {
	for _, test := range []struct{ name, limit, cursor string }{
		{name: "zero size", limit: "0"},
		{name: "negative size", limit: "-1"},
		{name: "oversize", limit: "1001"},
		{name: "overflow size", limit: "999999999999999999999999999"},
		{name: "nondecimal size", limit: "١٠"},
		{name: "negative page", limit: "2", cursor: "2:-1:0"},
		{name: "page overflow", limit: "2", cursor: "2:99999999999999999999999:0"},
		{name: "offset overflow", limit: "1000", cursor: "1000:9223372036854775807:0"},
		{name: "invalid direction", limit: "2", cursor: "2:1:2"},
		{name: "fractional cursor size", limit: "2", cursor: "2.5:1:0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := Query{Params: map[string]string{"per_page": test.limit}, Present: map[string]bool{"per_page": true}}
			if test.cursor != "" {
				request.Params["cursor"] = test.cursor
				request.Present["cursor"] = true
			}
			_, _, _, err := PageWindow(request)
			var invalid *HTTPError
			if !errors.As(err, &invalid) || invalid.Status != 400 {
				t.Fatalf("invalid pagination status: %v", err)
			}
		})
	}
}

func TestNativePaginationUsesCurrentSizeAcrossNavigation(t *testing.T) {
	limit, page, offset, err := PageWindow(Query{Params: map[string]string{"per_page": "2", "cursor": "5:1:1"}, Present: map[string]bool{"per_page": true, "cursor": true}})
	if err != nil || limit != 2 || page != 1 || offset != 2 {
		t.Fatalf("window=%d/%d/%d error=%v", limit, page, offset, err)
	}
}
