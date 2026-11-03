package backgroundtasks

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
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

func TestEmailMentionsKeepUnknownUsersAndContinue(t *testing.T) {
	const known = "11111111-1111-4111-8111-111111111111"
	const unknown = "22222222-2222-4222-8222-222222222222"
	input := `<p>before <mention-component entity_identifier="broken">invalid</mention-component> <mention-component entity_identifier="` + unknown + `">unknown</mention-component> <mention-component entity_identifier="` + known + `">old</mention-component> after</p>`
	got, err := replaceEmailMentions(input, map[uuid.UUID]string{uuid.MustParse(known): "Ada", uuid.Nil: "must not match malformed IDs"})
	require.NoError(t, err)
	require.Equal(t, "before invalid unknown @Ada after", got)
}
