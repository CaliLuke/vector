package workitems

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppDateFiltersRejectRelativeDurationOverflow(t *testing.T) {
	for _, value := range []string{
		strings.Repeat("9", 40) + "_weeks;after;fromnow",
		"9223372036854775807_weeks;before;fromnow",
		"9223372036854775807_months;after;fromnow",
		"142857143_weeks;after;fromnow",
		"33333334_months;before;fromnow",
	} {
		t.Run(value, func(t *testing.T) {
			_, _, err := BuildAppIssueFilters(map[string]string{"start_date": value}, nil, []string{"tenant_id = $1"}, []any{"owned-tenant"})
			require.Error(t, err, "a relative duration must not wrap into a different date range")
		})
	}
}
