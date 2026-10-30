//go:build integration

package projectstatereads

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStateNativeNonfiniteFieldsLeaveRowsAndRevisionsUnchanged(t *testing.T) {
	f := newStateMutationFixture(t)
	state := f.create(t, "Existing state")
	before := f.detail(t, state, true)
	for _, route := range []struct {
		name, field string
		app         bool
	}{
		{"API sequence", "sequence", false}, {"app sequence", "sequence", true}, {"app order", "order", true},
	} {
		for _, method := range []string{"POST", "PATCH"} {
			for _, value := range []string{"NaN", "Infinity", "-Infinity", "1e999"} {
				t.Run(route.name+"/"+method+"/"+value, func(t *testing.T) {
					input := f.input
					input.Method, input.App = method, route.app
					input.Fields = map[string]any{"name": "Rejected rename", "color": "#123456", "default": true, route.field: value}
					var err error
					if method == "POST" {
						_, err = f.repository.Create(t.Context(), input)
					} else {
						input.StateID = state
						_, err = f.repository.Update(t.Context(), input)
					}
					var failure *WriteFailure
					require.ErrorAs(t, err, &failure)
					require.Equal(t, 400, failure.Status)
					require.Equal(t, map[string][]string{route.field: {"A valid number is required."}}, failure.Body)
					require.Equal(t, before, f.detail(t, state, true))
					var states, effects, receipts int
					require.NoError(t, f.db.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM states),(SELECT count(*) FROM work_item_effects),(SELECT count(*) FROM mutation_receipts)`).Scan(&states, &effects, &receipts))
					require.Equal(t, 1, states)
					require.Zero(t, effects)
					require.Zero(t, receipts)
				})
			}
		}
	}
}

func TestStateNativeFiniteFractionalSequenceStillPersists(t *testing.T) {
	f := newStateMutationFixture(t)
	state := f.create(t, "Fractional")
	input := f.input
	input.StateID = state
	input.Fields = map[string]any{"sequence": json.Number("-0.25")}
	_, err := f.repository.Update(t.Context(), input)
	require.NoError(t, err)
	var sequence float64
	require.NoError(t, f.db.QueryRow(t.Context(), `SELECT sequence FROM states WHERE id=$1`, state).Scan(&sequence))
	require.Equal(t, -0.25, sequence)
}
