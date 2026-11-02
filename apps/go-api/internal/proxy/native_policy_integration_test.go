//go:build integration

package proxy_test

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/routing"
	"github.com/stretchr/testify/require"
)

// Forwarding is shared by all API families. Each shipping topology must preserve
// resource method headers while leaving authentication and CORS responses intact.
func TestNativeAPIProxyAndSharedResourceHeaders(t *testing.T) {
	policy, err := routing.Load()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	backend := newNamedUpstream(t, "go")
	parameter := regexp.MustCompile(`\{[^}]+\}`)
	for _, config := range []string{"../../../proxy/Caddyfile.ce", "../../../proxy/Caddyfile.aio.ce", "../../../proxy/Caddyfile.local"} {
		t.Run(config, func(t *testing.T) {
			proxyURL := startCaddyWithConfig(t, ctx, config, backend.port)
			for _, resource := range policy.Resources {
				path := parameter.ReplaceAllString(resource.Path, "11111111-1111-4111-8111-111111111111")
				assertGoResponseWithAllow(t, http.MethodOptions, proxyURL+path, strings.Join(resource.Methods, ", "))
				if len(resource.UUIDParameters) > 0 {
					literalPath := strings.Replace(resource.Path, "{"+resource.UUIDParameters[0]+"}", "leave", 1)
					literalPath = parameter.ReplaceAllString(literalPath, "11111111-1111-4111-8111-111111111111")
					assertGoResponseWithAllow(t, http.MethodOptions, proxyURL+literalPath, "partial methods")
				}
			}
			for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE"} {
				for _, path := range []string{"/api/new-native-feature/", "/api/v1/new-native-feature", "/auth/sign-in/", "/api/public/anchor/board/issues/item/votes/"} {
					assertGoProxyResponse(t, method, proxyURL+path)
				}
			}
			assertGoPreflightWithoutAllow(t, proxyURL+"/api/users/me/", map[string]string{
				"Origin": "http://localhost:8000", "Access-Control-Request-Method": "GET",
			})
			for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
				response := proxyRequestWithRetry(t, http.MethodGet, proxyURL+"/api/users/me/", map[string]string{"X-Test-Emit-Status": strconv.Itoa(status)})
				require.NoError(t, response.Body.Close())
				require.Equal(t, status, response.StatusCode)
				require.Empty(t, response.Header.Get("Allow"), "authorization failures must not gain method headers")
			}
			response := proxyRequestWithRetry(t, http.MethodGet, proxyURL+"/static/missing.css", nil)
			require.NoError(t, response.Body.Close())
			require.Equal(t, http.StatusNotFound, response.StatusCode)
			require.Empty(t, response.Header.Get("X-Test-Upstream"), "static paths must stay outside the API backend")
		})
	}
}
