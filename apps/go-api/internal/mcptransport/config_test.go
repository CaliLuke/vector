package mcptransport_test

import (
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/stretchr/testify/require"
)

func TestEnabledConfigurationRequiresCanonicalHTTPSResource(t *testing.T) {
	for _, resource := range []string{"", "http://vector.example/mcp", "https://vector.example/", "https://vector.example/mcp/", "https://user:secret@vector.example/mcp", "https://vector.example/mcp?token=secret", "https://vector.example/mcp?", "https://vector.example/mcp#fragment", "https://vector.example./mcp"} {
		t.Run(resource, func(t *testing.T) {
			c := config("header")
			c.ResourceURL = resource
			require.Error(t, c.Validate())
		})
	}
}

func TestEnabledConfigurationRequiresModeAndBounds(t *testing.T) {
	for _, mode := range []string{"", "invalid", "HEADER"} {
		t.Run("mode "+mode, func(t *testing.T) { require.Error(t, config(mode).Validate()) })
	}
	for _, limit := range []int64{-1, 0, 16<<20 + 1} {
		c := config("header")
		c.MaxRequestBytes = limit
		require.Error(t, c.Validate())
	}
	for _, timeout := range []time.Duration{-1, 0, 5*time.Minute + 1} {
		c := config("header")
		c.RequestTimeout = timeout
		require.Error(t, c.Validate())
	}
}

func TestTrustedOriginsRejectPathsAndNonHTTPS(t *testing.T) {
	for _, origin := range []string{"*", "http://agent.example", "https://agent.example/", "https://agent.example/path", "https://agent.example?token=secret"} {
		t.Run(origin, func(t *testing.T) {
			c := config("both")
			c.TrustedOrigins = []string{origin}
			require.Error(t, c.Validate())
		})
	}
}

func TestDisabledConfigurationNeedsNoCredentialsOrResource(t *testing.T) {
	require.NoError(t, mcptransport.DefaultConfig().Validate())
	for _, mode := range []string{"header", "oauth", "both"} {
		t.Run(mode, func(t *testing.T) { require.NoError(t, config(mode).Validate()) })
	}
}
