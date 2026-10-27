package backgroundtasks

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWebhookMilestoneRejectsMissingAndInvalidWorkspaceBinding(t *testing.T) {
	for _, raw := range []string{"", "null", "123", `""`, `"not-a-uuid"`, `"00000000-0000-0000-0000-000000000000"`} {
		t.Run(raw, func(t *testing.T) {
			input := WebhookTaskInput{EventID: uuid.NewString(), TaskName: webhookModelActivityTask, Kwargs: map[string]json.RawMessage{
				"model_name": []byte(`"milestone"`), "model_id": []byte(`"` + uuid.NewString() + `"`),
				"slug": []byte(`"milestones"`), "actor_id": []byte(`"` + uuid.NewString() + `"`),
				"requested_data": []byte(`{"title":"One"}`), "current_instance": []byte(`null`),
			}}
			if raw != "" {
				input.Kwargs["workspace_id"] = []byte(raw)
			}
			_, err := (&PostgresWebhookTaskStore{}).ModelActivities(t.Context(), input)
			require.ErrorIs(t, err, ErrPermanentWebhookTask)
			require.ErrorIs(t, validateWebhookModelTask(input), ErrPermanentWebhookTask)
		})
	}
	_, err := (&PostgresWebhookTaskStore{}).Fanout(t.Context(), webhookActivityInput{Event: "issue", Field: []byte(`"milestone"`)})
	require.ErrorIs(t, err, ErrPermanentWebhookTask)
	require.ErrorIs(t, (&PostgresWebhookTaskStore{}).Deliver(t.Context(), webhookSendInput{Event: "milestone"}, 1), ErrPermanentWebhookTask)
}

func TestWebhookWorkspaceBindingRejectsUnsupportedModels(t *testing.T) {
	for _, event := range []string{"user", "unknown"} {
		t.Run(event, func(t *testing.T) {
			_, err := (&PostgresWebhookTaskStore{}).Fanout(t.Context(), webhookActivityInput{Event: event, WorkspaceID: uuid.NewString()})
			require.ErrorIs(t, err, ErrPermanentWebhookTask)
		})
	}
}
