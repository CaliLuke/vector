package main

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func clearMCPEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{"MCP_ENABLED", "MCP_AUTH_MODE", "MCP_RESOURCE_URL", "MCP_TRUSTED_ORIGINS", "MCP_MAX_REQUEST_BYTES", "MCP_REQUEST_TIMEOUT"} {
		t.Setenv(key, "")
	}
}

func TestMCPEnvironmentDisabledByDefault(t *testing.T) {
	clearMCPEnvironment(t)
	cfg, err := mcpConfigFromEnvironment()
	require.NoError(t, err)
	require.False(t, cfg.Enabled)
	require.Empty(t, cfg.Mode)
	require.Empty(t, cfg.ResourceURL)
}

func TestMCPEnvironmentSelectsHeaderWithoutOAuthConfiguration(t *testing.T) {
	clearMCPEnvironment(t)
	t.Setenv("MCP_ENABLED", "true")
	t.Setenv("MCP_AUTH_MODE", "header")
	t.Setenv("MCP_RESOURCE_URL", "https://vector.example/mcp")
	t.Setenv("MCP_TRUSTED_ORIGINS", "https://agent.example, https://other.example")
	t.Setenv("MCP_MAX_REQUEST_BYTES", "2048")
	t.Setenv("MCP_REQUEST_TIMEOUT", "15s")
	cfg, err := mcpConfigFromEnvironment()
	require.NoError(t, err)
	require.True(t, cfg.Enabled)
	require.Equal(t, "header", cfg.Mode)
	require.Equal(t, []string{"https://agent.example", "https://other.example"}, cfg.TrustedOrigins)
	require.EqualValues(t, 2048, cfg.MaxRequestBytes)
	require.Equal(t, 15*time.Second, cfg.RequestTimeout)
}

func TestMCPEnvironmentRejectsInvalidSettings(t *testing.T) {
	for _, key := range []string{"MCP_ENABLED", "MCP_MAX_REQUEST_BYTES", "MCP_REQUEST_TIMEOUT"} {
		t.Run(key, func(t *testing.T) {
			clearMCPEnvironment(t)
			t.Setenv(key, "invalid-private-value")
			_, err := mcpConfigFromEnvironment()
			require.Error(t, err)
			require.NotContains(t, err.Error(), "invalid-private-value")
		})
	}
}

func TestMCPEnvironmentCannotEnableWithoutSelectedMode(t *testing.T) {
	clearMCPEnvironment(t)
	t.Setenv("MCP_ENABLED", "true")
	_, err := mcpConfigFromEnvironment()
	require.Error(t, err)
}

func TestMCPUnavailableAuthenticationFailsBeforeDatabaseStartup(t *testing.T) {
	clearMCPEnvironment(t)
	t.Setenv("MCP_ENABLED", "true")
	t.Setenv("MCP_AUTH_MODE", "header")
	t.Setenv("MCP_RESOURCE_URL", "https://vector.example/mcp")
	t.Setenv("DATABASE_URL", "not-a-database-url")
	err := run(context.Background(), []string{"--django-secret-key", "test-only-secret"}, io.Discard)
	require.ErrorContains(t, err, "MCP header authentication is not available")
}
