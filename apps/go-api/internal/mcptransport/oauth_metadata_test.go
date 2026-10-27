package mcptransport_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"github.com/stretchr/testify/require"
)

func TestOAuthMetadataAdvertisesOnlyConfiguredResource(t *testing.T) {
	h := handler(t, config("oauth"), options())
	r := httptest.NewRequest(http.MethodGet, "https://vector.example"+mcptransport.MetadataPath, nil)
	r.Header.Set("Origin", "https://client.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	require.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	var metadata oauthex.ProtectedResourceMetadata
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &metadata))
	require.Equal(t, "https://vector.example/mcp", metadata.Resource)
	require.Equal(t, []string{"https://issuer.example"}, metadata.AuthorizationServers)
	require.Equal(t, []string{"vector:read", "vector:write"}, metadata.ScopesSupported)
	require.Equal(t, []string{"header"}, metadata.BearerMethodsSupported)
}

func TestOAuthMetadataIsPublicButStrictlyRouted(t *testing.T) {
	for _, test := range []struct {
		name, method, target, mode string
		want                       int
	}{
		{"preflight", http.MethodOptions, "https://vector.example" + mcptransport.MetadataPath, "oauth", 204},
		{"post", http.MethodPost, "https://vector.example" + mcptransport.MetadataPath, "oauth", 405},
		{"head", http.MethodHead, "https://vector.example" + mcptransport.MetadataPath, "oauth", 405},
		{"query", http.MethodGet, "https://vector.example" + mcptransport.MetadataPath + "?token=secret", "oauth", 400},
		{"empty query", http.MethodGet, "https://vector.example" + mcptransport.MetadataPath + "?", "oauth", 400},
		{"host", http.MethodGet, "https://foreign.example" + mcptransport.MetadataPath, "oauth", 400},
		{"subpath", http.MethodGet, "https://vector.example" + mcptransport.MetadataPath + "/extra", "oauth", 404},
		{"header only", http.MethodGet, "https://vector.example" + mcptransport.MetadataPath, "header", 404},
	} {
		t.Run(test.name, func(t *testing.T) {
			opts := options()
			opts.OAuthAuth = func(context.Context, *http.Request) (context.Context, error) {
				t.Fatal("public metadata attempted authentication")
				return nil, nil
			}
			h := handler(t, config(test.mode), opts)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(test.method, test.target, nil))
			require.Equal(t, test.want, w.Code)
			require.NotContains(t, w.Body.String(), "secret")
		})
	}
	disabled := handler(t, mcptransport.DefaultConfig(), mcptransport.Options{})
	require.Equal(t, 404, call(disabled, mcptransport.MetadataPath, "", nil).Code)
}

func TestOAuthChallengesDistinguishCredentialsScopesAndDomainDenial(t *testing.T) {
	for _, test := range []struct {
		name, token string
		err         error
		status      int
		challenge   string
	}{
		{"missing", "", mcptransport.ErrInvalidCredential, 401, ""},
		{"invalid", "Bearer opaque", mcptransport.ErrInvalidCredential, 401, "invalid_token"},
		{"scope", "Bearer opaque", mcptransport.ErrInsufficientScope, 403, "insufficient_scope"},
		{"domain", "Bearer opaque", mcptransport.ErrAccessDenied, 403, "none"},
		{"outage", "Bearer opaque", errors.New("issuer unavailable"), 503, "none"},
		{"malformed", "Bearer one, two", mcptransport.ErrInvalidRequest, 400, "none"},
	} {
		t.Run(test.name, func(t *testing.T) {
			opts := options()
			opts.OAuthAuth = func(context.Context, *http.Request) (context.Context, error) { return nil, test.err }
			opts.HeaderAuth = opts.OAuthAuth
			h := handler(t, config("both"), opts)
			headers := http.Header{}
			if test.token != "" {
				headers.Set("Authorization", test.token)
			}
			w := call(h, "/mcp", "", headers)
			require.Equal(t, test.status, w.Code)
			challenge := w.Header().Get("WWW-Authenticate")
			if test.challenge == "none" {
				require.Empty(t, challenge)
			} else {
				require.Contains(t, challenge, `resource_metadata="https://vector.example/.well-known/oauth-protected-resource/mcp"`)
				require.Contains(t, challenge, `scope="vector:read"`)
				if test.challenge == "" {
					require.NotContains(t, challenge, "error=")
				} else {
					require.Contains(t, challenge, `error="`+test.challenge+`"`)
				}
			}
			require.Empty(t, call(h, "/mcp/api-key", "", headers).Header().Get("WWW-Authenticate"))
		})
	}
}
