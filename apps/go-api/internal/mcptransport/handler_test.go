package mcptransport_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	loom "github.com/CaliLuke/loom/pkg"

	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/CaliLuke/go-plane/internal/platform"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func config(mode string) mcptransport.Config {
	c := mcptransport.DefaultConfig()
	c.Enabled, c.Mode, c.ResourceURL = true, mode, "https://vector.example/mcp"
	return c
}

func verified(ctx context.Context, _ *http.Request) (context.Context, error) { return ctx, nil }

func options() mcptransport.Options {
	return mcptransport.Options{HeaderAuth: verified, OAuthAuth: verified, OAuthIssuer: "https://issuer.example"}
}

func handler(t *testing.T, c mcptransport.Config, opts mcptransport.Options) http.Handler {
	t.Helper()
	opts.Service = transportTestService{}
	h, err := mcptransport.New(c, opts, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) }))
	require.NoError(t, err)
	return h
}

type transportTestService struct{}

func (transportTestService) Member(context.Context, *api.MemberRequest) (*api.MemberResult, error) {
	return nil, errors.New("transport tests do not execute identity reads")
}

func call(h http.Handler, path, body string, headers http.Header) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "https://vector.example"+path, strings.NewReader(body))
	r.Header = headers.Clone()
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func initialize(version string) string {
	return `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"` + version + `","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`
}

func TestDisabledRoutesDoNotFallThroughToFrontend(t *testing.T) {
	h := handler(t, mcptransport.DefaultConfig(), mcptransport.Options{})
	for _, path := range []string{"/mcp", "/mcp/api-key", "/mcp/", "/mcp/unknown"} {
		t.Run(path, func(t *testing.T) { require.Equal(t, http.StatusNotFound, call(h, path, "", nil).Code) })
	}
	require.Equal(t, http.StatusTeapot, call(h, "/api/example", "", nil).Code)
}

func TestOnlySelectedAuthenticationRoutesAreMounted(t *testing.T) {
	for _, tc := range []struct{ mode, present, absent string }{
		{"header", "/mcp/api-key", "/mcp"}, {"oauth", "/mcp", "/mcp/api-key"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			h := handler(t, config(tc.mode), options())
			require.Equal(t, http.StatusOK, call(h, tc.present, initialize("2026-07-28"), nil).Code)
			require.Equal(t, http.StatusNotFound, call(h, tc.absent, initialize("2026-07-28"), nil).Code)
		})
	}
}

func TestGeneratedHandlerNegotiatesLegacyRevisionWithoutSessions(t *testing.T) {
	h := handler(t, config("both"), options())
	for _, path := range []string{"/mcp", "/mcp/api-key"} {
		t.Run(path, func(t *testing.T) {
			const version = "2025-11-25"
			w := call(h, path, initialize(version), nil)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.Empty(t, w.Header().Get("Mcp-Session-Id"))
			require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			var response struct {
				Result struct {
					ProtocolVersion string
					Capabilities    map[string]json.RawMessage
				}
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			require.Equal(t, version, response.Result.ProtocolVersion)
			require.Contains(t, response.Result.Capabilities, "tools")
			require.NotContains(t, response.Result.Capabilities, "prompts")
			require.NotContains(t, response.Result.Capabilities, "resources")
		})
	}
}

func TestGeneratedHandlerSupportsStatelessDiscovery(t *testing.T) {
	h := handler(t, config("both"), options())
	for _, path := range []string{"/mcp", "/mcp/api-key"} {
		t.Run(path, func(t *testing.T) {
			w := call(h, path, `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},"io.modelcontextprotocol/clientCapabilities":{}}}}`, http.Header{
				"Mcp-Protocol-Version": []string{"2026-07-28"}, "Mcp-Method": []string{"server/discover"},
			})
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			var response struct {
				Result struct {
					SupportedVersions []string
					CacheScope        string
					TTLMs             *int
				}
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			require.ElementsMatch(t, []string{"2026-07-28", "2025-11-25"}, response.Result.SupportedVersions)
			require.Equal(t, "private", response.Result.CacheScope)
			require.NotNil(t, response.Result.TTLMs)
			require.Zero(t, *response.Result.TTLMs)
			require.Empty(t, w.Header().Get("Mcp-Session-Id"))
		})
	}
}

func TestGeneratedHandlerRejectsUnsupportedProtocolHeader(t *testing.T) {
	h := handler(t, config("header"), options())
	w := call(h, "/mcp/api-key", `{"jsonrpc":"2.0","id":1,"method":"ping"}`, http.Header{"Mcp-Protocol-Version": []string{"2025-03-26"}})
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthenticationDoesNotFallBackAcrossRoutes(t *testing.T) {
	var calls atomic.Int64
	opts := mcptransport.Options{
		OAuthIssuer: "https://issuer.example",
		HeaderAuth: func(context.Context, *http.Request) (context.Context, error) {
			return nil, mcptransport.ErrInvalidCredential
		},
		OAuthAuth: func(ctx context.Context, _ *http.Request) (context.Context, error) { calls.Add(1); return ctx, nil },
	}
	h := handler(t, config("both"), opts)
	require.Equal(t, http.StatusUnauthorized, call(h, "/mcp/api-key", initialize("2026-07-28"), nil).Code)
	require.Zero(t, calls.Load())
	require.Equal(t, http.StatusOK, call(h, "/mcp", initialize("2026-07-28"), nil).Code)
	require.EqualValues(t, 1, calls.Load())
}

func TestAuthenticationRunsOnEveryRequestAndFailsClosed(t *testing.T) {
	var revoked atomic.Bool
	opts := options()
	opts.HeaderAuth = func(ctx context.Context, _ *http.Request) (context.Context, error) {
		if revoked.Load() {
			return nil, errors.New("private validator failure: secret")
		}
		return ctx, nil
	}
	h := handler(t, config("header"), opts)
	require.Equal(t, http.StatusOK, call(h, "/mcp/api-key", initialize("2026-07-28"), nil).Code)
	revoked.Store(true)
	w := call(h, "/mcp/api-key", `{"jsonrpc":"2.0","id":2,"method":"ping"}`, nil)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.NotContains(t, w.Body.String(), "secret")
}

func TestIngressRejectsUntrustedTargetsBeforeAuthentication(t *testing.T) {
	var calls atomic.Int64
	opts := options()
	opts.HeaderAuth = func(ctx context.Context, _ *http.Request) (context.Context, error) { calls.Add(1); return ctx, nil }
	h := handler(t, config("header"), opts)
	for _, origin := range []string{"https://evil.example", "http://vector.example", "null", ""} {
		t.Run("origin "+origin, func(t *testing.T) {
			w := call(h, "/mcp/api-key", initialize("2026-07-28"), http.Header{"Origin": []string{origin}})
			require.Equal(t, http.StatusForbidden, w.Code)
		})
	}
	require.Equal(t, http.StatusBadRequest, call(h, "/mcp/api-key?token=secret", "", nil).Code)
	r := httptest.NewRequest(http.MethodPost, "https://evil.example/mcp/api-key", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Zero(t, calls.Load())
}

func TestTrustedOriginUsesGeneratedOriginProtection(t *testing.T) {
	c := config("header")
	c.TrustedOrigins = []string{"https://agent.example"}
	h := handler(t, c, options())
	for _, origin := range []string{"https://vector.example", "https://agent.example"} {
		t.Run(origin, func(t *testing.T) {
			require.Equal(t, http.StatusOK, call(h, "/mcp/api-key", initialize("2026-07-28"), http.Header{"Origin": []string{origin}}).Code)
		})
	}
}

func TestRequestBodyBound(t *testing.T) {
	c := config("header")
	c.MaxRequestBytes = 32
	h := handler(t, c, options())
	require.Equal(t, http.StatusRequestEntityTooLarge, call(h, "/mcp/api-key", initialize("2026-07-28"), nil).Code)
}

func TestUnknownLengthRequestBodyIsAlsoBounded(t *testing.T) {
	c := config("header")
	c.MaxRequestBytes = 32
	h := handler(t, c, options())
	r := httptest.NewRequest(http.MethodPost, "https://vector.example/mcp/api-key", strings.NewReader(initialize("2025-11-25")))
	r.ContentLength = -1
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code, w.Body.String())
}

func TestMissingAuthenticatedContextFailsClosed(t *testing.T) {
	opts := options()
	opts.HeaderAuth = func(context.Context, *http.Request) (context.Context, error) { return nil, nil }
	h := handler(t, config("header"), opts)
	require.Equal(t, http.StatusServiceUnavailable, call(h, "/mcp/api-key", initialize("2025-11-25"), nil).Code)
}

func TestAuthenticationRateLimitHasSafeRetryAndNoOAuthChallenge(t *testing.T) {
	for _, tc := range []struct {
		wait time.Duration
		want string
	}{{0, "1"}, {1500 * time.Millisecond, "2"}} {
		opts := options()
		limited := &mcptransport.RateLimitError{RetryAfter: tc.wait}
		require.Equal(t, "MCP request rate exceeded", limited.Error())
		opts.HeaderAuth = func(context.Context, *http.Request) (context.Context, error) {
			return nil, limited
		}
		w := call(handler(t, config("header"), opts), "/mcp/api-key", initialize("2025-11-25"), nil)
		require.Equal(t, http.StatusTooManyRequests, w.Code)
		require.Equal(t, tc.want, w.Header().Get("Retry-After"))
		require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		require.Empty(t, w.Header().Get("WWW-Authenticate"))
		require.Equal(t, "rate limit exceeded\n", w.Body.String())
	}
}

func TestAuthenticationFailureRetainsRequestCorrelation(t *testing.T) {
	for _, incoming := range []string{"10000000-0000-4000-8000-000000000001", "private-untrusted-value", ""} {
		t.Run(incoming, func(t *testing.T) {
			var received string
			opts := options()
			opts.HeaderAuth = func(ctx context.Context, _ *http.Request) (context.Context, error) {
				received, _ = platform.RequestID(ctx)
				return nil, mcptransport.ErrInvalidCredential
			}
			w := call(handler(t, config("header"), opts), "/mcp/api-key", initialize("2025-11-25"), http.Header{
				"X-Request-Id": []string{incoming},
			})
			require.Equal(t, http.StatusUnauthorized, w.Code)
			_, err := uuid.Parse(received)
			require.NoError(t, err)
			require.Equal(t, received, w.Header().Get(platform.RequestIDHeader))
			if incoming == "10000000-0000-4000-8000-000000000001" {
				require.Equal(t, incoming, received)
			}
			require.NotContains(t, w.Header().Get(platform.RequestIDHeader), "private-untrusted-value")
			require.NotContains(t, w.Body.String(), "private-untrusted-value")
		})
	}
}

func TestAuthenticationFailureClassifications(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{"malformed request", mcptransport.ErrInvalidRequest, http.StatusBadRequest},
		{"invalid credential", mcptransport.ErrInvalidCredential, http.StatusUnauthorized},
		{"denied membership", mcptransport.ErrAccessDenied, http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := options()
			opts.HeaderAuth = func(context.Context, *http.Request) (context.Context, error) { return nil, tc.err }
			h := handler(t, config("header"), opts)
			require.Equal(t, tc.status, call(h, "/mcp/api-key", initialize("2025-11-25"), nil).Code)
		})
	}
}

func TestDuplicateOriginIsRejected(t *testing.T) {
	h := handler(t, config("header"), options())
	w := call(h, "/mcp/api-key", initialize("2025-11-25"), http.Header{"Origin": []string{"https://vector.example", "https://vector.example"}})
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequestTimeoutCancelsAuthentication(t *testing.T) {
	c := config("header")
	c.RequestTimeout = 10 * time.Millisecond
	cancelled := make(chan struct{})
	opts := options()
	opts.HeaderAuth = func(ctx context.Context, _ *http.Request) (context.Context, error) {
		<-ctx.Done()
		close(cancelled)
		return nil, ctx.Err()
	}
	w := call(handler(t, c, opts), "/mcp/api-key", initialize("2026-07-28"), nil)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("authentication did not receive cancellation")
	}
}

func TestSelectedAuthenticationMustBeAvailable(t *testing.T) {
	for _, mode := range []string{"header", "oauth", "both"} {
		t.Run(mode, func(t *testing.T) {
			_, err := mcptransport.New(config(mode), mcptransport.Options{}, http.NotFoundHandler())
			require.Error(t, err)
		})
	}
}

func TestEnabledTransportRequiresApplicationService(t *testing.T) {
	_, err := mcptransport.New(config("header"), options(), http.NotFoundHandler())
	require.ErrorContains(t, err, "application service is required")
}

func (transportTestService) Project(context.Context, *api.ProjectRequest) (loom.Nullable[api.ProjectResult], error) {
	return loom.Nullable[api.ProjectResult]{}, errors.New("this fixture does not execute project reads")
}

func (transportTestService) DiscoverActions(context.Context) (mcptransport.ActionVisibility, error) {
	return mcptransport.ActionVisibility{"member": {"me": true, "list_workspace": true, "list_project": true, "list_roles": true, "retrieve_role": true}, "project": {"list": true, "retrieve": true, "archive": true, "unarchive": true}}, nil
}
