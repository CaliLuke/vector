//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/CaliLuke/selvedge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type propertyWorkerEvent struct {
	id, kind string
	payload  json.RawMessage
}

func propertyWorkerEvents(t *testing.T, f milestoneFixture, cases []propertyWorkerCase) []propertyWorkerEvent {
	t.Helper()
	rows, err := f.pool.Query(t.Context(), `SELECT id::text,kind,payload FROM work_item_effects ORDER BY created_at,id`)
	require.NoError(t, err)
	defer rows.Close()
	var events []propertyWorkerEvent
	changes := map[string]int{}
	for rows.Next() {
		var event propertyWorkerEvent
		require.NoError(t, rows.Scan(&event.id, &event.kind, &event.payload))
		require.Contains(t, []string{"activity", "webhook"}, event.kind)
		if event.kind == "activity" {
			input := propertyWorkerActivity(t, event.payload)
			var requested map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(input.RequestedData, &requested))
			if input.Type == "issue.activity.updated" && requested["custom_properties"] != nil {
				require.NotNil(t, input.IssueID)
				changes[*input.IssueID]++
			}
		}
		events = append(events, event)
	}
	require.NoError(t, rows.Err())
	for _, value := range cases {
		require.Equal(t, 3, changes[value.itemID], "each committed value transition queues its native activity")
	}
	return events
}

func propertyWorkerActivity(t *testing.T, payload json.RawMessage) backgroundtasks.IssueActivityInput {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(payload, &fields))
	// Legacy producer envelopes carry these JSON documents as strings.
	for _, key := range []string{"requested_data", "current_instance"} {
		var document string
		if len(fields[key]) > 0 && fields[key][0] == '"' {
			require.NoError(t, json.Unmarshal(fields[key], &document))
			fields[key] = json.RawMessage(document)
		}
	}
	raw, err := json.Marshal(fields)
	require.NoError(t, err)
	var input backgroundtasks.IssueActivityInput
	require.NoError(t, json.Unmarshal(raw, &input))
	return input
}

func waitPropertyWorkerEvents(t *testing.T, ctx context.Context, f milestoneFixture, runtime *backgroundtasks.Runtime, events []propertyWorkerEvent) {
	t.Helper()
	ids := make([]string, len(events))
	for i, event := range events {
		ids[i] = event.id
	}
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		var admitted int
		err := f.pool.QueryRow(ctx, `SELECT count(*) FROM work_item_effects WHERE id=ANY($1::uuid[]) AND admitted_at IS NOT NULL AND last_error=''`, ids).Scan(&admitted)
		assert.NoError(collect, err)
		assert.Equal(collect, len(events), admitted)
	}, 30*time.Second, 25*time.Millisecond)
	for _, event := range events {
		var workflow string
		var err error
		if event.kind == "activity" {
			workflow, err = runtime.IssueActivities.Admit(ctx, event.id, propertyWorkerActivity(t, event.payload))
		} else {
			var task struct {
				Name   string
				Args   []json.RawMessage
				Kwargs map[string]json.RawMessage
			}
			require.NoError(t, json.Unmarshal(event.payload, &task))
			workflow, err = runtime.Webhooks.Admit(ctx, backgroundtasks.WebhookTaskInput{EventID: event.id, TaskName: task.Name, Args: task.Args, Kwargs: task.Kwargs})
		}
		require.NoError(t, err)
		result, err := selvedge.WaitWorkflow[struct{}](ctx, runtime.App, workflow)
		require.NoError(t, err)
		require.Equal(t, "completed", result.Status, result.Error)
	}
}
