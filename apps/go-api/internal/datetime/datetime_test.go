package datetime

import (
	"strings"
	"testing"
	"time"
)

func TestDateRequiresARealPaddedCalendarDate(t *testing.T) {
	for _, value := range []string{"0001-01-01", "2024-02-29", "2026-11-01", "9999-12-31"} {
		if got, err := Date(value); err != nil || got != value {
			t.Fatalf("Date(%q)=%q, %v", value, got, err)
		}
	}
	for _, value := range []string{"", "0000-01-01", "2026-02-29", "2026-1-02", "2026-01-2", "20261101", "2026-W44-7", "٢٠٢٦-١١-٠١", "2026-11-01T00:00:00Z", " 2026-11-01", "2026-11-01\n"} {
		if got, err := Date(value); err == nil {
			t.Fatalf("accepted %q as %q", value, got)
		}
	}
}

func TestParseCalendarValuesPreservesInstantsAndRequestTimezone(t *testing.T) {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ input, want string }{
		{"2026-11-01T12:30:00Z", "2026-11-01T12:30:00Z"},
		{"2026-11-01T12:30:00+05:45", "2026-11-01T06:45:00Z"},
		{"2026-11-01T12:30:00-08:00", "2026-11-01T20:30:00Z"},
		{"2026-11-01", "2026-11-01T04:00:00Z"},
		{"2026-07-01T12:30", "2026-07-01T16:30:00Z"},
		{"2026-12-01 12:30", "2026-12-01T17:30:00Z"},
		{"2026-07-01T12:30:59.123456789", "2026-07-01T16:30:59.123456Z"},
		{"2026-07-01 12:30:59.123400", "2026-07-01T16:30:59.1234Z"},
		{"2026-11-01T12:30:00.000000999Z", "2026-11-01T12:30:00Z"},
		{"2024-02-29", "2024-02-29T05:00:00Z"},
	} {
		t.Run(test.input, func(t *testing.T) {
			parsed, err := Parse(test.input, location)
			if err != nil || parsed.UTC().Format(time.RFC3339Nano) != test.want {
				t.Fatalf("timestamp=%s error=%v want=%s", parsed, err, test.want)
			}
		})
	}
	if parsed, err := Parse("2026-11-01T12:30:00Z", nil); err != nil || parsed.Hour() != 12 {
		t.Fatalf("aware timestamp=%s error=%v", parsed, err)
	}
}

func TestParseRejectsInvalidDatesAndLegacySpellings(t *testing.T) {
	for _, value := range []string{
		"", "2026-1-2T3:4:5Z", "٢٠٢٦-١-٢T٣:٤:٥Z", "2026W011", "2026-W01-1", "20260102",
		"2026-02-30", "0000-01-01", "0000-01-01T00:00:00Z", "2026-02-30T00:00:00Z",
		"2026-11-01T24:00:00Z", "2026-11-01T12:60:00Z", "2026-11-01T12:30:60Z",
		"2026-11-01T12:30:00+24:00", "2026-11-01T12:30:00+01:60", "2026-11-01T12:30:00+0530",
		"2026-11-01T12:30:00,123Z", "2026-11-01T12:30:00.1234567890Z", "2026-11-01T12:30Z",
		"2026-11-01x12:30:00Z", "2026-11-01T1:30:00Z", " 2026-11-01", "2026-11-01\n",
		"2026-11-01T24:30", "2026-11-01T12:30:00 ", strings.Repeat("1", 1000),
	} {
		t.Run(value, func(t *testing.T) {
			if parsed, err := Parse(value, time.UTC); err == nil {
				t.Fatalf("accepted %q as %s", value, parsed)
			}
		})
	}
	for _, value := range []string{"2026-11-01", "2026-11-01T12:30"} {
		if _, err := Parse(value, nil); err == nil {
			t.Fatalf("accepted local timestamp %q without a location", value)
		}
	}
}

func TestFormatUsesNativeISOAndDatabasePrecision(t *testing.T) {
	input := time.Date(2026, 11, 1, 12, 30, 0, 123400789, time.UTC)
	if got := Format(input, time.FixedZone("local", 3600)); got != "2026-11-01T13:30:00.1234+01:00" {
		t.Fatalf("timestamp=%q", got)
	}
	if got := Format(input.Truncate(time.Second), nil); got != "2026-11-01T12:30:00Z" {
		t.Fatalf("timestamp=%q", got)
	}
}
