package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	api "github.com/CaliLuke/go-plane/externalintegrations/gen/external_integrations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/stretchr/testify/require"
)

type unsplashCleanupBody struct {
	io.Reader
	closes int
}

func (body *unsplashCleanupBody) Close() error {
	body.closes++
	return errors.New("private upstream credential")
}

func TestUnsplashResponseCleanupPreservesPayloadAndRemoteFailure(t *testing.T) {
	for _, status := range []int{200, 429} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			payload := `{"results":[{"id":"photo"}]}`
			upstream := &unsplashCleanupBody{Reader: strings.NewReader(payload)}
			calls := 0
			client := &http.Client{Transport: externalIntegrationsRoundTripper(func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: upstream}, nil
			})}
			service, err := NewExternalIntegrations(externalIntegrationsTestRepository{configuration: map[string]string{"UNSPLASH_ACCESS_KEY": "fixture"}}, externalIntegrationsTestSessions{}, client)
			require.NoError(t, err)
			var output bytes.Buffer
			ctx := log.Context(t.Context(), log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}), log.WithDisableBuffering(func(context.Context) bool { return true }))
			_, body, err := service.Unsplash(ctx, &api.UnsplashPayload{})
			if status == 200 {
				require.NoError(t, err)
				data, readErr := io.ReadAll(body)
				testsupport.Close(t, body)
				require.NoError(t, readErr)
				require.JSONEq(t, payload, string(data))
			} else {
				var failure *api.ExternalIntegrationErrorResponse
				require.ErrorAs(t, err, &failure)
				require.Equal(t, "status_429", failure.LoomErrorName())
				require.JSONEq(t, payload, string(failure.Body))
			}
			require.Equal(t, 1, calls)
			require.Equal(t, 1, upstream.closes)
			require.NotEmpty(t, output.String())
			var event map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &event))
			require.Equal(t, "Response body cleanup failed", event[log.MessageKey])
			require.NotContains(t, output.String(), "private")
			require.NotContains(t, output.String(), "credential")
		})
	}
}
