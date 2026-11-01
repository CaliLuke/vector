package notionsync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/CaliLuke/loom/clue/log"
	"github.com/stretchr/testify/require"
)

type syncCleanupBody struct {
	io.Reader
	closes int
}

func (body *syncCleanupBody) Close() error {
	body.closes++
	return errors.New("private remote credentials")
}

type syncCleanupTransport struct {
	body          *syncCleanupBody
	status, calls int
}

func (transport *syncCleanupTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.calls++
	return &http.Response{StatusCode: transport.status, Header: make(http.Header), Body: transport.body, Request: request}, nil
}

func TestSyncResponseCleanupPreservesOutcomeWithoutReplayingWrites(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		status  int
		readErr error
	}{
		{"accepted write", http.StatusCreated, nil},
		{"uncertain write", http.StatusServiceUnavailable, nil},
		{"incomplete response", http.StatusCreated, io.ErrUnexpectedEOF},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			body := &syncCleanupBody{Reader: strings.NewReader(`{"id":"accepted"}`)}
			if scenario.readErr != nil {
				body.Reader = iotest.ErrReader(scenario.readErr)
			}
			transport := &syncCleanupTransport{body: body, status: scenario.status}
			client := newAPI("https://api.example", nil, 0)
			client.client = &http.Client{Transport: transport}
			var output bytes.Buffer
			ctx := log.Context(t.Context(), log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}), log.WithDisableBuffering(func(context.Context) bool { return true }))
			var result Object
			err := client.request(ctx, http.MethodPost, "/pages", map[string]any{"name": "one write"}, &result)
			if scenario.readErr != nil {
				require.ErrorIs(t, err, scenario.readErr)
			} else if scenario.status == http.StatusServiceUnavailable {
				var remote *responseError
				require.ErrorAs(t, err, &remote)
				require.Equal(t, http.StatusServiceUnavailable, remote.Status)
			} else {
				require.NoError(t, err)
				require.Equal(t, "accepted", result["id"])
			}
			require.Equal(t, 1, transport.calls)
			require.Equal(t, 1, body.closes)
			var event map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &event))
			require.NotEmpty(t, event["error_type"])
			require.NotContains(t, output.String(), "private")
			require.NotContains(t, output.String(), "credentials")
		})
	}
}
