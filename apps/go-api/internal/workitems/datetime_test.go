package workitems

import (
	"testing"
	"time"
)

func TestDateTimeInputPreservesDjangoISOForms(t *testing.T) {
	for _, tc := range []struct{ input, zone, instant string }{
		{"2026-10-10T12:30", "UTC", "2026-10-10T12:30:00Z"},
		{"2026-10-10T12:30Z", "UTC", "2026-10-10T12:30:00Z"},
		{"2026-10-10T12:30:00+0530", "UTC", "2026-10-10T07:00:00Z"},
		{"20261010T1230Z", "UTC", "2026-10-10T12:30:00Z"},
		{"2026-W41-6T12:30:00Z", "UTC", "2026-10-10T12:30:00Z"},
		{"2026-10-10T12:30", "America/New_York", "2026-10-10T16:30:00Z"},
		{"2026-10-10", "America/New_York", "2026-10-10T04:00:00Z"},
		{"2026-10-10T12:30:00.123456789Z", "UTC", "2026-10-10T12:30:00.123456Z"},
	} {
		t.Run(tc.input+"/"+tc.zone, func(t *testing.T) {
			got, err := parseQueryDateTime(tc.input, tc.zone)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := time.Parse(time.RFC3339Nano, got)
			want, _ := time.Parse(time.RFC3339Nano, tc.instant)
			if err != nil || !parsed.Equal(want) {
				t.Fatalf("input %q parsed as %q, want instant %s: %v", tc.input, got, tc.instant, err)
			}
		})
	}
}

func TestDateTimeInputRejectsInvalidDjangoForms(t *testing.T) {
	for _, input := range []string{"", "2026-02-30T12:30", "2026-10-10T12:30garbage", "2026-10-10T24:00Z", "2026-W54-1T12:30Z"} {
		if got, err := parseQueryDateTime(input, "UTC"); err == nil {
			t.Errorf("invalid input %q accepted as %q", input, got)
		}
	}
}

func TestBulkDatesAcceptDjangoUnpaddedCalendarValues(t *testing.T) {
	for _, tc := range []struct {
		start, target string
		valid         bool
	}{
		{"2026-1-2", "2026-1-3", true},
		{"2026-01-2", "2026-1-02", true},
		{"2026-1-3", "2026-1-2", false},
		{"2026-1- 2", "2026-1-3", true},
	} {
		got, err := validateBulkDates(tc.start, tc.target)
		if err != nil || got != tc.valid {
			t.Errorf("dates %q/%q: valid=%v err=%v, want %v", tc.start, tc.target, got, err, tc.valid)
		}
	}
}

func TestBulkDatePreparationRejectsSpacePaddingAfterComparison(t *testing.T) {
	if _, err := prepareBulkDate("2026-1- 2"); err == nil {
		t.Fatal("Django DateField rejects space-padded day during model preparation")
	}
	for _, input := range []any{nil, "2026-1-2"} {
		got, err := prepareBulkDate(input)
		if err != nil {
			t.Fatal(err)
		}
		if input != nil && got != "2026-01-02" {
			t.Fatalf("prepared date=%v", got)
		}
	}
}
