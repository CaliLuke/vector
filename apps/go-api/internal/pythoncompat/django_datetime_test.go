package pythoncompat

import (
	"testing"
	"time"
)

func TestPythonDateTimeString(t *testing.T) {
	for _, test := range []struct {
		name string
		at   time.Time
		want string
	}{
		{
			name: "no fractional seconds",
			at:   time.Date(2026, time.January, 15, 12, 0, 0, 0, time.FixedZone("UTC", 0)),
			want: "2026-01-15 12:00:00+00:00",
		},
		{
			name: "microseconds",
			at:   time.Date(2026, time.January, 15, 12, 0, 0, 123456000, time.FixedZone("UTC", 0)),
			want: "2026-01-15 12:00:00.123456+00:00",
		},
		{
			name: "truncate sub-microsecond precision",
			at:   time.Date(2026, time.January, 15, 12, 0, 0, 123456789, time.UTC),
			want: "2026-01-15 12:00:00.123456+00:00",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := PythonDateTimeString(test.at); got != test.want {
				t.Fatalf("PythonDateTimeString() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDRFDateTimeString(t *testing.T) {
	kathmandu, err := time.LoadLocation("Asia/Kathmandu")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		at       time.Time
		location *time.Location
		want     string
	}{
		{
			name:     "zero fraction",
			at:       time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC),
			location: time.UTC,
			want:     "2026-01-15T12:00:00Z",
		},
		{
			name:     "fraction retains six digits including trailing zeroes",
			at:       time.Date(2026, time.January, 15, 12, 0, 0, 123400000, time.UTC),
			location: time.UTC,
			want:     "2026-01-15T12:00:00.123400Z",
		},
		{
			name:     "sub-microsecond precision is truncated",
			at:       time.Date(2026, time.January, 15, 12, 0, 0, 123456789, time.UTC),
			location: time.UTC,
			want:     "2026-01-15T12:00:00.123456Z",
		},
		{
			name:     "actor timezone offset",
			at:       time.Date(2026, time.January, 15, 12, 0, 0, 123400000, time.UTC),
			location: kathmandu,
			want:     "2026-01-15T17:45:00.123400+05:45",
		},
		{
			name:     "truncated fraction becomes zero",
			at:       time.Date(2026, time.January, 15, 12, 0, 0, 999, time.UTC),
			location: time.UTC,
			want:     "2026-01-15T12:00:00Z",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := DRFDateTimeString(test.at, test.location); got != test.want {
				t.Fatalf("DRFDateTimeString() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDjangoDateTimeFallback(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"2026-1-2 3:4", "2026-01-02T03:04:00Z"},
		{"2026-1-2T3:4:5.123456789012 +0530", "2026-01-01T21:34:05.123456Z"},
		{"٢٠٢٦-١-٢T٣:٤:٥Z", "2026-01-02T03:04:05Z"},
		{"2026-1-2T3:4:5\u2003Z", "2026-01-02T03:04:05Z"},
		{"2026-1-2T3:4Z\n", "2026-01-02T03:04:00Z"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := DjangoDateTime(tc.input, time.UTC)
			if err != nil || got.Format(time.RFC3339Nano) != tc.want {
				t.Fatalf("got %s, %v; want %s", got.Format(time.RFC3339Nano), err, tc.want)
			}
		})
	}
	for _, input := range []string{"2026-2-30T3:4Z", "2026-1-2T24:4Z", "2026-1-2T3:60Z", "2026-1-2T3:4:5.1234567890123Z", "2026-1-2T3:4Zx"} {
		if got, err := DjangoDateTime(input, time.UTC); err == nil {
			t.Errorf("accepted %q as %v", input, got)
		}
	}
}
