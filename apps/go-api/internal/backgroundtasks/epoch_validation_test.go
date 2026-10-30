package backgroundtasks

import (
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIssueActivityEpochValidation(t *testing.T) {
	for _, test := range []struct {
		name  string
		epoch float64
		valid bool
	}{
		{"missing", 0, true}, {"before Unix epoch", -12.5, true}, {"fractional", 123.25, true}, {"largest finite", math.MaxFloat64, true},
		{"NaN", math.NaN(), false}, {"positive infinity", math.Inf(1), false}, {"negative infinity", math.Inf(-1), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			issue := uuid.NewString()
			input := IssueActivityInput{Type: "issue.activity.deleted", ProjectID: uuid.NewString(), IssueID: &issue, Epoch: test.epoch}
			err := validateIssueActivityInput(uuid.NewString(), input)
			if test.valid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrPermanentIssueActivity)
				// Neither worker may access PostgreSQL or Redis for invalid input.
				store := &PostgresIssueActivityStore{}
				ids, err := store.RecordIssueActivity(t.Context(), uuid.NewString(), input)
				require.ErrorIs(t, err, ErrPermanentIssueActivity)
				require.Empty(t, ids)
				require.ErrorIs(t, store.ProcessIssueNotifications(t.Context(), uuid.NewString(), input, []string{uuid.NewString()}), ErrPermanentIssueActivity)
			}
		})
	}
}
