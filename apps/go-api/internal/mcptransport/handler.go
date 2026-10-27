package mcptransport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/platform"
	generated "github.com/CaliLuke/go-plane/mcp/gen/mcp_vector_mcp"
	"github.com/CaliLuke/loom-mcp/v2/runtime/mcp/sdkbridge"
	"github.com/CaliLuke/loom/observability/transport"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Authentication providers return these classifications without exposing their
// internal errors. Unclassified failures produce a safe 503.
var (
	ErrInvalidRequest    = errors.New("invalid MCP authentication request")
	ErrInvalidCredential = errors.New("invalid MCP credential")
	ErrAccessDenied      = errors.New("MCP access denied")
)

// Authenticate verifies the live credential and installs a verified actor in
// context. Implementations own their route's credential syntax, account status,
// workspace membership and OAuth scopes. They must not fall back to another mode.
type Authenticate func(context.Context, *http.Request) (context.Context, error)

type Options struct {
	HeaderAuth Authenticate
	OAuthAuth  Authenticate
	Observer   transport.Observer
}

// ValidateStartup rejects unavailable authentication before starting databases,
// workers or listeners. Constructors repeat this check at the mount boundary.
func ValidateStartup(config Config, options Options) error {
	if err := config.Validate(); err != nil {
		return err
	}
	if !config.Enabled {
		return nil
	}
	if (config.Mode == "header" || config.Mode == "both") && options.HeaderAuth == nil {
		return fmt.Errorf("MCP header authentication is not available")
	}
	if (config.Mode == "oauth" || config.Mode == "both") && options.OAuthAuth == nil {
		return fmt.Errorf("MCP OAuth authentication is not available")
	}
	return nil
}

// New constructs the generated, stateless server. Its initial catalogue is empty
// until action implementations and their authorization contracts are available.
// A missing selected authentication provider is a startup error, never anonymous
// access. Disabled routes and paths beneath /mcp are always 404, not SPA pages.
func New(config Config, options Options, fallback http.Handler) (http.Handler, error) {
	if err := ValidateStartup(config, options); err != nil {
		return nil, err
	}
	if fallback == nil {
		return nil, fmt.Errorf("MCP fallback handler is required")
	}
	routes := make(map[string]http.Handler)
	if config.Enabled {
		server, err := generated.NewSDKServer(struct{}{}, &generated.SDKServerOptions{
			Server: &mcpsdk.ServerOptions{
				SupportedProtocolVersions: []string{"2026-07-28", "2025-11-25"},
				SetCacheable: func(_ context.Context, _ mcpsdk.Request, result *mcpsdk.Cacheable) {
					result.TTLMs, result.CacheScope = 0, "private"
				},
			},
			TransportObserver: options.Observer,
			OriginProtection:  &sdkbridge.OriginProtection{TrustedOrigins: slices.Clone(config.TrustedOrigins)},
			StreamableHTTP: &sdkbridge.StreamableHTTPOptions{
				Stateless: true, JSONResponse: true, MaxRequestBodyBytes: config.MaxRequestBytes,
				PropagateRequestCancellation: true,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("construct MCP server: %w", err)
		}
		if config.Mode == "header" || config.Mode == "both" {
			routes[HeaderPath] = platform.RequestIDMiddleware(protect(config, options.HeaderAuth, server.Handler))
		}
		if config.Mode == "oauth" || config.Mode == "both" {
			routes[OAuthPath] = platform.RequestIDMiddleware(protect(config, options.OAuthAuth, server.Handler))
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler := routes[r.URL.Path]; handler != nil {
			handler.ServeHTTP(w, r)
		} else if r.URL.Path == OAuthPath || strings.HasPrefix(r.URL.Path, OAuthPath+"/") {
			http.NotFound(w, r)
		} else {
			fallback.ServeHTTP(w, r)
		}
	}), nil
}

func protect(config Config, authenticate Authenticate, next http.Handler) http.Handler {
	resource, _ := url.Parse(config.ResourceURL) // Validated before construction.
	origins := append(slices.Clone(config.TrustedOrigins), "https://"+resource.Host)
	bounded := http.TimeoutHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := authenticate(r.Context(), r)
		var limited *RateLimitError
		switch {
		case errors.As(err, &limited):
			seconds := max(int64((limited.RetryAfter+time.Second-1)/time.Second), 1)
			w.Header().Set("Retry-After", strconv.FormatInt(seconds, 10))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		case errors.Is(err, ErrInvalidRequest):
			http.Error(w, "invalid_request", http.StatusBadRequest)
			return
		case errors.Is(err, ErrInvalidCredential):
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		case errors.Is(err, ErrAccessDenied):
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		case err != nil || ctx == nil:
			http.Error(w, "authentication unavailable", http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	}), config.RequestTimeout, "request timed out")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w = noStoreWriter{w}
		w.Header().Set("Cache-Control", "no-store")
		if r.Host != resource.Host || r.URL.RawQuery != "" || r.URL.ForceQuery {
			http.Error(w, "invalid MCP request target", http.StatusBadRequest)
			return
		}
		origin := r.Header.Values("Origin")
		if len(origin) > 1 || len(origin) == 1 && !slices.Contains(origins, origin[0]) {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
		if r.ContentLength > config.MaxRequestBytes {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, config.MaxRequestBytes)
		bounded.ServeHTTP(w, r)
	})
}

// Enforce the application's private response policy when the transport commits
// headers, including when TimeoutHandler copies its buffered response headers.
type noStoreWriter struct{ http.ResponseWriter }

func (w noStoreWriter) WriteHeader(status int) {
	w.Header().Set("Cache-Control", "no-store")
	w.ResponseWriter.WriteHeader(status)
}

func (w noStoreWriter) Write(body []byte) (int, error) {
	w.Header().Set("Cache-Control", "no-store")
	return w.ResponseWriter.Write(body)
}

func (w noStoreWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
