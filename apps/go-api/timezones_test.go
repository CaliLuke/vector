package systemapi

import (
	"context"
	"testing"
	"time"

	"strings"

	"github.com/stretchr/testify/require"
)

func TestTimezonesPublishValidNamesAndAccurateOffsets(t *testing.T) {
	frozen := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)
	result, err := newTimezones(func() time.Time { return frozen }).List(t.Context())
	require.NoError(t, err)
	require.Len(t, result.Timezones, len(timezoneLocations))
	for _, zone := range result.Timezones {
		location, err := time.LoadLocation(zone.Value)
		require.NoError(t, err, zone.Value)
		_, want := frozen.In(location).Zone()
		parsed, err := time.Parse("-07:00", strings.TrimPrefix(zone.UtcOffset, "UTC"))
		require.NoError(t, err, zone.Value)
		_, got := parsed.Zone()
		require.Equal(t, want, got, zone.Value)
		require.Equal(t, "GMT"+strings.TrimPrefix(zone.UtcOffset, "UTC"), zone.GmtOffset)
		require.NotEmpty(t, zone.Label)
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
		// Negative offsets retain their hour and minute magnitudes.
		{seconds: -3*3600 - 30*60, want: "-03:30"},
		{seconds: -30 * 60, want: "-00:30"},
		{seconds: 0, want: "+00:00"},
		{seconds: 5*3600 + 45*60, want: "+05:45"},
	} {
		if got := formatTimezoneOffset(test.seconds); got != test.want {
			t.Errorf("formatTimezoneOffset(%d) = %q, want %q", test.seconds, got, test.want)
		}
	}
}
