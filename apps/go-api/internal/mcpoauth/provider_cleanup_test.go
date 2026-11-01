package mcpoauth_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	oauth "github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/stretchr/testify/require"
)

type introspectionCleanupBody struct {
	io.ReadCloser
	closes *int
}

func (body *introspectionCleanupBody) Close() error {
	*body.closes++
	return errors.Join(body.ReadCloser.Close(), errors.New("private issuer credential"))
}

type introspectionCleanupTransport struct {
	base   http.RoundTripper
	closes int
}

func (transport *introspectionCleanupTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := transport.base.RoundTrip(request)
	if err == nil && request.URL.Path == "/introspect" {
		response.Body = &introspectionCleanupBody{ReadCloser: response.Body, closes: &transport.closes}
	}
	return response, err
}

func TestIntrospectionRejectsIssuerCleanupFailureWithoutDisclosingPrivateDetails(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		active bool
		status int
	}{
		{"active", true, http.StatusOK},
		{"revoked", false, http.StatusOK},
		{"unavailable", true, http.StatusServiceUnavailable},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			issuer := newIssuer(t)
			issuer.claims["active"], issuer.status = scenario.active, scenario.status
			client := *issuer.server.Client()
			transport := &introspectionCleanupTransport{base: client.Transport}
			client.Transport = transport
			provider, err := oauth.NewProvider(t.Context(), issuer.config(), &client)
			require.NoError(t, err)
			var output bytes.Buffer
			ctx := log.Context(t.Context(), log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}), log.WithDisableBuffering(func(context.Context) bool { return true }))
			identity, err := provider.Introspect(ctx, "access-user")
			require.ErrorIs(t, err, oauth.ErrUnavailable)
			require.Empty(t, identity)
			require.NotContains(t, err.Error(), "private")
			require.Equal(t, 1, issuer.calls)
			require.Equal(t, 1, transport.closes)
			require.NotContains(t, output.String(), "private")
			require.NotContains(t, output.String(), "credential")
		})
	}
}
