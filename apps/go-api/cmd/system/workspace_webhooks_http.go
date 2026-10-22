package main

import "net/http"

func newWebhookDetailProtocol() readServiceProtocol {
	protocol := newSessionServiceProtocol("/api/workspaces/{slug}/webhooks/{pk}/", "GET, POST, PATCH, DELETE, HEAD, OPTIONS", "Webhook Endpoint", true)
	// The source advertises POST through the shared collection view, but its
	// handler cannot accept pk. Reject it after the same authentication and role checks.
	protocol.protocolOnlyMethods = []string{http.MethodPost}
	return protocol
}
