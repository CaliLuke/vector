package issuequery_test

import (
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/stretchr/testify/require"
)

func TestPQLRelationshipFunctionsBindRequestCalendarAndEnumValues(t *testing.T) {
	args := []any{"scope"}
	_, err := issuequery.BuildPQL(`cycle IN activeCycle() AND stateGroup NOT IN closedStates()`, issuequery.PQLContext{
		Now: time.Date(2026, 10, 27, 2, 0, 0, 0, time.UTC), Timezone: "America/Los_Angeles",
	}, &args)
	require.NoError(t, err)
	require.Len(t, args, 5)
	require.Equal(t, "scope", args[0])
	require.True(t, time.Date(2026, 10, 27, 6, 59, 59, 999999000, time.UTC).Equal(args[1].(time.Time)))
	require.True(t, time.Date(2026, 10, 26, 7, 0, 0, 0, time.UTC).Equal(args[2].(time.Time)))
	require.Equal(t, []any{"completed", "cancelled"}, args[3:])
}

func TestPQLRejectsInvalidRelationshipOperatorsAndFunctionsAtomically(t *testing.T) {
	for _, query := range []string{
		`stateGroup = "triage"`, `stateGroup = "secret"`, `stateGroup = 1`, `stateGroup ~ "started"`,
		`stateGroup IS EMPTY`, `stateGroup > "started"`, `stateGroup IN openStates(1)`,
		`stateGroup IN (openStates())`, `stateGroup = openStates()`, `stateGroup IN unknown()`,
		`stateGroup IN currentUser()`, `state IN activeStates()`, `state = "secret"`,
		`type ~ "secret"`, `type IS EMPTY`, `type = currentUser()`,
		`label > "secret"`, `label ~ "secret"`, `label BETWEEN "secret" AND "secret"`,
		`label = currentUser()`, `label IN openStates()`, `module IN activeCycle()`,
		`cycle IN activeCycle(1)`, `cycle IN unknown()`, `cycle IN openStates()`,
		`cycle = activeCycle()`, `cycle IN (activeCycle())`, `cycle IN ("secret", activeCycle())`,
		`cycle IN activeCycle(today())`, `cycle IN activeCycle() trailing`, `cycle IN ()`,
		`isOverdue(1)`, `hasNoLabel(1)`, `isEpic(1)`, `isIntake(1)`, `hasStartAndDueDates(1)`,
		`isEpic = true`, `dueDate = isOverdue()`, `priority = hasNoLabel()`,
		`isOverdue() AND isEpic() AND isIntake() AND hasNoLabel() AND isDraft() AND isArchived()`,
	} {
		t.Run(query, func(t *testing.T) {
			backing := []any{"scope", "sentinel", "sentinel"}
			args := backing[:1]
			_, err := issuequery.BuildPQL(`title = "bound first" AND (`+query+`)`, issuequery.PQLContext{Now: time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)}, &args)
			var invalid *issuequery.FilterError
			require.ErrorAs(t, err, &invalid)
			require.Equal(t, "invalid_pql", invalid.Code)
			require.NotContains(t, err.Error(), "secret")
			require.Equal(t, []any{"scope"}, args)
			require.Equal(t, []any{"scope", "sentinel", "sentinel"}, backing)
		})
	}
}

func TestPQLCycleCalendarBoundariesFollowDST(t *testing.T) {
	for _, tc := range []struct {
		name, now, start, end string
	}{
		{"short day", "2026-03-08T12:00:00Z", "2026-03-08T05:00:00Z", "2026-03-09T03:59:59.999999Z"},
		{"long day", "2026-11-01T12:00:00Z", "2026-11-01T04:00:00Z", "2026-11-02T04:59:59.999999Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339Nano, tc.now)
			require.NoError(t, err)
			args := []any{}
			_, err = issuequery.BuildPQL(`cycle IN activeCycle()`, issuequery.PQLContext{Now: now, Timezone: "America/New_York"}, &args)
			require.NoError(t, err)
			require.Len(t, args, 2)
			require.Equal(t, tc.end, args[0].(time.Time).UTC().Format(time.RFC3339Nano))
			require.Equal(t, tc.start, args[1].(time.Time).UTC().Format(time.RFC3339Nano))
		})
	}
}

func TestPQLCycleCalendarRejectsMissingCivilBoundary(t *testing.T) {
	args := []any{"scope"}
	_, err := issuequery.BuildPQL(`cycle IN activeCycle()`, issuequery.PQLContext{
		Now: time.Date(2011, 12, 29, 12, 0, 0, 0, time.UTC), Timezone: "Pacific/Apia",
	}, &args)
	require.Error(t, err)
	require.Equal(t, []any{"scope"}, args)
}

func TestPQLRelationshipFunctionsRequireValidCapturedTime(t *testing.T) {
	for _, query := range []string{`cycle IN activeCycle()`, `cycle IN completedCycles()`, `cycle IN upcomingCycles()`, `isOverdue()`} {
		t.Run(query, func(t *testing.T) {
			args := []any{}
			_, err := issuequery.BuildPQL(query, issuequery.PQLContext{}, &args)
			require.Error(t, err)
			require.Empty(t, args)
		})
	}
}
