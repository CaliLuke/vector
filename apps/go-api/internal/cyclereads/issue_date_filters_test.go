package cyclereads

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCycleDateFiltersKeepRequesterCalendarRules(t *testing.T) {
	for _, test := range []struct {
		field         string
		requesterTime bool
	}{
		{"created_at", true},
		{"updated_at", true},
		{"completed_at", true},
		{"start_date", false},
		{"target_date", false},
	} {
		t.Run(test.field, func(t *testing.T) {
			args := []any{"requester", "scope"}
			predicates := issueFilterPredicates(map[string]string{test.field: "2026-11-01;after"}, &args)
			require.Len(t, predicates, 1)
			require.Contains(t, predicates[0], "issue."+test.field)
			require.Equal(t, test.requesterTime, strings.Contains(predicates[0], "requester_user.user_timezone"))
			require.True(t, strings.HasSuffix(predicates[0], "::date >= $3::date"), predicates[0])
			require.Equal(t, []any{"requester", "scope", "2026-11-01"}, args)
		})
	}
}

func TestCycleDateLookupsKeepLegacyCastsAndArgumentPositions(t *testing.T) {
	for _, test := range []struct {
		name, field, value string
		predicates         []string
		additionalArgs     []any
	}{
		{
			name: "timestamp exact retains database date cast", field: "created_at__exact", value: "2026-11-01",
			predicates: []string{"issue.created_at::date = $3::date"}, additionalArgs: []any{"2026-11-01"},
		},
		{
			name: "date exact preserves supplied value", field: "start_date__exact", value: " 2026-11-01 ",
			predicates: []string{"issue.start_date::date = $3::date"}, additionalArgs: []any{" 2026-11-01 "},
		},
		{
			name: "timestamp range binds one array", field: "updated_at__range", value: "2026-11-01,2026-11-30",
			predicates:     []string{"issue.updated_at::date BETWEEN ($3::text[])[1]::date AND ($3::text[])[2]::date"},
			additionalArgs: []any{[]string{"2026-11-01", "2026-11-30"}},
		},
		{
			name: "date range retains whitespace", field: "target_date__range", value: "2026-11-01, 2026-11-30",
			predicates:     []string{"issue.target_date::date BETWEEN ($3::text[])[1]::date AND ($3::text[])[2]::date"},
			additionalArgs: []any{[]string{"2026-11-01", " 2026-11-30"}},
		},
		{name: "one bound is ignored", field: "target_date__range", value: "2026-11-01"},
		{name: "extra bounds are ignored", field: "target_date__range", value: "2026-11-01,2026-11-02,2026-11-03"},
		{name: "empty value is ignored", field: "created_at__exact"},
		{name: "unsupported timestamp lookup is ignored", field: "completed_at__exact", value: "2026-11-01"},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := []any{"requester", "scope"}
			predicates := issueFilterPredicates(map[string]string{test.field: test.value}, &args)
			require.Equal(t, strings.Join(test.predicates, "\n"), strings.Join(predicates, "\n"))
			require.Equal(t, append([]any{"requester", "scope"}, test.additionalArgs...), args)
		})
	}
}
