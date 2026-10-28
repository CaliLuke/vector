package backgroundtasks

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTemplatedEmailRejectsInvalidArgumentsBeforeLoadingRows(t *testing.T) {
	cases := []struct {
		task  string
		count int
	}{
		{forgotPasswordTask, 5}, {magicLinkTask, 3},
		{emailUpdateMagicCodeTask, 2}, {emailUpdateConfirmationTask, 1},
		{userActivationEmailTask, 2}, {userDeactivationEmailTask, 2},
		{projectAddUserEmailTask, 3}, {webhookDeactivationEmailTask, 4},
		{workspaceInvitationEmailTask, 5}, {projectInvitationEmailTask, 5},
	}
	// No database is configured: invalid payloads must be rejected before any lookup.
	store := &templatedEmailStore{}
	for _, test := range cases {
		t.Run(test.task, func(t *testing.T) {
			args := make([]json.RawMessage, test.count)
			for i := range args {
				args[i] = json.RawMessage(`"valid"`)
			}
			for i := range args {
				original := args[i]
				args[i] = json.RawMessage(`42`)
				prepared, err := store.prepare(t.Context(), templatedEmailTaskInput{TaskName: test.task, Args: args})
				require.Nil(t, prepared)
				require.ErrorContains(t, err, fmt.Sprintf("%s argument %d must be a string", test.task, i+1))
				args[i] = original
			}
			args[0] = json.RawMessage(`42`)
			_, err := store.prepare(t.Context(), templatedEmailTaskInput{TaskName: test.task, Args: args[:len(args)-1]})
			require.EqualError(t, err, fmt.Sprintf("%s received %d arguments, want %d", test.task, test.count-1, test.count))
		})
	}
	_, err := store.prepare(t.Context(), templatedEmailTaskInput{TaskName: "unknown", Args: []json.RawMessage{json.RawMessage(`42`)}})
	require.EqualError(t, err, `unsupported templated email task "unknown"`)
}

func TestTemplatedEmailPreparationPreservesPayloadValues(t *testing.T) {
	store := &templatedEmailStore{}
	input := templatedEmailTaskInput{TaskName: forgotPasswordTask, Args: []json.RawMessage{
		json.RawMessage(`null`), json.RawMessage(`"person@example.test"`), json.RawMessage(`"user-id"`),
		json.RawMessage(`"token+value"`), json.RawMessage(`"https://example.test"`),
	}}
	prepared, err := store.prepare(t.Context(), input)
	require.NoError(t, err)
	require.Equal(t, &preparedEmail{
		To: "person@example.test", Subject: "A new password to your Vector account has been requested",
		Template: "auth/forgot_password.html", Values: map[string]string{
			"first_name": "", "email": "person@example.test",
			"forgot_password_url": "https://example.test/accounts/reset-password/?uidb64=user-id&token=token+value&email=person@example.test",
		},
	}, prepared)
}
