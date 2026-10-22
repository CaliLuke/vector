package main

import "net/http"

func notificationRemainingDetailProtocol() readServiceProtocol {
	protocol := newSessionServiceProtocol("/api/workspaces/{slug}/users/notifications/{pk}/", "GET, PATCH, DELETE, HEAD, OPTIONS", "Notification", true)
	protocol.methodAllow = map[string]string{http.MethodPatch: "PATCH, OPTIONS", http.MethodOptions: "PATCH, OPTIONS"}
	return protocol
}
