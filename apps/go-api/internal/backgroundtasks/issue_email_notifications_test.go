package backgroundtasks

import (
	"testing"
	"time"
)

func TestNextIssueEmailBeatUsesUTCFiveMinuteBoundaries(t *testing.T) {
	for _, test := range []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{name: "before boundary", now: time.Date(2026, 10, 22, 12, 4, 59, 0, time.UTC), want: time.Date(2026, 10, 22, 12, 5, 0, 0, time.UTC)},
		{name: "on boundary", now: time.Date(2026, 10, 22, 12, 5, 0, 0, time.UTC), want: time.Date(2026, 10, 22, 12, 10, 0, 0, time.UTC)},
		{name: "normalizes offset", now: time.Date(2026, 10, 22, 5, 1, 0, 0, time.FixedZone("PDT", -7*60*60)), want: time.Date(2026, 10, 22, 12, 5, 0, 0, time.UTC)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := nextIssueEmailBeat(test.now); !got.Equal(test.want) {
				t.Fatalf("next email Beat tick = %s, want %s", got, test.want)
			}
		})
	}
}
