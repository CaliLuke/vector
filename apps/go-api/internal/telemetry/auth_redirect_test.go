package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/CaliLuke/loom/clue/log"
	"github.com/stretchr/testify/require"
)

func TestAuthRedirectFailureClassifiesErrorsWithoutLeakingDetails(t *testing.T) {
	for _, test := range []struct {
		name, level, class string
		cause              error
	}{
		{"canceled", "info", "context_canceled", context.Canceled},
		{"deadline", "error", "deadline_exceeded", context.DeadlineExceeded},
		{"internal", "error", "internal", errors.New("dependency unavailable")},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			ctx := log.Context(t.Context(),
				log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}),
				log.WithDisableBuffering(func(context.Context) bool { return true }),
			)
			AuthRedirectFailure(ctx, "create_session", fmt.Errorf("private dependency data: %w", test.cause))

			var record map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &record))
			require.Equal(t, "Authentication redirect failed", record["msg"])
			require.Equal(t, test.level, record["level"])
			require.Equal(t, "create_session", record["auth.stage"])
			require.Equal(t, test.class, record["auth.error_class"])
			require.NotContains(t, output.String(), "private dependency data")
		})
	}
}

func TestPasswordSignInOutcomeDistinguishesRejectionFromSessionCreation(t *testing.T) {
	for _, test := range []struct {
		name           string
		code           int
		reason         string
		sessionCreated bool
		outcome        string
	}{
		{name: "rejected", code: 5065, reason: "AUTHENTICATION_FAILED_SIGN_IN", outcome: "rejected"},
		{name: "authenticated", sessionCreated: true, outcome: "session_created"},
		{name: "internal failure"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			ctx := log.Context(t.Context(),
				log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}),
				log.WithDisableBuffering(func(context.Context) bool { return true }),
			)
			PasswordSignInOutcome(ctx, test.code, test.reason, test.sessionCreated)

			if test.outcome == "" {
				require.Empty(t, output.String())
				return
			}
			var record map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &record))
			require.Equal(t, test.outcome, record["auth.outcome"])
			if test.code != 0 {
				require.Equal(t, float64(test.code), record["auth.rejection_code"])
				require.Equal(t, test.reason, record["auth.rejection_reason"])
			} else {
				require.NotContains(t, record, "auth.rejection_code")
			}
		})
	}
}
