package bootstrap

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

type releaseCleanupTransport func(*http.Request) (*http.Response, error)

func (transport releaseCleanupTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

type releaseCleanupBody struct {
	io.Reader
	closes int
}

func (body *releaseCleanupBody) Close() error {
	body.closes++
	return errors.New("private release response")
}

func TestLatestReleaseCleanupPreservesVersionAndFallback(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	for _, scenario := range []struct {
		name, body, want string
		status           int
	}{
		{name: "accepted", status: 200, body: `{"tag_name":"v1.2.3"}`, want: "v1.2.3"},
		{name: "invalid JSON", status: 200, body: `{`, want: "fallback"},
		{name: "remote failure", status: 503, body: `unavailable`, want: "fallback"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			body := &releaseCleanupBody{Reader: strings.NewReader(scenario.body)}
			http.DefaultTransport = releaseCleanupTransport(func(request *http.Request) (*http.Response, error) {
				require.Equal(t, "https://api.github.com/repos/CaliLuke/vector/releases/latest", request.URL.String())
				return &http.Response{StatusCode: scenario.status, Header: make(http.Header), Body: body}, nil
			})
			var output bytes.Buffer
			ctx := log.Context(t.Context(), log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}), log.WithDisableBuffering(func(context.Context) bool { return true }))
			require.Equal(t, scenario.want, latestReleaseVersion(ctx, "fallback"))
			require.Equal(t, 1, body.closes)
			require.NotEmpty(t, output.String())
			var event map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &event))
			require.Equal(t, "Response body cleanup failed", event[log.MessageKey])
			require.NotContains(t, output.String(), "private")
		})
	}
}
