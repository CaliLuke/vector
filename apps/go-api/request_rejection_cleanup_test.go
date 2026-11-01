package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	externalapi "github.com/CaliLuke/go-plane/externalintegrations/gen/external_integrations"
	feedbackapi "github.com/CaliLuke/go-plane/gen/public_feedback"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/stretchr/testify/require"
)

func TestRequestRejectionReportsOwnedBodyCleanupFailure(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		invoke func(context.Context, io.ReadCloser) (io.ReadCloser, error)
	}{
		{"deferred feedback cleanup", func(ctx context.Context, body io.ReadCloser) (io.ReadCloser, error) {
			_, response, err := (&publicFeedbackService{}).VoteCreate(ctx, &feedbackapi.VoteCreatePayload{}, body)
			return response, err
		}},
		{"immediate integration cleanup", func(ctx context.Context, body io.ReadCloser) (io.ReadCloser, error) {
			_, response, err := (&externalIntegrationsService{}).WorkspaceAiAssistant(ctx, &externalapi.WorkspaceAiAssistantPayload{}, body)
			return response, err
		}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			var output bytes.Buffer
			ctx := log.Context(t.Context(), log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}), log.WithDisableBuffering(func(context.Context) bool { return true }))
			body := &earlyCleanupBody{}
			response, err := scenario.invoke(ctx, body)
			if response != nil {
				testsupport.Close(t, response)
			}
			var failure interface{ LoomErrorName() string }
			require.ErrorAs(t, err, &failure)
			require.Equal(t, "status_401", failure.LoomErrorName())
			require.Zero(t, body.reads)
			require.Equal(t, 1, body.closes)
			var event map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &event))
			require.NotEmpty(t, event["error_type"])
			require.NotContains(t, output.String(), "private")
		})
	}
}
