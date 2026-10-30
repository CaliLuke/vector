package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/clue/log"
	"github.com/stretchr/testify/require"
)

type requestBodyCloser struct {
	calls int
	err   error
}

func (body *requestBodyCloser) Close() error { body.calls++; return body.err }

func TestCloseRequestBodyReportsFailureWithoutPrivateDetails(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		failure   error
		cancelled bool
		deadline  bool
	}{
		{name: "closed"},
		{name: "failed", failure: errors.New("private body contents")},
		{name: "cancelled", failure: fmt.Errorf("private body contents: %w", context.Canceled), cancelled: true},
		{name: "deadline", failure: fmt.Errorf("private body contents: %w", context.DeadlineExceeded), deadline: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			var output bytes.Buffer
			ctx := log.Context(t.Context(), log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}),
				log.WithDisableBuffering(func(context.Context) bool { return true }))
			ctx = log.With(ctx, log.KV{K: "request_id", V: "body-cleanup"})
			body := &requestBodyCloser{err: scenario.failure}

			CloseRequestBody(ctx, body)

			require.Equal(t, 1, body.calls)
			if scenario.failure == nil {
				require.Empty(t, output.String())
				return
			}
			var event map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &event))
			require.Equal(t, "body-cleanup", event["request_id"])
			require.NotEmpty(t, event["error_type"])
			require.Equal(t, scenario.cancelled, event["context_canceled"])
			require.Equal(t, scenario.deadline, event["deadline_exceeded"])
			require.False(t, strings.Contains(output.String(), "private"))
		})
	}
}

func TestCloseRequestBodyAcceptsAbsentInput(t *testing.T) {
	var output bytes.Buffer
	ctx := log.Context(t.Context(), log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}),
		log.WithDisableBuffering(func(context.Context) bool { return true }))
	CloseRequestBody(ctx, nil)
	require.Empty(t, output.String())
}
