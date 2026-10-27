package main

import (
	"context"
	"os"

	"github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/CaliLuke/loom/clue/log"
)

func mcpOAuthConfigFromEnvironment(resource string) mcpoauth.Config {
	return mcpoauth.Config{
		Resource:                  resource,
		Issuer:                    os.Getenv("MCP_OAUTH_ISSUER_URL"),
		IntrospectionURL:          os.Getenv("MCP_OAUTH_INTROSPECTION_URL"),
		IntrospectionClientID:     os.Getenv("MCP_OAUTH_INTROSPECTION_CLIENT_ID"),
		IntrospectionClientSecret: os.Getenv("MCP_OAUTH_INTROSPECTION_CLIENT_SECRET"),
		LinkClientID:              os.Getenv("MCP_OAUTH_LINK_CLIENT_ID"),
		LinkClientSecret:          os.Getenv("MCP_OAUTH_LINK_CLIENT_SECRET"),
	}
}

// The caller stops this worker before closing the primary database pool,
// including when application startup fails after linking is configured.
func startMCPLinkCleanup(ctx context.Context, links *mcpoauth.Links) func() {
	worker, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		links.RunCleanup(worker, func(err error) { log.Error(worker, err) })
	}()
	return func() { cancel(); <-done }
}
