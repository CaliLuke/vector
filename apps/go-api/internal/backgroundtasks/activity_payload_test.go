package backgroundtasks

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActivityPayloadRejectsMalformedCollections(t *testing.T) {
	for _, tc := range []struct {
		name, kind, requested, current string
	}{
		{"object shape", "cycle.activity.created", `{}`, `[]`},
		{"mixed issue identifiers", "cycle.activity.deleted", `{"issues":["issue",42]}`, `{}`},
		{"relation identifiers", "issue_relation.activity.created", `{"issues":42}`, `null`},
		{"cycle records", "cycle.activity.created", `{}`, `{"updated_cycle_issues":[42]}`},
		{"encoded cycle records", "cycle.activity.created", `{}`, `{"created_cycle_issues":"broken"}`},
		{"nested cycle fields", "cycle.activity.created", `{}`, `{"created_cycle_issues":[{"fields":42}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := buildIssueActivityRows(t.Context(), nil, IssueActivityInput{
				Type: tc.kind, RequestedData: json.RawMessage(tc.requested), CurrentInstance: json.RawMessage(tc.current),
			})
			require.ErrorIs(t, err, ErrPermanentIssueActivity)
			require.Empty(t, rows, "malformed input must not produce partial history")
		})
	}
}

func TestActivityPayloadRetainsEncodedCycleRecordsAndOptionalCollections(t *testing.T) {
	rows, err := buildIssueActivityRows(t.Context(), nil, IssueActivityInput{
		Type:            "cycle.activity.created",
		CurrentInstance: json.RawMessage(`{"created_cycle_issues":"[{\"fields\":{\"issue\":\"item-id\"}}]"}`),
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "item-id", *rows[0].IssueID)
	require.Equal(t, "created", rows[0].Verb)
	rows, err = buildIssueActivityRows(t.Context(), nil, IssueActivityInput{
		Type: "cycle.activity.deleted", RequestedData: json.RawMessage(`{"issues":null}`),
	})
	require.NoError(t, err)
	require.Empty(t, rows)
}
