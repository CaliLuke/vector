//go:build integration

package proxy_test

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestMCPRoutesReachGoInEveryTopology(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	other := newNamedUpstream(t, "must-not-receive-mcp")
	goAPI := newNamedUpstream(t, "go")
	for _, config := range []string{"../../../proxy/Caddyfile.ce", "../../../proxy/Caddyfile.aio.ce", "../../../proxy/Caddyfile.local"} {
		t.Run(config, func(t *testing.T) {
			proxyURL := startCaddyWithConfig(t, ctx, config, other.port, goAPI.port, nil)
			for _, path := range []string{"/mcp", "/mcp/api-key", "/mcp/", "/mcp/unknown", "/.well-known/oauth-protected-resource/mcp", "/.well-known/oauth-protected-resource/mcp/unknown"} {
				for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodOptions} {
					assertTarget(t, method, proxyURL+path, "go")
				}
				assertGoProxyResponse(t, http.MethodHead, proxyURL+path)
			}
		})
	}
}
