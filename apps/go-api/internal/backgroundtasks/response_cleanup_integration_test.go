//go:build integration

package backgroundtasks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/clue/log"
	"github.com/stretchr/testify/require"
)

type webhookCleanupBody struct {
	io.Reader
	closes int
}

func (body *webhookCleanupBody) Close() error {
	body.closes++
	return errors.New("private response body and credentials")
}

type webhookReadFailure struct{ err error }

func (reader webhookReadFailure) Read([]byte) (int, error) { return 0, reader.err }

func TestWebhookResponseCleanupPreservesDeliveryOutcomeAndSanitizesLogs(t *testing.T) {
	readFailure := errors.New("response read failed")
	for _, scenario := range []struct {
		name      string
		status    int
		readError bool
		wantLogs  int
	}{
		{name: "accepted", status: 200, wantLogs: 1},
		{name: "remote failure recorded", status: 503, wantLogs: 1},
		{name: "body read failed", status: 200, readError: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := newMilestoneWebhookFixture(t)
			id := f.hook(t, true)
			input := webhookDeactivationInput(id)
			body := &webhookCleanupBody{Reader: strings.NewReader("accepted body")}
			if scenario.readError {
				body.Reader = webhookReadFailure{readFailure}
			}
			var output bytes.Buffer
			ctx := log.Context(f.ctx, log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}), log.WithDisableBuffering(func(context.Context) bool { return true }))
			ctx = log.With(ctx, log.KV{K: "request_id", V: "webhook-cleanup"})
			requests := 0
			f.store.requestClient = func(context.Context, string, http.Header, []byte) (*http.Response, error) {
				requests++
				return &http.Response{StatusCode: scenario.status, Header: make(http.Header), Body: body}, nil
			}
			err := f.store.Deliver(ctx, input, 1)
			if scenario.readError {
				require.ErrorIs(t, err, readFailure)
			} else {
				require.NoError(t, err, "cleanup must not make an accepted delivery retryable")
			}
			require.Equal(t, 1, requests)
			require.Equal(t, 1, body.closes)
			var count int
			require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM webhook_logs WHERE webhook=$1`, id).Scan(&count))
			require.Equal(t, scenario.wantLogs, count)
			var active bool
			require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT is_active FROM webhooks WHERE id=$1`, id).Scan(&active))
			require.True(t, active)
			require.NotEmpty(t, output.String(), "response cleanup failures need a correlated diagnostic")
			var event map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &event))
			require.Equal(t, "Response body cleanup failed", event[log.MessageKey])
			require.Equal(t, "webhook-cleanup", event["request_id"])
			require.NotEmpty(t, event["error_type"])
			require.NotContains(t, output.String(), "private")
			require.NotContains(t, output.String(), "credentials")
			require.NotContains(t, output.String(), "accepted body")
		})
	}
}
