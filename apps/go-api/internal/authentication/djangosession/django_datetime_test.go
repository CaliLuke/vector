package djangosession

import (
	"testing"
	"time"
)

func TestDjangoDateTimeFromISOFormatMatchesPython312AwareForms(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		value string
		want  time.Time
	}{
		{
			name: "basic calendar and clock", value: "20260901T120102+0000",
			want: time.Date(2026, time.September, 1, 12, 1, 2, 0, time.UTC),
		},
		{
			name: "extended week date and arbitrary separator", value: "2026-W36-2🐍12:01:02Z",
			want: time.Date(2026, time.September, 1, 12, 1, 2, 0, time.UTC),
		},
		{
			name: "basic week without weekday", value: "2026W36 120102+00",
			want: time.Date(2026, time.August, 31, 12, 1, 2, 0, time.UTC),
		},
		{
			name: "offset seconds and fractional seconds", value: "20260901x12:01:02+01:02:03.5",
			want: time.Date(2026, time.September, 1, 10, 58, 58, 500_000_000, time.UTC),
		},
		{
			name: "fraction truncates to microseconds", value: "2026-09-01T12:01:02.123456789+00:00",
			want: time.Date(2026, time.September, 1, 12, 1, 2, 123_456_000, time.UTC),
		},
		{
			name: "fraction after minutes", value: "2026-09-01T12:01.5+00:00",
			want: time.Date(2026, time.September, 1, 12, 1, 0, 500_000_000, time.UTC),
		},
		{
			name: "fraction after hours and offset minutes", value: "2026-09-01T12.5+01:02.5",
			want: time.Date(2026, time.September, 1, 10, 58, 0, 0, time.UTC),
		},
		{
			name: "omitted minutes after colon", value: "2030-01-01T12:+00:00",
			want: time.Date(2030, time.January, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "omitted seconds after colon", value: "2030-01-01T12:30:+00:00",
			want: time.Date(2030, time.January, 1, 12, 30, 0, 0, time.UTC),
		},
		{
			name: "normalized offset minutes", value: "2030-01-01T00:00+00:60",
			want: time.Date(2029, time.December, 31, 23, 0, 0, 0, time.UTC),
		},
		{
			name: "normalized offset seconds", value: "2030-01-01T00:00+00:00:60",
			want: time.Date(2029, time.December, 31, 23, 59, 0, 0, time.UTC),
		},
		{
			name: "positive fractional zero offset", value: "2030-01-01T00:00+00:00:00.5",
			want: time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "negative fractional zero offset", value: "2030-01-01T00:00-00.5",
			want: time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "empty fraction after hours", value: "2030-01-01T12.+00:00",
			want: time.Date(2030, time.January, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "empty fraction after minutes", value: "2030-01-01T12:30,+00:00",
			want: time.Date(2030, time.January, 1, 12, 30, 0, 0, time.UTC),
		},
		{
			name: "empty fraction after seconds", value: "2030-01-01T12:30:45.+00:00",
			want: time.Date(2030, time.January, 1, 12, 30, 45, 0, time.UTC),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := djangoDateTimeFromISOFormat(testCase.value)
			if err != nil || !got.Equal(testCase.want) {
				t.Fatalf("djangoDateTimeFromISOFormat(%q)=%v err=%v want=%v", testCase.value, got, err, testCase.want)
			}
		})
	}
}

func TestDjangoDateTimeFromISOFormatRejectsValuesThatFailAwareArithmetic(t *testing.T) {
	for _, value := range []string{
		"2026-09-01",
		"2026-09-01T12:01:02",
		"2026-W54-1T12:01:02Z",
		"2026-09-01T12:01:02+24:00",
		"2030-01-01T12:00+01.",
	} {
		if _, err := djangoDateTimeFromISOFormat(value); err == nil {
			t.Fatalf("djangoDateTimeFromISOFormat(%q) unexpectedly succeeded", value)
		}
	}
}
