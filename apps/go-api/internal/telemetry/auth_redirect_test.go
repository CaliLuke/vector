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
