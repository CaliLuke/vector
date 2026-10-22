package systemapi

import (
	"bytes"
	"testing"
	"time"
)

func TestFormatProjectEstimateDateTimesUsesDRFTimezoneFormat(t *testing.T) {
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	got, err := formatProjectEstimateDateTimes([]byte(`[{"created_at":"2026-10-22T07:35:38.673523+00:00","updated_at":"2026-10-22T07:35:38.673523Z","deleted_at":null}]`), location)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte(`[{"created_at":"2026-10-22T00:35:38.673523-07:00","deleted_at":null,"updated_at":"2026-10-22T00:35:38.673523-07:00"}]`)
	if !bytes.Equal(got, want) {
		t.Fatalf("formatted estimate points = %s, want %s", got, want)
	}
}
