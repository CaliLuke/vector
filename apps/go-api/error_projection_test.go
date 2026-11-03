package systemapi

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/stretchr/testify/require"
)

func TestErrorProjectionsRejectInvalidJSON(t *testing.T) {
	for name, project := range map[string]func(int, map[string]any) error{
		"cycle": apiCycleError, "module": apiModuleError, "label": apiLabelError,
		"state": apiStateError, "identity": apiUserIdentityError,
		"membership": projectMembershipLifecycleError, "invitations": invitationsMembershipError,
		"activity": activityStatusError, "notification": notificationError,
		"state write": apiStateWriteError, "intake": intakeWriteError, "preferences": preferenceError,
	} {
		t.Run(name, func(t *testing.T) {
			err := project(400, map[string]any{"detail": json.RawMessage(`{"broken":`)})
			var syntax *json.SyntaxError
			require.True(t, errors.As(err, &syntax), "invalid projection must report its cause, not a partial typed response: %v", err)
		})
	}
}

func TestErrorProjectionsPreserveJSONAndClassification(t *testing.T) {
	for name, project := range map[string]func(int, map[string]any) error{
		"cycle": apiCycleError, "module": apiModuleError, "label": apiLabelError,
		"state": apiStateError, "identity": apiUserIdentityError,
		"membership": projectMembershipLifecycleError, "invitations": invitationsMembershipError,
		"activity": activityStatusError, "notification": notificationError,
	} {
		t.Run(name, func(t *testing.T) {
			err := project(403, map[string]any{"detail": map[string]any{"id": json.Number("9007199254740993"), "value": nil}})
			var classified interface{ LoomErrorName() string }
			require.ErrorAs(t, err, &classified)
			require.Equal(t, "status_403", classified.LoomErrorName())
			raw, marshalErr := json.Marshal(err)
			require.NoError(t, marshalErr)
			var result map[string]any
			require.NoError(t, apicontract.DecodeJSON(raw, &result))
			require.Equal(t, map[string]any{"detail": map[string]any{"id": json.Number("9007199254740993"), "value": nil}}, result)
		})
	}
}
