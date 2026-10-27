package main

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/stretchr/testify/require"
)

func clearMCPEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{"MCP_ENABLED", "MCP_AUTH_MODE", "MCP_RESOURCE_URL", "MCP_TRUSTED_ORIGINS", "MCP_MAX_REQUEST_BYTES", "MCP_REQUEST_TIMEOUT", "MCP_OAUTH_ISSUER_URL", "MCP_OAUTH_INTROSPECTION_URL", "MCP_OAUTH_INTROSPECTION_CLIENT_ID", "MCP_OAUTH_INTROSPECTION_CLIENT_SECRET", "MCP_OAUTH_LINK_CLIENT_ID", "MCP_OAUTH_LINK_CLIENT_SECRET"} {
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
	t.Setenv("MCP_AUTH_MODE", "oauth")
	t.Setenv("MCP_RESOURCE_URL", "https://vector.example/mcp")
	t.Setenv("DATABASE_URL", "not-a-database-url")
	err := run(context.Background(), []string{"--django-secret-key", "test-only-secret"}, io.Discard)
	require.ErrorContains(t, err, "MCP OAuth configuration is invalid")
}

func TestMCPPersonalTokensAvailableWithoutOAuth(t *testing.T) {
	clearMCPEnvironment(t)
	t.Setenv("MCP_ENABLED", "true")
	t.Setenv("MCP_AUTH_MODE", "header")
	t.Setenv("MCP_RESOURCE_URL", "https://vector.example/mcp")
	cfg, err := mcpConfigFromEnvironment()
	require.NoError(t, err)
	require.NoError(t, validateMCPAvailability(cfg, mcpoauth.Config{}))
	cfg.Mode = "both"
	require.ErrorContains(t, validateMCPAvailability(cfg, mcpoauth.Config{}), "OAuth configuration is invalid")
}

func TestMCPOAuthEnvironmentAcceptsExplicitConfidentialClients(t *testing.T) {
	clearMCPEnvironment(t)
	for key, value := range map[string]string{
		"MCP_OAUTH_ISSUER_URL":                  "https://issuer.example/tenant",
		"MCP_OAUTH_INTROSPECTION_URL":           "https://issuer.example/introspect",
		"MCP_OAUTH_INTROSPECTION_CLIENT_ID":     "introspection-client",
		"MCP_OAUTH_INTROSPECTION_CLIENT_SECRET": "test-introspection-secret",
		"MCP_OAUTH_LINK_CLIENT_ID":              "link-client",
		"MCP_OAUTH_LINK_CLIENT_SECRET":          "test-link-secret",
	} {
		t.Setenv(key, value)
	}
	oauth := mcpOAuthConfigFromEnvironment("https://vector.example/mcp")
	require.NoError(t, oauth.Validate())
	require.Equal(t, "https://vector.example/mcp", oauth.Resource)
	require.Equal(t, "https://issuer.example/tenant", oauth.Issuer)
	require.NotEqual(t, oauth.IntrospectionClientID, oauth.LinkClientID)
	for _, mode := range []string{"oauth", "both"} {
		t.Setenv("MCP_ENABLED", "true")
		t.Setenv("MCP_AUTH_MODE", mode)
		t.Setenv("MCP_RESOURCE_URL", oauth.Resource)
		cfg, err := mcpConfigFromEnvironment()
		require.NoError(t, err)
		require.NoError(t, validateMCPAvailability(cfg, oauth))
	}
}
