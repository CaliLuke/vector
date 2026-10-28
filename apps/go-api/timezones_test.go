package systemapi

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	timezoneapi "github.com/CaliLuke/go-plane/gen/timezones"
)

func TestTimezonesMatchRecordedDjangoResponse(t *testing.T) {
	fixtureBytes, err := os.ReadFile("internal/parity/testdata/fixtures/timezones/list.json")
	if err != nil {
		t.Fatalf("read Django timezone fixture: %v", err)
	}
	var fixture struct {
		Response struct {
			Body timezoneapi.TimezoneList `json:"body"`
		} `json:"response"`
	}
	if unmarshalErr := json.Unmarshal(fixtureBytes, &fixture); unmarshalErr != nil {
		t.Fatalf("decode Django timezone fixture: %v", unmarshalErr)
	}

	frozen := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)
	result, err := newTimezones(func() time.Time { return frozen }).List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result.Timezones) != 120 {
		t.Fatalf("List() timezone count = %d, want 120", len(result.Timezones))
	}
	for index, got := range result.Timezones {
		want := fixture.Response.Body.Timezones[index]
		if *got != *want {
			t.Fatalf("List() timezone %d = %#v, want %#v", index, got, want)
		}
	}
	got, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("encode Go timezone response: %v", err)
	}
	want, err := json.Marshal(fixture.Response.Body)
	if err != nil {
		t.Fatalf("encode Django timezone response: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("List() response differs from Django fixture")
	}
}

func TestTimezonesApplyDaylightSavingOffsets(t *testing.T) {
	frozen := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	result, err := newTimezones(func() time.Time { return frozen }).List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	want := map[string]string{
		"America/New_York": "UTC-04:00",
		"Australia/Sydney": "UTC+10:00",
		"Europe/Paris":     "UTC+02:00",
	}
	for _, item := range result.Timezones {
		if offset, exists := want[item.Value]; exists {
			if item.UtcOffset != offset {
				t.Errorf("%s offset = %q, want %q", item.Value, item.UtcOffset, offset)
			}
			delete(want, item.Value)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing timezone offsets: %v", want)
	}
}

func TestFormatTimezoneOffset(t *testing.T) {
	for _, test := range []struct {
		seconds int
		want    string
	}{
		// Django uses floor division for negative offsets. Keep its wire behavior.
		{seconds: -3*3600 - 30*60, want: "-04:30"},
		{seconds: 0, want: "+00:00"},
		{seconds: 5*3600 + 45*60, want: "+05:45"},
	} {
		if got := formatTimezoneOffset(test.seconds); got != test.want {
			t.Errorf("formatTimezoneOffset(%d) = %q, want %q", test.seconds, got, test.want)
		}
	}
}
