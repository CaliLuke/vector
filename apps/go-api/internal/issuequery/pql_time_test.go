package issuequery_test

import (
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/stretchr/testify/require"
)

func TestPQLCalendarFunctionsUseOneRequesterInstant(t *testing.T) {
	now := time.Date(2024, 3, 31, 18, 34, 56, 123456789, time.UTC)
	for _, tc := range []struct{ function, want string }{
		{"now()", "2024-03-31T11:34:56.123456-07:00"},
		{"today()", "2024-03-31T00:00:00-07:00"}, {"startOfDay()", "2024-03-31T00:00:00-07:00"},
		{"endOfDay()", "2024-03-31T23:59:59.999999-07:00"},
		{"startOfWeek()", "2024-03-25T00:00:00-07:00"}, {"endOfWeek()", "2024-03-31T23:59:59.999999-07:00"},
		{"startOfMonth()", "2024-03-01T00:00:00-08:00"}, {"endOfMonth()", "2024-03-31T23:59:59.999999-07:00"},
		{"startOfYear()", "2024-01-01T00:00:00-08:00"}, {"endOfYear()", "2024-12-31T23:59:59.999999-08:00"},
		{"daysAgo(1)", "2024-03-30T11:34:56.123456-07:00"}, {"daysFromNow(1)", "2024-04-01T11:34:56.123456-07:00"},
		{"weeksAgo(4)", "2024-03-03T11:34:56.123456-08:00"}, {"weeksFromNow(1)", "2024-04-07T11:34:56.123456-07:00"},
		{"monthsAgo(1)", "2024-02-29T11:34:56.123456-08:00"}, {"monthsFromNow(1)", "2024-04-30T11:34:56.123456-07:00"},
	} {
		t.Run(tc.function, func(t *testing.T) {
			args := []any{}
			_, err := issuequery.BuildPQL("createdAt >= "+tc.function, issuequery.PQLContext{Timezone: "America/Los_Angeles", Now: now}, &args)
			require.NoError(t, err)
			require.Equal(t, tc.want, args[0].(time.Time).Format(time.RFC3339Nano))
		})
	}
}

func TestPQLDateTypingAndMicroseconds(t *testing.T) {
	for _, tc := range []struct {
		query, zone string
		want        any
	}{
		{`dueDate >= now()`, "Pacific/Honolulu", "2026-10-26"},
		{`dueDate = "2024-02-29"`, "Pacific/Honolulu", "2024-02-29"},
		{`dueDate = "0001-01-01"`, "UTC", "0001-01-01"},
		{`createdAt = "2026-10-27"`, "America/Los_Angeles", time.Date(2026, 10, 27, 7, 0, 0, 0, time.UTC)},
		{`createdAt = "2026-10-27T03:00:00.123456789+02:00"`, "", time.Date(2026, 10, 27, 1, 0, 0, 123456000, time.UTC)},
	} {
		t.Run(tc.query, func(t *testing.T) {
			args := []any{}
			_, err := issuequery.BuildPQL(tc.query, issuequery.PQLContext{Timezone: tc.zone, Now: time.Date(2026, 10, 27, 1, 0, 0, 0, time.UTC)}, &args)
			require.NoError(t, err)
			if timestamp, ok := tc.want.(time.Time); ok {
				require.True(t, timestamp.Equal(args[0].(time.Time)))
			} else {
				require.Equal(t, tc.want, args[0])
			}
		})
	}
}

func TestPQLRejectsRelativeDatesOutsideCalendarRange(t *testing.T) {
	args := []any{}
	_, err := issuequery.BuildPQL(`dueDate = monthsAgo(10000)`, issuequery.PQLContext{Now: time.Date(1, 1, 2, 0, 0, 0, 0, time.UTC)}, &args)
	require.Error(t, err)
	require.Empty(t, args)
}

func TestPQLRejectsCivilGapsAndChoosesEarlierFold(t *testing.T) {
	for _, tc := range []struct {
		query, zone string
		now         time.Time
	}{
		{`createdAt = daysFromNow(1)`, "America/Los_Angeles", time.Date(2026, 3, 7, 10, 30, 0, 0, time.UTC)},
		{`createdAt = "2011-12-30"`, "Pacific/Apia", time.Time{}},
		{`createdAt = startOfDay()`, "America/Sao_Paulo", time.Date(2018, 11, 4, 12, 0, 0, 0, time.UTC)},
	} {
		t.Run(tc.zone, func(t *testing.T) {
			args := []any{}
			_, err := issuequery.BuildPQL(tc.query, issuequery.PQLContext{Timezone: tc.zone, Now: tc.now}, &args)
			require.ErrorContains(t, err, "local time does not exist")
			require.Empty(t, args)
		})
	}
	args := []any{}
	_, err := issuequery.BuildPQL(`createdAt = daysFromNow(1)`, issuequery.PQLContext{Timezone: "America/Los_Angeles", Now: time.Date(2026, 10, 31, 8, 30, 0, 0, time.UTC)}, &args)
	require.NoError(t, err)
	require.Equal(t, "2026-11-01T01:30:00-07:00", args[0].(time.Time).Format(time.RFC3339))
}
