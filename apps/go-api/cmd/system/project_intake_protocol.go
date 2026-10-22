package main

import (
	_ "embed"
	"net/http"
)

// Captured from the pinned Django IntakeSerializer's authenticated OPTIONS
// response. Both legacy collection aliases use this serializer.
//
//go:embed project_intake_writes_options_metadata.json
var projectIntakeWritesOptionsMetadata []byte

func projectMemberPreferencesProtocol() readServiceProtocol {
	protocol := newSessionServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/preferences/member/{member_id}/", "GET, PATCH, HEAD, OPTIONS", "Project Member Preference Endpoint", true)
	protocol.methodAllow = map[string]string{http.MethodPatch: "PATCH, OPTIONS", http.MethodOptions: "PATCH, OPTIONS"}
	return protocol
}

// Reads were retired separately; the retained Django URLconf binds only writes.
// Preserve each handler's Allow contract when they share the production path.
func newIntakeMixedProtocol(pattern, allow, name, action string, derivedHead bool) readServiceProtocol {
	protocol := newReadServiceProtocol(pattern, allow, name, action, derivedHead)
	protocol.authenticateMethod = true
	if action == http.MethodPost {
		protocol.methodAllow = map[string]string{http.MethodPost: "POST, OPTIONS"}
		protocol.optionsMetadata = projectIntakeWritesOptionsMetadata
	} else {
		protocol.methodAllow = map[string]string{http.MethodPatch: "PATCH, DELETE, OPTIONS", http.MethodDelete: "PATCH, DELETE, OPTIONS"}
	}
	return protocol
}

func isProjectIntakeWritesProtocol(pattern string) bool {
	switch pattern {
	case "/api/workspaces/{slug}/projects/{project_id}/inboxes/",
		"/api/workspaces/{slug}/projects/{project_id}/inboxes/{pk}/",
		"/api/workspaces/{slug}/projects/{project_id}/intakes/",
		"/api/workspaces/{slug}/projects/{project_id}/intakes/{pk}/":
		return true
	default:
		return false
	}
}
