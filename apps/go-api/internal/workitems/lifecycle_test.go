package workitems

import (
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/apicontract"
)

func TestBulkDateValidationRejectsInvalidValuesAndReversedDates(t *testing.T) {
	for _, tc := range []struct {
		start, target any
		valid, fails  bool
	}{
		{true, nil, false, true},
		{true, "2026-01-01", false, true},
		{"invalid", nil, false, true},
		{"2026-01-02", "2026-01-01", false, false},
		{"2026-01-01", "2026-01-02", true, false},
	} {
		got, err := validateBulkDates(tc.start, tc.target)
		if got != tc.valid || (err != nil) != tc.fails {
			t.Fatalf("dates %#v/%#v: valid=%v err=%v", tc.start, tc.target, got, err)
		}
	}
}

func TestBulkDatePersistenceRejectsSpacePaddedDayAsClientError(t *testing.T) {
	_, err := prepareBulkDateForPersistence("2026-1- 2")
	var httpErr *apicontract.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTP 400 error, got %v", err)
	}
	if httpErr.Status != 400 {
		t.Fatalf("status = %d, want 400", httpErr.Status)
	}
	if got := httpErr.Body["error"]; got != "Please provide valid detail" {
		t.Fatalf("error body = %#v", httpErr.Body)
	}

	got, err := prepareBulkDateForPersistence("2026-01-03")
	if err != nil || got != "2026-01-03" {
		t.Fatalf("valid date = %#v, %v; want 2026-01-03", got, err)
	}
}

func TestBulkDatePersistenceRejectsNonStringAsClientError(t *testing.T) {
	_, err := prepareBulkDateForPersistence(true)
	var response *apicontract.HTTPError
	if !errors.As(err, &response) || response.Status != 400 {
		t.Fatalf("expected client validation error, got %v", err)
	}
}
